
create table campaigns(
  -- identity
  id          binary(16),
  customer_id binary(16),

  type varchar(255),

  execute varchar(255),

  name      varchar(255),
  detail    text,

  status        varchar(255),
  service_level integer,
  end_handle  varchar(255),

  flow_id binary(16),
  actions json,

  outplan_id  binary(16),
  outdial_id  binary(16),
  queue_id    binary(16),

  next_campaign_id binary(16),

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_campaigns_customer_id on campaigns(customer_id);
create index idx_campaigns_flow_id on campaigns(flow_id);
create index idx_campaigns_outplan_id on campaigns(outplan_id);
create index idx_campaigns_outdial_id on campaigns(outdial_id);
create index idx_campaigns_queue_id on campaigns(queue_id);
