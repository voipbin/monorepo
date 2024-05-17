create table storage_files(
  -- identity
  id                binary(16),   -- id
  customer_id       binary(16),   -- customer id
  owner_id          binary(16), -- owner id

  name      varchar(255),
  detail    text,

  uri_bucket    text,
  uri_download  text,

  -- timestamps
  tm_download_expire  datetime(6),
  tm_create           datetime(6),  -- create
  tm_update           datetime(6),  -- update
  tm_delete           datetime(6),  -- delete

  primary key(id)
);

create index idx_storage_files_customer_id on storage_files(customer_id);
