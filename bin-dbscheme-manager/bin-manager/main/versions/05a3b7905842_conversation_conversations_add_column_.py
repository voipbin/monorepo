"""conversation_conversations add column owner_type owner_id

Revision ID: 05a3b7905842
Revises: dcebea23280e
Create Date: 2024-07-08 23:43:15.372982

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '05a3b7905842'
down_revision = 'dcebea23280e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conversation_conversations add column owner_type varchar(255) after customer_id;""")
    op.execute("""alter table conversation_conversations add column owner_id binary(16) after owner_type;""")
    op.execute("""create index idx_conversation_conversations_owner_id on conversation_conversations(owner_id);""")
    op.execute("""update conversation_conversations set owner_type = "";""")
    op.execute("""update conversation_conversations set owner_id = "";""")

def downgrade():
    op.execute("""alter table conversation_conversations drop column owner_type;""")
    op.execute("""alter table conversation_conversations drop column owner_id;""")
