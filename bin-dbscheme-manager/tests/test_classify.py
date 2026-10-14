import pytest

from classifier import classify as C
from classifier import source_gate as sg
from conftest import make_revision

A, B, C3, D = "aaaaaaaaaaaa", "bbbbbbbbbbbb", "cccccccccccc", "dddddddddddd"

CREATE = 'op.execute("CREATE TABLE t (id BINARY(16) NOT NULL, PRIMARY KEY (id));")'
ADD_COL = 'op.execute("ALTER TABLE t ADD COLUMN c1 VARCHAR(5) NULL")'
ADD_COL2 = 'op.execute("ALTER TABLE t ADD COLUMN c2 VARCHAR(5) NULL")'


def f(rev, down, body, **kw):
    return ("%s_x.py" % rev, make_revision(rev, down, body, **kw))


def chain(*items):
    return dict(items)


def classify(stream, from_rev=None, exceptions=None):
    return C.classify_range(stream, from_rev, exceptions=set() if exceptions is None else exceptions)


def test_additive_chain_yields_plans_in_order(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, ADD_COL)))
    res = classify(s)
    assert (res.klass, res.reason, res.blocked, res.pending) == ("additive", None, [], 2)
    assert [p.revision for p in res.plans] == [A, B]
    assert res.plans[0].statements[0].startswith("CREATE TABLE alembic_version(")
    assert res.plans[0].statements[1] == "CREATE TABLE t(id BINARY(16)NOT NULL,PRIMARY KEY(id))"
    assert res.plans[1].statements == [
        "ALTER TABLE t ADD COLUMN c1 VARCHAR(5)NULL",
        "UPDATE alembic_version SET version_num='%s' WHERE alembic_version.version_num = '%s'" % (B, A),
    ]


def test_pending_range_starts_after_the_current_revision(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, ADD_COL), f(C3, B, ADD_COL2)))
    res = classify(s, B)
    assert res.pending == 1 and [p.revision for p in res.plans] == [C3]
    assert classify(s, C3).klass == "na"


def test_empty_database_step_uses_insert_bookkeeping(make_stream):
    s = make_stream(chain(f(A, None, CREATE)))
    stmts = classify(s).plans[0].statements
    assert stmts[-1] == "INSERT INTO alembic_version(version_num)VALUES('%s')" % A


def test_unknown_current_revision(make_stream):
    s = make_stream(chain(f(A, None, CREATE)))
    with pytest.raises(C.UnknownRevision):
        classify(s, "zzzzzzzzzzzz")


def test_multiple_heads(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, ADD_COL), f(C3, A, ADD_COL2)))
    with pytest.raises(C.MultipleHeads) as exc:
        classify(s)
    assert exc.value.heads == sorted([B, C3])


def test_destructive_revision_blocks(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, 'op.execute("ALTER TABLE t DROP COLUMN c1")'), f(C3, B, ADD_COL2)))
    res = classify(s, A)
    assert (res.klass, res.reason, res.blocked, res.plans) == ("needs_human", "destructive", [B], [])


def test_data_changing_statements_block(make_stream):
    for sql in ["UPDATE t SET a = 1", "DELETE FROM t", "INSERT INTO t (a) VALUES (1)", "DROP TABLE t", "TRUNCATE TABLE t"]:
        s = make_stream(chain(f(A, None, CREATE), f(B, A, 'op.execute("%s")' % sql)))
        res = classify(s, A)
        assert (res.klass, res.reason) == ("needs_human", "destructive"), sql


def test_data_dependent_revision_blocks(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, "conn = op.get_bind()")))
    res = classify(s, A)
    assert (res.reason, res.blocked) == ("data_dependent", [B])


def test_not_renderable_revision_blocks(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, 'op.add_column("t", sa.Column("a", sa.Integer()), no_such_argument=1)')))
    res = classify(s, A)
    assert (res.reason, res.blocked) == ("not_renderable", [B])


def test_unparsable_revision_blocks(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, 'op.execute("ALTER TABLE t ADD COLUMN a INT NULL; ALTER TABLE t ADD COLUMN b INT NULL")')))
    assert classify(s, A).reason == "unparsable"
    s = make_stream(chain(f(A, None, CREATE), f(B, A, 'op.execute("ALTER TABLE t ADD COLUMN s serial")')))
    assert classify(s, A).reason == "unparsable"


def test_forged_bookkeeping_in_a_body_is_rejected(make_stream):
    forged = 'op.execute("UPDATE alembic_version SET version_num=\'%s\' WHERE alembic_version.version_num = \'%s\'")' % (B, A)
    s = make_stream(chain(f(A, None, CREATE), f(B, A, forged)))
    res = classify(s, A)
    assert res.klass == "needs_human" and res.blocked == [B]
    forged = 'op.execute("UPDATE alembic_version SET version_num=\'%s\' WHERE alembic_version.version_num = \'%s\'")' % (C3, A)
    s = make_stream(chain(f(A, None, CREATE), f(B, A, forged)))
    assert classify(s, A).klass == "needs_human"


