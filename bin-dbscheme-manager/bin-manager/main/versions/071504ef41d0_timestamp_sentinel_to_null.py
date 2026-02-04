"""timestamp_sentinel_to_null

Revision ID: 071504ef41d0
Revises: a1b2c3d4e5f6
Create Date: 2026-02-05 05:32:14.284576

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '071504ef41d0'
down_revision = 'a1b2c3d4e5f6'
branch_labels = None
depends_on = None

# List of all tables with tm_update and tm_delete columns
TABLES_WITH_TIMESTAMPS = [
    'agent_agents',
    'ai_ais',
    'ai_aicalls',
    'ai_messages',
    'ai_summaries',
    'billing_accounts',
    'billing_billings',
    'call_calls',
    'call_channels',
    'call_confbridges',
    'call_groupcalls',
    'call_recordings',
    'campaign_campaigns',
    'campaign_campaigncalls',
    'campaign_outplans',
    'conference_conferences',
    'conference_conferencecalls',
    'contact_contacts',
    'contact_lists',
    'conversation_accounts',
    'conversation_conversations',
    'conversation_medias',
    'customer_customers',
    'customer_accesskeys',
    'email_emails',
    'flow_flows',
    'flow_activeflows',
    'message_messages',
    'number_numbers',
    'outdial_outdials',
    'outdial_outdialtargets',
    'pipecat_pipecatcalls',
    'queue_queues',
    'queue_queuecalls',
    'registrar_sip_auths',
    'registrar_trunks',
    'route_routes',
    'storage_accounts',
    'storage_files',
    'tag_tags',
    'talk_chats',
    'talk_chatmembers',
    'talk_messages',
    'transcribe_transcribes',
    'transfer_transfers',
]

# Sentinel value patterns (both formats for safety)
SENTINEL_VALUES = [
    '9999-01-01 00:00:00.000000',
    '9999-01-01T00:00:00.000000Z',
]


def upgrade():
    """Convert sentinel timestamp values to NULL."""
    for table in TABLES_WITH_TIMESTAMPS:
        for sentinel in SENTINEL_VALUES:
            # Update tm_update
            op.execute(f"""
                UPDATE `{table}`
                SET tm_update = NULL
                WHERE tm_update = '{sentinel}'
            """)
            # Update tm_delete
            op.execute(f"""
                UPDATE `{table}`
                SET tm_delete = NULL
                WHERE tm_delete = '{sentinel}'
            """)


def downgrade():
    """Convert NULL back to sentinel timestamp values."""
    sentinel = '9999-01-01T00:00:00.000000Z'
    for table in TABLES_WITH_TIMESTAMPS:
        # Revert tm_update
        op.execute(f"""
            UPDATE `{table}`
            SET tm_update = '{sentinel}'
            WHERE tm_update IS NULL
        """)
        # Revert tm_delete
        op.execute(f"""
            UPDATE `{table}`
            SET tm_delete = '{sentinel}'
            WHERE tm_delete IS NULL
        """)
