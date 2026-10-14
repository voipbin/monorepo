"""Helpers that apply a whole stream to a scratch database and capture what it executes."""

import re

from alembic import command
from alembic.script import ScriptDirectory
from sqlalchemy import event
from sqlalchemy.engine import Engine

from classifier import classify as C
from classifier import source_gate
from classifier.render import NotRenderable, make_config, render_step
from classifier.replay import _TABLE_CHECK, _VERSION_READ
from classifier.sqltext import normalize


def chain_of(stream):
    script = ScriptDirectory.from_config(make_config(stream.root, stream.script_dir, stream.schema))
    return C._pending_chain(script, None)[0]


def additive_plans(stream, chain=None):
    """Plans of every revision that passes the source gate, renders and classifies additive."""
    plans = {}
    for rev in chain or chain_of(stream):
        with open(rev.path, "rb") as fh:
            if source_gate.check_upgrade(fh.read()):
                continue
        try:
            text = render_step(stream.root, stream.script_dir, stream.schema, rev.down_revision, rev.revision)
        except NotRenderable:
            continue
        reason, plan, _ = C.classify_rendered(text, stream.schema, rev.down_revision, rev.revision)
        if reason is None:
            plans[rev.revision] = plan
    return plans


def body_statements(captured):
    flat = [normalize(s, online=True) for s in captured]
    return [s for s in flat if s not in (_TABLE_CHECK, _VERSION_READ)]


_STEP_RE = re.compile(r"^(?:UPDATE alembic_version SET version_num='(\w+)'|INSERT INTO alembic_version \(version_num\) VALUES \('(\w+)'\))")


def apply_all(cfg, snapshot):
    """Upgrade a whole stream in one run. Returns {revision: (captured statements, schema before, schema after)}.

    The schema is snapshotted right before each bookkeeping statement, i.e. when the revision body has run.
    """
    steps = {}
    current = []
    state = {"before": snapshot()}

    def cb(conn, cursor, statement, parameters, context, executemany):
        current.append(statement)
        m = _STEP_RE.match(statement.strip())
        if m:
            rev = m.group(1) or m.group(2)
            after = snapshot()
            steps[rev] = (list(current), state["before"], after)
            state["before"] = after
            current.clear()

    event.listen(Engine, "before_cursor_execute", cb)
    try:
        command.upgrade(cfg, "head")
    finally:
        event.remove(Engine, "before_cursor_execute", cb)
    return steps


def schema_snapshot(connection, schema):
    """Columns (without ordinal position), indexes and tables of a schema."""
    cur = connection.cursor()
    cur.execute(
        "SELECT table_name, column_name, column_type, is_nullable, column_default, character_set_name, collation_name, column_comment, extra "
        "FROM information_schema.columns WHERE table_schema = %s",
        (schema,),
    )
    columns = {(r[0], r[1]): tuple(r[2:]) for r in cur.fetchall()}
    cur.execute("SELECT table_name, index_name, seq_in_index, column_name, non_unique FROM information_schema.statistics WHERE table_schema = %s", (schema,))
    indexes = {(r[0], r[1], r[2]): tuple(r[3:]) for r in cur.fetchall()}
    cur.execute("SELECT table_name FROM information_schema.tables WHERE table_schema = %s", (schema,))
    tables = {r[0] for r in cur.fetchall()}
    return tables, columns, indexes
