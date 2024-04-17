"""extensions add column domain_name username

Revision ID: 6673c8ced2a2
Revises: 67290d201bdd
Create Date: 2023-09-20 01:52:37.214667

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '6673c8ced2a2'
down_revision = '67290d201bdd'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table extensions add column domain_name varchar(255) after extension;""")
    op.execute("""update extensions set domain_name = "";""")

    op.execute("""alter table extensions add column username varchar(255) after domain_name;""")
    op.execute("""update extensions set username = "";""")

    op.execute("""alter table extensions drop column domain_id;""")


def downgrade():
    op.execute("""alter table extensions add column domain_id binary(16) after detail;""")
    op.execute("""update extensions set domain_id = "";""")

    op.execute("""alter table extensions drop column domain_name;""")
    op.execute("""alter table extensions drop column username;""")
