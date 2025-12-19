"""add column activeflows status

Revision ID: 453add0eb376
Revises: 25b1ed87e878
Create Date: 2023-03-23 03:07:04.637821

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '453add0eb376'
down_revision = '25b1ed87e878'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table activeflows add status varchar(16) after customer_id;""")


def downgrade():
    op.execute("""alter table activeflows drop column status;""")
