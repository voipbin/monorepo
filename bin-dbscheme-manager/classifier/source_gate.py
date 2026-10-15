"""Source gate for Alembic revision files.

Default deny. A revision file is accepted only when everything it executes at
import time, and everything its upgrade() can do, is on a short allow-list of
constant-only constructs. Anything else is "data_dependent" (it could behave
differently online than in the offline render) or "unparsable" (the revision
structure or the SQL text is outside the accepted forms).

Three entry points:
  check_directory(path)       the listing rules of a versions directory
  check_module_level(data)    encoding and import-time rules, for EVERY file of the chain
  check_upgrade(data)         the upgrade() body rules, for pending revisions
"""

import ast
import io
import os
import re
import tokenize
import unicodedata
from dataclasses import dataclass

from .sqltext import BIND_RE, SqlError, split_statements

DATA_DEPENDENT = "data_dependent"
UNPARSABLE = "unparsable"

FILE_NAME_RE = re.compile(r"[0-9A-Za-z]+_[0-9A-Za-z_]*\.py")

_PROTECTED = frozenset(("op", "sa", "mysql", "ENUM"))
_MAX_FOLDED = 64 * 1024
_MAX_STEPS = 100_000
_OK_CATEGORIES = frozenset("LMNPS")

_SA_TYPES = frozenset(
    """String VARCHAR CHAR Text TEXT Unicode UnicodeText Integer INTEGER BigInteger BIGINT SmallInteger
    SMALLINT Boolean BOOLEAN DateTime DATETIME TIMESTAMP Date DATE Time TIME Float FLOAT Numeric NUMERIC
    DECIMAL LargeBinary BINARY VARBINARY BLOB JSON Enum""".split()
)
_MYSQL_TYPES = frozenset(
    """BINARY VARBINARY TINYINT SMALLINT MEDIUMINT INTEGER BIGINT TINYTEXT MEDIUMTEXT LONGTEXT TINYBLOB
    MEDIUMBLOB LONGBLOB DATETIME TIMESTAMP TIME YEAR DECIMAL DOUBLE FLOAT VARCHAR CHAR TEXT BLOB JSON
    ENUM SET BIT BOOLEAN""".split()
)
_OP_CALLS = frozenset(("execute", "add_column", "create_table", "create_index"))
_ANNOTATION_NAMES = frozenset(("str", "int", "Union", "Sequence", "Optional"))
_TYPING_NAMES = frozenset(("Union", "Sequence", "Optional"))


@dataclass(frozen=True)
class Violation:
    reason: str  # data_dependent | unparsable
    msg: str
    exemptible: bool = False  # only module-level rules can be listed by hash


class _Fail(Exception):
    def __init__(self, reason, msg):
        super().__init__(msg)
        self.reason = reason
        self.msg = msg


def _dd(msg):
    return _Fail(DATA_DEPENDENT, msg)


# --------------------------------------------------------------------------
# directory listing and encoding
# --------------------------------------------------------------------------


def check_directory(versions_dir):
    """Every entry must be a regular file with a revision-style name."""
    out = []
    with os.scandir(versions_dir) as it:
        for entry in sorted(it, key=lambda e: e.name):
            if not entry.is_file(follow_symlinks=False) or not FILE_NAME_RE.fullmatch(entry.name):
                out.append((entry.name, Violation(DATA_DEPENDENT, "unexpected entry in versions directory")))
    return out


_CODING_RE = re.compile(rb"^[ \t\f]*#.*?coding[:=]")


def _check_encoding(data):
    """Encoding rules. Not exemptible: gate and interpreter must see the same text."""
    out = []

    def bad(msg):
        out.append(Violation(DATA_DEPENDENT, msg, exemptible=False))

    if data.startswith(b"\xef\xbb\xbf"):
        bad("byte order mark")
    if b"\x00" in data:
        bad("NUL byte")
    if b"\x0c" in data:
        bad("form feed")
    if re.search(rb"\r(?!\n)", data):
        bad("lone carriage return")
    for line in data.split(b"\n")[:2]:
        if _CODING_RE.match(line):
            bad("coding declaration")
    try:
        enc, _ = tokenize.detect_encoding(io.BytesIO(data).readline)
        if enc != "utf-8":
            bad("source encoding is not utf-8")
    except (SyntaxError, ValueError):
        bad("source encoding cannot be determined")
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError:
        bad("not valid UTF-8")
        return out, None
    for ch in text:
        o = ord(ch)
        if o < 128:
            if (o < 32 and ch not in "\t\n\r") or o == 127:
                bad("control character")
                break
        elif unicodedata.category(ch)[0] not in _OK_CATEGORIES:
            bad("invisible or unassigned character")
            break
    try:
        for tok in tokenize.generate_tokens(io.StringIO(text).readline):
            if tok.type == tokenize.NAME and not tok.string.isascii():
                bad("non-ASCII identifier")
                break
    except (tokenize.TokenError, SyntaxError, IndentationError):
        bad("source cannot be tokenized")
    return out, text


