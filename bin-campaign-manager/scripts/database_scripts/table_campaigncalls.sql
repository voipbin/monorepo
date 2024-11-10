
create table campaign_campaigncalls(
  -- identity
  id                binary(16),
  customer_id       binary(16),
  campaign_id       binary(16),

  outplan_id        binary(16),
  outdial_id        binary(16),
  outdial_target_id binary(16),
  queue_id          binary(16),

  activeflow_id     binary(16),
  flow_id           binary(16),

  reference_type  varchar(255),
  reference_id    binary(16),

  status  varchar(255),
  result  varchar(255),

  source            json,
  destination       json,
  destination_index integer,
  try_count         integer,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_campaign_campaigncalls_customer_id on campaign_campaigncalls(customer_id);
create index idx_campaign_campaigncalls_campaign_id on campaign_campaigncalls(campaign_id);
create index idx_campaign_campaigncalls_outdial_target_id on campaign_campaigncalls(outdial_target_id);
create index idx_campaign_campaigncalls_activeflow_id on campaign_campaigncalls(activeflow_id);
create index idx_campaign_campaigncalls_reference_id on campaign_campaigncalls(reference_id);
create index idx_campaign_campaigncalls_campaign_id_status on campaign_campaigncalls(campaign_id, status);
