create table extensions (
  -- identity
  id            binary(16), -- id
  customer_id   binary(16), -- owner's id

  name    varchar(255),
  detail  varchar(255),

  endpoint_id   varchar(255),
  aor_id        varchar(255),
  auth_id       varchar(255),

  extension     varchar(255),
  domain_name   varchar(255),

  realm         varchar(255),
  username      varchar(255),
  password      varchar(255),

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)

);

create index idx_extensions_customerid on extensions(customer_id);
create index idx_extensions_extension on extensions(extension);
create index idx_extensions_domain_name on extensions(domain_name);
create index idx_extensions_username on extensions(username);
create index idx_extensions_realm on extensions(realm);
