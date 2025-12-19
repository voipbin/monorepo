"""chats update column agentid

Revision ID: dcebea23280e
Revises: a68766c6b227
Create Date: 2024-07-01 00:25:36.635995

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'dcebea23280e'
down_revision = 'a68766c6b227'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table chatrooms change owner_id room_owner_id binary(16);""")
    op.execute("""alter table chatrooms change agent_id owner_id binary(16);""")
    op.execute("""alter table chatrooms add column owner_type varchar(255) after customer_id;""")
    op.execute("""create index idx_chatrooms_room_owner_id on chatrooms(room_owner_id);""")
    op.execute("""update chatrooms SET owner_type = ''""")

    op.execute("""alter table chats change owner_id room_owner_id binary(16);""")

    op.execute("""alter table messagechatrooms change agent_id owner_id binary(16);""")
    op.execute("""alter table messagechatrooms add column owner_type varchar(255) after customer_id;""")
    op.execute("""update messagechatrooms SET owner_type = ''""")



def downgrade():
    op.execute("""alter table chatrooms drop column owner_type;""")
    op.execute("""alter table chatrooms change owner_id agent_id binary(16);""")
    op.execute("""alter table chatrooms change room_owner_id owner_id binary(16);""")

    op.execute("""alter table chats change room_owner_id owner_id binary(16);""")

    op.execute("""alter table messagechatrooms drop column owner_type;""")
    op.execute("""alter table messagechatrooms change owner_id agent_id binary(16);""")