# --------------------------------------------------------------------------
# constant folding
# --------------------------------------------------------------------------


class _Opaque:
    """A SQLAlchemy construct (column, type, text) whose arguments were checked."""

    def __init__(self, sql=None):
        self.sql = sql


class _Steps:
    def __init__(self):
        self.n = 0

    def tick(self):
        self.n += 1
        if self.n > _MAX_STEPS:
            raise _dd("folding budget exceeded")


def _check_str(s):
    if len(s) > _MAX_FOLDED:
        raise _dd("folded string is too long")
    try:
        s.encode("utf-8")
    except UnicodeEncodeError:
        raise _dd("string is not valid UTF-8") from None
    return s


def _is_num(v):
    return isinstance(v, (int, float)) and not isinstance(v, bool)


_FORMAT_SPEC_RE = re.compile(r"%(.?)", re.DOTALL)


def _fold(node, env, steps):
    steps.tick()
    if isinstance(node, ast.Constant):
        v = node.value
        if isinstance(v, str):
            return _check_str(v)
        if v is None or isinstance(v, (bool, int, float)):
            return v
        raise _dd("constant kind")
    if isinstance(node, ast.Name):
        if node.id in env and not isinstance(env[node.id], _Opaque):
            return env[node.id]
        raise _dd("name does not resolve to a tracked constant")
    if isinstance(node, (ast.Tuple, ast.List)):
        vals = []
        for e in node.elts:
            if isinstance(e, ast.Starred):
                inner = _fold(e.value, env, steps)
                if not isinstance(inner, (tuple, list)):
                    raise _dd("star-arg of a non-sequence")
                vals.extend(inner)
            else:
                vals.append(_fold(e, env, steps))
        if len(vals) > 10_000:
            raise _dd("sequence is too long")
        return tuple(vals) if isinstance(node, ast.Tuple) else list(vals)
    if isinstance(node, ast.JoinedStr):
        parts = []
        for p in node.values:
            if isinstance(p, ast.Constant) and isinstance(p.value, str):
                parts.append(p.value)
            elif isinstance(p, ast.FormattedValue):
                if p.conversion != -1 or p.format_spec is not None:
                    raise _dd("f-string conversion or format spec")
                v = _fold(p.value, env, steps)
                if not isinstance(v, (str, int)) or isinstance(v, bool):
                    raise _dd("f-string value kind")
                parts.append(str(v))
            else:
                raise _dd("f-string part")
        return _check_str("".join(parts))
    if isinstance(node, ast.BinOp):
        left = _fold(node.left, env, steps)
        right = _fold(node.right, env, steps)
        if isinstance(node.op, ast.Add):
            if isinstance(left, str) and isinstance(right, str):
                return _check_str(left + right)
            if _is_num(left) and _is_num(right):
                return left + right
            if isinstance(left, list) and isinstance(right, list):
                return left + right
            if isinstance(left, tuple) and isinstance(right, tuple):
                return left + right
            raise _dd("operand kinds of +")
        if isinstance(node.op, ast.Mod) and isinstance(left, str):
            for m in _FORMAT_SPEC_RE.finditer(left):
                if m.group(1) not in ("s", "d", "%"):
                    raise _dd("format specifier")
            args = right if isinstance(right, tuple) else (right,)
            if any(not isinstance(a, (str, int)) or isinstance(a, bool) for a in args):
                raise _dd("format argument kind")
            try:
                return _check_str(left % (args if isinstance(right, tuple) else right))
            except (TypeError, ValueError):
                raise _dd("format failed") from None
        raise _dd("binary operator")
    if isinstance(node, ast.UnaryOp):
        v = _fold(node.operand, env, steps)
        if isinstance(node.op, ast.Not):
            return not v
        if isinstance(node.op, (ast.USub, ast.UAdd)) and _is_num(v):
            return -v if isinstance(node.op, ast.USub) else +v
        raise _dd("unary operator")
    if isinstance(node, ast.BoolOp):
        vals = [_fold(v, env, steps) for v in node.values]
        # Python semantics: the value of the deciding operand, not a bool.
        want = isinstance(node.op, ast.Or)
        for v in vals[:-1]:
            if bool(v) == want:
                return v
        return vals[-1]
    if isinstance(node, ast.Compare):
        left = _fold(node.left, env, steps)
        result = True
        for op, comp in zip(node.ops, node.comparators):
            right = _fold(comp, env, steps)
            try:
                if isinstance(op, ast.Eq):
                    ok = left == right
                elif isinstance(op, ast.NotEq):
                    ok = left != right
                elif isinstance(op, ast.In):
                    ok = left in right
                elif isinstance(op, ast.NotIn):
                    ok = left not in right
                elif isinstance(op, ast.Lt):
                    ok = left < right
                elif isinstance(op, ast.Gt):
                    ok = left > right
                elif isinstance(op, ast.LtE):
                    ok = left <= right
                elif isinstance(op, ast.GtE):
                    ok = left >= right
                else:
                    raise _dd("comparison operator")
            except TypeError:
                raise _dd("comparison operands") from None
            result = result and ok
            left = right
        return result
    raise _dd("expression kind " + type(node).__name__)


