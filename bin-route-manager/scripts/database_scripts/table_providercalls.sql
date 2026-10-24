create table route_providercalls(
  -- identity
  id          binary(16),
  customer_id binary(16),

  provider_id binary(16),
  flow_id     binary(16),

  source        json,
  destinations  json,
  anonymous     varchar(16),

  call_ids      json,
  groupcall_ids json,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_route_providercalls_customer_id on route_providercalls(customer_id);
create index idx_route_providercalls_provider_id on route_providercalls(provider_id);
create index idx_route_providercalls_tm_delete_tm_create on route_providercalls(tm_delete, tm_create);
