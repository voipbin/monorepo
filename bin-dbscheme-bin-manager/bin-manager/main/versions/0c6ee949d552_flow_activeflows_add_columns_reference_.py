"""flow_activeflows add columns reference_activeflow_id

Revision ID: 0c6ee949d552
Revises: 7c00d1572ff0
Create Date: 2025-03-24 00:13:53.589994

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0c6ee949d552'
down_revision = '7c00d1572ff0'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table flow_activeflows add column reference_activeflow_id binary(16) after reference_id;""")
    op.execute("""update flow_activeflows set reference_activeflow_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")


def downgrade():
    op.execute("""alter table flow_activeflows drop column reference_activeflow_id;""")
