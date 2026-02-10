"""number_add_type_column

Revision ID: fc1a2b3d4e5f
Revises: fb5b8d1359c9
Create Date: 2026-02-10 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fc1a2b3d4e5f'
down_revision = 'fb5b8d1359c9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE number_numbers
            ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT 'normal' AFTER number;
    """)
    op.execute("""
        CREATE INDEX idx_number_numbers_type ON number_numbers (type);
    """)


def downgrade():
    op.execute("""DROP INDEX idx_number_numbers_type ON number_numbers;""")
    op.execute("""ALTER TABLE number_numbers DROP COLUMN type;""")
