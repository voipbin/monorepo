"""storage_files create table

Revision ID: 58557426bd74
Revises: 20fbe7fc5157
Create Date: 2024-05-19 23:45:38.141844

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '58557426bd74'
down_revision = '20fbe7fc5157'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table storage_files(
            -- identity
            id                binary(16),   -- id
            customer_id       binary(16),   -- customer id
            owner_id          binary(16),   -- owner id

            reference_type  varchar(255),   -- reference type
            reference_id    binary(16),     -- reference id

            name      varchar(255),
            detail    text,

            bucket_name text,
            filepath    text,

            uri_bucket    text,
            uri_download  text,

            -- timestamps
            tm_download_expire  datetime(6),
            tm_create           datetime(6),  -- create
            tm_update           datetime(6),  -- update
            tm_delete           datetime(6),  -- delete

            primary key(id)
        );
    ;""")
    op.execute("""create index idx_storage_files_customer_id on storage_files(customer_id);""");
    op.execute("""create index idx_storage_files_owner_id on storage_files(owner_id);""");
    op.execute("""create index idx_storage_files_reference_id on storage_files(reference_id);""");

def downgrade():
    op.execute("""drop table storage_files;""")
