create table number_providers(
  provider_name     varchar(32) not null,
  enabled           boolean not null default 0,
  priority          integer not null default 100,
  config            json,
  secret_ciphertext blob,
  key_version       integer not null default 0,
  updated_by        binary(16),
  secret_updated_by binary(16),
  tm_secret_update  datetime(6),
  tm_create         datetime(6),
  tm_update         datetime(6),

  primary key(provider_name)
);
