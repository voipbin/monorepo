"""add table activeflows

Revision ID: d07e532d2a28
Revises: 45acd7839db4
Create Date: 2022-03-27 17:18:43.178429

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd07e532d2a28'
down_revision = '45acd7839db4'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
      create table activeflows(
        -- identity
        id          binary(16),
        customer_id binary(16),
        flow_id     binary(16),

        reference_type  varchar(255),
        reference_id    binary(16),

        current_action        json,
        execute_count         integer,
        forward_action_id     binary(16),

        actions           json,
        executed_actions  json,

        -- timestamps
        tm_create datetime(6),  -- create
        tm_update datetime(6),  -- update
        tm_delete datetime(6),  -- delete

        primary key(id)
      );
    """)
    op.execute("""create index idx_activeflows_customer_id on activeflows(customer_id);""")
    op.execute("""create index idx_activeflows_flow_id on activeflows(flow_id);""")
    op.execute("""create index idx_activeflows_reference_id on activeflows(reference_id);""")


def downgrade():
    op.execute("""drop table activeflows;""")

