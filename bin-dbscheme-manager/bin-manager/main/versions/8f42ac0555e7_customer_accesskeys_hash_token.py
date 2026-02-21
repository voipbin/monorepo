"""customer_accesskeys_hash_token

Revision ID: 8f42ac0555e7
Revises: a3b4c5d6e7f8
Create Date: 2026-02-22 10:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '8f42ac0555e7'
down_revision = 'a3b4c5d6e7f8'
branch_labels = None
depends_on = None


def upgrade():
    # Add new columns
    op.execute("""ALTER TABLE customer_accesskeys ADD COLUMN token_hash CHAR(64);""")
    op.execute("""ALTER TABLE customer_accesskeys ADD COLUMN token_prefix VARCHAR(16);""")

    # Backfill: hash existing plain-text tokens
    op.execute("""
        UPDATE customer_accesskeys
        SET token_hash = SHA2(token, 256),
            token_prefix = LEFT(token, 11)
        WHERE token IS NOT NULL AND token != '';
    """)

    # Add index on token_hash
    op.execute("""CREATE INDEX idx_customer_accesskeys_token_hash ON customer_accesskeys(token_hash);""")

    # Drop old token index and column
    op.execute("""DROP INDEX idx_customer_accesskeys_token ON customer_accesskeys;""")
    op.execute("""ALTER TABLE customer_accesskeys DROP COLUMN token;""")


def downgrade():
    # Add back token column
    op.execute("""ALTER TABLE customer_accesskeys ADD COLUMN token VARCHAR(1023);""")
    op.execute("""CREATE INDEX idx_customer_accesskeys_token ON customer_accesskeys(token);""")

    # Drop new columns
    op.execute("""DROP INDEX idx_customer_accesskeys_token_hash ON customer_accesskeys;""")
    op.execute("""ALTER TABLE customer_accesskeys DROP COLUMN token_hash;""")
    op.execute("""ALTER TABLE customer_accesskeys DROP COLUMN token_prefix;""")
