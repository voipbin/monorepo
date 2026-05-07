"""call_outbound_configs_bootstrap_fix_customer_condition

Revision ID: 98abdec1a745
Revises: b43193683818
Create Date: 2026-05-07 22:32:52.731329

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '98abdec1a745'
down_revision = 'b43193683818'
branch_labels = None
depends_on = None


def upgrade():
    # The previous bootstrap migration (b43193683818) used
    # `WHERE c.tm_delete = '9999-01-01 00:00:00.000000'` to identify active
    # customers — but customer_customers.tm_delete was migrated to NULL for
    # active records by 071504ef41d0_timestamp_sentinel_to_null.py, so zero
    # rows were inserted. This migration corrects that using `tm_delete IS NULL`.
    op.execute("""
        INSERT IGNORE INTO call_outbound_configs
            (id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update)
        SELECT UUID(), c.id, '', '', '[]', '', NOW(), NOW()
        FROM customer_customers c
        WHERE c.tm_delete IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM call_outbound_configs oc
              WHERE oc.customer_id = c.id AND oc.tm_delete IS NULL
          )
    """)


def downgrade():
    # Remove auto-bootstrapped configs (those with empty name/detail/whitelist/codecs).
    op.execute("""
        DELETE FROM call_outbound_configs
        WHERE name = '' AND detail = '' AND codecs = ''
          AND JSON_LENGTH(destination_whitelist) = 0
          AND tm_delete IS NULL
    """)
