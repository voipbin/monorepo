"""timeline_analyses_add_tables

Revision ID: a63b82d73655
Revises: e234a24addec
Create Date: 2026-06-23 08:42:28.603019

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a63b82d73655'
down_revision = 'e234a24addec'
branch_labels = None
depends_on = None


def upgrade():
    # Live table: exactly one row per activeflow.
    # NULL timestamp convention (see 071504ef41d0_timestamp_sentinel_to_null);
    # single-column UNIQUE(activeflow_id) enforces "one analysis per activeflow".
    # Delete is a hard delete (no tm_delete column); re-analyze resets the row
    # in place. There is no history/archive table (the analysis is a reproducible
    # derivative of the timeline, so superseded verdicts are not retained).
    op.execute("""
        CREATE TABLE IF NOT EXISTS timeline_analyses (
            id            BINARY(16)   NOT NULL,
            customer_id   BINARY(16)   NOT NULL,
            activeflow_id BINARY(16)   NOT NULL,

            status        VARCHAR(32)  NOT NULL,
            result        JSON         NULL,
            model         VARCHAR(255) NOT NULL DEFAULT '',
            error         TEXT         NULL,

            tm_create     DATETIME(6)  NOT NULL,
            tm_update     DATETIME(6)  NULL,

            PRIMARY KEY (id),
            UNIQUE KEY uq_timeline_analyses_activeflow (activeflow_id),
            INDEX idx_timeline_analyses_customer_create (customer_id, tm_create)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS timeline_analyses")
