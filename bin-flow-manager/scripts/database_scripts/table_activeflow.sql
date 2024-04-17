create table activeflows(
  -- identity
  id          binary(16),
  customer_id binary(16),

  status    binary(16),
  flow_id   binary(16),

  reference_type  varchar(255),
  reference_id    binary(16),

  stack_map json,

  current_stack_id  binary(16),
  current_action    json,

  forward_stack_id    binary(16),
  forward_action_id   binary(16),

  execute_count     integer,
  executed_actions  json,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_activeflows_customer_id on activeflows(customer_id);
create index idx_activeflows_flow_id on activeflows(flow_id);
create index idx_activeflows_reference_id on activeflows(reference_id);
