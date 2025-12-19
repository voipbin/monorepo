"""chatbot_add_column_engine_model

Revision ID: 4723f143cfe1
Revises: 3363754338da
Create Date: 2025-02-18 01:33:58.821157

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '4723f143cfe1'
down_revision = '3363754338da'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table chatbot_chatbots add engine_model varchar(255) after engine_type;""")
    op.execute("""alter table chatbot_chatbots add credential_base64 text after init_prompt;""")
    op.execute("""alter table chatbot_chatbots add credential_project_id varchar(255) after credential_base64;""")
    op.execute("""update chatbot_chatbots set engine_model = '';""")
    op.execute("""update chatbot_chatbots set credential_base64 = '';""")
    op.execute("""update chatbot_chatbots set credential_project_id = '';""")

    op.execute("""alter table chatbot_chatbotcalls add chatbot_engine_model varchar(255) after chatbot_engine_type;""")
    op.execute("""update chatbot_chatbotcalls set chatbot_engine_model = '';""")

def downgrade():
    op.execute("""alter table chatbot_chatbots drop column engine_model;""")
    op.execute("""alter table chatbot_chatbots drop column credential_base64;""")
    op.execute("""alter table chatbot_chatbots drop column credential_project_id;""")

    op.execute("""alter table chatbot_chatbotcalls drop column chatbot_engine_model;""")