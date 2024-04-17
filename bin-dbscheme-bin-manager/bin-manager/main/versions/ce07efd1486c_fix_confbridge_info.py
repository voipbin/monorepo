"""fix confbridge info

Revision ID: ce07efd1486c
Revises: fd9ebdbd7acc
Create Date: 2023-01-21 16:53:27.167889

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ce07efd1486c'
down_revision = 'fd9ebdbd7acc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table confbridges add customer_id binary(16) default '' after id;""")
    op.execute("""alter table calls drop column asterisk_id;""")


def downgrade():
    op.execute("""alter table confbridges drop column customer_id;""")
    op.execute("""alter table calls add asterisk_id varchar(255) default '' after id;""")

