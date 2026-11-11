create table push_chat_mutes(
  agent_id      binary(16),
  chat_id       binary(16),
  customer_id   binary(16),

  tm_create datetime(6),

  primary key(agent_id, chat_id)
);

create index idx_push_chat_mutes_customer_id on push_chat_mutes(customer_id);
