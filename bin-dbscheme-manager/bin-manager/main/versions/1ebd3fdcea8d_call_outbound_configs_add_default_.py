"""call_outbound_configs_add_default_outgoing_source_number_id

Revision ID: 1ebd3fdcea8d
Revises: dd500d4e8cb7
Create Date: 2026-05-08 11:40:56.383258

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1ebd3fdcea8d'
down_revision = 'dd500d4e8cb7'
branch_labels = None
depends_on = None


def upgrade():
    # Step 1: Add the column as NULLable so existing rows accept it.
    op.execute("""
        ALTER TABLE call_outbound_configs
          ADD COLUMN default_outgoing_source_number_id BINARY(16) NULL
    """)

    # Step 2: Copy each customer's default_outgoing_source_number_id into the new column.
    # No tm_delete filter — soft-deleted customers still own outbound_config rows
    # (per uq_customer_id bootstrapped in dd500d4e8cb7), and step 3's NOT NULL would
    # otherwise reject those orphan rows.
    op.execute("""
        UPDATE call_outbound_configs o
          JOIN customer_customers c ON o.customer_id = c.id
          SET o.default_outgoing_source_number_id = c.default_outgoing_source_number_id
    """)

    # Step 2b: Zero-fill remaining NULLs. Two cases reach this step:
    #   (a) Customers with NULL source — customer_customers.default_outgoing_source_number_id
    #       is NULLABLE (added in 317f486116c9 with no DEFAULT). Customers who never
    #       configured a default have NULL there, and the JOIN above faithfully copies it.
    #       In practice this is the majority case.
    #   (b) Orphan rows — outbound_configs whose customer was hard-deleted (no JOIN match).
    # Without this, Step 3's MODIFY NOT NULL would fail on those rows.
    op.execute("""
        UPDATE call_outbound_configs
          SET default_outgoing_source_number_id = UNHEX('00000000000000000000000000000000')
          WHERE default_outgoing_source_number_id IS NULL
    """)

    # Step 3: Tighten to NOT NULL.
    op.execute("""
        ALTER TABLE call_outbound_configs
          MODIFY default_outgoing_source_number_id BINARY(16) NOT NULL
    """)


def downgrade():
    op.execute("""
        ALTER TABLE call_outbound_configs
          DROP COLUMN default_outgoing_source_number_id
    """)
