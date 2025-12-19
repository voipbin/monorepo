"""add column confbridges flags

Revision ID: 328dce97f0d8
Revises: 10f7389f7db9
Create Date: 2023-04-10 02:49:46.452031

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '328dce97f0d8'
down_revision = '10f7389f7db9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table confbridges add flags json after bridge_id;""")


def downgrade():
    op.execute("""alter table confbridges drop column flags;""")

