create table agents(
  -- identity
  id            binary(16),  -- id
  customer_id   binary(16),
  username      varchar(255), -- username
  password_hash varchar(255), -- password hash

  -- basic info
  name            varchar(255),
  detail          text,

  ring_method   varchar(255), -- ring method

  status      varchar(255),  -- agent's status
  permission  integer,
  tag_ids     json,
  addresses   json,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_agents_customerid on agents(customer_id);
create index idx_agents_username on agents(username);
