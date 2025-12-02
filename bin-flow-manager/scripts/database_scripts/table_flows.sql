create table flow_flows(
  -- identity
  id          binary(16),
  customer_id binary(16),
  type        varchar(255),

  name      varchar(255),
  detail    text,

  actions json,

  on_complete_flow_id BINARY(16) NOT NULL,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_flow_flows_customer_id on flow_flows(customer_id);
