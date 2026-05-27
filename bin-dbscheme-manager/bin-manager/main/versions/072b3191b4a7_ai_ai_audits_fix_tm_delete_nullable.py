"""ai_ai_audits_fix_tm_delete_nullable

Revision ID: 072b3191b4a7
Revises: aed365a9e29e
Create Date: 2026-05-27 16:19:10.744968

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '072b3191b4a7'
down_revision = 'aed365a9e29e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE ai_ai_audits
        MODIFY tm_delete DATETIME(6) NULL DEFAULT NULL
    """)
    op.execute("""
        UPDATE ai_ai_audits
        SET tm_delete = NULL
        WHERE tm_delete = '9999-01-01 00:00:00.000000'
    """)


def downgrade():
    op.execute("""
        UPDATE ai_ai_audits
        SET tm_delete = '9999-01-01 00:00:00.000000'
        WHERE tm_delete IS NULL
    """)
    op.execute("""
        ALTER TABLE ai_ai_audits
        MODIFY tm_delete DATETIME(6) NOT NULL DEFAULT '9999-01-01 00:00:00.000000'
    """)
