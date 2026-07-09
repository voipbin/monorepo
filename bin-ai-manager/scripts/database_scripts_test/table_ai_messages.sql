create table ai_messages(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id
  aicall_id     binary(16),
  activeflow_id binary(16),

  direction     varchar(255),   -- direction
  role          varchar(255),   -- role
  content       text,           -- content

  -- tool
  tool_calls    json,
  tool_call_id  varchar(255),

  -- active ai
  active_ai_id  binary(16),

  -- delivery tracking
  pipecatcall_id  binary(16),
  delivery_status varchar(16) not null default 'delivered',

  -- reply correlation
  in_reply_to_message_id binary(16) not null default 0x00000000000000000000000000000000,

  -- timestamps
  tm_create datetime(6),  --
  tm_delete datetime(6),

  primary key(id)
);

create index idx_ai_messages_create on ai_messages(tm_create);
create index idx_ai_messages_customer_id on ai_messages(customer_id);
create index idx_ai_messages_aicall_id on ai_messages(aicall_id);
create index idx_ai_messages_activeflow_id on ai_messages(activeflow_id);
create index idx_ai_messages_pcc_delivery on ai_messages(pipecatcall_id, delivery_status);
