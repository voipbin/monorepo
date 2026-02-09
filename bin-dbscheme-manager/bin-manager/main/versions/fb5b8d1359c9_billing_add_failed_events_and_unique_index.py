"""billing_add_failed_events_and_unique_index

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
    # Create billing_failed_events table for retry persistence
    op.execute("""
        CREATE TABLE billing_failed_events(
            id              binary(16),
            event_type      varchar(255) NOT NULL,
            event_publisher varchar(255) NOT NULL,
            event_data      mediumblob   NOT NULL,
            error_message   text         NOT NULL,

            retry_count     int          NOT NULL DEFAULT 0,
            max_retries     int          NOT NULL DEFAULT 5,
            next_retry_at   datetime(6)  NOT NULL,
            status          varchar(32)  NOT NULL DEFAULT 'pending',

            tm_create       datetime(6),
            tm_update       datetime(6),

            PRIMARY KEY(id)
        );
    """)
    op.execute("""CREATE INDEX idx_billing_failed_events_status_next_retry ON billing_failed_events(status, next_retry_at);""")

    # Migrate existing NULL tm_delete rows to sentinel value so the unique index works.
    # MySQL treats NULL != NULL, so NULLs would bypass the unique constraint.
    op.execute("""UPDATE billing_billings SET tm_delete = '9999-01-01 00:00:00.000000' WHERE tm_delete IS NULL;""")

    # Add unique index to prevent duplicate active billings per reference_type + reference_id.
    op.execute("""CREATE UNIQUE INDEX idx_billings_ref_type_id_active ON billing_billings(reference_type, reference_id, tm_delete);""")


def downgrade():
    op.execute("""DROP INDEX idx_billings_ref_type_id_active ON billing_billings;""")
    op.execute("""UPDATE billing_billings SET tm_delete = NULL WHERE tm_delete = '9999-01-01 00:00:00.000000';""")
    op.execute("""DROP INDEX idx_billing_failed_events_status_next_retry ON billing_failed_events;""")
    op.execute("""DROP TABLE billing_failed_events;""")
