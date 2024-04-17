"""remove transcribes webhook

Revision ID: 5c2ed766c8ff
Revises: 1a8a38f9c167
Create Date: 2022-02-05 03:29:36.829347

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '5c2ed766c8ff'
down_revision = '1a8a38f9c167'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table transcribes drop column webhook_method;""")
    op.execute("""alter table transcribes drop column webhook_uri;""")
    op.execute("""alter table transcribes add direction varchar(255) after language;""")


def downgrade():
    op.execute("""alter table transcribes add webhook_method varchar(255) after language;""")
    op.execute("""alter table transcribes add webhook_uri varchar(1023) after webhook_method;""")
    op.execute("""alter table transcribes drop column direction;""")
