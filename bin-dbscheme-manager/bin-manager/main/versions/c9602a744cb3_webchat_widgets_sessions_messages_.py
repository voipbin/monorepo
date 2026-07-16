"""webchat_widgets_sessions_messages_create_tables

Revision ID: c9602a744cb3
Revises: 43947d3f7312
Create Date: 2026-07-16 23:00:11.836228

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c9602a744cb3'
down_revision = '43947d3f7312'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table webchat_widgets(
            -- identity
            id                    binary(16),
            customer_id           binary(16),

            -- basic info
            name                  varchar(255),
            status                varchar(16),

            -- direct hash
            direct_id             binary(16),

            welcome_message       text,
            flow_id               binary(16),
            session_idle_timeout  integer,

            theme_config          json,

            tm_create             datetime(6),
            tm_update             datetime(6),
            tm_delete             datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_webchat_widgets_customer_id_tm_create on webchat_widgets(customer_id, tm_create);
    """)

    op.execute("""
        create index idx_webchat_widgets_customer_id_tm_delete on webchat_widgets(customer_id, tm_delete);
    """)

    op.execute("""
        create table webchat_sessions(
            -- identity
            id                binary(16),
            customer_id       binary(16),
            widget_id         binary(16),

            status            varchar(16),
            activeflow_id     binary(16),

            tm_last_activity  datetime(6),
            tm_create         datetime(6),
            tm_update         datetime(6),
            tm_end            datetime(6),
            tm_delete         datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_webchat_sessions_customer_id_tm_create on webchat_sessions(customer_id, tm_create);
    """)

    op.execute("""
        create index idx_webchat_sessions_customer_id_tm_delete on webchat_sessions(customer_id, tm_delete);
    """)

    op.execute("""
        create index idx_webchat_sessions_widget_id_status on webchat_sessions(widget_id, status);
    """)

    op.execute("""
        create index idx_webchat_sessions_status_tm_last_activity on webchat_sessions(status, tm_last_activity);
    """)

    op.execute("""
        create table webchat_messages(
            -- identity
            id             binary(16),
            customer_id    binary(16),
            widget_id      binary(16),
            session_id     binary(16),

            direction      varchar(16),
            status         varchar(16),
            text           varchar(4000),
            sender_id      binary(16),
            activeflow_id  binary(16),

            tm_create      datetime(6),
            tm_delete      datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_webchat_messages_session_id_tm_create on webchat_messages(session_id, tm_create);
    """)

    op.execute("""
        create index idx_webchat_messages_customer_id_tm_create on webchat_messages(customer_id, tm_create);
    """)


def downgrade():
    op.execute("""drop table if exists webchat_messages;""")
    op.execute("""drop table if exists webchat_sessions;""")
    op.execute("""drop table if exists webchat_widgets;""")
