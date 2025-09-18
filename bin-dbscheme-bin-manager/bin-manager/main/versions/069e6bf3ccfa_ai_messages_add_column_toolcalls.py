"""ai_messages_add_column_toolcalls

Revision ID: 069e6bf3ccfa
Revises: 6efdc5294d5e
Create Date: 2025-09-19 02:11:17.548984

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '069e6bf3ccfa'
down_revision = '6efdc5294d5e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table ai_messages add column tool_calls json after content;""")
    op.execute("""update ai_messages set tool_calls = '[]';""")

    op.execute("""alter table ai_messages add column tool_call_id varchar(255) after tool_calls;""")
    op.execute("""update ai_messages set tool_call_id = '';""")


def downgrade():
    op.execute("""alter table ai_messages drop column tool_calls;""")
    op.execute("""alter table ai_messages drop column tool_call_id;""")
