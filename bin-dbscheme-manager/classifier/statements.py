"""Statement classifier for rendered DDL.

A statement is "additive" only when it matches the small allow-list below.
Everything else is "destructive" (a recognised destructive form) or
"unparsable" (anything outside the grammar). Unknown is never additive.
"""

import re
from dataclasses import dataclass

from .sqltext import SqlError, ident_name, normalize, split_statements

ADDITIVE = "additive"
BOOKKEEPING = "bookkeeping"
DESTRUCTIVE = "destructive"
UNPARSABLE = "unparsable"

_NAME_RE = re.compile(r"[A-Za-z0-9_]+")
_ID_RE = re.compile(r"[0-9A-Za-z_]+")

# Unquoted words that are never accepted anywhere in a statement.
_FORBIDDEN_WORDS = frozenset(
    "FOREIGN REFERENCES CHECK GENERATED SELECT WITH PARTITION AS LIKE OR TEMPORARY ALGORITHM LOCK".split()
)
# Unquoted words that cannot be a column name (a clause keyword).
_CLAUSE_WORDS = frozenset(
    "UNIQUE PRIMARY FULLTEXT SPATIAL FOREIGN CONSTRAINT CHECK PERIOD INDEX KEY PARTITION SYSTEM".split()
)
_DESTRUCTIVE_STATEMENT_WORDS = frozenset("DROP TRUNCATE DELETE UPDATE INSERT REPLACE RENAME".split())
_DESTRUCTIVE_CLAUSE_WORDS = frozenset(
    "DROP MODIFY CHANGE RENAME ALTER CONVERT DISCARD IMPORT TRUNCATE ENGINE ROW_FORMAT DEFAULT "
    "CHARACTER CHARSET COLLATE ORDER FORCE".split()
)

_TYPES = frozenset(
    """tinyint smallint int bigint decimal mediumint integer bool numeric dec real year float double
    varchar char tinytext text mediumtext longtext tinyblob blob mediumblob longblob binary varbinary
    datetime timestamp date time json enum set bit boolean""".split()
)
_STRING_ARG_TYPES = frozenset(("enum", "set"))

_HEX_DEFAULT_RE = re.compile(r"0[xX][0-9A-Fa-f]+")
_HEX_STR_RE = re.compile(r"'[0-9A-Fa-f]*'")
_DIGITS_RE = re.compile(r"[0-9]+")


class Reject(Exception):
    def __init__(self, kind, msg):
        super().__init__(msg)
        self.kind = kind
        self.msg = msg


def _bad(msg):
    return Reject(UNPARSABLE, msg)


@dataclass(frozen=True)
class Verdict:
    kind: str
    detail: str = ""


@dataclass(frozen=True)
class Context:
    """schema: the only schema qualifier a table name may carry.
    from_rev/to_rev: the expected bookkeeping pair of the step (from_rev None on an empty database).
    """

    schema: str
    from_rev: str | None = None
    to_rev: str | None = None


class _Cursor:
    def __init__(self, tokens):
        self.t = list(tokens)
        self.i = 0

    def done(self):
        return self.i >= len(self.t)

    def peek(self, k=0):
        j = self.i + k
        return self.t[j] if j < len(self.t) else None

    def next(self):
        if self.done():
            raise _bad("unexpected end of statement")
        tok = self.t[self.i]
        self.i += 1
        return tok

    def is_kw(self, *words, k=0):
        tok = self.peek(k)
        return tok is not None and tok.kind == "word" and tok.val.upper() in words

    def kw(self, *words):
        if not self.is_kw(*words):
            raise _bad("expected " + "/".join(words))
        return self.next().val.upper()

    def accept_kw(self, *words):
        if self.is_kw(*words):
            return self.next().val.upper()
        return None

    def is_punct(self, ch, k=0):
        tok = self.peek(k)
        return tok is not None and tok.kind == "punct" and tok.val == ch

    def punct(self, ch):
        if not self.is_punct(ch):
            raise _bad("expected " + ch)
        self.next()

    def accept_punct(self, ch):
        if self.is_punct(ch):
            self.next()
            return True
        return False


