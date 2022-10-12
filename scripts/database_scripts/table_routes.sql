create table routes(
  -- identity
  id          binary(16),
  customer_id binary(16),

  provider_id binary(16),
  priority    int,

  target varchar(255),

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update

  primary key(id)
);

create index idx_routes_customer_id on routes(customer_id);
