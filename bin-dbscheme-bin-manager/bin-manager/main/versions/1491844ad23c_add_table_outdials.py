"""add table outdials

Revision ID: 1491844ad23c
Revises: ca9787b1985a
Create Date: 2022-04-08 22:03:20.657206

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1491844ad23c'
down_revision = 'ca9787b1985a'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table outdials(
            -- identity
            id          binary(16),
            customer_id binary(16),

            campaign_id binary(16),

            name      varchar(255),
            detail    text,

            data  text,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_outdials_customer_id on outdials(customer_id);""")
    op.execute("""create index idx_outdials_campaign_id on outdials(campaign_id);""")


def downgrade():
    op.execute("""drop table outdials;""")

