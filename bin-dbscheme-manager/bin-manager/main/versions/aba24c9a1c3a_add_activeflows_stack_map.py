"""Add activeflows stack_map

Revision ID: aba24c9a1c3a
Revises: 476c036bd925
Create Date: 2022-05-18 18:04:16.587290

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'aba24c9a1c3a'
down_revision = '476c036bd925'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table activeflows drop column actions;""")

    op.execute("""alter table activeflows add column stack_map json after reference_id;""")
    op.execute("""alter table activeflows add column current_stack_id binary(16) default '' after stack_map;""")
    op.execute("""alter table activeflows add column forward_stack_id binary(16) default '' after execute_count;""")


def downgrade():
    op.execute("""alter table activeflows add column actions json after forward_action_id;""")

    op.execute("""alter table activeflows drop stack_map actions;""")
    op.execute("""alter table activeflows drop current_stack_id actions;""")
    op.execute("""alter table activeflows drop forward_stack_id actions;""")

