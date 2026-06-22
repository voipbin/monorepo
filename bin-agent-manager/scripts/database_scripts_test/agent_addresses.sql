create table agent_addresses(
  -- identity
  id          binary(16),   -- surrogate id
  agent_id    binary(16),   -- owning agent (agent_agents.id)
  customer_id binary(16),   -- denormalized for the by-(customer,addr) lookup + uniqueness

  -- address (commonaddress.Address)
  type        varchar(255),
  target      varchar(255),
  target_name varchar(255),
  name        varchar(255),
  detail      text,

  idx         int,          -- preserves address order within an agent (ring order)

  tm_create   datetime(6),
  tm_update   datetime(6),

  primary key(id)
);

create index idx_agent_addresses_agent_id on agent_addresses(agent_id);
-- UNIQUE lookup index. The UNIQUE(customer_id, type, target) constraint was
-- promoted in ffeb3808129c after the backfill + duplicate-resolution gate.
create unique index idx_agent_addresses_owner on agent_addresses(customer_id, type, target);
