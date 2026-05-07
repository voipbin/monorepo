"""outbound_configs_create_table

Revision ID: 60e68bfd6442
Revises: 86dd44956fa5
Create Date: 2026-05-07 11:28:42.331134

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '60e68bfd6442'
down_revision = '86dd44956fa5'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE outbound_configs (
            id                    VARCHAR(36)  NOT NULL,
            customer_id           VARCHAR(36)  NOT NULL,
            name                  VARCHAR(255) NOT NULL DEFAULT '',
            detail                TEXT         NOT NULL DEFAULT '',
            destination_whitelist JSON         NOT NULL DEFAULT ('[]'),
            codecs                VARCHAR(255) NOT NULL DEFAULT '',
            tm_create             DATETIME(6)  DEFAULT NULL,
            tm_update             DATETIME(6)  DEFAULT NULL,
            tm_delete             DATETIME(6)  DEFAULT NULL,
            PRIMARY KEY (id),
            UNIQUE KEY uq_customer_id (customer_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS outbound_configs")
