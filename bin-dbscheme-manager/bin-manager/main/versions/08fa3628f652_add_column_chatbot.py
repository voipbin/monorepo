"""add column chatbot

Revision ID: 08fa3628f652
Revises: 71484d30e651
Create Date: 2023-05-20 01:37:25.145451

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '08fa3628f652'
down_revision = '71484d30e651'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table chatbotcalls add column chatbot_engine_type varchar(255) after chatbot_id;""")
    op.execute("""alter table chatbotcalls add column messages json after language;""")

    op.execute("""update chatbotcalls set chatbot_engine_type = "chatGPT";""")

    op.execute("""alter table chatbots add column init_prompt text after engine_type;""")

    op.execute("""update chatbots set init_prompt = "";""")


def downgrade():
    op.execute("""alter table chatbotcalls drop column chatbot_engine_type;""")
    op.execute("""alter table chatbotcalls drop column messages;""")

    op.execute("""alter table chatbots drop column init_prompt;""")
