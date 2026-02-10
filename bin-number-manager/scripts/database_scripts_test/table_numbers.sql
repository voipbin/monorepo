create table number_numbers(
  -- identity
  id            binary(16),     -- id
  number        varchar(255),   -- number
  type          varchar(255) not null default 'normal', -- number type (normal, virtual)
  customer_id   binary(16),     -- customer id

  call_flow_id      binary(16), --
  message_flow_id   binary(16), --

  name    varchar(255),
  detail  text,

  provider_name         varchar(255), -- number provider's name
  provider_reference_id varchar(255), -- reference id for searching the number info from the provider

  status  varchar(32),  -- number's registration status

  t38_enabled         boolean,    -- T38 eanbled
  emergency_enabled   boolean,    -- Emergency available

  -- timestamps
  tm_purchase datetime(6),
  tm_renew    datetime(6),
  tm_create   datetime(6),  --
  tm_update   datetime(6),  --
  tm_delete   datetime(6),  --

  primary key(id)
);

create index idx_number_numbers_number on number_numbers(number);
create index idx_number_numbers_type on number_numbers(type);
create index idx_number_numbers_customer_id on number_numbers(customer_id);
create index idx_number_numbers_call_flow_id on number_numbers(call_flow_id);
create index idx_number_numbers_message_flow_id on number_numbers(message_flow_id);
create index idx_number_numbers_provider_name on number_numbers(provider_name);
create index idx_number_numbers_tm_renew on number_numbers(tm_renew);
