"""add table groupcalls

Revision ID: 25b1ed87e878
Revises: aa8826443972
Create Date: 2023-03-08 16:06:34.778390

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '25b1ed87e878'
down_revision = 'aa8826443972'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table groupcalls(
            -- identity
            id                binary(16),   -- id
            customer_id       binary(16),   -- customer id

            source        json,
            destinations  json,

            ring_method   varchar(255),
            answer_method varchar(255),

            answer_call_id    binary(16),
            call_ids          json,

            -- timestamps
            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)
    op.execute("""create index idx_groupcalls_customer_id on groupcalls(customer_id);""")


def downgrade():
    op.execute("""drop table groupcalls;""")
