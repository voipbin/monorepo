create table ai_ais(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  -- info
  name        varchar(255),   -- name
  detail      text,           -- detail description

  engine_type   varchar(255),   --
  engine_model  varchar(255),
  engine_data   json,

  init_prompt   text,           -- initial prompt

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_ai_ais_create on ai_ais(tm_create);
create index idx_ai_ais_customer_id on ai_ais(customer_id);
