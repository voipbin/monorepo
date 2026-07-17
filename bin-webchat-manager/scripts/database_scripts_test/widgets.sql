create table webchat_widgets(
  -- identity
  id                    binary(16),
  customer_id           binary(16),

  -- basic info
  name                  varchar(255),
  status                varchar(16),

  -- direct hash
  direct_id             binary(16),
  direct_hash           varchar(255),

  welcome_message       text,
  session_flow_id       binary(16),
  message_flow_id       binary(16),
  session_idle_timeout  integer,

  theme_config          json,

  tm_create             datetime(6),
  tm_update             datetime(6),
  tm_delete             datetime(6),

  primary key(id)
);

create index idx_webchat_widgets_customer_id_tm_create on webchat_widgets(customer_id, tm_create);
create index idx_webchat_widgets_customer_id_tm_delete on webchat_widgets(customer_id, tm_delete);
