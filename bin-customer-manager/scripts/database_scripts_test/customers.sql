create table customers(
  -- identity
  id  binary(16) primary key,  -- id

  name      varchar(255),
  detail    text,

  email         varchar(255), -- email
  phone_number  varchar(255), -- phone number
  address       varchar(1024), -- address

  -- webhook info
  webhook_method  varchar(255),
  webhook_uri     varchar(1023),

  billing_account_id binary(16),

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6)
);
