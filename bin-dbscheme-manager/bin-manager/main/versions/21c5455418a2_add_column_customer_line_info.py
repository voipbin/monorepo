"""add column customer line info

Revision ID: 21c5455418a2
Revises: aba24c9a1c3a
Create Date: 2022-06-07 12:47:10.483629

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '21c5455418a2'
down_revision = 'aba24c9a1c3a'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table customers add column line_secret text after webhook_uri;""")
    op.execute("""alter table customers add column line_token text after line_secret;""")


def downgrade():
    op.execute("""alter table customers drop column line_secret;""")
    op.execute("""alter table customers drop column line_token;""")
