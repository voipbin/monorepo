"""storage_files_add_type_column

Revision ID: fd2a3b4c5d6e
Revises: fc1a2b3d4e5f
Create Date: 2026-03-14 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fd2a3b4c5d6e'
down_revision = 'fc1a2b3d4e5f'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE storage_files
        ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT '' AFTER reference_id
    """)


def downgrade():
    op.execute("""
        ALTER TABLE storage_files
        DROP COLUMN type
    """)
