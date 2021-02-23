"""add table extensions

Revision ID: c939ba877f8f
Revises: 6a7807e971e1
Create Date: 2021-02-23 14:17:17.716292

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = 'c939ba877f8f'
down_revision = '6a7807e971e1'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'extensions' in tables:
        return

    op.execute("""
        create table extensions (
            -- identity
            id            binary(16), -- id
            user_id       integer, -- user id

            name    varchar(255),
            detail  varchar(255),

            domain_id     binary(16),
            endpoint_id   varchar(255),
            aor_id        varchar(255),
            auth_id       varchar(255),

            extension     varchar(255),
            password      varchar(255),

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_extensions_user_id on extensions(user_id);
    """)

    op.execute("""
        create index idx_extensions_domain_id on extensions(domain_id);
    """)


def downgrade():
    op.execute("""
        drop table extensions;
    """)
