"""Add queues execute

Revision ID: e93a0bfa95fe
Revises: e10a36a73850
Create Date: 2022-05-12 15:39:18.548220

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e93a0bfa95fe'
down_revision = 'e10a36a73850'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queues add column execute varchar(255) after tag_ids;""")


def downgrade():
    op.execute("""alter table queues drop column execute;""")
