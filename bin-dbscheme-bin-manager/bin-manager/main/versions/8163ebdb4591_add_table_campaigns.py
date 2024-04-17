"""add table campaigns

Revision ID: 8163ebdb4591
Revises: 4ee7f2705167
Create Date: 2022-04-08 22:54:26.584105

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '8163ebdb4591'
down_revision = '4ee7f2705167'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table campaigns(
            -- identity
            id          binary(16),
            customer_id binary(16),

            name      varchar(255),
            detail    text,

            status          varchar(255),
            service_level   integer,

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
    op.execute("""create index idx_campaigns_outplan_id on campaigns(outplan_id);""")
    op.execute("""create index idx_campaigns_outdial_id on campaigns(outdial_id);""")
    op.execute("""create index idx_campaigns_queue_id on campaigns(queue_id);""")


def downgrade():
    op.execute("""drop table campaigns;""")
