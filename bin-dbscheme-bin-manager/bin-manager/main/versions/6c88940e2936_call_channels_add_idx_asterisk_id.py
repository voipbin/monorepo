"""call_channels: add index on (asterisk_id, type, tm_create)

Revision ID: 6c88940e2936
Revises: 04877993e57e
Create Date: 2025-06-14 15:07:07.550007

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '6c88940e2936'
down_revision = '04877993e57e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create index idx_call_channels_asterisk_id_type_create 
        on call_channels(asterisk_id, type, tm_create);
    """)


def downgrade():
    op.execute("""
        alter table call_channels 
        drop index idx_call_channels_asterisk_id_type_create;
    """)
