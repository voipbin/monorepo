"""add table domains

Revision ID: 6a7807e971e1
Revises: 890ee70f6ffd
Create Date: 2021-02-23 14:12:25.516482

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.engine.reflection import Inspector


# revision identifiers, used by Alembic.
revision = '6a7807e971e1'
down_revision = '890ee70f6ffd'
branch_labels = None
depends_on = None


def upgrade():

    conn = op.get_bind()
    inspector = Inspector.from_engine(conn)
    tables = inspector.get_table_names()

    if 'domains' in tables:
        return

    op.execute("""
        create table domains(
            -- identity
            id        binary(16), -- id
            user_id   integer,      -- user id

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

    op.execute("""
        create index idx_domains_user_id on domains(user_id);
    """)

    op.execute("""
        create index idx_domains_domain_name on domains(domain_name);
    """)



def downgrade():
    op.execute("""
        drop table domains;
    """)
