"""add_table_bridge

Revision ID: c11929334165
Revises: df5313d6701e
Create Date: 2021-02-23 12:24:32.438501

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c11929334165'
down_revision = 'df5313d6701e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table if not exists bridges(
        -- identity
        asterisk_id varchar(255), -- Asterisk id
        id          varchar(255), -- bridge id
        name        varchar(255), -- bridge name

        -- info
        type    varchar(255),   -- bridge's type
        tech    varchar(255),   -- bridge's technology
        class   varchar(255),   -- bridge's class
        creator varchar(255),   -- bridge creator

        -- video info
        video_mode      varchar(255), -- video's mode.
        video_source_id varchar(255),

        -- joined channel info
        channel_ids json, -- joined channel ids

        -- record info
        record_channel_id varchar(255), -- recording channel id
        record_files      json,         -- recording filenames

        -- conference info
        conference_id   binary(16),   -- conference's id
        conference_type varchar(255), -- conference's type
        conference_join boolean,      -- true it this bridge is joining

        -- timestamps
        tm_create datetime(6),  --
        tm_update datetime(6),  --
        tm_delete datetime(6),  --

        primary key(id)
        );
    """)

    op.execute("""
        create index idx_bridges_create on bridges(tm_create);
    """)


def downgrade():
    op.execute("""
        drop table bridges;
    """)
