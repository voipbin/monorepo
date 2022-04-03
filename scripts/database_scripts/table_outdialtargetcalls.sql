create table outdialtargetcalls(
  -- identity
  id                binary(16),
  customer_id       binary(16),
  campaign_id       binary(16),
  outdial_id        binary(16),
  outdial_target_id binary(16),

  activeflow_id     binary(16),
  reference_type  varchar(255),
  reference_id    binary(16),

  status          varchar(255),

  destination       json,
  destination_index integer,
  try_count         integer,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_outdialtargetcalls_customer_id on outdialtargetcalls(customer_id);
create index idx_outdialtargetcalls_campaign_id on outdialtargetcalls(campaign_id);
create index idx_outdialtargetcalls_outdial_target_id on outdialtargetcalls(outdial_target_id);
create index idx_outdialtargetcalls_activeflow_id on outdialtargetcalls(activeflow_id);
create index idx_outdialtargetcalls_reference_id on outdialtargetcalls(reference_id);
