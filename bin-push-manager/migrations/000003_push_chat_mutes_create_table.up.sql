-- push_chat_mutes holds the chats an agent muted. A row means "muted"; unmuting deletes the row. A chat
-- that is deleted leaves its rows behind, which no event matches any more.
CREATE TABLE IF NOT EXISTS push_chat_mutes (
  agent_id binary(16) NOT NULL,
  chat_id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL,
  PRIMARY KEY (agent_id, chat_id),
  KEY idx_push_chat_mutes_customer_id (customer_id)
);
