create table fm_flows(
  -- identity
  id binary(16),
  revision binary(16),

  name varchar(255),
  detail text,

  actions json,

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),

  primary key(id, revision)
)
