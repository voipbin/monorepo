"""billing_add_failed_events_table

Revision ID: fb5b8d1359c9
Revises: b2e4f8a91c03
Create Date: 2026-02-09 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fb5b8d1359c9'
down_revision = 'b2e4f8a91c03'
branch_labels = None
depends_on = None


def upgrade():
    # Create billing_failed_events table for retry persistence.
    # IF NOT EXISTS for idempotency in case of re-run after partial failure.
    op.execute("""
        CREATE TABLE IF NOT EXISTS billing_failed_events(
            id              binary(16),
            event_type      varchar(255) NOT NULL,
            event_publisher varchar(255) NOT NULL,
            event_data      text         NOT NULL,
            error_message   text         NOT NULL,

            retry_count     int          NOT NULL DEFAULT 0,
            max_retries     int          NOT NULL DEFAULT 5,
            next_retry_at   datetime(6)  NOT NULL,
            status          varchar(32)  NOT NULL DEFAULT 'pending',

            tm_create       datetime(6),
            tm_update       datetime(6),

            PRIMARY KEY(id),
            INDEX idx_billing_failed_events_status_next_retry(status, next_retry_at)
        );
    """)

    # Revert sentinel values from a previous partial run of this migration.
    # The earlier version converted NULL tm_delete to '9999-01-01' for a unique index
    # that has been removed. Restore the original NULLs.
    op.execute("""UPDATE billing_billings SET tm_delete = NULL WHERE tm_delete = '9999-01-01 00:00:00.000000';""")


def downgrade():
    op.execute("""DROP TABLE IF EXISTS billing_failed_events;""")
