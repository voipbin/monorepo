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
-- Non-unique lookup index, matching the expand-phase Alembic migration
-- (14bca52528f5). The UNIQUE(customer_id, type, target) constraint is promoted
-- in a separate operational step AFTER the backfill + duplicate-resolution gate,
-- so it is intentionally NOT present here.
create index idx_agent_addresses_owner on agent_addresses(customer_id, type, target);
