"""customer_customers_drop_default_outgoing_source_number_id

Revision ID: ff02d243a799
Revises: 1ebd3fdcea8d
Create Date: 2026-05-08 13:34:54.836140

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ff02d243a799'
down_revision = '1ebd3fdcea8d'
branch_labels = None
depends_on = None


def upgrade():
    # Drop the column. By the time this migration runs (24h+ after
    # bin-customer-manager rollout), all live customer-manager pods have been
    # re-deployed without the DefaultOutgoingSourceNumberID Go field, so SELECT
    # statements no longer reference this column. The column's data has already
    # been preserved into call_outbound_configs.default_outgoing_source_number_id
    # by Mig 1 (1ebd3fdcea8d).
    op.execute("""
        ALTER TABLE customer_customers
          DROP COLUMN default_outgoing_source_number_id
    """)


def downgrade():
    # Re-adds the column shape but cannot restore data.
    # Original values are preserved on call_outbound_configs.default_outgoing_source_number_id
    # (set by Mig 1's JOIN backfill); a forward "restore" migration could copy them back
    # if needed before this Mig 2 was applied.
    # Default form matches prior art in e2e30543f2fc_*.py.
    op.execute("""
        ALTER TABLE customer_customers
          ADD COLUMN default_outgoing_source_number_id BINARY(16) NOT NULL
            DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', '')))
    """)
