create table messagechatrooms(
  -- identity
  id          binary(16),
  customer_id binary(16),

  chatroom_id binary(16),

  messagechat_id binary(16),

  message json,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_messagechatrooms_customer_id on messagechatrooms(customer_id);
create index idx_messagechatrooms_chat_id on messagechatrooms(chat_id);
create index idx_messagechatrooms_chatroom_id on messagechatrooms(chatroom_id);
create index idx_messagechatrooms_messagechat_id on messagechatrooms(messagechat_id);
