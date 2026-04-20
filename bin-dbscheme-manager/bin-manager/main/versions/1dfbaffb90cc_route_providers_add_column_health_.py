"""route_providers add column health_status health_checked_at

Revision ID: 1dfbaffb90cc
Revises: 7a4eed7c79ed
Create Date: 2026-04-20 01:07:38.220386

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1dfbaffb90cc'
down_revision = '7a4eed7c79ed'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE route_providers
        ADD COLUMN health_status VARCHAR(64) NOT NULL DEFAULT 'unknown',
        ADD COLUMN health_checked_at DATETIME(6)
    """)


def downgrade():
    op.execute("""
        ALTER TABLE route_providers
        DROP COLUMN health_status,
        DROP COLUMN health_checked_at
    """)
