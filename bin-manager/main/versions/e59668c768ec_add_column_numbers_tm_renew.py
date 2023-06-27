"""add column numbers tm_renew

Revision ID: e59668c768ec
Revises: 59932cd17004
Create Date: 2023-06-27 02:41:27.488736

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e59668c768ec'
down_revision = '59932cd17004'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table numbers add column tm_renew datetime(6) after tm_purchase;""")
    op.execute("""create index idx_numbers_tm_renew on numbers(tm_renew);""")

    op.execute("""update numbers set tm_renew = tm_purchase;""")


def downgrade():
    op.execute("""alter table numbers drop column tm_renew;""")
