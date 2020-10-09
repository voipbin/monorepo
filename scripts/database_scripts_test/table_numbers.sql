create table numbers(
  -- identity
  id        binary(16),     -- id
  number    varchar(255),   -- number
  flow_id   binary(16),     -- flow id
  user_id   integer,        -- user id

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_numbers_number on numbers(number);