def test_merge_revision_is_unparsable(make_stream):
    files = chain(f(A, None, CREATE), f(B, A, ADD_COL), f(C3, A, ADD_COL2))
    files["dddddddddddd_merge.py"] = make_revision(D, (B, C3)).replace("down_revision = ('%s', '%s')" % (B, C3), "down_revision = ('%s', '%s')" % (B, C3))
    s = make_stream(files)
    res = classify(s, A)
    assert (res.reason, res.blocked) == ("unparsable", [D])


def test_range_reason_priority(make_stream):
    bodies = {
        "data_dependent": "conn = op.get_bind()",
        "not_renderable": 'op.add_column("t", sa.Column("a", sa.Integer()), no_such_argument=1)',
        "unparsable": 'op.execute("ALTER TABLE t ADD COLUMN s serial")',
        "destructive": 'op.execute("ALTER TABLE t DROP COLUMN c")',
    }
    order = ["data_dependent", "not_renderable", "unparsable", "destructive"]
    revs = ["bbbbbbbbbbbb", "cccccccccccc", "dddddddddddd", "eeeeeeeeeeee"]
    files = [f(A, None, CREATE)]
    down = A
    # oldest is the lowest priority reason, so the order of the range cannot decide the result
    for name, rev in zip(reversed(order), revs):
        files.append(f(rev, down, bodies[name]))
        down = rev
    res = classify(make_stream(chain(*files)), A)
    assert res.reason == "data_dependent" and len(res.blocked) == 4
    # without the data_dependent one the next in order wins
    files = [f(A, None, CREATE)]
    down = A
    for name, rev in zip(reversed(order[1:]), revs):
        files.append(f(rev, down, bodies[name]))
        down = rev
    res = classify(make_stream(chain(*files)), A)
    assert res.reason == "not_renderable"


def test_blocked_lists_every_blocking_revision(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, 'op.execute("DROP TABLE t")'), f(C3, B, ADD_COL), f(D, C3, 'op.execute("DELETE FROM t")')))
    assert classify(s, A).blocked == [B, D]


# --- chain-wide scan -----------------------------------------------------------


def test_module_level_violation_blocks_even_when_nothing_is_pending(make_stream):
    s = make_stream(chain(f(A, None, CREATE), f(B, A, ADD_COL, extra="import os\n")))
    res = classify(s, B)
    assert (res.klass, res.reason, res.blocked, res.pending) == ("needs_human", "data_dependent", [B], None)


def test_module_level_call_does_not_run_during_classification(make_stream, tmp_path):
    marker = tmp_path / "ran"
    evil = "import os\nopen(%r, 'w').write('x')\n" % str(marker)
    s = make_stream(chain(f(A, None, CREATE), f(B, A, ADD_COL, extra=evil)))
    res = classify(s, A)
    assert res.klass == "needs_human" and not marker.exists()


def test_exception_list_exempts_only_module_level_rules(make_stream):
    files = chain(f(A, None, CREATE), f(B, A, ADD_COL, extra="import os\n"))
    s = make_stream(files)
    digest = C.sha256_file(files[B + "_x.py"].encode())
    assert classify(s, B, exceptions={digest}).klass == "na"
    # a changed hash is not covered
    assert classify(s, B, exceptions={"0" * 64}).klass == "needs_human"
    # the upgrade body rules still apply when the file is pending
    files = chain(f(A, None, CREATE), f(B, A, "conn = op.get_bind()", extra="import os\n"))
    s = make_stream(files)
    digest = C.sha256_file(files[B + "_x.py"].encode())
    res = classify(s, A, exceptions={digest})
    assert (res.klass, res.reason) == ("needs_human", "data_dependent")


def test_encoding_violation_is_not_exemptible(make_stream):
    data = b"\xef\xbb\xbf" + make_revision(B, A, ADD_COL).encode()
    files = {A + "_x.py": make_revision(A, None, CREATE), B + "_x.py": data}
    s = make_stream(files)
    res = classify(s, B, exceptions={C.sha256_file(data)})
    assert (res.klass, res.blocked) == ("needs_human", [B])


def test_stray_files_in_versions_block(make_stream):
    for stray in [".hid.py", "__init__.py", "notes.txt"]:
        files = chain(f(A, None, CREATE))
        files[stray] = "x = 1\n"
        res = classify(make_stream(files), A)
        assert (res.klass, res.blocked) == ("needs_human", ["chain"]), stray


def test_unresolvable_revision_id_is_reported_as_chain(make_stream):
    files = chain(f(A, None, CREATE))
    files["bbbbbbbbbbbb_x.py"] = "import os\nrevision = something()\n"
    res = classify(make_stream(files), A)
    assert res.blocked == ["chain"]


