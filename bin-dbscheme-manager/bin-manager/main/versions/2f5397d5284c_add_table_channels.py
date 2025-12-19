"""add table channels

Revision ID: 2f5397d5284c
Revises: c7ff733dd349
Create Date: 2021-02-23 13:40:33.642370

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = '2f5397d5284c'
down_revision = 'c7ff733dd349'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'channels' in tables:
        return

    op.execute("""
        create table channels(
            -- identity
            id          varchar(255), -- channel id
            asterisk_id varchar(255), -- Asterisk id
            name        varchar(255), -- channel name
            type        varchar(255), -- channel's type. this context sets by call/conference handler.
            tech        varchar(255), -- channel driver. pjsip, snoop, sip, ...

            -- sip info
            sip_call_id     varchar(255), -- sip call id.
            sip_transport   varchar(255), -- sip transport type. udp, tcp, tls, wss, ...

            -- src/dst
            src_name    varchar(255), -- source name
            src_number  varchar(255), -- source number
            dst_name    varchar(255), -- destination name
            dst_number  varchar(255), -- destination number

            -- info
            state     varchar(255), -- current state.
            data      json,         -- additional data. sip headers, and so on...
            stasis    varchar(255), -- stasis application name.
            bridge_id varchar(255), -- bridge id

            dial_result       varchar(255), -- dial result. answer, busy, cancel, ...
            hangup_cause      int,          -- hangup cause code.

            direction varchar(255), -- channel's direction. incoming, outgoing

            -- timestamps
            tm_create datetime(6),  -- created timestamp
            tm_update datetime(6),  -- last updated timestamp

            tm_answer datetime(6),  -- answer timestamp
            tm_ringing datetime(6), -- rining timestamp
            tm_end datetime(6),     -- end timestamp

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_channels_create on channels(tm_create);
    """)

    op.execute("""
        create index idx_channels_src_number on channels(src_number);
    """)

    op.execute("""
        create index idx_channels_dst_number on channels(dst_number);
    """)

    op.execute("""
        create index idx_channels_sip_call_id on channels(sip_call_id);
    """)

def downgrade():
    op.execute("""
        drop table channels;
    """)
