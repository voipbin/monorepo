"""add table conferences

Revision ID: 3743a43a215d
Revises: 2f5397d5284c
Create Date: 2021-02-23 13:42:49.014520

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = '3743a43a215d'
down_revision = '2f5397d5284c'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'conferences' in tables:
        return

    op.execute("""
        create table conferences(
            -- identity
            id        binary(16),   -- id
            user_id   integer,      -- user id
            type      varchar(255), -- type
            bridge_id varchar(255), -- conference's bridge id

            -- info
            status  varchar(255),   -- status
            name    varchar(255),   -- conference's name
            detail  text,           -- conference's detail description
            data    json,           -- additional data
            timeout int,            -- timeout. second

            call_ids   json, -- calls in the conference

            -- record info
            recording_id   binary(16),  -- current record id
            recording_ids  json,        -- record ids

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_conferences_create on conferences(tm_create);
    """)

    op.execute("""
        create index idx_conferences_user_id on conferences(user_id);
    """)

def downgrade():
    op.execute("""
        drop table conferences;
    """)
