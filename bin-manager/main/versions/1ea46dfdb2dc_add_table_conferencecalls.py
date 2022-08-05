"""add table conferencecalls

Revision ID: 1ea46dfdb2dc
Revises: 1c016f68f899
Create Date: 2022-08-06 00:01:21.573883

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1ea46dfdb2dc'
down_revision = '1c016f68f899'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conferences add conferencecall_ids json after post_actions;""")
    op.execute("""alter table conferences drop column call_ids;""")

    op.execute("""
        create table conferencecalls(
            -- identity
            id            binary(16),   -- id
            customer_id   binary(16),   -- customer id
            conference_id binary(16),

            reference_type    varchar(255),
            reference_id      binary(16),

            status  varchar(255),   -- status

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_conferencecalls_customer_id on conferencecalls(customer_id);""")
    op.execute("""create index idx_conferencecalls_conference_id on conferencecalls(conference_id);""")
    op.execute("""create index idx_conferencecalls_reference_id on conferencecalls(reference_id);""")
    op.execute("""create index idx_conferencecalls_create on conferencecalls(tm_create);""")



def downgrade():
    op.execute("""alter table conferences add call_ids json after post_actions;""")
    op.execute("""alter table conferences drop column conferencecall_ids;""")

    op.execute("""drop table conferencecalls;""")

