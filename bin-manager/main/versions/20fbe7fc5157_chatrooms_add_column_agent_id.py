"""chatrooms add column agent_id

Revision ID: 20fbe7fc5157
Revises: 9ebf10b79e54
Create Date: 2024-03-05 00:00:44.642963

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '20fbe7fc5157'
down_revision = '9ebf10b79e54'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table chatrooms add column agent_id binary(16) after customer_id;""")
    op.execute("""update chatrooms set agent_id=owner_id;""")

    op.execute("""alter table messagechatrooms add column agent_id binary(16) after customer_id;""")
    op.execute("""update messagechatrooms t1
        inner join chatrooms t2 
            on t1.chatroom_id = t2.id
        set
            t1.agent_id = t2.agent_id;""")


def downgrade():
    op.execute("""alter table chatrooms drop column agent_id;""")
    op.execute("""alter table messagechatrooms drop column agent_id;""")
