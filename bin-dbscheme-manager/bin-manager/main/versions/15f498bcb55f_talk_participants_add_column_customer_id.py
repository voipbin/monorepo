"""talk_participants add column customer_id

Revision ID: 15f498bcb55f
Revises: f6d7d64e259e
Create Date: 2026-01-18 12:37:13.878015

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '15f498bcb55f'
down_revision = 'f6d7d64e259e'
branch_labels = None
depends_on = None


def upgrade():
    # Add customer_id column as nullable first
    op.execute("""
        ALTER TABLE talk_participants
        ADD COLUMN customer_id BINARY(16) NULL AFTER id
    """)

    # Populate customer_id from parent chat for existing rows
    op.execute("""
        UPDATE talk_participants p
        INNER JOIN talk_chats c ON p.chat_id = c.id
        SET p.customer_id = c.customer_id
    """)

    # Make customer_id NOT NULL
    op.execute("""
        ALTER TABLE talk_participants
        MODIFY COLUMN customer_id BINARY(16) NOT NULL
    """)

    # Add index for customer_id
    op.execute("""
        CREATE INDEX idx_customer_id ON talk_participants(customer_id)
    """)


def downgrade():
    # Remove index
    op.execute("""
        DROP INDEX idx_customer_id ON talk_participants
    """)

    # Remove customer_id column
    op.execute("""
        ALTER TABLE talk_participants
        DROP COLUMN customer_id
    """)
