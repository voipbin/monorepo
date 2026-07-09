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
active_reference_key had already succeeded in production before
CREATE UNIQUE INDEX failed with 1062. A naive retry of the original
unconditional upgrade() would hit "Duplicate column name" (1060) on
the ADD COLUMN this time instead. upgrade()/downgrade() now guard each
statement with information_schema existence checks (the same
_index_exists()-style pattern already established by
h2b3c4d5e6f7_billing_billings_idempotency_key_unique.py) so the
migration is idempotent and safe to retry from this partially-applied
state, or from a clean state, or after a downgrade.

A second production retry (after the zero-UUID fix above) still hit a
1062 on CREATE UNIQUE INDEX, this time with a real (non-zero, non-NULL)
hash value. Direct inspection confirmed there is currently no *steady
state* duplicate for any real (customer_id, reference_type,
reference_id) tuple -- ai_aicalls is a live, actively-written table,
and the CREATE UNIQUE INDEX scan can observe a transient window where
two rows are concurrently "active" for the same real reference (the
exact race this migration exists to prevent going forward) before one
of them is naturally cleaned up (idle-expire, normal termination,
etc.) moments later. Because DDL scan timing cannot be controlled and
the table cannot be quiesced for this migration, the fix is a
backfill step that runs immediately before CREATE UNIQUE INDEX: for
every group of currently-active rows sharing a real (customer_id,
reference_type, reference_id), keep only the most recently updated row
active and mark the rest 'terminated'. This makes CREATE UNIQUE INDEX
apply against a table with no coexisting active duplicates regardless
of what transient state produced them, without guessing at or
depending on the root cause of any individual duplicate. The backfill
only runs while the index does not yet exist (i.e. only during the
one-time transition to enforcing the constraint) since once the index
exists the constraint itself prevents new active duplicates from
accumulating.

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


def _column_exists(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    return result.scalar() > 0


def _index_exists(conn, table, index):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.statistics "
        "WHERE table_schema = DATABASE() AND table_name = :table AND index_name = :idx"
    ), {'table': table, 'idx': index})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()

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
        # One-time backfill: collapse any group of currently-active rows
        # sharing a real reference down to a single active row (keep the
        # most recently updated, terminate the rest) so the unique index
        # below can be created cleanly regardless of how those active
        # duplicates arose.
        op.execute("""
            UPDATE ai_aicalls a
            JOIN (
                SELECT id, ROW_NUMBER() OVER (
                    PARTITION BY customer_id, reference_type, reference_id
                    ORDER BY tm_update DESC, tm_create DESC, id DESC
                ) AS rn
                FROM ai_aicalls
                WHERE status NOT IN ('terminated', 'terminating')
                  AND reference_id != UNHEX('00000000000000000000000000000000')
            ) ranked ON a.id = ranked.id AND ranked.rn > 1
            SET a.status = 'terminated',
                a.tm_end = COALESCE(a.tm_end, NOW(6)),
                a.tm_update = NOW(6)
        """)

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
