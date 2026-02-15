"""billing_allowances_fix_tm_delete_sentinel

Revision ID: a1b2c3d4e5f7
Revises: fd3b4c5d6e7f
Create Date: 2026-02-15 18:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f7'
down_revision = 'fd3b4c5d6e7f'
branch_labels = None
depends_on = None

SENTINEL = '9999-01-01 00:00:00.000000'


def upgrade():
    # Make tm_delete nullable (currently NOT NULL with sentinel default)
    op.execute("ALTER TABLE `billing_allowances` MODIFY `tm_delete` DATETIME(6) NULL")

    # Convert existing sentinel values to NULL
    op.execute(f"""
        UPDATE `billing_allowances`
        SET `tm_delete` = NULL
        WHERE `tm_delete` = '{SENTINEL}'
    """)


def downgrade():
    # Restore sentinel values from NULL
    op.execute(f"""
        UPDATE `billing_allowances`
        SET `tm_delete` = '{SENTINEL}'
        WHERE `tm_delete` IS NULL
    """)

    # Restore NOT NULL constraint with sentinel default
    op.execute(
        f"ALTER TABLE `billing_allowances` MODIFY `tm_delete` DATETIME(6) NOT NULL DEFAULT '{SENTINEL}'"
    )
