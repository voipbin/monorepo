
create table conversation_accounts(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id

  type varchar(32),

  name    varchar(255),
  detail  text,

  secret  text,
  token   text,

  -- timestamps
  tm_create   datetime(6),  --
  tm_update   datetime(6),  --
  tm_delete   datetime(6),  --

  primary key(id)
);

create index idx_conversation_accounts_customer_id on conversation_accounts(customer_id);