# --------------------------------------------------------------------------
# module-level rules (every file of the chain)
# --------------------------------------------------------------------------


def _annotation_ok(node):
    if node is None:
        return True
    if isinstance(node, ast.Constant):
        return node.value is None or isinstance(node.value, str)
    if isinstance(node, ast.Name):
        return node.id in _ANNOTATION_NAMES
    if isinstance(node, ast.Subscript):
        return _annotation_ok(node.value) and _annotation_ok(node.slice)
    if isinstance(node, ast.Tuple):
        return all(_annotation_ok(e) for e in node.elts)
    return False


def _check_import(node):
    if isinstance(node, ast.Import):
        if len(node.names) == 1 and node.names[0].name == "sqlalchemy" and node.names[0].asname == "sa":
            return
    elif isinstance(node, ast.ImportFrom) and node.level == 0:
        names = [(a.name, a.asname) for a in node.names]
        if node.module == "alembic" and names == [("op", None)]:
            return
        if node.module == "sqlalchemy.dialects" and names == [("mysql", None)]:
            return
        if node.module == "sqlalchemy.dialects.postgresql" and names == [("ENUM", None)]:
            return
        if node.module == "__future__" and all(a is None for _, a in names):
            return
        if node.module == "typing" and all(n in _TYPING_NAMES and a is None for n, a in names):
            return
    raise _dd("import form")


def _check_function_header(node, env, steps):
    """Decorators, default arguments and annotations run at import time."""
    if node.decorator_list:
        raise _dd("decorator")
    a = node.args
    for d in list(a.defaults) + [d for d in a.kw_defaults if d is not None]:
        _fold(d, env, steps)
    for arg in list(a.posonlyargs) + list(a.args) + list(a.kwonlyargs) + [x for x in (a.vararg, a.kwarg) if x]:
        if not _annotation_ok(arg.annotation):
            raise _dd("argument annotation")
    if not _annotation_ok(node.returns):
        raise _dd("return annotation")


def _check_rebinding(tree):
    """op, sa, mysql and ENUM come from the template imports and are never rebound."""
    allowed_imports = set()
    for stmt in tree.body:
        if isinstance(stmt, (ast.Import, ast.ImportFrom)):
            allowed_imports.add(id(stmt))
    for n in ast.walk(tree):
        names = []
        if isinstance(n, ast.Name) and isinstance(n.ctx, (ast.Store, ast.Del)):
            names.append(n.id)
        elif isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            names.append(n.name)
        elif isinstance(n, ast.arg):
            names.append(n.arg)
        elif isinstance(n, ast.ExceptHandler) and n.name:
            names.append(n.name)
        elif isinstance(n, (ast.Global, ast.Nonlocal)):
            raise _dd("global or nonlocal")
        elif isinstance(n, (ast.Import, ast.ImportFrom)):
            if id(n) not in allowed_imports:
                raise _dd("import outside module level")
            for a in n.names:
                if (a.asname or a.name.split(".")[0]) in _PROTECTED and not _is_template_import(n):
                    raise _dd("template name rebound")
        for nm in names:
            if nm in _PROTECTED or nm in ("__file__", "__builtins__", "__debug__"):
                raise _dd("template name rebound")


