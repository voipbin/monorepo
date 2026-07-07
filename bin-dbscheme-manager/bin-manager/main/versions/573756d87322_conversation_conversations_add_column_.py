"""conversation_conversations_add_column_metadata

Revision ID: 573756d87322
Revises: 5c7bc362be27
Create Date: 2026-07-07 09:55:53.875223

Adds a nullable metadata JSON column to conversation_conversations
(VOIP-1228), mirroring bin-customer-manager's customer_customers.metadata
column exactly (c78cf0c45f54_customer_customers_add_column_metadata.py).

Used by the contact-case-management design's §4.3/§4.4 unified
case-linking mechanism: bin-contact-manager writes Metadata.ContactCaseID
here (via a new dedicated ConversationUpdateMetadata RPC, not the
general PUT allowlist) to claim a Conversation for a Case; every
inbound/outbound message event then echoes this value as a case_id
hint. Purely additive and inert to conversation-manager's own dispatch
logic.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '573756d87322'
down_revision = '5c7bc362be27'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE conversation_conversations ADD COLUMN metadata JSON DEFAULT NULL;""")


def downgrade():
    op.execute("""ALTER TABLE conversation_conversations DROP COLUMN metadata;""")
