create table conversation_conversations(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id
  owner_type    varchar(255),   -- owner type
  owner_id      binary(16),     -- owner id

  account_id    binary(16),     -- account id

  name    varchar(255),
  detail  text,

  type  varchar(255),

  dialog_id    varchar(255),

  self    json,
  peer    json,

  -- timestamps
  tm_create   datetime(6),  --
  tm_update   datetime(6),  --
  tm_delete   datetime(6),  --

  primary key(id)
);

create index idx_conversation_conversations_customer_id on conversation_conversations(customer_id);
create index idx_conversation_conversations_reference_type_reference_id on conversation_conversations(type, dialog_id);
create index idx_conversation_conversations_owner_id on conversation_conversations(owner_id);
