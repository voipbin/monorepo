"""add table registrar_sip_auths

Revision ID: 8648107ccabb
Revises: 066748327ce8
Create Date: 2024-02-18 00:55:09.696361

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '8648107ccabb'
down_revision = '066748327ce8'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table registrar_sip_auths(
            id              binary(16),
            reference_type  varchar(255),

            auth_types  json,
            realm       varchar(255),

            username varchar(255),
            password varchar(255),

            allowed_ips json,

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_registrar_sip_auths_realm on registrar_sip_auths(realm);""")

    # data migration
    op.execute("""
                insert into registrar_sip_auths(
                    id, reference_type, auth_types, realm, username, password, allowed_ips, tm_create, tm_update
                )
                select id, "trunk", auth_types, realm, username, password, allowed_ips, tm_create, tm_update from registrar_trunks where tm_delete >= '9999-01-01 00:00:00.000000';
                """)
    op.execute("""
                insert into registrar_sip_auths(
                    id, reference_type, auth_types, realm, username, password, allowed_ips, tm_create, tm_update
                )
                select id, "extension", '["basic"]', realm, username, password, '[]', tm_create, tm_update from extensions where tm_delete >= '9999-01-01 00:00:00.000000';
                """)


def downgrade():
    op.execute("""
        drop table registrar_sip_auths;
    """)

