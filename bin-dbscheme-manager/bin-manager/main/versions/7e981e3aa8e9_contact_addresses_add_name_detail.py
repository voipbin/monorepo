"""contact_addresses_add_name_detail

Add name and detail columns to contact_addresses table.
- name: optional human-readable label for the address (e.g. "Main Office")
- detail: optional free-form notes about the address (e.g. "Primary contact number")

Revises: adb8daac2bb0
Revision ID: 7e981e3aa8e9
"""
from alembic import op

# revision identifiers, used by Alembic.
revision = '7e981e3aa8e9'
down_revision = 'adb8daac2bb0'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE contact_addresses
            ADD COLUMN name   VARCHAR(255) NOT NULL DEFAULT '' AFTER target_name,
            ADD COLUMN detail VARCHAR(255) NOT NULL DEFAULT '' AFTER name
    """)


def downgrade():
    op.execute("""
        ALTER TABLE contact_addresses
            DROP COLUMN detail,
            DROP COLUMN name
    """)
