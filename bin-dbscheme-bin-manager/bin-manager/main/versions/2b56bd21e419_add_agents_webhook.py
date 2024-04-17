"""add agents webhook

Revision ID: 2b56bd21e419
Revises: dfb5f6effd7a
Create Date: 2022-01-20 12:33:54.127629

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '2b56bd21e419'
down_revision = 'dfb5f6effd7a'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table agents add column webhook_method varchar(255) default '' after detail;""")
    op.execute("""alter table agents add column webhook_uri varchar(1023) default '' after webhook_method;""")


def downgrade():
    op.execute("""alter table agents drop column webhook_method;""")
    op.execute("""alter table agents drop column webhook_uri;""")
