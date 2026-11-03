CREATE TABLE IF NOT EXISTS talk_mentions (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  chat_id binary(16) NOT NULL,
  message_id binary(16) NOT NULL,
  agent_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL DEFAULT current_timestamp(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_talk_mentions_message_agent (message_id, agent_id),
  KEY idx_talk_mentions_agent_create (customer_id, agent_id, tm_create),
  KEY idx_talk_mentions_chat_id (chat_id),
  CONSTRAINT talk_mentions_ibfk_1 FOREIGN KEY (chat_id) REFERENCES talk_chats (id) ON DELETE CASCADE,
  CONSTRAINT talk_mentions_ibfk_2 FOREIGN KEY (message_id) REFERENCES talk_messages (id) ON DELETE CASCADE
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
