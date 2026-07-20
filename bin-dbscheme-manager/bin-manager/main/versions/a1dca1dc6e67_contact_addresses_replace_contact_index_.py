"""contact_addresses_replace_contact_index_with_type

Revision ID: a1dca1dc6e67
Revises: 2bdab4b6f1a4
Create Date: 2026-07-21 05:09:45.219658

Replaces the single-column idx_contact_addresses_contact (contact_id)
index with a composite idx_contact_addresses_contact_type (contact_id,
type). This supports AddressListByContactID's new `WHERE contact_id=? AND
type IN (?)` filter (see
docs/plans/2026-07-21-contact-address-reachable-type-filter-design.md)
with an index range scan instead of a full per-contact table scan.

Empirically verified via EXPLAIN against a seeded MySQL 8.0 instance
(analysis doc §8.1): the composite index cuts scanned rows from 502 to 2
in a stress scenario (one contact_id carrying 500 non-reachable-type
rows). The leftmost-prefix (contact_id) of the new composite index fully
covers every existing bare `WHERE contact_id=?` caller, so this is a safe
1:1 replacement, not a narrowing of any existing query's index coverage.

Single combined ALTER TABLE (not two separate DROP/ADD statements) so the
change is one ALGORITHM=INPLACE, LOCK=NONE online DDL operation on
MySQL 8.0 / MariaDB 10.2+, regardless of table size.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a1dca1dc6e67'
down_revision = '2bdab4b6f1a4'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE contact_addresses
            DROP INDEX idx_contact_addresses_contact,
            ADD INDEX idx_contact_addresses_contact_type (contact_id, type);
    """)


def downgrade():
    op.execute("""
        ALTER TABLE contact_addresses
            DROP INDEX idx_contact_addresses_contact_type,
            ADD INDEX idx_contact_addresses_contact (contact_id);
    """)
