create table conversation_messages(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id

  conversation_id binary(16),

  status varchar(16),

  reference_type  varchar(255),
  reference_id    varchar(255),

  source_target   varchar(255),

  data  binary(4096),

  -- timestamps
  tm_create   datetime(6),  --
  tm_update   datetime(6),  --
  tm_delete   datetime(6),  --

  primary key(id)
);

create index idx_conversation_messages_customerid on conversation_messages(customer_id);
create index idx_conversation_messages_reference_type_reference_id on conversation_messages(reference_type, reference_id);
