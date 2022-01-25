
create table permissions(
  -- identity
  id        binary(16) primary key,  -- id

  name      varchar(255),
  detail    text,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6)
);
