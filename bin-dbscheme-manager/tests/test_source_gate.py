import os

import pytest

from classifier import source_gate as sg
from conftest import make_revision

REV = "0123456789ab"
DOWN = "ba9876543210"


def revs(body="pass", **kw):
    return make_revision(REV, DOWN, body, **kw)


def rev(body="pass", **kw):
    return revs(body, **kw).encode("utf-8")


def upgrade_reasons(data):
    return sorted({v.reason for v in sg.check_upgrade(data)})


def module_reasons(data):
    return sorted({v.reason for v in sg.check_module_level(data)})


# --- accepted forms ----------------------------------------------------------

ACCEPTED_BODIES = [
    "pass",
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL")',
    'op.execute("""\n    CREATE TABLE t (\n        id BINARY(16) NOT NULL, -- id\n        PRIMARY KEY (id)\n    );\n""")',
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL;;")',
    'op.execute(sa.text("ALTER TABLE t ADD COLUMN a INT NULL"))',
    'T = "t"\nop.execute("ALTER TABLE " + T + " ADD COLUMN a INT NULL")',
    'T = "t"\nop.execute(f"ALTER TABLE {T} ADD COLUMN a INT NULL")',
    'op.execute("ALTER TABLE %s ADD COLUMN a INT NULL" % "t")',
    'op.execute("ALTER TABLE %s ADD COLUMN %s INT NULL" % ("t", "a"))',
    'for c in ("a", "b"):\n    op.execute("ALTER TABLE t ADD COLUMN " + c + " INT NULL")',
    'if 1 == 1:\n    op.execute("ALTER TABLE t ADD COLUMN a INT NULL")\nelse:\n    op.execute("ALTER TABLE t ADD COLUMN b INT NULL")',
    'op.add_column("t", sa.Column("a", sa.String(255), nullable=True, server_default=sa.text("0"), comment="c"))',
    'op.add_column("t", sa.Column("a", sa.Integer(), nullable=True))',
    'op.create_table("t", sa.Column("id", sa.Integer(), nullable=False), sa.Column("e", sa.Enum("a", "b", name="n")))',
    'op.create_index("i", "t", ["a"], unique=False)',
    'cols = ["a", "b"]\nop.create_index("i", "t", cols)',
    'op.add_column("t", sa.Column("a", mysql.TIMESTAMP(), nullable=True))',
    'op.execute("ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL DEFAULT \'12:30:00\'")',
]


@pytest.mark.parametrize("body", ACCEPTED_BODIES)
def test_accepted_upgrade_bodies(body):
    imports = "from alembic import op\nimport sqlalchemy as sa\nfrom sqlalchemy.dialects import mysql\n"
    assert sg.check_upgrade(rev(body, imports=imports)) == []


def test_comments_with_visible_non_ascii_are_accepted():
    src = '"""Revision \u2014 note"""\nfrom alembic import op\nimport sqlalchemy as sa\n# see \u00a7 4.7\nrevision = "a"\ndown_revision = None\nbranch_labels = None\ndepends_on = None\n\n\ndef upgrade():\n    op.execute("ALTER TABLE t ADD COLUMN a INT NULL")  # \ud55c\uae00\n\n\ndef downgrade():\n    pass\n'
    assert sg.check_module_level(src.encode()) == []
    assert sg.check_upgrade(src.encode()) == []


def test_module_level_constants_and_annotated_assignments_are_accepted():
    src = rev('op.execute("ALTER TABLE t ADD COLUMN " + COL + " INT NULL")', extra='COL = "a"\nVALUES = ["x", "y"]\nNAME = f"{COL}_x"\n')
    assert sg.check_module_level(src) == []
    assert sg.check_upgrade(src) == []
    src = revs("pass").replace("revision = ", "revision: str = ", 1).encode()
    assert sg.check_upgrade(src) == []


def test_postgresql_enum_import_is_accepted():
    src = rev("pass", imports="from alembic import op\nimport sqlalchemy as sa\nfrom sqlalchemy.dialects.postgresql import ENUM\n")
    assert sg.check_module_level(src) == []


# --- upgrade() body rules ----------------------------------------------------

