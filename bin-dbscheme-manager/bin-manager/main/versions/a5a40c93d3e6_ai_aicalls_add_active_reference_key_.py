"""ai_aicalls_add_active_reference_key_unique

Revision ID: a5a40c93d3e6
Revises: 573756d87322
Create Date: 2026-07-09 10:24:01.420747

Adds active_reference_key to ai_aicalls (VOIP-1234 subtask 1) to enforce
"at most one active AIcall per (customer, reference_type, reference_id)"
at the DB level via a STORED generated column, mirroring the
contact_cases.open_peer_uk pattern (f718e26f2c44).

active_reference_key carries a value only when status is NOT
'terminated'/'terminating' AND reference_id is a real (non-zero) UUID;
terminated/terminating rows AND legacy rows with a zero-value
reference_id (uuid.UUID{} in Go, produced by older code paths that
never set a reference) all compute NULL, which MySQL treats as
distinct under UNIQUE, so any number of such rows for the same
(customer_id, reference_type, reference_id) may coexist while at most
one genuinely-active, genuinely-referenced row may exist.
SHA2(..., 256) (not a raw CONCAT) avoids silent truncation of an
oversized concatenation into a fixed-size BINARY column, which could
create false-positive uniqueness collisions between genuinely
different tuples.

The zero-UUID exclusion was added after the first upgrade attempt
against the production ai_aicalls table failed with a 1062 on
CREATE UNIQUE INDEX: 200+ pre-existing rows share
customer_id + reference_type='' + reference_id=0x00..00 (legacy
AIcalls created without a reference, some stuck in 'progressing'/
'initiating' for over a year -- a separate stale-cleanup issue,
tracked outside this migration). Those rows are not "the same active
AIcall re-created" in any meaningful sense; they are unrelated rows
that happen to share the degenerate all-zero key. Excluding
reference_id = 0x00..00 from the generated column (in addition to the
status check) makes the migration apply cleanly without needing any
data backfill/cleanup as a precondition, and does not weaken the
invariant we actually want to enforce (uniqueness of a *real*
reference).

Because that first upgrade attempt failed partway through -- MySQL DDL
statements commit individually and are not rolled back by a failed
statement later in the same op.execute() sequence -- ADD COLUMN
active_reference_key had already succeeded in production (with the
FIRST, pre-zero-UUID-exclusion definition) before CREATE UNIQUE INDEX
failed with 1062.

Two follow-up "fixes" (backfill, then a bounded retry loop around
backfill+CREATE INDEX) were attempted and both still failed with 1062
against the identical, deterministic hash value on every retry over
several separate runs spanning minutes -- with zero new rows written
to ai_aicalls in that window (the table's last row create/update
timestamp predates all of these attempts by over a week). That combination
-- same hash, every time, on a table receiving no new writes -- ruled out
a live-traffic race entirely. Root cause: upgrade()'s ADD COLUMN was
guarded with a bare "does the column exist" check (_column_exists),
which is true from the FIRST failed attempt onward. Every subsequent
retry saw the column already present and skipped ADD COLUMN entirely,
so the production column's GENERATED ALWAYS AS expression was still
the original one WITHOUT the zero-UUID exclusion (confirmed via
information_schema.columns.GENERATION_EXPRESSION against production).
The 200+ zero-reference legacy rows were therefore still colliding on
the unique index every time, deterministically, regardless of any
backfill of genuinely-duplicate real references (of which there were
none). upgrade() now checks the existing column's generation
expression against the current definition (via
_column_has_current_definition, matched on the literal zero-UUID hex
substring) and drops+recreates the column if it does not match, before
proceeding -- so a column left over from an earlier, pre-fix attempt
of this same migration gets brought current instead of silently kept.
The backfill and retry-loop scaffolding from the second and third
attempts is removed: with the column definition correct, the
zero-reference rows compute NULL (excluded) and there is no evidence
of any real, non-zero-UUID active duplicate in production, so
CREATE UNIQUE INDEX needs no retry loop and no backfill precondition.

A real-world spike (20 concurrent INSERTs against a MySQL 8.0
container) confirmed the optimistic-INSERT + DB-level unique index
combination alone is sufficient here (1 success / 19 duplicate-key
1062 / 0 deadlocks) -- unlike contact_cases, no SELECT...FOR UPDATE
pre-check or in-process keyed lock is needed for the create path.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a5a40c93d3e6'
down_revision = '573756d87322'
branch_labels = None
depends_on = None

# Substring unique to the CURRENT generated-column expression (the
# zero-UUID exclusion clause). Used to detect a stale column left over
# from an earlier, pre-fix attempt at this same migration.
_CURRENT_DEFINITION_MARKER = '00000000000000000000000000000000'


def _column_exists(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    return result.scalar() > 0


def _column_has_current_definition(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT GENERATION_EXPRESSION FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    row = result.first()
    if row is None:
        return False
    expression = row[0] or ''
    return _CURRENT_DEFINITION_MARKER in expression


def _index_exists(conn, table, index):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.statistics "
        "WHERE table_schema = DATABASE() AND table_name = :table AND index_name = :idx"
    ), {'table': table, 'idx': index})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()

    if _column_exists(conn, 'ai_aicalls', 'active_reference_key') \
            and not _column_has_current_definition(conn, 'ai_aicalls', 'active_reference_key'):
        # Leftover column from an earlier, pre-fix attempt at this
        # migration (created before the zero-UUID exclusion existed).
        # No index depends on it yet in that state, so it is safe to
        # drop and recreate with the current definition.
        op.execute("""
            ALTER TABLE ai_aicalls
            DROP COLUMN active_reference_key
        """)

    if not _column_exists(conn, 'ai_aicalls', 'active_reference_key'):
        op.execute("""
            ALTER TABLE ai_aicalls
            ADD COLUMN active_reference_key BINARY(32) GENERATED ALWAYS AS (
                IF(status NOT IN ('terminated', 'terminating')
                   AND reference_id != UNHEX('00000000000000000000000000000000'),
                   UNHEX(SHA2(CONCAT_WS('|', customer_id, reference_type, reference_id), 256)),
                   NULL)
            ) STORED
        """)

    if not _index_exists(conn, 'ai_aicalls', 'uq_aicall_active_reference_key'):
        op.execute("""
            CREATE UNIQUE INDEX uq_aicall_active_reference_key
            ON ai_aicalls(active_reference_key)
        """)


def downgrade():
    conn = op.get_bind()

    if _index_exists(conn, 'ai_aicalls', 'uq_aicall_active_reference_key'):
        op.execute("""
            DROP INDEX uq_aicall_active_reference_key ON ai_aicalls
        """)

    if _column_exists(conn, 'ai_aicalls', 'active_reference_key'):
        op.execute("""
            ALTER TABLE ai_aicalls
            DROP COLUMN active_reference_key
        """)
