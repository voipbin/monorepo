"""campaigncalls add column tm_delete

Revision ID: 9ebf10b79e54
Revises: b39053ce4bd6
Create Date: 2024-02-24 22:55:17.644854

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9ebf10b79e54'
down_revision = 'b39053ce4bd6'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table campaigncalls add column tm_delete datetime(6) after tm_update;""")
    op.execute("""update campaigncalls SET tm_delete = '9999-01-01 00:00:00.000000'""")


def downgrade():
    op.execute("""alter table campaigncalls drop column tm_delete;""")
