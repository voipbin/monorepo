"""fix wrong transcribe handle

Revision ID: 55bafce12bc5
Revises: d3765d22fa1d
Create Date: 2023-01-20 12:03:36.564939

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '55bafce12bc5'
down_revision = 'd3765d22fa1d'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table transcribes add streaming_ids json after direction;""")


def downgrade():
    op.execute("""alter table transcribes drop column streaming_ids;""")
