create table webchat_sessions(
  -- identity
  id                binary(16),
  customer_id       binary(16),
  widget_id         binary(16),

  status            varchar(16),
  activeflow_id     binary(16),

  tm_last_activity  datetime(6),
  tm_create         datetime(6),
  tm_update         datetime(6),
  tm_end            datetime(6),
  tm_delete         datetime(6),

  primary key(id)
);

create index idx_webchat_sessions_customer_id_tm_create on webchat_sessions(customer_id, tm_create);
create index idx_webchat_sessions_customer_id_tm_delete on webchat_sessions(customer_id, tm_delete);
create index idx_webchat_sessions_widget_id_status on webchat_sessions(widget_id, status);
create index idx_webchat_sessions_status_tm_last_activity on webchat_sessions(status, tm_last_activity);
