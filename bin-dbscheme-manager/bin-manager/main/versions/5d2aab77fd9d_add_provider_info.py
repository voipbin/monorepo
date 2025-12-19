"""add provider info

Revision ID: 5d2aab77fd9d
Revises: c939ba877f8f
Create Date: 2021-03-01 01:53:00.395262

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '5d2aab77fd9d'
down_revision = 'c939ba877f8f'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('numbers', sa.Column('provider_name', sa.VARCHAR(255)))
    op.add_column('numbers', sa.Column('provider_reference_id', sa.VARCHAR(255)))

    op.add_column('numbers', sa.Column('status', sa.VARCHAR(32)))

    op.add_column('numbers', sa.Column('t38_enabled', sa.BOOLEAN))
    op.add_column('numbers', sa.Column('emergency_enabled', sa.BOOLEAN))

    op.execute("""
        alter table numbers add tm_purchase datetime(6)
    """)

    # indices
    op.create_index('idx_numbers_user_id', 'numbers', ['user_id'])
    op.create_index('idx_numbers_flow_id', 'numbers', ['flow_id'])
    op.create_index('idx_numbers_provider_name', 'numbers', ['provider_name'])


def downgrade():
    op.drop_column('numbers', 'provider_name')
    op.drop_column('numbers', 'provider_reference_id')

    op.drop_column('numbers', 'status')

    op.drop_column('numbers', 't38_enabled')
    op.drop_column('numbers', 'emergency_enabled')

    op.drop_column('numbers', 'tm_purchase')

    op.drop_index('idx_numbers_user_id', 'numbers')
    op.drop_index('idx_numbers_flow_id', 'numbers')
    op.drop_index('idx_numbers_provider_name', 'numbers')

