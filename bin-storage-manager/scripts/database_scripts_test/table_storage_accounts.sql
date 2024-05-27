create table storage_accounts(
  -- identity
  id                binary(16),   -- id
  customer_id       binary(16),   -- customer id

  total_file_count integer, -- total file count
  total_file_size  integer, -- total file size

  tm_create           datetime(6),  -- create
  tm_update           datetime(6),  -- update
  tm_delete           datetime(6),  -- delete

  primary key(id)
);

create index idx_storage_accounts_customer_id on storage_accounts(customer_id);
