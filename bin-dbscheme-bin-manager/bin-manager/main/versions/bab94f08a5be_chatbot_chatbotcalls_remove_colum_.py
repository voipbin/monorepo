"""chatbot_chatbotcalls_remove_colum_messages

Revision ID: bab94f08a5be
Revises: 403c1ab3552c
Create Date: 2025-03-02 02:33:20.912275

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'bab94f08a5be'
down_revision = '403c1ab3552c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE chatbot_chatbotcalls DROP COLUMN messages;""")


def downgrade():
    op.execute("""ALTER TABLE chatbot_chatbotcalls ADD COLUMN messages JSON AFTER language;""")
