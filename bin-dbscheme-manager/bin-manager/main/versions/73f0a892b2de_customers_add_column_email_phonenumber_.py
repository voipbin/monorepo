"""customers add column email phonenumber address

Revision ID: 73f0a892b2de
Revises: e59668c768ec
Create Date: 2023-09-02 02:23:59.856868

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '73f0a892b2de'
down_revision = 'e59668c768ec'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table customers add column email varchar(255) after detail;""")
    op.execute("""update customers set email = "";""")

    op.execute("""alter table customers add column phone_number varchar(255) after email;""")
    op.execute("""update customers set phone_number = "";""")

    op.execute("""alter table customers add column address varchar(1024) after phone_number;""")
    op.execute("""update customers set address = "";""")


def downgrade():
    op.execute("""alter table customers drop column email;""")
    op.execute("""alter table customers drop column phone_number;""")
    op.execute("""alter table customers drop column address;""")
