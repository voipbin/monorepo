create table chat_chatrooms(
  -- identity
  id          binary(16),
  customer_id binary(16),
  owner_type  varchar(255),
  owner_id    binary(16),

  type        varchar(255),
  chat_id     binary(16),

  room_owner_id     binary(16),
  participant_ids   json,

  name      varchar(255),
  detail    text,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_chat_chatrooms_customer_id on chat_chatrooms(customer_id);
create index idx_chat_chatrooms_chat_id on chat_chatrooms(chat_id);
create index idx_chat_chatrooms_owner_id on chat_chatrooms(owner_id);
create index idx_chat_chatrooms_chat_id_owner_id on chat_chatrooms(chat_id, owner_id);
create index idx_chat_chatrooms_room_owner_id on chat_chatrooms(room_owner_id);
