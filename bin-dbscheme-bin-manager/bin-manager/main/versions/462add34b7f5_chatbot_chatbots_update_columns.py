"""chatbot_chatbots update columns

Revision ID: 462add34b7f5
Revises: 982e06d7691e
Create Date: 2025-03-12 14:36:24.612663

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '462add34b7f5'
down_revision = '982e06d7691e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table chatbot_chatbots add engine_data json after engine_model;""")
    op.execute("""update chatbot_chatbots SET engine_data = '{}';""")

    op.execute("""alter table chatbot_chatbots drop column credential_base64;""")
    op.execute("""alter table chatbot_chatbots drop column credential_project_id;""")

    op.execute("""alter table chatbot_chatbotcalls add chatbot_engine_data json after chatbot_engine_model;""")
    op.execute("""update chatbot_chatbotcalls SET chatbot_engine_data = '{}';""")


def downgrade():
    op.execute("""alter table chatbot_chatbots drop column engine_data;""")

    op.execute("""alter table chatbot_chatbots add credential_base64 text after init_prompt;""")
    op.execute("""update chatbot_chatbots SET credential_base64 = '';""")

    op.execute("""alter table chatbot_chatbots add credential_project_id text after credential_base64;""")
    op.execute("""update chatbot_chatbots SET credential_project_id = '';""")

    op.execute("""alter table chatbot_chatbotcalls drop column chatbot_engine_data;""")
