"""conversation change columns

Revision ID: 7a27decc13da
Revises: 0341ff217465
Create Date: 2025-04-17 03:15:32.954460

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '7a27decc13da'
down_revision = '0341ff217465'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE conversation_conversations
            CHANGE COLUMN reference_type type varchar(255),
            CHANGE COLUMN reference_id dialog_id varchar(255),
            CHANGE COLUMN source self json,
            CHANGE COLUMN participants peer json;
    """)
    op.execute("""
        ALTER TABLE conversation_messages
            DROP COLUMN source;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE conversation_conversations
            CHANGE COLUMN type reference_type varchar(255),
            CHANGE COLUMN dialog_id reference_id varchar(255),
            CHANGE COLUMN self source json,
            CHANGE COLUMN peer participants json;
    """)
    op.execute("""
        ALTER TABLE conversation_messages
            ADD COLUMN source json;
    """)
