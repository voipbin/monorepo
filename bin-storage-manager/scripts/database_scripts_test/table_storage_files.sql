create table storage_files(
  -- identity
  id                binary(16),   -- id
  customer_id       binary(16),   -- customer id
  owner_id          binary(16),   -- owner id

  reference_type  varchar(255),   -- reference type
  reference_id    binary(16),     -- reference id

  name      varchar(255),
  detail    text,
  filename  varchar(1023),

  bucket_name text,
  filepath    text,
  filesize    integer,

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
create index idx_storage_files_owner_id on storage_files(owner_id);
create index idx_storage_files_reference_id on storage_files(reference_id);
