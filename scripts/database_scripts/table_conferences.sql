create table conferences(
  -- identity
  id    binary(16),       -- id
  type  varchar(255),     -- type
  bridge_id varchar(255), -- conference's bridge id

  -- info
  status  varchar(255),   -- status
  name    varchar(255),   -- conference's name
  detail  text,           -- conference's detail description
  data    json,           -- additional data

  bridge_ids json, -- bridges related with this conference
  call_ids   json, -- calls in the conference

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_conferences_create on conferences(tm_create);
