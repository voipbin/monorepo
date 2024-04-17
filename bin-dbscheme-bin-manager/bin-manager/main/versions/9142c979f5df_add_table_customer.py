"""add table customer

Revision ID: 9142c979f5df
Revises: 2b56bd21e419
Create Date: 2022-01-28 23:58:33.468533

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9142c979f5df'
down_revision = '2b56bd21e419'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table customers(
        -- identity
        id            binary(16) primary key,  -- id

        username      varchar(255),     -- username
        password_hash varchar(1023),    -- password hash

        name            varchar(255),
        detail          text,
        webhook_method  varchar(255),
        webhook_uri     varchar(1023),

        permission_ids json,

        tm_create datetime(6),
        tm_update datetime(6),
        tm_delete datetime(6)
        );
    """)

    op.execute("""
        create index idx_customers_username on customers(username);
    """)



def downgrade():
    op.execute("""
        drop table customers;
    """)

