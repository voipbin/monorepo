create table flows(
  -- identity
  id binary(16),
  user_id int(10),

  name varchar(255),
  detail text,
  
  persist bool,

  actions json,

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),

  primary key(id)
)