def test_listing_symlink_is_rejected(make_stream, tmp_path):
    import os

    s = make_stream(chain(f(A, None, CREATE)))
    os.symlink(os.path.join(s.versions_dir, A + "_x.py"), os.path.join(s.versions_dir, B + "_link.py"))
    assert classify(s, A).blocked == ["chain"]


def test_importing_revisions_leaves_no_bytecode(make_stream):
    import os

    s = make_stream(chain(f(A, None, CREATE), f(B, A, ADD_COL)))
    classify(s)
    assert sorted(os.listdir(s.versions_dir)) == [A + "_x.py", B + "_x.py"]


# --- merging streams ---------------------------------------------------------------


def test_merge_results_one_needs_human_blocks_both():
    ok = C.RangeResult("additive", pending=1)
    na = C.RangeResult("na", pending=0)
    bad = C.RangeResult("needs_human", "destructive", blocked=["x"], pending=1)
    assert C.merge_results([ok, bad]) == ("needs_human", "destructive", ["x"])
    assert C.merge_results([na, ok]) == ("additive", None, [])
    assert C.merge_results([na, na]) == ("na", None, [])
    worse = C.RangeResult("needs_human", "data_dependent", blocked=["y"], pending=None)
    assert C.merge_results([bad, worse]) == ("needs_human", "data_dependent", ["x", "y"])


def test_rendered_bind_style_text_is_unparsable():
    reason, plan, _ = C.classify_rendered("ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL DEFAULT ':x';\nUPDATE alembic_version SET version_num='b' WHERE alembic_version.version_num = 'a';\n", "bin_manager", "a", "b")
    assert (reason, plan) == (C.UNPARSABLE, None)


def test_rendered_text_without_bookkeeping_is_unparsable():
    reason, _, _ = C.classify_rendered("ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL;\n", "bin_manager", "a", "b")
    assert reason == C.UNPARSABLE
    twice = "UPDATE alembic_version SET version_num='b' WHERE alembic_version.version_num = 'a';\n" * 2
    assert C.classify_rendered(twice, "bin_manager", "a", "b")[0] == C.UNPARSABLE
    not_last = "UPDATE alembic_version SET version_num='b' WHERE alembic_version.version_num = 'a';\nALTER TABLE t ADD COLUMN a VARCHAR(5) NULL;\n"
    assert C.classify_rendered(not_last, "bin_manager", "a", "b")[0] == C.UNPARSABLE


def test_rendered_alembic_version_create_must_be_once_and_first():
    create = "CREATE TABLE alembic_version (\n\tversion_num VARCHAR(32) NOT NULL, \n\tCONSTRAINT alembic_version_pkc PRIMARY KEY (version_num)\n);\n"
    other = "CREATE TABLE t (id INT NOT NULL, PRIMARY KEY (id));\n"
    stamp = "INSERT INTO alembic_version (version_num) VALUES ('b');\n"
    assert C.classify_rendered(create + other + stamp, "bin_manager", None, "b")[0] is None
    assert C.classify_rendered(other + create + stamp, "bin_manager", None, "b")[0] == C.UNPARSABLE
    assert C.classify_rendered(create + create + other + stamp, "bin_manager", None, "b")[0] == C.UNPARSABLE


def test_range_reason_unparsable_outranks_destructive(make_stream):
    files = chain(
        f(A, None, CREATE),
        f(B, A, 'op.execute("ALTER TABLE t ADD COLUMN s serial")'),
        f(C3, B, 'op.execute("ALTER TABLE t DROP COLUMN c")'),
    )
    res = classify(make_stream(files), A)
    assert res.reason == "unparsable" and res.blocked == [B, C3]


def _hook_source(body):
    return make_revision(B, A, "pass", extra="\n" + body + "\n")


@pytest.mark.parametrize(
    "hook",
    [
        'def __getattr__(name):\n    open(MARK, "w").write(name)\n    raise AttributeError(name)',
        'def helper():\n    pass',
        'def upgrade_twice():\n    pass',
    ],
)
def test_module_level_function_other_than_upgrade_and_downgrade_is_never_executed(make_stream, tmp_path, hook):
    """Alembic reads module attributes on load, so a module level __getattr__ would run during classification."""
    mark = tmp_path / "executed"
    files = chain(f(A, None, CREATE), ("%s_x.py" % B, _hook_source(hook.replace("MARK", repr(str(mark))))))
    res = classify(make_stream(files), A)
    assert res.klass == "needs_human" and res.blocked == [B]
    assert not mark.exists()
    # an already applied revision is scanned as well
    files = chain(("%s_x.py" % A, make_revision(A, None, CREATE, extra="\n" + hook.replace("MARK", repr(str(mark))) + "\n")), f(B, A, ADD_COL))
    res = classify(make_stream(files), A)
    assert res.klass == "needs_human"
    assert not mark.exists()


def test_source_gate_reasons_are_the_classifier_reasons():
    assert (sg.DATA_DEPENDENT, sg.UNPARSABLE) == (C.DATA_DEPENDENT, C.UNPARSABLE)