def _name(cur, allow_clause_word=False):
    tok = cur.next()
    name = ident_name(tok)
    if name is None:
        raise _bad("not a simple name")
    if tok.kind == "word" and not allow_clause_word and tok.val.upper() in _CLAUSE_WORDS:
        raise _bad("clause keyword used as a name")
    return name


def _table_name(cur, ctx):
    first = _name(cur)
    if cur.accept_punct("."):
        if first.lower() != ctx.schema.lower():
            raise _bad("table qualified with another schema")
        first = _name(cur)
    if first.lower() == "alembic_version":
        raise _bad("alembic_version is not a target of a body statement")
    return first


def _index_cols(cur):
    cur.punct("(")
    while True:
        _name(cur, allow_clause_word=True)
        if cur.accept_punct("("):
            n = cur.next()
            if n.kind != "word" or not _DIGITS_RE.fullmatch(n.val):
                raise _bad("index prefix length")
            cur.punct(")")
        if cur.accept_punct(","):
            continue
        cur.punct(")")
        return


def _type_args(cur, type_name):
    if not cur.accept_punct("("):
        return
    while True:
        tok = cur.next()
        if tok.kind == "word" and _DIGITS_RE.fullmatch(tok.val):
            pass
        elif tok.kind == "str" and type_name in _STRING_ARG_TYPES:
            pass
        else:
            raise _bad("type argument")
        if cur.accept_punct(","):
            continue
        cur.punct(")")
        return


def _column_type(cur):
    tok = cur.next()
    if tok.kind != "word" or tok.val.lower() not in _TYPES:
        raise _bad("column type")
    type_name = tok.val.lower()
    _type_args(cur, type_name)
    return type_name


def _default_value(cur):
    tok = cur.next()
    if tok.kind == "str":
        return "string"
    if tok.kind == "word":
        up = tok.val.upper()
        if up == "NULL":
            return "null"
        if up in ("TRUE", "FALSE"):
            return "bool"
        if _HEX_DEFAULT_RE.fullmatch(tok.val):
            return "hex"
        if up == "CURRENT_TIMESTAMP" or up == "NOW":
            if cur.accept_punct("("):
                if not cur.is_punct(")"):
                    n = cur.next()
                    if n.kind != "word" or not _DIGITS_RE.fullmatch(n.val):
                        raise _bad("timestamp precision")
                cur.punct(")")
            elif up == "NOW":
                raise _bad("now() needs parentheses")
            return "expr"
        if _DIGITS_RE.fullmatch(tok.val):
            if cur.is_punct(".") and cur.peek(1) is not None and cur.peek(1).kind == "word" and _DIGITS_RE.fullmatch(cur.peek(1).val):
                cur.next()
                cur.next()
            return "number"
        raise _bad("default value")
    if tok.kind == "punct" and tok.val in "+-":
        n = cur.next()
        if n.kind != "word" or not _DIGITS_RE.fullmatch(n.val):
            raise _bad("default value")
        if cur.is_punct(".") and cur.peek(1) is not None and cur.peek(1).kind == "word" and _DIGITS_RE.fullmatch(cur.peek(1).val):
            cur.next()
            cur.next()
        return "number"
    if tok.kind == "punct" and tok.val == "(":
        fn = cur.next()
        if fn.kind != "word":
            raise _bad("default expression")
        up = fn.val.upper()
        cur.punct("(")
        if up in ("JSON_OBJECT", "JSON_ARRAY"):
            pass
        elif up == "UNHEX":
            arg = cur.next()
            if arg.kind != "str" or not _HEX_STR_RE.fullmatch(arg.val):
                raise _bad("unhex argument")
        else:
            raise _bad("default expression")
        cur.punct(")")
        cur.punct(")")
        return "expr"
    raise _bad("default value")


