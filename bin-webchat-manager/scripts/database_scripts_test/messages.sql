create table webchat_messages(
  -- identity
  id             binary(16),
  customer_id    binary(16),
  widget_id      binary(16),
  session_id     binary(16),

  direction      varchar(16),
  status         varchar(16),
  text           varchar(4000),
  sender_id      binary(16),
  activeflow_id  binary(16),

  tm_create      datetime(6),
  tm_delete      datetime(6),

  primary key(id)
);

create index idx_webchat_messages_session_id_tm_create on webchat_messages(session_id, tm_create);
create index idx_webchat_messages_customer_id_tm_create on webchat_messages(customer_id, tm_create);
