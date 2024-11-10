create table conversation_messages(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id

  conversation_id binary(16),
  direction       varchar(255),
  status          varchar(16),

  reference_type  varchar(255),
  reference_id    varchar(255),

  transaction_id  varchar(255),

  source  json,

  text    text,
  medias  json,

  -- timestamps
  tm_create   datetime(6),  --
  tm_update   datetime(6),  --
  tm_delete   datetime(6),  --

  primary key(id)
);

create index idx_conversation_messages_customer_id on conversation_messages(customer_id);
create index idx_conversation_messages_reference_type_reference_id on conversation_messages(reference_type, reference_id);
create index idx_conversation_messages_transaction_id on conversation_messages(transaction_id);
