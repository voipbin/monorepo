"""add tts_manager_speaking

Revision ID: b2c3d4e5f6a7
Revises: c3fb144b1c95
Create Date: 2026-02-14 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'b2c3d4e5f6a7'
down_revision = 'c3fb144b1c95'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE tts_manager_speaking (
            id binary(16) NOT NULL,
            customer_id binary(16) NOT NULL,
            reference_type varchar(32) NOT NULL DEFAULT '',
            reference_id binary(16) NOT NULL,
            language varchar(16) NOT NULL DEFAULT '',
            provider varchar(32) NOT NULL DEFAULT '',
            voice_id varchar(255) NOT NULL DEFAULT '',
            direction varchar(8) NOT NULL DEFAULT '',
            status varchar(16) NOT NULL DEFAULT '',
            pod_id varchar(64) NOT NULL DEFAULT '',
            tm_create datetime(6) DEFAULT NULL,
            tm_update datetime(6) DEFAULT NULL,
            tm_delete datetime(6) DEFAULT NULL,
            PRIMARY KEY (id),
            INDEX idx_tts_manager_speaking_customer_id (customer_id),
            INDEX idx_tts_manager_speaking_reference (reference_type, reference_id),
            INDEX idx_tts_manager_speaking_status (status)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS tts_manager_speaking;""")
