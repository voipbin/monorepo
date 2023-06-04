create table customers(
  -- identity
  id            binary(16) primary key,  -- id

  username      varchar(255),   -- username
  password_hash varchar(1023),  -- password hash

  name            varchar(255),
  detail          text,

  -- webhook info
  webhook_method  varchar(255),
  webhook_uri     varchar(1023),

  permission_ids json,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6)
);

create index idx_customers_username on customers(username);
