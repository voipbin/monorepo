"""add conversation_messages column direction

Revision ID: 1c016f68f899
Revises: 122b2ba1b2b0
Create Date: 2022-06-24 12:06:58.636681

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1c016f68f899'
down_revision = '122b2ba1b2b0'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conversation_messages add direction varchar(255) default '' after conversation_id;""")


def downgrade():
    op.execute("""alter table conversation_messages drop column direction;""")
