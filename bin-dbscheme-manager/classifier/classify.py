"""Classification of a pending revision range of one Alembic stream."""

import ast
import hashlib
import os
from dataclasses import dataclass, field

from alembic.script import ScriptDirectory

from . import source_gate
from .render import NotRenderable, make_config, render_step
from .source_gate import DATA_DEPENDENT, UNPARSABLE
from .sqltext import BIND_RE, SqlError, normalize, split_statements
from .statements import ADDITIVE, BOOKKEEPING, DESTRUCTIVE, Context, classify_statement

NOT_RENDERABLE = "not_renderable"

# Reason priority over a whole range.
PRIORITY = (DATA_DEPENDENT, NOT_RENDERABLE, UNPARSABLE, DESTRUCTIVE)

SERVICE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
EXCEPTIONS_FILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "module_level_exceptions.txt")


@dataclass(frozen=True)
class Stream:
    name: str  # schema name, also the stream key
    root: str  # directory with alembic.ini.sample
    script_dir: str  # relative to root: env.py and versions/

    @property
    def schema(self):
        return self.name

    @property
    def versions_dir(self):
        return os.path.join(self.root, self.script_dir, "versions")


def default_streams(service_dir=SERVICE_DIR):
    return (
        Stream("bin_manager", os.path.join(service_dir, "bin-manager"), "main"),
        Stream("asterisk", os.path.join(service_dir, "asterisk_config"), "config"),
    )


class MultipleHeads(Exception):
    def __init__(self, heads):
        super().__init__("multiple heads")
        self.heads = heads


class UnknownRevision(Exception):
    pass


@dataclass
class StepPlan:
    revision: str
    down_revision: object
    statements: list = field(default_factory=list)  # normalized rendered statements, bookkeeping last


@dataclass
class RangeResult:
    klass: str  # na | additive | needs_human
    reason: object = None  # one of PRIORITY when needs_human
    blocked: list = field(default_factory=list)
    pending: object = 0  # int, or None when the chain-wide scan stopped first
    plans: list = field(default_factory=list)  # StepPlan list when additive
    findings: dict = field(default_factory=dict)  # revision -> list[(reason, message)]


def load_exceptions(path=EXCEPTIONS_FILE):
    """Set of sha256 hex digests of files exempted from the module-level rules."""
    out = set()
    if not os.path.exists(path):
        return out
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            out.add(line.split()[0].lower())
    return out


def sha256_file(data):
    return hashlib.sha256(data).hexdigest()


def _revision_id(data):
    """The revision id a file declares, or None when it cannot be resolved."""
    try:
        tree = ast.parse(data)
    except (SyntaxError, ValueError, RecursionError, MemoryError):
        return None
    for stmt in tree.body:
        if isinstance(stmt, (ast.Assign, ast.AnnAssign)):
            target = stmt.targets[0] if isinstance(stmt, ast.Assign) else stmt.target
            if isinstance(target, ast.Name) and target.id == "revision":
                v = stmt.value
                if isinstance(v, ast.Constant) and isinstance(v.value, str):
                    return v.value
    return None


def chain_scan(stream, exceptions):
    """Chain-wide source scan (module-level, encoding and listing rules).

    Returns dict revision-id-or-"chain" -> list of messages. Empty means clean.
    """
    found = {}
    for name, viol in source_gate.check_directory(stream.versions_dir):
        found.setdefault("chain", []).append(viol.msg)
    with os.scandir(stream.versions_dir) as it:
        entries = sorted(it, key=lambda e: e.name)
    for entry in entries:
        if not entry.is_file(follow_symlinks=False) or not source_gate.FILE_NAME_RE.fullmatch(entry.name):
            continue
        with open(entry.path, "rb") as fh:
            data = fh.read()
        viols = source_gate.check_module_level(data)
        digest = sha256_file(data)
        remaining = [v for v in viols if not (v.exemptible and digest in exceptions)]
        if remaining:
            key = _revision_id(data) or "chain"
            found.setdefault(key, []).extend(v.msg for v in remaining)
    return found