def _is_template_import(node):
    try:
        _check_import(node)
    except _Fail:
        return False
    return True


def _module_level(tree):
    env = {}
    steps = _Steps()
    for stmt in tree.body:
        if isinstance(stmt, (ast.Import, ast.ImportFrom)):
            _check_import(stmt)
        elif isinstance(stmt, ast.Expr):
            if not (isinstance(stmt.value, ast.Constant) and isinstance(stmt.value.value, str)):
                raise _dd("module-level expression")
            _check_str(stmt.value.value)
        elif isinstance(stmt, (ast.Assign, ast.AnnAssign)):
            if isinstance(stmt, ast.Assign):
                targets = stmt.targets
            else:
                targets = [stmt.target]
                if not _annotation_ok(stmt.annotation):
                    raise _dd("annotation")
                if stmt.value is None:
                    raise _dd("annotation without value")
            if len(targets) != 1 or not isinstance(targets[0], ast.Name):
                raise _dd("assignment target")
            env[targets[0].id] = _fold(stmt.value, env, steps)
        elif isinstance(stmt, ast.FunctionDef):
            # Alembic reads module attributes while loading, so any other function (a module level
            # __getattr__ for one) could run code at import time.
            if stmt.name not in ("upgrade", "downgrade"):
                raise _dd("helper function")
            _check_function_header(stmt, env, steps)
        else:
            raise _dd("module-level statement " + type(stmt).__name__)
    return env


def check_module_level(data):
    """Encoding and import-time rules for one file. Returns list[Violation]."""
    out, text = _check_encoding(data)
    if text is None:
        return out
    try:
        tree = ast.parse(data)
    except (SyntaxError, ValueError, RecursionError, MemoryError):
        out.append(Violation(DATA_DEPENDENT, "source does not parse", exemptible=False))
        return out
    try:
        _check_rebinding(tree)
        _module_level(tree)
    except _Fail as exc:
        out.append(Violation(exc.reason, exc.msg, exemptible=True))
    return out


# --------------------------------------------------------------------------
# upgrade() body rules (pending revisions)
# --------------------------------------------------------------------------


def _call_key(func):
    """'op.execute', 'sa.Column', 'mysql.TIMESTAMP' or 'ENUM'; anything else is rejected."""
    if isinstance(func, ast.Name) and func.id == "ENUM":
        return "ENUM"
    if isinstance(func, ast.Attribute) and isinstance(func.value, ast.Name) and func.value.id in ("op", "sa", "mysql"):
        return func.value.id + "." + func.attr
    raise _dd("call target")


def _check_one_statement(sql):
    if BIND_RE.search(sql):
        raise _Fail(UNPARSABLE, "bind-style parameter in SQL text")
    # the offline renderer strips the text and appends ';'
    try:
        stmts = split_statements(sql.strip() + ";")
    except SqlError as exc:
        raise _Fail(UNPARSABLE, str(exc)) from None
    if len(stmts) != 1:
        raise _Fail(UNPARSABLE, "op.execute must carry exactly one non-empty statement")


def _value(node, env, steps, level):
    """A constant, or an allowed SQLAlchemy construct."""
    steps.tick()
    if isinstance(node, ast.Call):
        return _call(node, env, steps, level)
    if isinstance(node, ast.Attribute) and isinstance(node.value, ast.Name) and node.value.id in ("sa", "mysql"):
        types = _SA_TYPES if node.value.id == "sa" else _MYSQL_TYPES
        if node.attr in types:
            return _Opaque()
        raise _dd("attribute")
    if isinstance(node, ast.Name) and node.id in env and isinstance(env[node.id], _Opaque):
        return env[node.id]
    return _fold(node, env, steps)


def _args(node, env, steps):
    for a in node.args:
        if isinstance(a, ast.Starred):
            v = _fold(a.value, env, steps)
            if not isinstance(v, (tuple, list)):
                raise _dd("star-arg of a non-sequence")
        else:
            _value(a, env, steps, "expr")
    for kw in node.keywords:
        if kw.arg is None:
            raise _dd("keyword unpacking")
        _value(kw.value, env, steps, "expr")


