"""contact_drop_legacy_phone_numbers_and_emails_tables

Revision ID: 1d0f4d07ff58
Revises: 2d8f0ea90565
Create Date: 2026-07-12 19:57:24.395973

Drops contact_phone_numbers and contact_emails, the two legacy tables
created by a1b2c3d4e5f6_contact_create_tables.py and superseded by the
unified contact_addresses table (VOIP-1207, migration ac5d4e18060c +
data migration bbcf80d332eb). Deferred at the time per
docs/plans/2026-06-27-voip-1207-crm-m1-address-migration.md's "Out of
scope" section ("keeps rollback cheap" while M1 was unproven) -- M1 has
since shipped, been in production for multiple further migrations
(contact_cases, contact_case_notes, contact_address_ownership_periods),
and no Go code anywhere in the monorepo references either table
(grep-confirmed monorepo-wide). Safe to drop now.

No data migration needed: bbcf80d332eb already copied every row from
both tables into contact_addresses at M1 time: any data still physically
present in these two tables is a stale, already-superseded duplicate of
what contact_addresses holds, not a distinct data set.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1d0f4d07ff58'
down_revision = '2d8f0ea90565'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""DROP TABLE IF EXISTS contact_phone_numbers;""")
    op.execute("""DROP TABLE IF EXISTS contact_emails;""")


def downgrade():
    # Recreates the original (VOIP-1207-era) schema, per
    # a1b2c3d4e5f6_contact_create_tables.py's upgrade(). Data is NOT
    # restored -- if this table held any physical rows at drop time,
    # they were already stale duplicates of contact_addresses (see
    # upgrade()'s docstring); a downgrade recreates empty tables only.
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_phone_numbers (
            id BINARY(16) NOT NULL,
            contact_id BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,

            number VARCHAR(50) NOT NULL,
            number_e164 VARCHAR(20) NOT NULL,
            type VARCHAR(20) NOT NULL DEFAULT '',
            is_primary BOOLEAN NOT NULL DEFAULT FALSE,

            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (id),
            INDEX idx_contact_phone_numbers_contact (contact_id),
            INDEX idx_contact_phone_numbers_customer_number (customer_id, number_e164),
            UNIQUE INDEX idx_contact_phone_numbers_unique (customer_id, number_e164)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_emails (
            id BINARY(16) NOT NULL,
            contact_id BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,

            address VARCHAR(255) NOT NULL,
            type VARCHAR(20) NOT NULL DEFAULT '',
            is_primary BOOLEAN NOT NULL DEFAULT FALSE,

            tm_create DATETIME(6) NOT NULL,

            PRIMARY KEY (id),
            INDEX idx_contact_emails_contact (contact_id),
            INDEX idx_contact_emails_customer_address (customer_id, address)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    """)
