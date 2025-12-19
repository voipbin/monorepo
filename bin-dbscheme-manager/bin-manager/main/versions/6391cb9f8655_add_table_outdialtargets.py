"""add table outdialtargets

Revision ID: 6391cb9f8655
Revises: 1491844ad23c
Create Date: 2022-04-08 22:27:58.330452

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '6391cb9f8655'
down_revision = '1491844ad23c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table outdialtargets(
            -- identity
            id          binary(16),
            outdial_id  binary(16),

            name      varchar(255),
            detail    text,

            data    text,
            status  varchar(255),

            -- destinations
            destination_0 json,
            destination_1 json,
            destination_2 json,
            destination_3 json,
            destination_4 json,

            -- try counts
            try_count_0 integer,
            try_count_1 integer,
            try_count_2 integer,
            try_count_3 integer,
            try_count_4 integer,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_outdialtargets_outdial_id on outdialtargets(outdial_id);""")


def downgrade():
    op.execute("""drop table outdialtargets;""")
