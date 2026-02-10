create table registrar_directs (
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

create unique index idx_registrar_directs_extension_id on registrar_directs(extension_id);
create unique index idx_registrar_directs_hash on registrar_directs(hash);
create index idx_registrar_directs_customer_id on registrar_directs(customer_id);
