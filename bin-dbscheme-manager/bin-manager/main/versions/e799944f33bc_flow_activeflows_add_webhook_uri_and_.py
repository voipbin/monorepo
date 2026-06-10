"""flow_activeflows_add_webhook_uri_and_webhook_method

Revision ID: e799944f33bc
Revises: 7ccfe8109c43
Create Date: 2026-06-10 22:46:41.828621

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e799944f33bc'
down_revision = '7ccfe8109c43'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE flow_activeflows ADD COLUMN webhook_uri VARCHAR(1024) AFTER on_complete_flow_id;""")
    op.execute("""ALTER TABLE flow_activeflows ADD COLUMN webhook_method VARCHAR(16) AFTER webhook_uri;""")


def downgrade():
    op.execute("""ALTER TABLE flow_activeflows DROP COLUMN webhook_method;""")
    op.execute("""ALTER TABLE flow_activeflows DROP COLUMN webhook_uri;""")
