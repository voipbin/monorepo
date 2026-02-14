"""number_update_virtual_prefix_899

Revision ID: fd9ebdbd7baa
Revises: fd9ebdbd7acc
Create Date: 2026-02-14 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fd9ebdbd7baa'
down_revision = 'fd9ebdbd7acc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        UPDATE number_numbers
        SET number = REPLACE(number, '+999', '+899')
        WHERE type = 'virtual';
    """)


def downgrade():
    op.execute("""
        UPDATE number_numbers
        SET number = REPLACE(number, '+899', '+999')
        WHERE type = 'virtual';
    """)
