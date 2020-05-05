create table bridges(
  -- identity
  asterisk_id varchar(255), -- Asterisk id
  id          varchar(255),       -- bridge id
  name        varchar(255),       -- bridge name

  -- info
  type    varchar(255),   -- bridge's type
  tech    varchar(255),   -- bridge's technology
  class   varchar(255),   -- bridge's class
  creator varchar(255),   -- bridge creator

  video_mode      varchar(255),
  video_source_id varchar(255),

  channels json,

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(asterisk_id, id)
);

create index idx_create on channels(tm_create);
