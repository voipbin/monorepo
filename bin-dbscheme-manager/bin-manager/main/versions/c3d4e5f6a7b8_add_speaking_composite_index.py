"""add speaking composite index for duplicate check

Revision ID: c3d4e5f6a7b8
Revises: b2c3d4e5f6a7
Create Date: 2026-02-14 01:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'c3d4e5f6a7b8'
down_revision = 'b2c3d4e5f6a7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE INDEX idx_tts_manager_speaking_ref_status
        ON tts_manager_speaking (customer_id, reference_type, reference_id, status);
    """)


def downgrade():
    op.execute("""
        DROP INDEX idx_tts_manager_speaking_ref_status
        ON tts_manager_speaking;
    """)
