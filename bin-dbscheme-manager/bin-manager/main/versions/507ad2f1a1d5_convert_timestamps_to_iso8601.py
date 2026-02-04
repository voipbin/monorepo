"""convert timestamps to iso8601

Revision ID: 507ad2f1a1d5
Revises: a1b2c3d4e5f6
Create Date: 2026-02-04 18:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '507ad2f1a1d5'
down_revision = 'a1b2c3d4e5f6'
branch_labels = None
depends_on = None


# Tables and their timestamp columns to convert
# Format: (table_name, [list of timestamp columns])
TABLES_WITH_TIMESTAMPS = [
    ('activeflows', ['tm_create', 'tm_update', 'tm_delete']),
    ('agent_resources', ['tm_create', 'tm_update', 'tm_delete']),
    ('agents', ['tm_create', 'tm_update', 'tm_delete']),
    ('ai_aicalls', ['tm_create', 'tm_update', 'tm_delete', 'tm_end']),
    ('ai_ais', ['tm_create', 'tm_update', 'tm_delete']),
    ('ai_messages', ['tm_create', 'tm_delete']),
    ('ai_summaries', ['tm_create', 'tm_update', 'tm_delete']),
    ('billing_accounts', ['tm_create', 'tm_update', 'tm_delete']),
    ('billing_billings', ['tm_create', 'tm_update', 'tm_delete']),
    ('bridges', ['tm_create', 'tm_update', 'tm_delete']),
    ('calls', ['tm_create', 'tm_update', 'tm_delete', 'tm_ringing', 'tm_progressing', 'tm_hangup']),
    ('campaigncalls', ['tm_create', 'tm_update', 'tm_delete']),
    ('campaigns', ['tm_create', 'tm_update', 'tm_delete']),
    ('channels', ['tm_create', 'tm_update', 'tm_delete']),
    ('chatbotcalls', ['tm_create', 'tm_update', 'tm_delete']),
    ('chatbot_messages', ['tm_create', 'tm_delete']),
    ('chatbots', ['tm_create', 'tm_update', 'tm_delete']),
    ('confbridges', ['tm_create', 'tm_update', 'tm_delete']),
    ('conferencecalls', ['tm_create', 'tm_update', 'tm_delete']),
    ('conferences', ['tm_create', 'tm_update', 'tm_delete']),
    ('contact_contacts', ['tm_create', 'tm_update', 'tm_delete']),
    ('contact_emails', ['tm_create', 'tm_update', 'tm_delete']),
    ('contact_numbers', ['tm_create', 'tm_update', 'tm_delete']),
    ('contact_addresses', ['tm_create', 'tm_update', 'tm_delete']),
    ('contact_lists', ['tm_create', 'tm_update', 'tm_delete']),
    ('contact_list_members', ['tm_create', 'tm_update', 'tm_delete']),
    ('contact_dnc_lists', ['tm_create', 'tm_update', 'tm_delete']),
    ('conversation_accounts', ['tm_create', 'tm_update', 'tm_delete']),
    ('conversation_conversations', ['tm_create', 'tm_update', 'tm_delete']),
    ('conversation_medias', ['tm_create', 'tm_update', 'tm_delete']),
    ('customer_accesskeys', ['tm_create', 'tm_update', 'tm_delete']),
    ('customers', ['tm_create', 'tm_update', 'tm_delete']),
    ('email_emails', ['tm_create', 'tm_update', 'tm_delete']),
    ('extensions', ['tm_create', 'tm_update', 'tm_delete']),
    ('flows', ['tm_create', 'tm_update', 'tm_delete']),
    ('groupcalls', ['tm_create', 'tm_update', 'tm_delete']),
    ('messages', ['tm_create', 'tm_update', 'tm_delete']),
    ('numbers', ['tm_create', 'tm_update', 'tm_delete', 'tm_renew']),
    ('outdials', ['tm_create', 'tm_update', 'tm_delete']),
    ('outdialtargets', ['tm_create', 'tm_update', 'tm_delete']),
    ('outplans', ['tm_create', 'tm_update', 'tm_delete']),
    ('pipecat_pipecatcalls', ['tm_create', 'tm_update', 'tm_delete']),
    ('queuecalls', ['tm_create', 'tm_update', 'tm_delete', 'tm_end']),
    ('queues', ['tm_create', 'tm_update', 'tm_delete']),
    ('recordings', ['tm_create', 'tm_update', 'tm_delete']),
    ('registrar_sip_auths', ['tm_create', 'tm_update', 'tm_delete']),
    ('registrar_trunks', ['tm_create', 'tm_update', 'tm_delete']),
    ('routes', ['tm_create', 'tm_update', 'tm_delete']),
    ('storage_accounts', ['tm_create', 'tm_update', 'tm_delete']),
    ('storage_files', ['tm_create', 'tm_update', 'tm_delete']),
    ('tags', ['tm_create', 'tm_update', 'tm_delete']),
    ('talk_chats', ['tm_create']),
    ('talk_messages', ['tm_create']),
    ('transcribes', ['tm_create', 'tm_update', 'tm_delete']),
    ('transcripts', ['tm_create', 'tm_delete', 'tm_transcript']),
    ('users', ['tm_create', 'tm_update', 'tm_delete']),
]


def upgrade():
    """Convert all timestamp columns from custom format to ISO 8601.

    Old format: 2024-01-15 10:30:45.123456
    New format: 2024-01-15T10:30:45.123456Z
    """
    for table_name, columns in TABLES_WITH_TIMESTAMPS:
        for column in columns:
            # Convert: "2024-01-15 10:30:45.123456" -> "2024-01-15T10:30:45.123456Z"
            # SUBSTRING(col, 1, 10) = date part "2024-01-15"
            # SUBSTRING(col, 12) = time part "10:30:45.123456"
            sql = f"""
                UPDATE {table_name}
                SET {column} = CONCAT(SUBSTRING({column}, 1, 10), 'T', SUBSTRING({column}, 12), 'Z')
                WHERE {column} IS NOT NULL
                  AND {column} != ''
                  AND {column} NOT LIKE '%T%Z';
            """
            try:
                op.execute(sql)
            except Exception:
                # Table or column might not exist in some environments
                pass


def downgrade():
    """Convert all timestamp columns from ISO 8601 back to custom format.

    Old format: 2024-01-15T10:30:45.123456Z
    New format: 2024-01-15 10:30:45.123456
    """
    for table_name, columns in TABLES_WITH_TIMESTAMPS:
        for column in columns:
            # Convert: "2024-01-15T10:30:45.123456Z" -> "2024-01-15 10:30:45.123456"
            # SUBSTRING(col, 1, 10) = date part "2024-01-15"
            # SUBSTRING(col, 12, 15) = time part "10:30:45.123456" (excludes trailing Z)
            sql = f"""
                UPDATE {table_name}
                SET {column} = CONCAT(SUBSTRING({column}, 1, 10), ' ', SUBSTRING({column}, 12, 15))
                WHERE {column} IS NOT NULL
                  AND {column} != ''
                  AND {column} LIKE '%T%Z';
            """
            try:
                op.execute(sql)
            except Exception:
                # Table or column might not exist in some environments
                pass
