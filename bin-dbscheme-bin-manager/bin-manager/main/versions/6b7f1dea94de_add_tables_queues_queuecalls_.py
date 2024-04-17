"""add tables queues queuecalls queuereferences

Revision ID: 6b7f1dea94de
Revises: d984e97d123b
Create Date: 2021-12-20 11:20:43.932776

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '6b7f1dea94de'
down_revision = 'd984e97d123b'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table queues(
            -- identity
            id                binary(16),   -- id
            user_id           integer,      -- user's id
            flow_id           binary(16),
            confbridge_id     binary(16),
            forward_action_id binary(16),

            -- basic info
            name            varchar(255),
            detail          text,
            webhook_uri     varchar(1023),  -- webhook uri
            webhook_method  varchar(16),

            routing_method  varchar(16),
            tag_ids         json,

            -- wait/service info
            wait_actions            json,
            wait_queue_call_ids     json,
            wait_timeout            integer,  -- wait timeout(ms)
            service_queue_call_ids  json,
            service_timeout         integer,  -- service timeout(ms)

            total_incoming_count    integer,  -- total incoming count
            total_serviced_count    integer,  -- total serviced count
            total_abandoned_count   integer,  -- total abandoned count
            total_wait_duration     integer,  -- total wait duration(ms)
            total_service_duration  integer,  -- total service duration(ms)

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)
    op.execute("""
        create index idx_queues_userid on queues(user_id);
    """)
    op.execute("""
        create index idx_queues_flowid on queues(flow_id);
    """)
    op.execute("""
        create index idx_queues_confbridgeid on queues(confbridge_id);
    """)

    op.execute("""
        create table queuecalls(
            -- identity
            id                binary(16),   -- id
            user_id           integer,      -- user's id

            queue_id          binary(16),   -- queue id
            reference_type    varchar(255), -- reference's type
            reference_id      binary(16),   -- reference's id
            forward_action_id binary(16),   -- action id for forward.
            exit_action_id    binary(16),   -- action id for queue exit.
            confbridge_id     binary(16),

            webhook_uri     varchar(1023),
            webhook_method  varchar(255),

            source            json,
            routing_method    varchar(255),
            tag_ids           json,

            status            varchar(255), --
            service_agent_id  binary(16),   --

            timeout_wait      integer,  --
            timeout_service   integer,  --

            tm_create   datetime(6),
            tm_service  datetime(6),
            tm_update   datetime(6),
            tm_delete   datetime(6),

            primary key(id)
        );
    """)
    op.execute("""
        create index idx_queuecalls_userid on queuecalls(user_id);
    """)
    op.execute("""
        create index idx_queuecalls_queueid on queuecalls(queue_id);
    """)
    op.execute("""
        create index idx_queuecalls_referenceid on queuecalls(reference_id);
    """)
    op.execute("""
        create index idx_queuecalls_serviceagentid on queuecalls(service_agent_id);
    """)

    op.execute("""
        create table queuecallreferences(
            -- identity
            id        binary(16),   -- id
            user_id   integer,
            type      varchar(255),

            current_queuecall_id  binary(16),
            queuecall_ids         json,

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)


def downgrade():
    op.execute("""
        drop table queues;
    """)

    op.execute("""
        drop table queuecalls;
    """)

    op.execute("""
        drop table queuecallreferences;
    """)