DATA_DEPENDENT_BODIES = [
    "conn = op.get_bind()",
    "op.get_bind().execute(sa.text('DROP TABLE x'))",
    "ctx = op.get_context()",
    "op.batch_alter_table('t')",
    "op.alter_column('t', 'a', existing_type=sa.Integer())",
    "op.drop_column('t', 'a')",
    "op.drop_table('t')",
    "op.create_foreign_key('f', 't', 'u', ['a'], ['id'])",
    "op.create_unique_constraint('u', 't', ['a'])",
    "op.impl.bind",
    "b = op.get_bind",
    "op.execute(os.environ['X'])",
    "getattr(op, 'get' + '_bind')()",
    "try:\n    getattr(op, 'get' + '_bind')()\nexcept Exception:\n    pass",
    "eval('1')",
    "exec('1')",
    "__import__('os')",
    "compile('1', 'x', 'exec')",
    "vars()",
    "globals()",
    "locals()",
    "print('x')",
    "x = lambda: 1",
    "x = [a for a in (1, 2)]",
    "def inner():\n    pass",
    "class C:\n    pass",
    "return",
    "with open('x'):\n    pass",
    "try:\n    pass\nexcept Exception:\n    pass",
    "assert True",
    "raise Exception()",
    "del x",
    "x += 1",
    "global y",
    "downgrade()",
    "upgrade()",
    "sa.func.now()",
    "op.execute(sa.func.now())",
    "sa.true()",
    "sa.PrimaryKeyConstraint('a')",
    "sa.UniqueConstraint('a')",
    "sa.ForeignKey('a.b')",
    "op.add_column('t', sa.Column('a', sa.Integer(), sa.ForeignKey('u.id')))",
    "x = op.execute",
    "op.execute('a').foo()",
    "op.execute(x=1)",
    "op.execute()",
    "op.execute('a', 'b')",
    "op.execute(*['a'])",
    "op.execute(UNKNOWN)",
    "x = UNKNOWN",
    "x = __file__",
    "x = __builtins__",
    "op.execute(session.query(1))",
    "conn.execute('select 1')",
    "inspect(op.get_bind())",
    "Inspector.from_engine(1)",
    "op.add_column('t', sa.Column(**kw))",
    "(lambda: 1)()",
    "ops = [op.execute]\nops[0]('x')",
    "sa.Column('a', sa.Integer())()",
    "x = ('abc' * 100000)",
    "x = '%999999999d' % 1",
    "x = '%5d' % 1",
    "x = '%-5s' % 'a'",
    "x = '%(a)s' % {'a': 1}",
    "x = '{}'.format(1)",
    "x = 'a' if 1 else 'b'",
    "for i in range(3):\n    pass",
    "for i in (1, 2):\n    pass\nelse:\n    pass",
    "for i in (1, 2):\n    break",
    "if UNKNOWN:\n    pass",
    "if 1:\n    pass\nelse:\n    conn = op.get_bind()",
    "if 0:\n    conn = op.get_bind()",
    "x = {'a': 1}",
    "x = b'abc'",
    "x = 1j",
    "x = ...",
    "x = '\\ud800'",
    "if 1 == 1:\n    conn = op.get_bind()\nelse:\n    pass",
    "op.execute('ALTER TABLE t ADD COLUMN a INT NULL', execution_options=op.get_bind())",
    "op.execute(sa.text('ALTER TABLE t ADD COLUMN a INT NULL', bind=op.get_bind()))",
    "op.add_column('t', sa.Column('a', sa.Integer(), nullable=op.get_bind()))",
    "op.create_index('i', 't', *op.get_bind())",
    "a, b = 1, 2",
    "x = sa.Foo",
    "x = mysql.os",
    "a = b = 1",
]


@pytest.mark.parametrize("body", DATA_DEPENDENT_BODIES)
def test_data_dependent_upgrade_bodies(body):
    got = upgrade_reasons(rev(body))
    assert got == [sg.DATA_DEPENDENT], (body, got)


def test_boolean_operators_follow_python_values_so_branches_are_not_misjudged():
    """`("x" or "y") == "x"` is true in Python; a gate that folded it to a bool would inspect the wrong branch."""
    body = (
        'if (A or "y") == "x":\n'
        '    L = ("a",)\n'
        "else:\n"
        "    L = ()\n"
        "for s in L:\n"
        '    __import__("os").system("true")\n'
        'op.execute("CREATE TABLE t (a int)")'
    )
    assert upgrade_reasons(rev(body, extra='A = "x"\n')) == [sg.DATA_DEPENDENT]
    body = 'if ("" or "x") == "x":\n    conn = op.get_bind()\nop.execute("CREATE TABLE t (a int)")'
    assert upgrade_reasons(rev(body)) == [sg.DATA_DEPENDENT]
    body = 'if ("x" and "y") == "y":\n    conn = op.get_bind()\nop.execute("CREATE TABLE t (a int)")'
    assert upgrade_reasons(rev(body)) == [sg.DATA_DEPENDENT]


