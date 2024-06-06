create table agent_resources(
  -- identity
  id            binary(16),  -- id
  customer_id   binary(16),
  owner_id      binary(16),

  reference_type varchar(255), --
  reference_id   binary(16), --

  data text,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_agent_resources_customerid on agent_resources(customer_id);
create index idx_agent_resources_ownerid on agent_resources(owner_id);
create index idx_agent_resources_referenceid on agent_resources(reference_id);
create index idx_agent_resources_referencetype_referenceid on agent_resources(reference_type, reference_id);

