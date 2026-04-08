"""customer_customers add column default_outgoing_source_number_id

Revision ID: 317f486116c9
Revises: 68bc2fb5141f
Create Date: 2026-04-08 11:59:25.687752

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '317f486116c9'
down_revision = '68bc2fb5141f'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('customer_customers',
        sa.Column('default_outgoing_source_number_id', sa.LargeBinary(length=16), nullable=True)
    )


def downgrade():
    op.drop_column('customer_customers', 'default_outgoing_source_number_id')
