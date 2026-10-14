"""Quote-aware SQL tokenizer, statement splitter and normalize().

Everything here is deliberately narrow: anything outside the small, fully
specified grammar raises SqlError (kind "unparsable") instead of being guessed
at. The tokenizer is shared by the statement parser and by normalize(), so the
rendered text and the executed text are judged by the same rules.
"""

import re
import unicodedata
from dataclasses import dataclass

# Whitespace is exactly this ASCII set. VT, FF and NBSP are not whitespace
# here: the server accepts some of them in the middle of a statement, so they
# are rejected instead of being tolerated.
ASCII_WS = " \t\n\r"

# SQLAlchemy bind patterns without their lookahead and with a wider net. It is
# applied to the text including quoted strings and comments (the safe direction).
BIND_RE = re.compile(r"(?<![\w\\]):[\w$]+|\\:")

_WORD_CHARS = frozenset("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_$")
_COMMENT_BODY_CATEGORIES = frozenset("LMNPS")
_SIMPLE_IDENT_RE = re.compile(r"[A-Za-z0-9_]+")


class SqlError(Exception):
    """Text outside the accepted grammar."""

    kind = "unparsable"


@dataclass(frozen=True)
class Tok:
    kind: str  # word | ident | str | punct | comment
    val: str
    start: int
    end: int

    @property
    def upper(self):
        return self.val.upper() if self.kind == "word" else None


def replace_tabs(text):
    """The offline renderer replaces every tab by four spaces, quoted or not."""
    return text.replace("\t", "    ")


def _check_comment_body(body):
    for ch in body:
        if ch == " ":
            continue
        if unicodedata.category(ch)[0] not in _COMMENT_BODY_CATEGORIES:
            raise SqlError("comment contains a character that is not visible text")


def tokenize(text, keep_comments=False):
    """Tokenize text. The text must already have tabs replaced (replace_tabs)."""
    tokens = []
    i = 0
    n = len(text)
    while i < n:
        c = text[i]
        if c in ASCII_WS:
            i += 1
            continue
        if c == "-" and text.startswith("--", i):
            if text[i + 2 : i + 3] != " ":
                raise SqlError("a -- comment must be followed by a space")
            nl = text.find("\n", i)
            if nl < 0:
                raise SqlError("a -- comment must be closed by a line feed")
            _check_comment_body(text[i + 3 : nl])
            if keep_comments:
                tokens.append(Tok("comment", text[i:nl], i, nl))
            i = nl + 1
            continue
        if c == "#":
            raise SqlError("# comments are not accepted")
        if c == "/" and text.startswith("/*", i):
            raise SqlError("/* is not accepted")
        if c in ("'", '"'):
            j = i + 1
            while True:
                if j >= n:
                    raise SqlError("unterminated string")
                d = text[j]
                if d == "\\":
                    raise SqlError("backslash in a string literal")
                if d == "\x00":
                    raise SqlError("NUL in a string literal")
                if d == c:
                    if text[j + 1 : j + 2] == c:
                        j += 2
                        continue
                    break
                j += 1
            tokens.append(Tok("str", text[i : j + 1], i, j + 1))
            i = j + 1
            continue
        if c == "`":
            j = i + 1
            while True:
                if j >= n:
                    raise SqlError("unterminated identifier quote")
                d = text[j]
                if d == "\x00":
                    raise SqlError("NUL in an identifier")
                if d == "`":
                    if text[j + 1 : j + 2] == "`":
                        j += 2
                        continue
                    break
                j += 1
            tokens.append(Tok("ident", text[i : j + 1], i, j + 1))
            i = j + 1
            continue
        if c in _WORD_CHARS:
            j = i + 1
            while j < n and text[j] in _WORD_CHARS:
                j += 1
            tokens.append(Tok("word", text[i:j], i, j))
            i = j
            continue
        if " " <= c <= "~":
            tokens.append(Tok("punct", c, i, i + 1))
            i += 1
            continue
        raise SqlError("character outside the accepted set")
    return tokens


def ident_name(tok):
    """Name of a word or backticked identifier token, or None if not a simple name."""
    if tok.kind == "word":
        return tok.val if _SIMPLE_IDENT_RE.fullmatch(tok.val) else None
    if tok.kind == "ident":
        inner = tok.val[1:-1]
        return inner if _SIMPLE_IDENT_RE.fullmatch(inner) else None
    return None


@dataclass(frozen=True)
class Stmt:
    text: str  # statement text with comments kept, without the terminating ';'
    tokens: tuple  # tokens without comments


def split_statements(text):
    """Split rendered text on top-level ';'. Empty statements are ignored."""
    text = replace_tabs(text)
    stmts = []
    cur = []
    begin = 0
    for tok in tokenize(text):
        if tok.kind == "punct" and tok.val == ";":
            if cur:
                stmts.append(Stmt(text[begin : tok.start], tuple(cur)))
            cur = []
            begin = tok.end
        else:
            cur.append(tok)
    if cur:
        stmts.append(Stmt(text[begin:], tuple(cur)))
    return stmts


def normalize(text, online=False):
    """Canonical form used to compare a rendered statement with an executed one.

    Both sides use the same function. The online side arrives pyformat-escaped,
    so "%%" is turned back into "%" there only.
    """
    text = replace_tabs(text)
    if online:
        text = text.replace("%%", "%")
    tokens = tokenize(text)
    out = []
    prev = None
    for tok in tokens:
        if tok.kind == "ident":
            name = ident_name(tok)
            piece = name if name is not None else tok.val
        else:
            piece = tok.val
        if prev is not None:
            gap = text[prev.end : tok.start]
            if gap:
                tight = (prev.kind == "punct" and prev.val in "(),") or (
                    tok.kind == "punct" and tok.val in "(),"
                )
                if not tight:
                    out.append(" ")
        out.append(piece)
        prev = tok
    while out and out[-1] in (";", " "):
        out.pop()
    return "".join(out)
