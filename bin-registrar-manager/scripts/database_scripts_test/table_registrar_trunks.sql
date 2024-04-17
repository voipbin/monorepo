create table registrar_trunks(
  -- identity
  id            binary(16), -- id
  customer_id   binary(16), -- customer's id

  name    varchar(255),
  detail  varchar(255),

  domain_name varchar(255),
  auth_types  json,

  -- basic auth
  realm    varchar(255),
  username varchar(255),
  password varchar(255),

  -- ip addresses
  allowed_ips json,

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_registrar_trunks_customer_id on registrar_trunks(customer_id);
create index idx_registrar_trunks_domain_name on registrar_trunks(domain_name);
create index idx_registrar_trunks_realm on registrar_trunks(realm);
