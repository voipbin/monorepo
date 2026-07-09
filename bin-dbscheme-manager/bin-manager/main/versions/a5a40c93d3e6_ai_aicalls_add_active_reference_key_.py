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


def upgrade():
    op.execute("""
        ALTER TABLE ai_aicalls
        ADD COLUMN active_reference_key BINARY(32) GENERATED ALWAYS AS (
            IF(status NOT IN ('terminated', 'terminating')
               AND reference_id != UNHEX('00000000000000000000000000000000'),
               UNHEX(SHA2(CONCAT_WS('|', customer_id, reference_type, reference_id), 256)),
               NULL)
        ) STORED
    """)

    op.execute("""
        CREATE UNIQUE INDEX uq_aicall_active_reference_key
        ON ai_aicalls(active_reference_key)
    """)


def downgrade():
    op.execute("""
        DROP INDEX uq_aicall_active_reference_key ON ai_aicalls
    """)

    op.execute("""
        ALTER TABLE ai_aicalls
        DROP COLUMN active_reference_key
    """)
