"""webchat_sessions_add_column_page_url

Revision ID: 04b99363284c
Revises: b41d1b2317af
Create Date: 2026-07-22 11:45:01.636791

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '04b99363284c'
down_revision = 'b41d1b2317af'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE webchat_sessions ADD COLUMN page_url VARCHAR(2048) NULL;""")


def downgrade():
    op.execute("""ALTER TABLE webchat_sessions DROP COLUMN page_url;""")
