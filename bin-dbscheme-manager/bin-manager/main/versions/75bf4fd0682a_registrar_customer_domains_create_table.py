"""registrar_customer_domains_create_table

Create the registrar_customer_domains table (VOIP-1385): one row per customer
mapping the customer to its SIP domain label and full realm
(<label>.reg.<base domain>). Owned by bin-registrar-manager.

Deviation note: no tm_delete column (hard delete on customer_deleted) — a pure
mapping row has no soft-delete consumer; deliberate deviation from sibling
registrar tables.

Revision ID: 75bf4fd0682a
Revises: ede50012c416
Create Date: 2026-08-23 08:39:53.159670

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '75bf4fd0682a'
down_revision = 'ede50012c416'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE registrar_customer_domains (
            customer_id   BINARY(16)   NOT NULL,
            domain_label  VARCHAR(64)  NOT NULL,
            realm         VARCHAR(255) NOT NULL,

            tm_create DATETIME(6),
            tm_update DATETIME(6),

            PRIMARY KEY (customer_id),
            UNIQUE KEY ux_registrar_customer_domains_domain_label (domain_label),
            UNIQUE KEY ux_registrar_customer_domains_realm (realm)
        )
    """)


def downgrade():
    op.execute("""
        DROP TABLE registrar_customer_domains
    """)
