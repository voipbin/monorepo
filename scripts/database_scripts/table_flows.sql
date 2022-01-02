create table flows(
  -- identity
  id        binary(16),
  user_id   int(10),
  type      varchar(255),

  name      varchar(255),
  detail    text,

  webhook_uri   varchar(1023),

  actions json,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
)
