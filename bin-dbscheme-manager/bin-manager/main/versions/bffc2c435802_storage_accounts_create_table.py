"""storage_accounts create table

Revision ID: bffc2c435802
Revises: 68a9b4bd0e5c
Create Date: 2024-05-24 16:34:10.301279

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'bffc2c435802'
down_revision = '68a9b4bd0e5c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""create table storage_accounts(
        -- identity
        id                binary(16),   -- id
        customer_id       binary(16),   -- customer id

        total_file_count    bigint, -- total file count
        total_file_size     bigint, -- total file size

        tm_create           datetime(6),  -- create
        tm_update           datetime(6),  -- update
        tm_delete           datetime(6),  -- delete

        primary key(id)
        );
    ;""")
    op.execute("""create index idx_storage_accounts_customer_id on storage_accounts(customer_id);""")

    # create storage_accounts rows from the customers table
    op.execute("""
        insert into storage_accounts(
            id, customer_id, total_file_count, total_file_size, tm_create, tm_update, tm_delete
        )
        select unhex(replace(UUID(), '-', '')), id, 0, 0, now(6), now(6), "9999-01-01 00:00:000" from customers;
    """)


    # modify the storage_files table
    op.execute("""alter table storage_files modify column filename varchar(1023) after bucket_name;""")
    op.execute("""alter table storage_files add column filesize bigint after filepath;""")
    op.execute("""update storage_files set filesize = 0;""")

    op.execute("""alter table storage_files add column account_id binary(16) after customer_id;""")
    op.execute("""create index idx_storage_files_account_id on storage_files(account_id);""")
    op.execute("""update storage_files set account_id = customer_id;""")




def downgrade():
    op.execute("""drop table storage_accounts;""")
    op.execute("""alter table storage_files drop column filesize;""")
    op.execute("""alter table storage_files drop column account_id;""")