def _call(node, env, steps, level):
    key = _call_key(node.func)
    kind, _, attr = key.partition(".")
    if key == "op.execute":
        if level != "stmt" or len(node.args) != 1 or node.keywords or isinstance(node.args[0], ast.Starred):
            raise _dd("op.execute form")
        arg = _value(node.args[0], env, steps, "expr")
        sql = arg.sql if isinstance(arg, _Opaque) else arg
        if not isinstance(sql, str):
            raise _dd("op.execute argument")
        _check_one_statement(sql)
        return _Opaque()
    if kind == "op":
        if level != "stmt" or attr not in _OP_CALLS:
            raise _dd("op call")
        _args(node, env, steps)
        return _Opaque()
    if key == "sa.text":
        if len(node.args) != 1 or node.keywords or isinstance(node.args[0], ast.Starred):
            raise _dd("sa.text form")
        sql = _fold(node.args[0], env, steps)
        if not isinstance(sql, str):
            raise _dd("sa.text argument")
        return _Opaque(sql)
    if key == "sa.Column" or key == "ENUM" or (kind == "sa" and attr in _SA_TYPES) or (kind == "mysql" and attr in _MYSQL_TYPES):
        _args(node, env, steps)
        return _Opaque()
    raise _dd("call target")


def _exec_block(body, env, steps):
    for s in body:
        _exec_stmt(s, env, steps)


def _exec_stmt(s, env, steps):
    steps.tick()
    if isinstance(s, ast.Pass):
        return
    if isinstance(s, ast.Expr):
        if isinstance(s.value, ast.Call):
            _call(s.value, env, steps, "stmt")
            return
        if isinstance(s.value, ast.Constant) and isinstance(s.value.value, str):
            return
        raise _dd("expression statement")
    if isinstance(s, ast.Assign):
        if len(s.targets) != 1 or not isinstance(s.targets[0], ast.Name):
            raise _dd("assignment target")
        env[s.targets[0].id] = _value(s.value, env, steps, "expr")
        return
    if isinstance(s, ast.For):
        if s.orelse or not isinstance(s.target, ast.Name):
            raise _dd("for form")
        seq = _fold(s.iter, env, steps)
        if not isinstance(seq, (tuple, list)):
            raise _dd("for over a non-sequence")
        if not seq:
            raise _dd("for over an empty sequence")
        for item in seq:
            env[s.target.id] = item
            _exec_block(s.body, env, steps)
        return
    if isinstance(s, ast.If):
        taken = bool(_fold(s.test, env, steps))
        live, dead = (s.body, s.orelse) if taken else (s.orelse, s.body)
        other = dict(env)
        _exec_block(live, env, steps)
        _exec_block(dead, other, steps)
        return
    raise _dd("statement " + type(s).__name__)


def _structure(env):
    """Revision structure: single linear chain, no labels, no dependencies."""
    if not isinstance(env.get("revision"), str):
        raise _Fail(UNPARSABLE, "revision is not a string")
    if "down_revision" not in env:
        raise _Fail(UNPARSABLE, "down_revision is missing")
    down = env["down_revision"]
    if down is not None and not isinstance(down, str):
        raise _Fail(UNPARSABLE, "down_revision is not a single revision")
    if env.get("depends_on") is not None:
        raise _Fail(UNPARSABLE, "depends_on is not accepted")
    if env.get("branch_labels") is not None:
        raise _Fail(UNPARSABLE, "branch_labels is not accepted")


def check_upgrade(data):
    """upgrade() body and structure rules for one pending revision. Returns list[Violation]."""
    try:
        tree = ast.parse(data)
        env = _module_level(tree)
    except _Fail as exc:
        return [Violation(exc.reason, exc.msg)]
    except (SyntaxError, ValueError, RecursionError, MemoryError):
        return [Violation(DATA_DEPENDENT, "source does not parse")]
    out = []
    steps = _Steps()
    try:
        _check_rebinding(tree)
        defs = {s.name: s for s in tree.body if isinstance(s, ast.FunctionDef)}
        if "upgrade" not in defs or "downgrade" not in defs:
            raise _dd("upgrade or downgrade is missing")
        for name, fn in defs.items():
            if name not in ("upgrade", "downgrade"):
                raise _dd("helper function")
            a = fn.args
            if a.posonlyargs or a.args or a.vararg or a.kwonlyargs or a.kwarg:
                raise _dd("function parameters")
        _exec_block(defs["upgrade"].body, dict(env), steps)
    except _Fail as exc:
        out.append(Violation(exc.reason, exc.msg))
    except (RecursionError, MemoryError):
        out.append(Violation(DATA_DEPENDENT, "source is too deeply nested"))
    try:
        _structure(env)
    except _Fail as exc:
        out.append(Violation(exc.reason, exc.msg))
    return out
