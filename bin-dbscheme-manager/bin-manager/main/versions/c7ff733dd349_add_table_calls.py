"""add table calls

Revision ID: c7ff733dd349
Revises: c11929334165
Create Date: 2021-02-23 13:25:22.387743

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = 'c7ff733dd349'
down_revision = 'c11929334165'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'calls' in tables:
        return

    op.execute("""
        create table calls(
            -- identity
            id                binary(16),   -- id
            user_id           integer,      -- user id
            asterisk_id       varchar(255), -- Asterisk id
            channel_id        varchar(255), -- channel id
            flow_id           binary(16),   -- flow id
            conference_id     binary(16),   -- currently joined conference id
            type              varchar(16),  -- type of call

            -- etc info
            master_call_id    binary(16),   -- master call id
            chained_call_ids  json,         -- chained call ids

            -- recording info
            recording_id  binary(16),     -- current recording id
            recording_ids json,           -- recording ids

            -- source/destination
            source        json, -- source's type, target, number, name, ...
            source_target varchar(1024),

            destination         json, -- destination's type, target, number, name, ...
            destination_target  varchar(1024),

            -- info
            status            varchar(255), -- current status of call.
            data              json,         -- additional data. sip headers, and so on...
            action            json,         -- current action
            direction         varchar(16),  -- direction of call. incoming/outgoing
            hangup_by         varchar(16),  -- local/remote/empty for not sure.
            hangup_reason     varchar(16),  -- reason

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --

            tm_progressing  datetime(6), -- progrssing timestamp
            tm_ringing      datetime(6), -- rining timestamp
            tm_hangup       datetime(6), -- end timestamp

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_calls_channelid on calls(channel_id);
    """)

    op.execute("""
        create index idx_calls_flowid on calls(flow_id);
    """)

    op.execute("""
        create index idx_calls_create on calls(tm_create);
    """)

    op.execute("""
        create index idx_calls_hangup on calls(tm_hangup);
    """)

    op.execute("""
        create index idx_calls_source_target on calls(source_target);
    """)

    op.execute("""
        create index idx_calls_destination_target on calls(destination_target);
    """)

    op.execute("""
        create index idx_calls_user_id on calls(user_id);
    """)


def downgrade():
    op.execute("""
        drop table calls;
    """)
