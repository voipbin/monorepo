create table chats(
  -- identity
  id          binary(16),
  customer_id binary(16),

  type        varchar(255),

  room_owner_id   binary(16),
  participant_ids json,

  name      varchar(255),
  detail    text,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_chats_customer_id on chats(customer_id);
create index idx_chats_owner_id on chats(owner_id);
