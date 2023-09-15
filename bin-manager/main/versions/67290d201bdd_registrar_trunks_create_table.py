"""registrar_trunks create table

Revision ID: 67290d201bdd
Revises: 73f0a892b2de
Create Date: 2023-09-15 01:30:43.754311

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '67290d201bdd'
down_revision = '73f0a892b2de'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table registrar_trunks(
            -- identity
            id            binary(16), -- id
            customer_id   binary(16), -- customer's id

            name    varchar(255),
            detail  varchar(255),

            domain_name varchar(255),
            auth_types  json,

            -- basic auth
            username varchar(255),
            password varchar(255),

            -- ip addresses
            allowed_ips json,

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_registrar_trunks_customer_id on registrar_trunks(customer_id);""")
    op.execute("""create index idx_registrar_trunks_domain_name on registrar_trunks(domain_name);""")


def downgrade():
    op.execute("""drop table registrar_trunks;""")
