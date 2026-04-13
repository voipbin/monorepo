"""call_manager_groupcalls add column anonymous

Add anonymous column to call_manager_groupcalls table to persist the anonymous
caller ID option across linear ring retries. Without this column, the anonymous
flag is lost when dialNextDestination creates chained groupcalls/calls.

Tri-state values: "yes", "no", "auto" (or empty string for default/auto).

Revision ID: 7a4eed7c79ed
Revises: 8268e08fecd4
Create Date: 2026-04-13 21:47:39.223671

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '7a4eed7c79ed'
down_revision = '8268e08fecd4'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE call_manager_groupcalls
        ADD COLUMN anonymous VARCHAR(10) NOT NULL DEFAULT ''
        AFTER answer_method
    """)


def downgrade():
    op.execute("""
        ALTER TABLE call_manager_groupcalls
        DROP COLUMN anonymous
    """)
