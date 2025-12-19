"""add table confbridge

Revision ID: 6102c5d4793b
Revises: 311a5418bf20
Create Date: 2021-11-01 01:22:03.037062

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector



# revision identifiers, used by Alembic.
revision = '6102c5d4793b'
down_revision = '311a5418bf20'
branch_labels = None
depends_on = None


def upgrade():
    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'confbridges' in tables:
        return

    op.execute("""
        create table confbridges(
            -- identity
            id            binary(16),   -- id
            conference_id binary(16),   -- conference id
            type          varchar(255), -- type
            bridge_id     varchar(255), -- conference's bridge id

            channel_call_ids  json, -- channel_id-call_id in the confbridge

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

    op.execute("""create index idx_confbridges_create on confbridges(tm_create);""")
    op.execute("""create index idx_confbridges_conference_id on confbridges(conference_id);""")
    op.execute("""create index idx_confbridges_bridge_id on confbridges(bridge_id);""")



def downgrade():
    op.execute("""
        drop table confbridges;
    """)

