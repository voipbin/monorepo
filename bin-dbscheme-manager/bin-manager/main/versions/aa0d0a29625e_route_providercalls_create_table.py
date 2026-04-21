"""route_providercalls create table

Revision ID: aa0d0a29625e
Revises: 1dfbaffb90cc
Create Date: 2026-04-21 23:35:03.145731

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'aa0d0a29625e'
down_revision = '1dfbaffb90cc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table route_providercalls(
            -- identity
            id             binary(16),

            -- requested
            customer_id    binary(16),
            provider_id    binary(16),
            flow_id        binary(16),
            source         json,
            destinations   json,
            anonymous      varchar(16),

            -- created
            call_ids       json,
            groupcall_ids  json,

            -- timestamps
            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_route_providercalls_customer_id on route_providercalls(customer_id);
    """)
    op.execute("""
        create index idx_route_providercalls_provider_id on route_providercalls(provider_id);
    """)
    # Compound index for the list query pattern: WHERE tm_delete IS NULL ORDER BY tm_create DESC.
    # MySQL can use the leading column for the equality predicate and the trailing column for ordering,
    # so as soft-deleted rows accumulate, list queries stay index-backed instead of scanning.
    op.execute("""
        create index idx_route_providercalls_tm_delete_tm_create on route_providercalls(tm_delete, tm_create);
    """)


def downgrade():
    op.execute("""
        drop table route_providercalls;
    """)
