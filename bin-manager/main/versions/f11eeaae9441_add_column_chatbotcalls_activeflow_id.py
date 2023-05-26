"""add column chatbotcalls activeflow_id

Revision ID: f11eeaae9441
Revises: 08fa3628f652
Create Date: 2023-05-26 18:18:07.136180

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f11eeaae9441'
down_revision = '08fa3628f652'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table chatbotcalls add column activeflow_id binary(16) after chatbot_engine_type;""")
    op.execute("""create index idx_chatbotcalls_activeflow_id on chatbotcalls(activeflow_id);""")

    op.execute("""update chatbotcalls set activeflow_id = "";""")


def downgrade():
    op.execute("""alter table chatbotcalls drop column activeflow_id;""")