def _column_attrs(cur, ctx, in_create):
    """Parse column attributes up to a clause boundary (',' or ')' or end)."""
    seen = set()
    nullability = None
    default = None
    while not cur.done() and not cur.is_punct(",") and not cur.is_punct(")"):
        tok = cur.peek()
        if tok.kind != "word":
            raise _bad("column attribute")
        up = tok.val.upper()
        if up == "NOT":
            cur.next()
            cur.kw("NULL")
            attr = "NULLABILITY"
            value = "not null"
        elif up == "NULL":
            cur.next()
            attr = "NULLABILITY"
            value = "null"
        elif up == "UNSIGNED":
            cur.next()
            attr = up
            value = None
        elif up == "AUTO_INCREMENT" and in_create:
            cur.next()
            attr = up
            value = None
        elif up == "CHARACTER":
            cur.next()
            cur.kw("SET")
            attr = "CHARSET"
            value = _name(cur)
        elif up == "COLLATE":
            cur.next()
            attr = up
            value = _name(cur)
        elif up == "COMMENT":
            cur.next()
            s = cur.next()
            if s.kind != "str":
                raise _bad("comment text")
            attr = up
            value = s.val
        elif up == "DEFAULT":
            cur.next()
            attr = up
            value = _default_value(cur)
        elif up == "AFTER" and not in_create:
            cur.next()
            attr = up
            value = _name(cur, allow_clause_word=True)
        else:
            raise _bad("column attribute")
        if attr in seen:
            raise _bad("repeated column attribute")
        seen.add(attr)
        if attr == "NULLABILITY":
            nullability = value
        elif attr == "DEFAULT":
            default = value
    if not in_create and nullability == "not null" and (default is None or default == "null"):
        raise _bad("NOT NULL column added without a default")
    return nullability, default


def _split_top_level(cur_tokens):
    """Split tokens on top-level commas (outside parentheses)."""
    parts = []
    cur = []
    depth = 0
    for tok in cur_tokens:
        if tok.kind == "punct" and tok.val == "(":
            depth += 1
        elif tok.kind == "punct" and tok.val == ")":
            depth -= 1
            if depth < 0:
                raise _bad("unbalanced parentheses")
        if depth == 0 and tok.kind == "punct" and tok.val == ",":
            parts.append(cur)
            cur = []
        else:
            cur.append(tok)
    if depth != 0:
        raise _bad("unbalanced parentheses")
    parts.append(cur)
    return parts


def _column_def(cur, ctx, in_create):
    _name(cur)
    _column_type(cur)
    _column_attrs(cur, ctx, in_create)
    if not cur.done():
        raise _bad("trailing tokens in column definition")


def _check_alter_clause(tokens, ctx):
    cur = _Cursor(tokens)
    if cur.done():
        raise _bad("empty clause")
    first = cur.peek()
    if first.kind == "word" and first.val.upper() in _DESTRUCTIVE_CLAUSE_WORDS:
        raise Reject(DESTRUCTIVE, "ALTER clause " + first.val.upper())
    cur.kw("ADD")
    if cur.is_kw("INDEX", "KEY"):
        cur.next()
        _name(cur)
        _index_cols(cur)
        if not cur.done():
            raise _bad("trailing tokens in ADD INDEX")
        return
    cur.accept_kw("COLUMN")
    if cur.is_kw("IF"):
        cur.next()
        cur.kw("NOT")
        cur.kw("EXISTS")
    _column_def(cur, ctx, in_create=False)


def _create_def(tokens, ctx):
    cur = _Cursor(tokens)
    if cur.done():
        raise _bad("empty definition")
    first = cur.peek()
    if first.kind == "word" and first.val.upper() in _CLAUSE_WORDS:
        up = first.val.upper()
        if up == "PRIMARY":
            cur.next()
            cur.kw("KEY")
            _index_cols(cur)
        elif up in ("KEY", "INDEX"):
            cur.next()
            if not cur.is_punct("("):
                _name(cur)
            _index_cols(cur)
        elif up == "UNIQUE":
            cur.next()
            cur.accept_kw("KEY", "INDEX")
            if not cur.is_punct("("):
                _name(cur)
            _index_cols(cur)
        elif up == "CONSTRAINT":
            cur.next()
            _name(cur)
            kind = cur.kw("UNIQUE", "PRIMARY")
            if kind == "PRIMARY":
                cur.kw("KEY")
            _index_cols(cur)
        else:
            raise _bad("definition " + up)
        if not cur.done():
            raise _bad("trailing tokens in definition")
        return
    _column_def(cur, ctx, in_create=True)


