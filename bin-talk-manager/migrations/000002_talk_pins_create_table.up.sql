CREATE TABLE IF NOT EXISTS talk_pins (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  owner_type varchar(255) NOT NULL,
  owner_id binary(16) NOT NULL,
  chat_id binary(16) NOT NULL,
  target_type varchar(16) NOT NULL,
  target_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL DEFAULT current_timestamp(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_talk_pins_owner_target (owner_type, owner_id, target_type, target_id),
  KEY idx_talk_pins_owner_create (customer_id, owner_type, owner_id, tm_create),
  KEY idx_talk_pins_chat_id (chat_id),
  CONSTRAINT talk_pins_ibfk_1 FOREIGN KEY (chat_id) REFERENCES talk_chats (id) ON DELETE CASCADE
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
