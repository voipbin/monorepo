"""add confbridge transcribe

Revision ID: fd9ebdbd7acc
Revises: 55bafce12bc5
Create Date: 2023-01-20 22:37:59.380243

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'fd9ebdbd7acc'
down_revision = '55bafce12bc5'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conferences add transcribe_id binary(16) default '' after recording_ids;""")
    op.execute("""alter table conferences add transcribe_ids json after transcribe_id;""")


def downgrade():
    op.execute("""alter table conferences drop column transcribe_id;""")
    op.execute("""alter table conferences drop column transcribe_ids;""")
