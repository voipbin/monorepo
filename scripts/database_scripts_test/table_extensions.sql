create table extensions (
  -- identity
  id            binary(16), -- id
  user_id       integer, -- user id

  domain_id     binary(16),
  endpoint_id   varchar(255),
  aor_id        varchar(255),
  auth_id       varchar(255),

  extension     varchar(255),
  password      varchar(255),

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)

);

create index idx_extensions_user_id on extensions(user_id);
create index idx_extensions_domain_id on extensions(domain_id);