def test_not_yields_a_bool_like_python():
    body = 'if (not "") == True:\n    conn = op.get_bind()\nop.execute("CREATE TABLE t (a int)")'
    assert upgrade_reasons(rev(body)) == [sg.DATA_DEPENDENT]


def test_a_loop_over_an_empty_sequence_is_rejected():
    body = 'for s in ():\n    __import__("os").system("true")\nop.execute("CREATE TABLE t (a int)")'
    assert upgrade_reasons(rev(body)) == [sg.DATA_DEPENDENT]


def test_long_folded_string_is_rejected():
    body = 'x = "a" * 2\nop.execute("ALTER TABLE t ADD COLUMN a INT NULL")'
    assert upgrade_reasons(rev(body)) == [sg.DATA_DEPENDENT]
    big = 'X = "' + "a" * 40000 + '"\nY = X + X\nop.execute("ALTER TABLE t ADD COLUMN a INT NULL")'
    assert upgrade_reasons(rev(big)) == [sg.DATA_DEPENDENT]


def test_unrolled_loops_are_bounded():
    body = "for a in (" + ",".join(["1"] * 500) + "):\n    for b in (" + ",".join(["1"] * 500) + "):\n        for c in (1, 2, 3, 4):\n            pass"
    assert upgrade_reasons(rev(body)) == [sg.DATA_DEPENDENT]


UNPARSABLE_BODIES = [
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL; DROP TABLE x")',
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL; ALTER TABLE t ADD COLUMN b INT NULL")',
    'op.execute("")',
    'op.execute("  ;  ")',
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL DEFAULT :v")',
    'op.execute("SELECT :30")',
    'op.execute("SELECT \\\\:x")',
    'op.execute("SELECT :$x")',
    'op.execute("SELECT :\u00e9x")',
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL -- c")',
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL -- c\\n")',
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL # c")',
    'op.execute("ALTER TABLE t ADD COLUMN a INT NULL /* c */")',
    'op.execute("ALTER TABLE t\\x0bADD COLUMN a INT NULL")',
]


@pytest.mark.parametrize("body", UNPARSABLE_BODIES)
def test_unparsable_upgrade_bodies(body):
    assert upgrade_reasons(rev(body)) == [sg.UNPARSABLE], body


@pytest.mark.parametrize(
    "text",
    [
        'op.execute("ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL DEFAULT \'12:30:00\'")',
        'op.execute("ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL COMMENT \'ratio 1:2\'")',
        'op.execute("ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL DEFAULT \'9999-01-01 00:00:00.000000\'")',
    ],
)
def test_colon_after_a_word_character_is_not_a_bind(text):
    assert sg.check_upgrade(rev(text)) == []


def test_structure_rules():
    assert upgrade_reasons(make_revision(REV, ("a", "b")).encode()) == [sg.UNPARSABLE]
    assert upgrade_reasons(rev(depends_on="'x'")) == [sg.UNPARSABLE]
    assert upgrade_reasons(rev(branch_labels="'x'")) == [sg.UNPARSABLE]
    src = revs().replace("down_revision = %r\n" % DOWN, "").encode()
    assert upgrade_reasons(src) == [sg.UNPARSABLE]
    src = revs().replace("revision = %r" % REV, "revision = 5", 1).encode()
    assert upgrade_reasons(src) == [sg.UNPARSABLE]
    assert sg.check_upgrade(make_revision("a", None).encode()) == []


def test_missing_or_parameterised_upgrade_is_rejected():
    assert upgrade_reasons(revs().replace("def upgrade():", "def upgrade(x):").encode()) == [sg.DATA_DEPENDENT]
    assert upgrade_reasons(revs().replace("def upgrade():", "def up():").encode()) == [sg.DATA_DEPENDENT]


def test_helper_function_is_rejected():
    src = rev(extra="def helper():\n    pass\n")
    assert upgrade_reasons(src) == [sg.DATA_DEPENDENT]


def test_syntax_error_is_rejected():
    assert upgrade_reasons(b"def (:\n") == [sg.DATA_DEPENDENT]
    assert module_reasons(b"def (:\n") == [sg.DATA_DEPENDENT]


