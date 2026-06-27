"""conversation_messages_add_source_destination

Revision ID: 04e92612d9c7
Revises: ac5d4e18060c
Create Date: 2026-06-27 06:35:45.882997

Adds source/destination (commonaddress.Address, stored as JSON) to
conversation_messages. These are the absolute endpoints the message carried
(source = sending party, destination = receiving party), set once at message
creation, and emitted on the conversation_message webhook event. They feed the
CRM interaction timeline (VOIP-1204 / VOIP-1208), which needs the remote party
per message.

History note: conversation_messages.source (json) existed before (added 2022-06,
122b2ba1b2b0) and was deliberately dropped 2025-04 (7a27decc13da), which moved
the endpoint pair onto the conversation as self/peer. This RE-introduces source
(plus destination) for a different purpose: a per-message immutable record of the
absolute endpoints for the CRM read-model. Self/Peer on the conversation are
immutable after create, so the per-message copy never goes stale.

No backfill: from cutover new messages carry the endpoints; historical rows stay
NULL (consistent with VOIP-1204 M2 "no retroactive backfill"). JSON nullable,
matching conversation_conversations.self/peer convention.
"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '04e92612d9c7'
down_revision = 'ac5d4e18060c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE conversation_messages
            ADD COLUMN source      JSON DEFAULT NULL AFTER reference_id,
            ADD COLUMN destination  JSON DEFAULT NULL AFTER source;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE conversation_messages
            DROP COLUMN destination,
            DROP COLUMN source;
    """)
