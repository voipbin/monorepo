"""add table recordings

Revision ID: 4c348a96c3b8
Revises: 520eee292397
Create Date: 2021-02-23 13:58:27.037461

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = '4c348a96c3b8'
down_revision = '520eee292397'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'recordings' in tables:
        return

    op.execute("""
        create table recordings(
            -- identity
            id                binary(16),     -- recording's id(name)
            user_id           integer,        -- user id
            type              varchar(16),    -- type of record. call, conference
            reference_id      binary(16),     -- referenced id. call-id, conference-id
            status            varchar(255),   -- current status of record.
            format            varchar(16),    -- recording's format. wav, ...
            filename          varchar(255),   -- recording's filename.

            -- asterisk info
            asterisk_id       varchar(255), -- Asterisk id
            channel_id        varchar(255), -- channel id

            -- timestamps
            tm_start  datetime(6),    -- record started timestamp.
            tm_end    datetime(6),    -- record ended timestamp.

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_recordings_user_id on recordings(user_id);
    """)

    op.execute("""
        create index idx_recordings_tm_start on recordings(tm_start);
    """)

    op.execute("""
        create index idx_recordings_filename on recordings(filename);
    """)


def downgrade():
    op.execute("""
        drop table recordings;
    """)
