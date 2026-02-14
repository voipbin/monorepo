"""update_billing_billings_cost_fields

Revision ID: fd3b4c5d6e7f
Revises: fc2a3b4d5e6f
Create Date: 2026-02-14 12:01:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fd3b4c5d6e7f'
down_revision = 'fc2a3b4d5e6f'
branch_labels = None
depends_on = None


def upgrade():
    # Drop old cost columns
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_per_unit;""")
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_total;""")
    op.execute("""ALTER TABLE billing_billings DROP COLUMN billing_unit_count;""")

    # Add new cost columns
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_type VARCHAR(32) NOT NULL DEFAULT '';""")
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_unit_count FLOAT NOT NULL DEFAULT 0;""")
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_token_per_unit INT NOT NULL DEFAULT 0;""")
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_token_total INT NOT NULL DEFAULT 0;""")
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_credit_per_unit FLOAT NOT NULL DEFAULT 0;""")
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_credit_total FLOAT NOT NULL DEFAULT 0;""")


def downgrade():
    # Remove new cost columns
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_type;""")
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_unit_count;""")
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_token_per_unit;""")
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_token_total;""")
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_credit_per_unit;""")
    op.execute("""ALTER TABLE billing_billings DROP COLUMN cost_credit_total;""")

    # Restore old cost columns
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_per_unit FLOAT NOT NULL DEFAULT 0;""")
    op.execute("""ALTER TABLE billing_billings ADD COLUMN cost_total FLOAT NOT NULL DEFAULT 0;""")
    op.execute("""ALTER TABLE billing_billings ADD COLUMN billing_unit_count FLOAT NOT NULL DEFAULT 0;""")
