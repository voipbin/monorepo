"""add table route

Revision ID: 32b74b818313
Revises: 0eb33809a60c
Create Date: 2022-10-16 21:52:59.614590

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '32b74b818313'
down_revision = '0eb33809a60c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table routes(
            -- identity
            id          binary(16),
            customer_id binary(16),

            provider_id binary(16),
            priority    int,

            target varchar(255),

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update

            primary key(id)
        );
    """)
    op.execute("""
        create index idx_routes_customer_id on routes(customer_id);
    """)
    op.execute("""
        create index idx_routes_provider_id on routes(provider_id);
    """)

    op.execute("""
        create table providers(
            -- identity
            id          binary(16),

            type      varchar(255),
            hostname  varchar(255),

            tech_prefix   varchar(255),
            tech_postfix  varchar(255),
            tech_headers  json,

            name      varchar(255),
            detail    text,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)


def downgrade():
    op.execute("""drop table routes;""")
    op.execute("""drop table providers;""")

