create table chat_messagechatrooms(
  -- identity
  id          binary(16),
  customer_id binary(16),
  owner_type  varchar(255),
  owner_id    binary(16),

  chatroom_id binary(16),
  messagechat_id binary(16),

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

create index idx_chat_messagechatrooms_customer_id on chat_messagechatrooms(customer_id);
create index idx_chat_messagechatrooms_chatroom_id on chat_messagechatrooms(chatroom_id);
create index idx_chat_messagechatrooms_messagechat_id on chat_messagechatrooms(messagechat_id);
