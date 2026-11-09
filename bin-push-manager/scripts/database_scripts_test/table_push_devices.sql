create table push_devices(
  -- identity
  id            binary(16),  -- id
  customer_id   binary(16),

  -- owner and device
  agent_id      binary(16),
  platform      varchar(16),
  token         varbinary(512),

  tm_create datetime(6),
  tm_update datetime(6),

  primary key(id)
);

create unique index idx_push_devices_token on push_devices(token);
create index idx_push_devices_customer_id on push_devices(customer_id);
create index idx_push_devices_agent_id on push_devices(agent_id);
