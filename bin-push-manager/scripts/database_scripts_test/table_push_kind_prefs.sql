create table push_kind_prefs(
  agent_id      binary(16),
  kind          varchar(32),
  customer_id   binary(16),

  tm_create datetime(6),

  primary key(agent_id, kind)
);

create index idx_push_kind_prefs_customer_id on push_kind_prefs(customer_id);
