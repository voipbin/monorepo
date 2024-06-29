"""agent_resources remove table

Revision ID: a68766c6b227
Revises: c3fb144b1c95
Create Date: 2024-06-28 21:25:06.078811

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a68766c6b227'
down_revision = 'c3fb144b1c95'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""drop table agent_resources;""")


def downgrade():
    op.execute("""
        create table agent_resources(
            -- identity
            id            binary(16),
            customer_id   binary(16),
            owner_id      binary(16),

            reference_type varchar(255),
            reference_id   binary(16),

            data json,

            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)
    op.execute("""create index idx_agent_resources_customerid on agent_resources(customer_id);""")
    op.execute("""create index idx_agent_resources_ownerid on agent_resources(owner_id);""")
    op.execute("""create index idx_agent_resources_referenceid on agent_resources(reference_id);""")
    op.execute("""create index idx_agent_resources_referencetype_referenceid on agent_resources(reference_type, reference_id);""")

