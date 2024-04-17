create table messagechats(
  -- identity
  id          binary(16),
  customer_id binary(16),

  chat_id binary(16),

  source  json,
  type    varchar(255),
  text    text,
  medias  json,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_messagechats_customer_id on messagechats(customer_id);
create index idx_messagechats_chat_id on messagechats(chat_id);