def _table_options(cur):
    seen = set()
    while not cur.done():
        up = cur.kw("ENGINE", "ROW_FORMAT", "DEFAULT", "CHARSET", "CHARACTER", "COLLATE", "COMMENT")
        if up == "DEFAULT":
            up = cur.kw("CHARSET", "CHARACTER", "COLLATE")
        if up == "CHARACTER":
            cur.kw("SET")
            up = "CHARSET"
        cur.accept_punct("=")
        if up == "ENGINE":
            val = _name(cur)
            if val.lower() != "innodb":
                raise _bad("table engine")
        elif up == "ROW_FORMAT":
            val = _name(cur)
            if val.upper() not in ("DYNAMIC", "COMPACT"):
                raise _bad("row format")
        elif up in ("CHARSET", "COLLATE"):
            _name(cur)
        else:
            s = cur.next()
            if s.kind != "str":
                raise _bad("table comment")
        if up in seen:
            raise _bad("repeated table option")
        seen.add(up)


def _create_table(cur, ctx):
    cur.kw("TABLE")
    if cur.is_kw("IF"):
        cur.next()
        cur.kw("NOT")
        cur.kw("EXISTS")
    _table_name(cur, ctx)
    cur.punct("(")
    # find the matching close parenthesis
    depth = 1
    j = cur.i
    while j < len(cur.t):
        tok = cur.t[j]
        if tok.kind == "punct" and tok.val == "(":
            depth += 1
        elif tok.kind == "punct" and tok.val == ")":
            depth -= 1
            if depth == 0:
                break
        j += 1
    if depth != 0:
        raise _bad("unbalanced parentheses")
    inner = cur.t[cur.i : j]
    for part in _split_top_level(inner):
        _create_def(part, ctx)
    cur.i = j + 1
    _table_options(cur)


def _create_index(cur, ctx):
    cur.kw("INDEX")
    _name(cur)
    cur.kw("ON")
    _table_name(cur, ctx)
    _index_cols(cur)
    if not cur.done():
        raise _bad("trailing tokens in CREATE INDEX")


def _alter_table(cur, ctx):
    cur.kw("TABLE")
    _table_name(cur, ctx)
    rest = cur.t[cur.i :]
    cur.i = len(cur.t)
    parts = _split_top_level(rest)
    verdict = None
    for part in parts:
        try:
            _check_alter_clause(part, ctx)
        except Reject as exc:
            if exc.kind == DESTRUCTIVE:
                raise
            verdict = exc
    if verdict is not None:
        raise verdict


def _quoted_value(tok):
    """Value of a single-quoted string token, or None when it is not one."""
    if tok.kind != "str" or tok.val[0] != "'":
        return None
    return tok.val[1:-1].replace("''", "'")


