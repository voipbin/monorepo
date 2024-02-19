"""domains drop table

Revision ID: b39053ce4bd6
Revises: 8648107ccabb
Create Date: 2024-02-19 18:55:43.729094

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'b39053ce4bd6'
down_revision = '8648107ccabb'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""drop table domains;""")


def downgrade():
    op.execute("""
        create table domains(
            -- identity
            id          binary(16), -- id
            customer_id binary(16),
            user_id     integer,    -- user id

            name    varchar(255),
            detail  varchar(255),

            domain_name varchar(255),

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)

    op.execute("""create index idx_domains_user_id on domains(user_id);""")
    op.execute("""create index idx_domains_domain_name on domains(domain_name);""")
    op.execute("""create index idx_domains_customerid on domains(customer_id);""")
