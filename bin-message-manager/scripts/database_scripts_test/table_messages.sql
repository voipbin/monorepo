create table message_messages(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id
  type          varchar(255),

  source  json,
  targets json,

  provider_name         varchar(255), -- message provider's name
  provider_reference_id varchar(255), -- reference id for searching the message info from the provider

  text          text,
  medias        json,
  direction     varchar(255),

  -- timestamps
  tm_create   datetime(6),  --
  tm_update   datetime(6),  --
  tm_delete   datetime(6),  --

  primary key(id)
);

create index idx_message_messages_customer_id on message_messages(customer_id);
create index idx_message_messages_provider_name on message_messages(provider_name);
create index idx_message_messages_provider_reference_id on message_messages(provider_reference_id);
