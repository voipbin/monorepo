create table chatbotcalls(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  chatbot_id          binary(16),   -- chatbot id
  chatbot_engine_type varchar(255), -- chatbot engine type

  activeflow_id     binary(16),
  reference_type    varchar(255),
  reference_id      binary(16),

  confbridge_id     binary(16),
  transcribe_id     binary(16),

  status  varchar(255),   -- status

  gender    varchar(255), -- gender
  language  varchar(255), -- language

  messages json,

  -- timestamps
  tm_end    datetime(6),  --
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_chatbotcalls_customer_id on chatbotcalls(customer_id);
create index idx_chatbotcalls_chatbot_id on chatbotcalls(chatbot_id);
create index idx_chatbotcalls_reference_type on chatbotcalls(reference_type);
create index idx_chatbotcalls_reference_id on chatbotcalls(reference_id);
create index idx_chatbotcalls_transcribe_id on chatbotcalls(transcribe_id);
create index idx_chatbotcalls_create on chatbotcalls(tm_create);
create index idx_chatbotcalls_activeflow_id on chatbotcalls(activeflow_id);

