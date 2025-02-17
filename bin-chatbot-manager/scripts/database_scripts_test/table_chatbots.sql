create table chatbot_chatbots(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  -- info
  name        varchar(255),   -- name
  detail      text,           -- detail description

  engine_type   varchar(255),   --
  engine_model  varchar(255),
  init_prompt   text,           -- initial prompt

  credential_base64       text,           -- credential base64
  credential_project_id   varchar(255),   -- credential project id

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_chatbot_chatbots_create on chatbot_chatbots(tm_create);
create index idx_chatbot_chatbots_customer_id on chatbot_chatbots(customer_id);
