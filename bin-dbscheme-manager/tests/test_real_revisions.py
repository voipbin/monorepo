"""Checks that run over the real revision files of both streams.

Assertions are written as plain booleans or sets of ids so that a failure message never prints counts.
"""

import os
import shutil

import pymysql
import pytest

from alembic import command
from classifier import classify as C
from classifier import source_gate
from classifier.render import make_config
from classifier.classify import default_streams, load_exceptions, sha256_file
from conftest import SERVICE_DIR
import chainutil


@pytest.fixture(scope="module")
def tree(tmp_path_factory):
    """A pristine copy of the service tree (no bytecode directories left by other tools)."""
    dst = tmp_path_factory.mktemp("service") / "bin-dbscheme-manager"
    shutil.copytree(
        SERVICE_DIR,
        dst,
        ignore=shutil.ignore_patterns("__pycache__", "tests", "classifier", ".git", "alembic.ini", "*.pyc", ".pytest_cache"),
    )
    return str(dst)


@pytest.fixture(scope="module")
def streams(tree):
    return default_streams(tree)


def test_chain_scan_is_clean_with_the_exception_list(streams):
    exceptions = load_exceptions()
    for s in streams:
        assert C.chain_scan(s, exceptions) == {}, s.name


def test_without_the_exception_list_the_legacy_files_are_found(streams):
    assert any(C.chain_scan(s, set()) for s in streams)


def test_exception_list_agrees_with_the_files(streams):
    """Every entry matches a present file, and every module-level violator is listed."""
    entries = {}
    with open(C.EXCEPTIONS_FILE, encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if line and not line.startswith("#"):
                digest, name = line.split(None, 1)
                assert digest not in entries
                entries[digest] = name
    present = {}
    violators = set()
    for s in streams:
        for entry in os.scandir(s.versions_dir):
            data = open(entry.path, "rb").read()
            digest = sha256_file(data)
            present[digest] = "%s/%s" % (s.name, entry.name)
            viols = source_gate.check_module_level(data)
            if viols:
                assert all(v.exemptible for v in viols), entry.name
                violators.add(digest)
    assert set(entries) == violators
    for digest, name in entries.items():
        assert present[digest] == name


def test_every_revision_gets_exactly_one_outcome(streams, classified):
    for s in streams:
        res = C.classify_range(s, None)
        script_chain, plans = classified[s.name]
        assert res.klass == "needs_human"
        pending_matches = res.pending == len(script_chain)
        assert pending_matches, s.name
        all_ids = {r.revision for r in script_chain}
        assert set(res.blocked).isdisjoint(plans)
        assert set(res.blocked) | set(plans) == all_ids


def test_empty_range_is_na(streams):
    s = streams[0]
    head = chainutil.chain_of(s)[-1].revision
    assert C.classify_range(s, head).klass == "na"


def test_unknown_current_revision_is_an_error(streams):
    with pytest.raises(C.UnknownRevision):
        C.classify_range(streams[0], "doesnotexist")


@pytest.fixture(scope="module")
def classified(streams):
    out = {}
    for s in streams:
        chain = chainutil.chain_of(s)
        out[s.name] = (chain, chainutil.additive_plans(s, chain))
    return out


@pytest.fixture(scope="module")
def applied(streams, classified, db_server):
    """Apply each whole stream to a scratch database; per revision the captured statements and schema snapshots."""
    admin = pymysql.connect(host=db_server.host, port=db_server.port or 3306, user=db_server.username, password=db_server.password or "", autocommit=True)
    out = {}
    created = []
    try:
        for s in streams:
            name = "dbscheme_c_%s_%d" % (s.name, os.getpid())
            admin.cursor().execute("DROP DATABASE IF EXISTS `%s`" % name)
            admin.cursor().execute("CREATE DATABASE `%s`" % name)
            created.append(name)
            url = db_server.set(drivername="mariadb+pymysql", database=name, query={"charset": "utf8mb4"}).render_as_string(hide_password=False)
            cfg = make_config(s.root, s.script_dir, s.schema, url=url)
            probe = pymysql.connect(host=db_server.host, port=db_server.port or 3306, user=db_server.username, password=db_server.password or "", database=name, autocommit=True)
            try:
                out[s.name] = chainutil.apply_all(cfg, lambda: chainutil.schema_snapshot(probe, name))
            finally:
                probe.close()
    finally:
        for name in created:
            admin.cursor().execute("DROP DATABASE IF EXISTS `%s`" % name)
        admin.close()
    return out


def test_whole_chain_applies_online(applied, classified):
    for stream, steps in applied.items():
        chain, _ = classified[stream]
        missing = {r.revision for r in chain} - set(steps)
        assert not missing, stream


def test_rendered_statements_equal_executed_statements_over_the_chain(applied, classified):
    """Render and the captured online run are equal after normalize(), for every additive revision."""
    for stream, steps in applied.items():
        _, plans = classified[stream]
        assert plans
        for rev, plan in plans.items():
            assert plan.statements and plan.statements[-1].startswith(("UPDATE alembic_version", "INSERT INTO alembic_version")), (stream, rev)
            assert chainutil.body_statements(steps[rev][0]) == plan.statements, (stream, rev)


def test_additive_revisions_only_add(applied, classified):
    """Execute every statement classified additive and assert the schema diff only adds."""
    for stream, steps in applied.items():
        _, plans = classified[stream]
        for rev in plans:
            _, before, after = steps[rev]
            tables0, cols0, idx0 = before
            tables1, cols1, idx1 = after
            assert tables0 <= tables1, (stream, rev)
            for key, val in cols0.items():
                assert cols1.get(key) == val, (stream, rev, key)
            for key, val in idx0.items():
                assert idx1.get(key) == val, (stream, rev, key)
