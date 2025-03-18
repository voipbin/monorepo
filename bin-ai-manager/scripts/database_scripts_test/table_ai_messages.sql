create table ai_messages(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id
  aicall_id binary(16),

  direction     varchar(255),   -- direction
  role          varchar(255),   -- role
  content       text,           -- content

  -- timestamps
  tm_create datetime(6),  --
  tm_delete datetime(6),

  primary key(id)
);

create index idx_ai_messages_create on ai_messages(tm_create);
create index idx_ai_messages_customer_id on ai_messages(customer_id);
create index idx_ai_messages_aicall_id on ai_messages(aicall_id);
