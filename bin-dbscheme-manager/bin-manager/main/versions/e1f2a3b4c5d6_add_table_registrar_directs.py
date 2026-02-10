"""add_table_registrar_directs

Revision ID: e1f2a3b4c5d6
Revises: d4e5f6a7b8c9
Create Date: 2026-02-11 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'e1f2a3b4c5d6'
down_revision = 'd4e5f6a7b8c9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table registrar_directs(
            -- identity
            id            binary(16),
            customer_id   binary(16),

            extension_id  binary(16),
            hash          varchar(255),

            -- timestamps
            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)

    op.execute("create unique index idx_registrar_directs_extension_id on registrar_directs(extension_id);")
    op.execute("create unique index idx_registrar_directs_hash on registrar_directs(hash);")
    op.execute("create index idx_registrar_directs_customer_id on registrar_directs(customer_id);")


def downgrade():
    op.execute("drop table if exists registrar_directs;")
