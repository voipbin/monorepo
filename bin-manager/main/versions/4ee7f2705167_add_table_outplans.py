"""add table outplans

Revision ID: 4ee7f2705167
Revises: 6391cb9f8655
Create Date: 2022-04-08 22:31:42.090090

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '4ee7f2705167'
down_revision = '6391cb9f8655'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table outplans(
            -- identity
            id          binary(16),
            customer_id binary(16),

            name      varchar(255),
            detail    text,

            actions       json,
            source        json,
            dial_timeout  integer,
            end_handle    varchar(255),
            try_interval  integer,

            max_try_count_0 integer,
            max_try_count_1 integer,
            max_try_count_2 integer,
            max_try_count_3 integer,
            max_try_count_4 integer,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_outplans_customer_id on outplans(customer_id);""")


def downgrade():
    op.execute("""drop table outplans;""")