# --- module-level rules (every file of the chain) ----------------------------

MODULE_LEVEL_NEGATIVES = [
    ('op.execute("DROP TABLE x")\n', {}),
    ("import os\n", {}),
    ("from os import path\n", {}),
    ("from alembic import op as o\n", {}),
    ("from sqlalchemy import *\n", {}),
    ("from sqlalchemy.engine.reflection import Inspector\n", {}),
    ("from alembic import context\n", {}),
    ("def ENUM(*a, **k):\n    return op.get_bind()\n", {}),
    ("def op():\n    pass\n", {}),
    ("op = 1\n", {}),
    ("sa = 1\n", {}),
    ("mysql = 1\n", {}),
    ("ENUM = 1\n", {}),
    ("import os as sa\n", {}),
    ("if __debug__:\n    pass\n", {}),
    ("class C:\n    pass\n", {}),
    ("x = [op.get_bind()]\n", {}),
    ("x = sa.Enum('a')\n", {}),
    ("x = ENUM('a', name='n')\n", {}),
    ("x = __file__\n", {}),
    ("x = __builtins__\n", {}),
    ("print(1)\n", {}),
    ("for i in (1, 2):\n    pass\n", {}),
    ("try:\n    pass\nexcept Exception:\n    pass\n", {}),
    ("with open('x'):\n    pass\n", {}),
    ("x, y = 1, 2\n", {}),
    ("x = y = 1\n", {}),
    ("x: int\n", {}),
    ("x = {'a': 1}\n", {}),
    ("x = open('f')\n", {}),
    ("x = 'a' * 3\n", {}),
    ("def downgrade(a=op.get_bind()):\n    pass\n", {}),
    ("def downgrade(a=1, *, b=os.getcwd()):\n    pass\n", {}),
    ("def downgrade(a: op.get_bind()):\n    pass\n", {}),
    ("def downgrade() -> op.get_bind():\n    pass\n", {}),
    ("@decorator\ndef downgrade():\n    pass\n", {}),
    ("from __future__ import annotations as op\n", {}),
    ("import sqlalchemy as sa, os\n", {}),
    ("from .alembic import op\n", {}),
    ("from . import op\n", {}),
    ("from sidefx import op\n", {}),
    ("from sidefx import mysql\n", {}),
    ("from sidefx import Union\n", {}),
    ("from sidefx.postgresql import ENUM\n", {}),
    ("from alembic import op, context\n", {}),
    ("from sqlalchemy.dialects import mysql, postgresql\n", {}),
    ("from sqlalchemy.dialects import postgresql\n", {}),
    ("from sqlalchemy.dialects.postgresql import ENUM, UUID\n", {}),
    ("from typing import cast\n", {}),
    ("def downgrade(*a: print(1)):\n    pass\n", {}),
    ("def downgrade(**k: print(1)):\n    pass\n", {}),
    ("def downgrade(*, a: print(1) = 1):\n    pass\n", {}),
    ("def downgrade(a: print(1), /):\n    pass\n", {}),
    ("x: Union[str, print(1)] = 'a'\n", {}),
    ("x: print(1)[str] = 'a'\n", {}),
    ("def downgrade(a: str[print(1)] = 1):\n    pass\n", {}),
    ("def downgrade(a=print(1)):\n    pass\n", {}),
    ("def downgrade(*, b=print(1)):\n    pass\n", {}),
    ("def downgrade(a: print(1) = 1):\n    pass\n", {}),
    ("def downgrade() -> print(1):\n    pass\n", {}),
    ("def downgrade() -> str[print(1)]:\n    pass\n", {}),
    ("x: str[print(1)] = 1\n", {}),
    ("x: print(1) = 1\n", {}),
    ("def f():\n    global x\n", {}),
    ("def f():\n    import os\n", {}),
    ("def f(op):\n    pass\n", {}),
    ("def f():\n    op = 1\n", {}),
    ("def f():\n    for sa in (1,):\n        pass\n", {}),
    ("def f():\n    with open('x') as mysql:\n        pass\n", {}),
    ("def f():\n    try:\n        pass\n    except Exception as ENUM:\n        pass\n", {}),
    ("if True:\n    from alembic import op\n", {}),
    ("x = (lambda: 1)\n", {}),
    ("x = [i for i in (1,)]\n", {}),
    ("x = f'{op.get_bind()}'\n", {}),
    ("x = f'{1:5}'\n", {}),
    ("x = f'{1!r}'\n", {}),
    ("x = '\\ud800'\n", {}),
]


