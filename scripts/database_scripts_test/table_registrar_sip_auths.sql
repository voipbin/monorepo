create table registrar_sip_auths(
  -- identity
  id              binary(16), -- id
  reference_type  varchar(255),

  auth_types  json,
  realm       varchar(255),

  username varchar(255),
  password varchar(255),

  allowed_ips json,

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --

  primary key(id)
);

create index idx_registrar_sip_auths_realm on registrar_sip_auths(realm);
