"""Runtime replay check against a real MariaDB server."""

import pymysql
import pytest
from alembic import command
from sqlalchemy import create_engine

from classifier import classify as C
from classifier.render import make_config
from classifier.replay import ReplayGuard, StatementNotAllowed
from conftest import make_revision

A, B = "aaaaaaaaaaaa", "bbbbbbbbbbbb"
CREATE = 'op.execute("CREATE TABLE t (id BINARY(16) NOT NULL, PRIMARY KEY (id));")'
ADD_COL = 'op.execute("ALTER TABLE t ADD COLUMN c1 VARCHAR(5) NULL DEFAULT \'50%\'")'


def connect(server, database):
    return pymysql.connect(host=server.host, port=server.port or 3306, user=server.username, password=server.password or "", database=database, autocommit=True)


@pytest.fixture
def scratch(make_stream, fresh_db, db_server):
    name, url = fresh_db()
    stream = make_stream(
        {A + "_x.py": make_revision(A, None, CREATE), B + "_x.py": make_revision(B, A, ADD_COL)},
        schema=name,
    )
    cfg = make_config(stream.root, stream.script_dir, stream.schema, url=url)
    plans = {p.revision: p for p in C.classify_range(stream, None, exceptions=set()).plans}
    return stream, cfg, plans, url, name, db_server


def tables(server, name):
    c = connect(server, name)
    try:
        cur = c.cursor()
        cur.execute("SHOW TABLES")
        return sorted(r[0] for r in cur.fetchall())
    finally:
        c.close()


def test_classified_steps_run_through_the_guard(scratch):
    stream, cfg, plans, url, name, server = scratch
    for rev in (A, B):
        guard = ReplayGuard(plans[rev].statements, stream.schema)
        guard.install()
        try:
            command.upgrade(cfg, rev)
        finally:
            guard.remove()
        assert guard.remaining == 0
    assert tables(server, name) == ["alembic_version", "t"]


def test_statement_that_differs_from_the_plan_is_not_sent(scratch):
    stream, cfg, plans, url, name, server = scratch
    tampered = list(plans[A].statements)
    tampered[1] = tampered[1].replace("BINARY(16)", "BINARY(17)")
    guard = ReplayGuard(tampered, stream.schema)
    guard.install()
    try:
        with pytest.raises(StatementNotAllowed):
            command.upgrade(cfg, A)
    finally:
        guard.remove()
    assert "t" not in tables(server, name)


def test_extra_statement_is_rejected_before_execution(scratch):
    stream, cfg, plans, url, name, server = scratch
    command.upgrade(cfg, A)
    engine = create_engine(url)
    guard = ReplayGuard(plans[B].statements, stream.schema)
    guard.install()
    try:
        with engine.connect() as conn:
            for sql in ["DROP TABLE t", "ALTER TABLE t ADD COLUMN x INT NULL", "SELECT 1", "TRUNCATE t"]:
                with pytest.raises(StatementNotAllowed):
                    conn.exec_driver_sql(sql)
    finally:
        guard.remove()
        engine.dispose()
    assert tables(server, name) == ["alembic_version", "t"]


def test_statements_must_arrive_in_order(scratch):
    stream, cfg, plans, url, name, server = scratch
    guard = ReplayGuard(plans[A].statements, stream.schema)
    with pytest.raises(StatementNotAllowed):
        guard.check(plans[A].statements[2])


def test_forged_bookkeeping_is_rejected(scratch):
    stream, cfg, plans, url, name, server = scratch
    guard = ReplayGuard(plans[B].statements, stream.schema)
    with pytest.raises(StatementNotAllowed):
        guard.check("UPDATE alembic_version SET version_num='%s' WHERE alembic_version.version_num = '%s'" % ("cccccccccccc", A))
    with pytest.raises(StatementNotAllowed):
        guard.check("INSERT INTO alembic_version (version_num) VALUES ('x')")
    with pytest.raises(StatementNotAllowed):
        guard.check("DELETE FROM alembic_version")


def test_read_only_alembic_statements_are_anchored(scratch):
    stream, cfg, plans, url, name, server = scratch
    guard = ReplayGuard(plans[B].statements, stream.schema)
    table_check = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = %(table_schema)s AND table_name = %(table_name)s"
    guard.check(table_check, {"table_schema": stream.schema, "table_name": "alembic_version"})
    guard.check("SELECT alembic_version.version_num \nFROM alembic_version", {})
    for params in [{"table_schema": "mysql", "table_name": "alembic_version"}, {"table_schema": stream.schema, "table_name": "user"}, {}, None]:
        with pytest.raises(StatementNotAllowed):
            guard.check(table_check, params)
    with pytest.raises(StatementNotAllowed):
        guard.check("SELECT alembic_version.version_num FROM alembic_version WHERE 1=1", {})
    with pytest.raises(StatementNotAllowed):
        guard.check("SELECT alembic_version.version_num FROM alembic_version", {"a": 1})
    assert guard.remaining == len(plans[B].statements)


def test_executemany_and_exotic_text_are_rejected(scratch):
    stream, cfg, plans, url, name, server = scratch
    guard = ReplayGuard(plans[A].statements, stream.schema)
    with pytest.raises(StatementNotAllowed):
        guard(None, None, plans[A].statements[0], [], None, True)
    for text in ["ALTER TABLE t\x0bADD COLUMN x INT NULL", "ALTER TABLE t ADD COLUMN x INT NULL /*M! , drop column b */", "x\u00a0"]:
        with pytest.raises(StatementNotAllowed):
            guard.check(text)


def test_percent_default_survives_the_round_trip(scratch):
    stream, cfg, plans, url, name, server = scratch
    assert any("'50%'" in s for s in plans[B].statements)
    for rev in (A, B):
        guard = ReplayGuard(plans[rev].statements, stream.schema)
        guard.install()
        try:
            command.upgrade(cfg, rev)
        finally:
            guard.remove()
        assert guard.remaining == 0
