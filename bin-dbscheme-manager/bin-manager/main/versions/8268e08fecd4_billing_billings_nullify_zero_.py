"""billing_billings_nullify_zero_idempotency_key

Convert any remaining zero-value UUID idempotency_key records to NULL.
Migration h2b3c4d5e6f7 converted existing zeros to NULL, but the Go code
continued inserting zero-value UUIDs for non-Paddle billing records.
This cleans up the single row that may have been inserted between
h2b3c4d5e6f7 and the code fix, unblocking new inserts immediately.

Revision ID: 8268e08fecd4
Revises: 317f486116c9
Create Date: 2026-04-09 04:20:29.807043

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '8268e08fecd4'
down_revision = '317f486116c9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        UPDATE billing_billings
        SET idempotency_key = NULL
        WHERE idempotency_key = X'00000000000000000000000000000000'
    """)


def downgrade():
    # No-op: re-introducing zero-value UUIDs would violate the UNIQUE constraint
    # and is not desirable. The code fix ensures new records use proper keys.
    pass
