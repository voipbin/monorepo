create table conferences(
  -- identity
  id        binary(16),   -- id
  user_id   integer,      -- user id
  type      varchar(255), -- type
  bridge_id varchar(255), -- conference's bridge id

  -- info
  status  varchar(255),   -- status
  name    varchar(255),   -- conference's name
  detail  text,           -- conference's detail description
  data    json,           -- additional data
  timeout int,            -- timeout. second

  call_ids   json, -- calls in the conference

  -- record info
  record_id   varchar(255),   -- current record id
  record_ids  json,           -- record ids

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_conferences_create on conferences(tm_create);
create index idx_conferences_user_id on conferences(user_id);
