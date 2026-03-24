"""ai_aicalls drop column gender

Revision ID: 3783e1841f4b
Revises: 0c1a75da0b58
Create Date: 2026-03-24 13:49:03.279141

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '3783e1841f4b'
down_revision = '0c1a75da0b58'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN gender""")


def downgrade():
    op.execute("""ALTER TABLE ai_aicalls ADD COLUMN gender VARCHAR(255) NOT NULL DEFAULT ''""")
