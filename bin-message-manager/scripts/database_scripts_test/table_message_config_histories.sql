create table message_config_histories (
  id              binary(16)   not null,
  config_key      varchar(128) not null,
  action          varchar(16)  not null,
  is_secret       tinyint(1)   not null,
  old_value_json  json,
  new_value_json  json,
  old_key_version integer      not null default 0,
  new_key_version integer      not null default 0,
  actor_id        binary(16)   not null,
  source          varchar(16)  not null,
  tm_create       datetime(6)  not null,

  primary key(id)
);
