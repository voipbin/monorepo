create table direct_directs(
  -- identity
  id            binary(16),
  customer_id   binary(16),

  -- basic info
  resource_type varchar(32),
  resource_id   binary(16),
  hash          varchar(255),

  tm_create datetime(6),
  tm_update datetime(6),

  primary key(id)
);

create unique index idx_direct_directs_hash on direct_directs(hash);
create unique index idx_direct_directs_resource on direct_directs(resource_type, resource_id);
create index idx_direct_directs_customer_id on direct_directs(customer_id);
