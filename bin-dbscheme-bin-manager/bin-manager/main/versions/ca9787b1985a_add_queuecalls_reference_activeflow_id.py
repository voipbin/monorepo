"""add queuecalls reference_activeflow_id

Revision ID: ca9787b1985a
Revises: d07e532d2a28
Create Date: 2022-03-29 20:32:05.077486

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ca9787b1985a'
down_revision = 'd07e532d2a28'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queuecalls add reference_activeflow_id binary(16) after reference_id;""")
    op.execute("""create index idx_queuecalls_reference_activeflow_id on queuecalls(reference_activeflow_id);""")


def downgrade():
    op.execute("""alter table queuecalls drop column reference_activeflow_id;""")
