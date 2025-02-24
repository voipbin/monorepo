create table chatbot_messages(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id
  chatbotcall_id binary(16),    -- chatbotcall id

  direction     varchar(255),   -- direction
  role          varchar(255),   -- role
  content       text,           -- content

  -- timestamps
  tm_create datetime(6),  --
  tm_delete datetime(6),

  primary key(id)
);

create index idx_chatbot_messages_create on chatbot_messages(tm_create);
create index idx_chatbot_messages_customer_id on chatbot_messages(customer_id);
create index idx_chatbot_messages_chatbotcall_id on chatbot_messages(chatbotcall_id);
