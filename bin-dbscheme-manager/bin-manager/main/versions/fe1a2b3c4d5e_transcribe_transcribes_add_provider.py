"""transcribe_transcribes add provider column

Revision ID: fe1a2b3c4d5e
Revises: ebbb033028db
Create Date: 2026-02-24 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'fe1a2b3c4d5e'
down_revision = 'ebbb033028db'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE transcribe_transcribes ADD COLUMN provider VARCHAR(16) DEFAULT '' AFTER direction;""")


def downgrade():
    op.execute("""ALTER TABLE transcribe_transcribes DROP COLUMN provider;""")
