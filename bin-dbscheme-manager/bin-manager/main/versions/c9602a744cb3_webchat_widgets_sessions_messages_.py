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
            id                    binary(16)   not null,
            customer_id           binary(16)   not null,

            -- basic info
            name                  varchar(255),
            status                varchar(16)  not null,

            -- direct hash -- nullable: a Widget with no DirectID is
            -- "provisioning incomplete" (see widget.go doc comment).
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

    # direct_id is the visitor-facing linkage to bin-direct-manager
    # (resource_type=webchat_widget, resource_id=Widget.ID). Two
    # widgets must never share the same direct_id -- a unique index
    # enforces that invariant at the DB layer (defense in depth on
    # top of bin-direct-manager's own uniqueness), and doubles as the
    # index needed for any future direct_id lookup path.
    op.execute("""
        create unique index idx_webchat_widgets_direct_id on webchat_widgets(direct_id);
    """)

    op.execute("""
        create table webchat_sessions(
            -- identity
            id                binary(16)   not null,
            customer_id       binary(16)   not null,
            widget_id         binary(16)   not null,

            status            varchar(16)  not null,
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

    # widget-scoped session listing (WHERE widget_id = ? AND tm_delete
    # IS NULL AND tm_create < ? ORDER BY tm_create DESC, per
    # dbhandler.SessionList) isn't covered by the (widget_id, status)
    # index above -- mirrors the analogous fix on webchat_messages.
    op.execute("""
        create index idx_webchat_sessions_widget_id_tm_create on webchat_sessions(widget_id, tm_create);
    """)

    op.execute("""
        create index idx_webchat_sessions_status_tm_last_activity on webchat_sessions(status, tm_last_activity);
    """)

    op.execute("""
        create table webchat_messages(
            -- identity
            id             binary(16)   not null,
            customer_id    binary(16)   not null,
            widget_id      binary(16)   not null,
            session_id     binary(16)   not null,

            direction      varchar(16)  not null,
            status         varchar(16)  not null,
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

    # widget_id is denormalized onto every message so downstream event
    # consumers can build Conversation.Self without a join back through
    # webchat_sessions -- index it for any widget-scoped message query.
    op.execute("""
        create index idx_webchat_messages_widget_id_tm_create on webchat_messages(widget_id, tm_create);
    """)


def downgrade():
    op.execute("""drop table if exists webchat_messages;""")
    op.execute("""drop table if exists webchat_sessions;""")
    op.execute("""drop table if exists webchat_widgets;""")
