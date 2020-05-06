create table cm_conferences(
  -- identity
  id    binary(16),   -- id
  type  varchar(255), -- type

  -- info
  name    varchar(255),   -- conference's name
  detail  text,           -- conference's detail description
  data    json,           -- additional data

  bridges json, -- bridges related with this conference
  calls   json, -- calls in the conference

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_create on cm_confreneces(tm_create);