def _pending_chain(script, from_rev):
    """Linear walk from the head down to from_rev. Returns (chain oldest first, complete).

    A revision with a tuple down_revision (a merge) ends the walk: the chain is not linear,
    and that revision is reported as unparsable by the source gate.
    """
    heads = script.get_heads()
    if len(heads) != 1:
        raise MultipleHeads(sorted(heads))
    chain = []
    complete = True
    rev = script.get_revision(heads[0])
    while rev is not None and rev.revision != from_rev:
        chain.append(rev)
        down = rev.down_revision
        if down is None:
            rev = None
        elif isinstance(down, str):
            rev = script.get_revision(down)
        else:
            rev = None
            complete = False
    if from_rev is not None and rev is None and complete:
        raise UnknownRevision("current revision is not in the chain")
    chain.reverse()
    return chain, complete


def classify_rendered(text, schema, down, rev):
    """Classify the rendered text of one step. Returns (reason or None, StepPlan, messages)."""
    msgs = []
    if BIND_RE.search(text):
        return UNPARSABLE, None, ["bind-style parameter in rendered text"]
    try:
        stmts = split_statements(text)
    except SqlError as exc:
        return UNPARSABLE, None, [str(exc)]
    ctx = Context(schema=schema, from_rev=down, to_rev=rev)
    kinds = []
    for s in stmts:
        v = classify_statement(s, ctx)
        kinds.append(v.kind)
        if v.kind not in (ADDITIVE, BOOKKEEPING):
            msgs.append(v.kind + ": " + v.detail)
    stamps = [i for i, s in enumerate(stmts) if kinds[i] == BOOKKEEPING and classify_statement(s, ctx).detail == ""]
    tables = [i for i, s in enumerate(stmts) if kinds[i] == BOOKKEEPING and classify_statement(s, ctx).detail != ""]
    if UNPARSABLE in kinds:
        return UNPARSABLE, None, msgs
    if DESTRUCTIVE in kinds:
        return DESTRUCTIVE, None, msgs
    if len(stamps) != 1 or stamps[0] != len(stmts) - 1 or len(tables) > 1 or (tables and tables[0] != 0):
        return UNPARSABLE, None, ["bookkeeping statement is not exactly once, last"]
    try:
        plan = StepPlan(rev, down, [normalize(s.text) for s in stmts])
    except SqlError as exc:
        return UNPARSABLE, None, [str(exc)]
    return None, plan, msgs


def classify_range(stream, from_rev, exceptions=None):
    """Classify the pending range (from_rev, head] of one stream. from_rev None means an empty database."""
    if exceptions is None:
        exceptions = load_exceptions()
    scan = chain_scan(stream, exceptions)
    if scan:
        return RangeResult(
            "needs_human",
            DATA_DEPENDENT,
            blocked=sorted(scan),
            pending=None,
            findings={k: [(DATA_DEPENDENT, m) for m in v] for k, v in scan.items()},
        )
    script = ScriptDirectory.from_config(make_config(stream.root, stream.script_dir, stream.schema))
    chain, _ = _pending_chain(script, from_rev)
    if not chain:
        return RangeResult("na", pending=0)
    reasons = {}
    findings = {}
    plans = []
    for rev in chain:
        with open(rev.path, "rb") as fh:
            data = fh.read()
        viols = source_gate.check_upgrade(data)
        if viols:
            reasons[rev.revision] = DATA_DEPENDENT if any(v.reason == DATA_DEPENDENT for v in viols) else UNPARSABLE
            findings[rev.revision] = [(v.reason, v.msg) for v in viols]
            continue
        down = rev.down_revision
        try:
            text = render_step(stream.root, stream.script_dir, stream.schema, down, rev.revision)
        except NotRenderable as exc:
            reasons[rev.revision] = NOT_RENDERABLE
            findings[rev.revision] = [(NOT_RENDERABLE, str(exc))]
            continue
        reason, plan, msgs = classify_rendered(text, stream.schema, down, rev.revision)
        if reason:
            reasons[rev.revision] = reason
            findings[rev.revision] = [(reason, m) for m in msgs]
        else:
            plans.append(plan)
    if reasons:
        top = next(r for r in PRIORITY if r in reasons.values())
        return RangeResult("needs_human", top, blocked=[r.revision for r in chain if r.revision in reasons], pending=len(chain), findings=findings)
    return RangeResult("additive", pending=len(chain), plans=plans)


def merge_results(results):
    """Merge the per-stream results: one needs_human blocks both."""
    blocked = [r for res in results for r in res.blocked]
    for reason in PRIORITY:
        if any(res.reason == reason for res in results):
            return "needs_human", reason, blocked
    if any(res.klass == "additive" for res in results):
        return "additive", None, []
    return "na", None, []
