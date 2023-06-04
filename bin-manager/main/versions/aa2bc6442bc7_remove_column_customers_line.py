"""remove column customers line

Revision ID: aa2bc6442bc7
Revises: 0744fdb96a63
Create Date: 2023-06-05 00:58:52.381452

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'aa2bc6442bc7'
down_revision = '0744fdb96a63'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table customers drop column line_secret;""")
    op.execute("""alter table customers drop column line_token;""")


def downgrade():
    op.execute("""alter table customers add column line_secret text after webhook_uri;""")
    op.execute("""alter table customers add column line_token text after line_secret;""")

