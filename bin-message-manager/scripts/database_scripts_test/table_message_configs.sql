create table message_configs (
  config_key        varchar(128) not null,
  value_type        varchar(16)  not null,
  value_json        json,
  secret_ciphertext blob,
  key_version       integer      not null default 0,
  revision          integer      not null default 1,
  updated_by        binary(16),
  tm_create         datetime(6)  not null,
  tm_update         datetime(6)  not null,

  primary key(config_key)
);