@pytest.mark.parametrize("snippet", [c[0] for c in MODULE_LEVEL_NEGATIVES])
def test_module_level_negatives(snippet):
    src = make_revision(REV, DOWN, extra=snippet).encode("utf-8")
    viols = sg.check_module_level(src)
    assert viols, snippet
    assert {v.reason for v in viols} == {sg.DATA_DEPENDENT}


def test_module_level_violations_of_the_rules_are_exemptible_but_encoding_is_not():
    v = sg.check_module_level(make_revision(REV, DOWN, extra="import os\n").encode())
    assert v and all(x.exemptible for x in v)
    v = sg.check_module_level(b"\xef\xbb\xbf" + make_revision(REV, DOWN).encode())
    assert v and not any(x.exemptible for x in v)


ENCODING_NEGATIVES = {
    "bom": b"\xef\xbb\xbf" + make_revision(REV, DOWN).encode(),
    "utf7-declaration": b"# -*- coding: utf-7 -*-\n" + make_revision(REV, DOWN).encode(),
    "latin1-declaration": b"# -*- coding: latin-1 -*-\n" + make_revision(REV, DOWN).encode(),
    "utf8-upper": b"# coding: UTF-8\n" + make_revision(REV, DOWN).encode(),
    "utf8-short": b"# coding: utf8\n" + make_revision(REV, DOWN).encode(),
    "declaration-line2": b"# a\n# coding=utf-8\n" + make_revision(REV, DOWN).encode(),
    "lone-cr": make_revision(REV, DOWN).encode() + b"# a\rb\n",
    "nul": make_revision(REV, DOWN).encode() + b"\x00",
    "form-feed": make_revision(REV, DOWN).encode().replace(b"\n\n\ndef upgrade", b"\n\x0c\n\ndef upgrade", 1),
    "vertical-tab-in-comment": make_revision(REV, DOWN).encode() + b"# a\x0bb\n",
    "invalid-utf8": make_revision(REV, DOWN).encode() + b"# \xff\xfe\n",
    "bidi-control-in-comment": make_revision(REV, DOWN).encode() + "# a\u202eb\n".encode(),
    "zero-width-in-string": make_revision(REV, DOWN).encode() + 'X = "a\u200bb"\n'.encode(),
    "nbsp-in-comment": make_revision(REV, DOWN).encode() + "# a\u00a0b\n".encode(),
    "tag-char-in-comment": make_revision(REV, DOWN).encode() + "# a\U000e0041\n".encode(),
    "non-ascii-identifier": make_revision(REV, DOWN).encode() + "\uff4fp = 1\n".encode(),
    "non-ascii-identifier-lookalike": make_revision(REV, DOWN).encode() + "caf\u00e9 = 1\n".encode(),
}


@pytest.mark.parametrize("name", sorted(ENCODING_NEGATIVES))
def test_encoding_negatives_are_not_exemptible(name):
    viols = sg.check_module_level(ENCODING_NEGATIVES[name])
    assert viols, name
    assert not all(v.exemptible for v in viols), name


def test_template_files_pass():
    assert sg.check_module_level(make_revision(REV, DOWN).encode()) == []


# --- directory listing ----------------------------------------------------------


def test_listing_rules(tmp_path):
    d = tmp_path / "versions"
    d.mkdir()
    (d / "0123456789ab_ok.py").write_text("x")
    assert sg.check_directory(str(d)) == []
    (d / ".hid.py").write_text("x")
    (d / "__init__.py").write_text("x")
    (d / "notes.txt").write_text("x")
    (d / "a_x.py").mkdir()
    os.symlink(str(d / "0123456789ab_ok.py"), str(d / "ffffffffffff_link.py"))
    (d / "__pycache__").mkdir()
    (d / "nonewline.py\n").write_text("x")
    (d / "UPPER-dash.py").write_text("x")
    (d / "0123456789ab_x.py\n").write_text("x")
    (d / "0123456789ab_x.pyc").write_text("x")
    names = sorted(n for n, _ in sg.check_directory(str(d)))
    assert names == sorted([".hid.py", "__init__.py", "__pycache__", "a_x.py", "ffffffffffff_link.py", "nonewline.py\n", "notes.txt", "UPPER-dash.py", "0123456789ab_x.py\n", "0123456789ab_x.pyc"])
