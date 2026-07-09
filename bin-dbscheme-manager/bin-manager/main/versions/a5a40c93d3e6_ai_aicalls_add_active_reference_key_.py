"""ai_aicalls_add_active_reference_key_unique

Revision ID: a5a40c93d3e6
Revises: 573756d87322
Create Date: 2026-07-09 10:24:01.420747

Adds active_reference_key to ai_aicalls (VOIP-1234 subtask 1) to enforce
"at most one active AIcall per (customer, reference_type, reference_id)"
at the DB level via a STORED generated column, mirroring the
contact_cases.open_peer_uk pattern (f718e26f2c44).

active_reference_key carries a value only when status is NOT
'terminated'/'terminating'; terminated/terminating rows compute NULL,
which MySQL treats as distinct under UNIQUE, so any number of
terminated rows for the same (customer_id, reference_type,
reference_id) may coexist while at most one active row may exist.
SHA2(..., 256) (not a raw CONCAT) avoids silent truncation of an
oversized concatenation into a fixed-size BINARY column, which could
create false-positive uniqueness collisions between genuinely
different tuples.

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
            IF(status NOT IN ('terminated', 'terminating'),
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
