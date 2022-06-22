"""add conversation columnds transaction_id

Revision ID: 122b2ba1b2b0
Revises: caae1605498b
Create Date: 2022-06-22 13:23:48.617046

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '122b2ba1b2b0'
down_revision = 'caae1605498b'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conversation_conversations add column source json after reference_id;""")

    op.execute("""alter table conversation_messages drop column source_target;""")
    op.execute("""alter table conversation_messages add column transaction_id varchar(255) default '' after reference_id;""")
    op.execute("""alter table conversation_messages add column source json after transaction_id;""")



def downgrade():
    op.execute("""alter table conversation_messages drop column source;""")

    op.execute("""alter table conversation_conversations add column source_target varchar(255) default '' after reference_id;""")
    op.execute("""alter table conversation_messages drop column transaction_id;""")
    op.execute("""alter table conversation_messages drop column source;""")

