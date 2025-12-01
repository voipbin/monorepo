"""flow_flows add column on_complete_flow_id

Revision ID: 1618235a36a1
Revises: ec31df0c8e6a
Create Date: 2025-12-01 16:50:51.239554

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1618235a36a1'
down_revision = 'ec31df0c8e6a'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE flow_flows
        ADD on_complete_flow_id BINARY(16) NOT NULL DEFAULT UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''));
    """)

    op.execute("""
        ALTER TABLE flow_activeflows
        ADD on_complete_flow_id BINARY(16) NOT NULL DEFAULT UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''));
    """)
        

def downgrade():
    op.execute("""
        ALTER TABLE flow_flows
        DROP COLUMN on_complete_flow_id;
    """)
    
    op.execute("""
        ALTER TABLE flow_activeflows
        DROP COLUMN on_complete_flow_id;
    """)
