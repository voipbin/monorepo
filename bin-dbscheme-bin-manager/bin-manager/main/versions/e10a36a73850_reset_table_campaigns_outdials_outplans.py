"""reset table campaigns outdials outplans

Revision ID: e10a36a73850
Revises: e8b72d3bf93e
Create Date: 2022-04-26 10:37:06.052764

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e10a36a73850'
down_revision = 'e8b72d3bf93e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""drop table campaigns;""")
    op.execute("""drop table campaigncalls;""")
    op.execute("""drop table outplans;""")
    op.execute("""drop table outdials;""")
    op.execute("""drop table outdialtargets;""")

    op.execute("""
        create table campaigncalls(
            -- identity
            id                binary(16),
            customer_id       binary(16),
            campaign_id       binary(16),

            outplan_id        binary(16),
            outdial_id        binary(16),
            outdial_target_id binary(16),
            queue_id          binary(16),

            activeflow_id     binary(16),
            flow_id           binary(16),

            reference_type  varchar(255),
            reference_id    binary(16),

            status  varchar(255),
            result  varchar(255),

            source            json,
            destination       json,
            destination_index integer,
            try_count         integer,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update

            primary key(id)
        );
    """)
    op.execute("""create index idx_campaigncalls_customer_id on campaigncalls(customer_id);""")
    op.execute("""create index idx_campaigncalls_campaign_id on campaigncalls(campaign_id);""")
    op.execute("""create index idx_campaigncalls_outdial_target_id on campaigncalls(outdial_target_id);""")
    op.execute("""create index idx_campaigncalls_activeflow_id on campaigncalls(activeflow_id);""")
    op.execute("""create index idx_campaigncalls_reference_id on campaigncalls(reference_id);""")
    op.execute("""create index idx_campaigncalls_campaign_id_status on campaigncalls(campaign_id, status);""")

    op.execute("""
        create table campaigns(
            -- identity
            id          binary(16),
            customer_id binary(16),

            type varchar(255),

            execute varchar(255),

            name      varchar(255),
            detail    text,

            status        varchar(255),
            service_level integer,
            end_handle  varchar(255),

            flow_id binary(16),
            actions json,

            outplan_id  binary(16),
            outdial_id  binary(16),
            queue_id    binary(16),

            next_campaign_id binary(16),

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_campaigns_customer_id on campaigns(customer_id);""")
    op.execute("""create index idx_campaigns_flow_id on campaigns(flow_id);""")
    op.execute("""create index idx_campaigns_outplan_id on campaigns(outplan_id);""")
    op.execute("""create index idx_campaigns_outdial_id on campaigns(outdial_id);""")
    op.execute("""create index idx_campaigns_queue_id on campaigns(queue_id);""")

    op.execute("""
        create table outplans(
            -- identity
            id          binary(16),
            customer_id binary(16),

            name    varchar(255),
            detail  text,

            source  json,

            dial_timeout  integer,
            try_interval  integer,
            max_try_count_0 integer,
            max_try_count_1 integer,
            max_try_count_2 integer,
            max_try_count_3 integer,
            max_try_count_4 integer,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_outplans_customer_id on outplans(customer_id);""")

    op.execute("""
        create table outdials(
            -- identity
            id          binary(16),
            customer_id binary(16),

            campaign_id binary(16),

            name      varchar(255),
            detail    text,

            data  text,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_outdials_customer_id on outdials(customer_id);""")
    op.execute("""create index idx_outdials_campaign_id on outdials(campaign_id);""")

    op.execute("""
        create table outdialtargets(
            -- identity
            id          binary(16),
            outdial_id  binary(16),

            name      varchar(255),
            detail    text,

            data    text,
            status  varchar(255),

            -- destinations
            destination_0 json,
            destination_1 json,
            destination_2 json,
            destination_3 json,
            destination_4 json,

            -- try counts
            try_count_0 integer,
            try_count_1 integer,
            try_count_2 integer,
            try_count_3 integer,
            try_count_4 integer,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_outdialtargets_outdial_id on outdialtargets(outdial_id);""")


def downgrade():
    op.execute("""drop table campaigns;""")
    op.execute("""drop table campaigncalls;""")
    op.execute("""drop table outplans;""")
    op.execute("""drop table outdials;""")
    op.execute("""drop table outdialtargets;""")

