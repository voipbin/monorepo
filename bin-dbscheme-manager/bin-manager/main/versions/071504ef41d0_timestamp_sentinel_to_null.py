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

# Sentinel value as stored in MySQL DATETIME(6) columns.
# The Go code uses "9999-01-01T00:00:00.000000Z" (ISO 8601), but the Go MySQL
# driver converts it to MySQL format when using prepared statements.
SENTINEL = '9999-01-01 00:00:00.000000'

# Exact mapping of each table to the columns that use the sentinel value.
# Only columns that were initialized to DefaultTimeStamp in Go Create functions
# are listed here. Columns like tm_create, tm_billing_start, tm_start are
# always set to real timestamps and are NOT included.
TABLE_COLUMNS = {
    'agent_agents': ['tm_update', 'tm_delete'],
    'ai_ais': ['tm_update', 'tm_delete'],
    'ai_aicalls': ['tm_update', 'tm_delete', 'tm_end'],
    'ai_messages': ['tm_delete'],
    'ai_summaries': ['tm_update', 'tm_delete'],
    'billing_accounts': ['tm_update', 'tm_delete'],
    'billing_billings': ['tm_update', 'tm_delete', 'tm_billing_end'],
    'call_calls': ['tm_update', 'tm_delete', 'tm_progressing', 'tm_ringing', 'tm_hangup'],
    'call_channels': ['tm_update', 'tm_delete', 'tm_end', 'tm_answer', 'tm_ringing'],
    'call_confbridges': ['tm_update', 'tm_delete'],
    'call_groupcalls': ['tm_update', 'tm_delete'],
    'call_recordings': ['tm_update', 'tm_delete', 'tm_end'],
    'campaign_campaigns': ['tm_update', 'tm_delete'],
    'campaign_campaigncalls': ['tm_update', 'tm_delete'],
    'campaign_outplans': ['tm_update', 'tm_delete'],
    'conference_conferences': ['tm_update', 'tm_delete', 'tm_end'],
    'conference_conferencecalls': ['tm_update', 'tm_delete'],
    'contact_contacts': ['tm_update', 'tm_delete'],
    'contact_lists': ['tm_update', 'tm_delete'],
    'conversation_accounts': ['tm_update', 'tm_delete'],
    'conversation_conversations': ['tm_update', 'tm_delete'],
    'conversation_medias': ['tm_update', 'tm_delete'],
    'customer_customers': ['tm_update', 'tm_delete'],
    'customer_accesskeys': ['tm_update', 'tm_delete'],
    'email_emails': ['tm_update', 'tm_delete'],
    'flow_flows': ['tm_update', 'tm_delete'],
    'flow_activeflows': ['tm_update', 'tm_delete'],
    'message_messages': ['tm_update', 'tm_delete'],
    'number_numbers': ['tm_update', 'tm_delete'],
    'outdial_outdials': ['tm_update', 'tm_delete'],
    'outdial_outdialtargets': ['tm_update', 'tm_delete'],
    'pipecat_pipecatcalls': ['tm_update', 'tm_delete'],
    'queue_queues': ['tm_update', 'tm_delete'],
    'queue_queuecalls': ['tm_update', 'tm_delete', 'tm_end', 'tm_service'],
    'registrar_sip_auths': ['tm_update'],
    'registrar_trunks': ['tm_update', 'tm_delete'],
    'route_routes': ['tm_update', 'tm_delete'],
    'storage_accounts': ['tm_update', 'tm_delete'],
    'storage_files': ['tm_update', 'tm_delete'],
    'tag_tags': ['tm_update', 'tm_delete'],
    'talk_chats': ['tm_update', 'tm_delete'],
    'talk_chatmembers': [],
    'talk_messages': ['tm_update', 'tm_delete'],
    'transcribe_transcribes': ['tm_update', 'tm_delete'],
}


def upgrade():
    """Convert sentinel timestamp values to NULL.

    Step 1: ALTER columns to allow NULL (some columns have NOT NULL constraints).
    Step 2: UPDATE sentinel values to NULL.
    """
    for table, columns in TABLE_COLUMNS.items():
        if not columns:
            continue

        for column in columns:
            # Step 1: Make column nullable
            print(f"  ALTER {table}.{column} -> nullable")
            op.execute(f"ALTER TABLE `{table}` MODIFY `{column}` DATETIME(6) NULL")

            # Step 2: Convert sentinel to NULL
            print(f"  UPDATE {table}.{column}: sentinel -> NULL")
            op.execute(f"""
                UPDATE `{table}`
                SET `{column}` = NULL
                WHERE `{column}` = '{SENTINEL}'
            """)


def downgrade():
    """Convert NULL back to sentinel timestamp values.

    Step 1: UPDATE NULL values back to sentinel.
    Step 2: Restore NOT NULL constraints.
    """
    for table, columns in TABLE_COLUMNS.items():
        if not columns:
            continue

        for column in columns:
            # Step 1: Restore sentinel values
            print(f"  UPDATE {table}.{column}: NULL -> sentinel")
            op.execute(f"""
                UPDATE `{table}`
                SET `{column}` = '{SENTINEL}'
                WHERE `{column}` IS NULL
            """)

            # Step 2: Restore NOT NULL constraint
            print(f"  ALTER {table}.{column} -> NOT NULL")
            op.execute(
                f"ALTER TABLE `{table}` MODIFY `{column}` DATETIME(6) NOT NULL DEFAULT '{SENTINEL}'"
            )
