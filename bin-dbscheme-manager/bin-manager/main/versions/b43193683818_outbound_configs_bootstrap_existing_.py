"""call_outbound_configs_bootstrap_existing_customers

Revision ID: b43193683818
Revises: 60e68bfd6442
Create Date: 2026-05-07 18:17:54.047479

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'b43193683818'
down_revision = '60e68bfd6442'
branch_labels = None
depends_on = None


def upgrade():
    # Insert an empty OutboundConfig for every active customer that doesn't have one.
    # customer_customers uses '9999-01-01 00:00:00.000000' for active records (sentinel pattern).
    # call_outbound_configs uses NULL for active records.
    # INSERT IGNORE skips customers that already have an outbound_config row (idempotent).
    op.execute("""
        INSERT IGNORE INTO call_outbound_configs (id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update)
        SELECT UUID(), c.id, '', '', '[]', '', NOW(), NOW()
        FROM customer_customers c
        WHERE c.tm_delete = '9999-01-01 00:00:00.000000'
          AND NOT EXISTS (
              SELECT 1 FROM call_outbound_configs oc
              WHERE oc.customer_id = c.id AND oc.tm_delete IS NULL
          )
    """)


def downgrade():
    # Remove auto-bootstrapped configs (those with empty name/detail/whitelist/codecs).
    # Note: this does NOT remove configs that were manually configured by customers.
    op.execute("""
        DELETE FROM call_outbound_configs
        WHERE name = '' AND detail = '' AND codecs = ''
          AND JSON_LENGTH(destination_whitelist) = 0
          AND tm_delete IS NULL
    """)
