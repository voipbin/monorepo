"""customer_customers_add_webhook_secret

Revision ID: 902325885953
Revises: 0c037bf0a362
Create Date: 2026-08-04 06:24:37.371996

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '902325885953'
down_revision = '0c037bf0a362'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE customer_customers ADD COLUMN webhook_secret VARCHAR(255) NOT NULL DEFAULT '' AFTER webhook_uri;""")
    # Backfill existing rows so webhook signing works immediately after deploy.
    # RAND() is not a CSPRNG, but the UUID `id` supplies the real entropy here;
    # new customers going forward get their secret from crypto/rand via
    # utilHandler.StringGenerateRandom (see bin-customer-manager/pkg/customerhandler).
    op.execute("""UPDATE customer_customers SET webhook_secret = SHA2(CONCAT(id, RAND(), NOW(6)), 256) WHERE tm_delete IS NULL AND webhook_secret = '';""")


def downgrade():
    op.execute("""ALTER TABLE customer_customers DROP COLUMN webhook_secret;""")
