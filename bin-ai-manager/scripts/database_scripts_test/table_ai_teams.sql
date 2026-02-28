create table ai_teams(
  -- identity
  id              binary(16),   -- id
  customer_id     binary(16),   -- customer id

  -- info
  name            varchar(255),   -- name
  detail          text,           -- detail description
  start_member_id binary(16),     -- starting member id
  members         json,           -- members as json array
  parameter       json,           -- parameter as json object

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_ai_teams_create on ai_teams(tm_create);
create index idx_ai_teams_customer_id on ai_teams(customer_id);