def _bookkeeping(stmt, ctx):
    """Return True when the statement is exactly the expected alembic_version statement."""
    cur = _Cursor(stmt.tokens)
    first = cur.peek()
    if first is None or first.kind != "word":
        return False
    up = first.val.upper()
    if up == "UPDATE" and ctx.from_rev is not None:
        cur.next()
        if not (cur.peek() and ident_name(cur.peek()) and ident_name(cur.next()).lower() == "alembic_version"):
            return False
        cur.kw("SET")
        col = cur.next()
        if ident_name(col) is None or ident_name(col).lower() != "version_num":
            raise _bad("bookkeeping column")
        cur.punct("=")
        to_val = _quoted_value(cur.next())
        cur.kw("WHERE")
        t = cur.next()
        cur.punct(".")
        c = cur.next()
        if ident_name(t) is None or ident_name(t).lower() != "alembic_version" or ident_name(c) is None or ident_name(c).lower() != "version_num":
            raise _bad("bookkeeping where clause")
        cur.punct("=")
        from_val = _quoted_value(cur.next())
        if not cur.done():
            raise _bad("trailing tokens in bookkeeping statement")
        if to_val != ctx.to_rev or from_val != ctx.from_rev or not _ID_RE.fullmatch(to_val or "") or not _ID_RE.fullmatch(from_val or ""):
            raise _bad("bookkeeping pair differs from the step")
        return True
    if up == "INSERT" and ctx.from_rev is None:
        cur.next()
        cur.kw("INTO")
        t = cur.next()
        if ident_name(t) is None or ident_name(t).lower() != "alembic_version":
            return False
        cur.punct("(")
        col = cur.next()
        if ident_name(col) is None or ident_name(col).lower() != "version_num":
            raise _bad("bookkeeping column")
        cur.punct(")")
        cur.kw("VALUES")
        cur.punct("(")
        to_val = _quoted_value(cur.next())
        cur.punct(")")
        if not cur.done():
            raise _bad("trailing tokens in bookkeeping statement")
        if to_val != ctx.to_rev or not _ID_RE.fullmatch(to_val or ""):
            raise _bad("bookkeeping value differs from the step")
        return True
    return False


_ALEMBIC_CREATE_RE = re.compile(
    r"CREATETABLEalembic_version\(version_numVARCHAR\(32\)NOTNULL,CONSTRAINTalembic_version_pkcPRIMARYKEY\(version_num\)\)",
    re.IGNORECASE,
)


def _alembic_version_create(stmt, ctx):
    """The one table alembic creates by itself, accepted on an empty database only."""
    if ctx.from_rev is not None:
        return False
    flat = normalize(stmt.text).replace(" ", "")
    return bool(_ALEMBIC_CREATE_RE.fullmatch(flat))


def classify_statement(stmt, ctx):
    """Classify one split statement. Returns Verdict."""
    try:
        return _classify(stmt, ctx)
    except Reject as exc:
        return Verdict(exc.kind, exc.msg)
    except SqlError as exc:
        return Verdict(UNPARSABLE, str(exc))


def _classify(stmt, ctx):
    toks = stmt.tokens
    if not toks:
        raise _bad("empty statement")
    for tok in toks:
        if tok.kind == "word" and tok.val.upper() in _FORBIDDEN_WORDS:
            first = toks[0].val.upper() if toks[0].kind == "word" else ""
            if first in _DESTRUCTIVE_STATEMENT_WORDS:
                raise Reject(DESTRUCTIVE, first + " statement")
            raise _bad("forbidden word " + tok.val.upper())
    first = toks[0]
    if first.kind != "word":
        raise _bad("statement start")
    up = first.val.upper()
    if up in _DESTRUCTIVE_STATEMENT_WORDS:
        if _bookkeeping(stmt, ctx):
            return Verdict(BOOKKEEPING)
        raise Reject(DESTRUCTIVE, up + " statement")
    if _alembic_version_create(stmt, ctx):
        return Verdict(BOOKKEEPING, "alembic_version table")
    cur = _Cursor(toks)
    cur.next()
    if up == "CREATE":
        if cur.is_kw("TABLE"):
            _create_table(cur, ctx)
        elif cur.is_kw("INDEX"):
            _create_index(cur, ctx)
        else:
            raise _bad("CREATE form")
    elif up == "ALTER":
        _alter_table(cur, ctx)
    else:
        raise _bad("statement form")
    if not cur.done():
        raise _bad("trailing tokens")
    return Verdict(ADDITIVE)


def classify_text(text, ctx):
    """Classify every statement of a rendered text. Returns list[(Stmt, Verdict)]."""
    return [(s, classify_statement(s, ctx)) for s in split_statements(text)]
