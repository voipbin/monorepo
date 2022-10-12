create table providers(
  -- identity
  id          binary(16),

  type      varchar(255),
  hostname  varchar(255),

  tech_prefix   varchar(255),
  tech_postfix  varchar(255),
  tech_headers  json,

  name      varchar(255),
  detail    text,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);
