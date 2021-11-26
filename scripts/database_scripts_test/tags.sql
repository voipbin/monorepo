create table tags(
  -- identity
  id            binary(16),  -- id
  user_id       integer,

  -- basic info
  name        varchar(255),
  detail      text,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_tags_userid on tags(user_id);
create index idx_tags_name on tags(name);
