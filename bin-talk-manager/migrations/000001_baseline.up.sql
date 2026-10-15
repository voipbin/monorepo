-- Baseline of the tables owned by bin-talk-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the tables
-- (created by the former Alembic stream) are left untouched.
-- The table option DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci is written out so that the
-- collation of the tables and of their text columns does not depend on the database default.
-- The foreign keys carry the names <table>_ibfk_N that the tables of existing databases have.
CREATE TABLE IF NOT EXISTS talk_chats (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  type varchar(255) NOT NULL,
  name varchar(255) DEFAULT '',
  detail text DEFAULT NULL,
  member_count int(11) NOT NULL DEFAULT 0,
  tm_create datetime(6) NOT NULL DEFAULT current_timestamp(6),
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_customer_id (customer_id)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS talk_participants (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  chat_id binary(16) NOT NULL,
  owner_type varchar(255) NOT NULL,
  owner_id binary(16) NOT NULL,
  tm_joined datetime(6) NOT NULL DEFAULT current_timestamp(6),
  PRIMARY KEY (id),
  UNIQUE KEY unique_participant (chat_id, owner_type, owner_id),
  KEY idx_customer_id (customer_id),
  KEY idx_chat_id (chat_id),
  KEY idx_owner (owner_type, owner_id),
  CONSTRAINT talk_participants_ibfk_1 FOREIGN KEY (chat_id) REFERENCES talk_chats (id) ON DELETE CASCADE
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS talk_messages (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  chat_id binary(16) NOT NULL,
  parent_id binary(16) DEFAULT NULL,
  owner_type varchar(255) NOT NULL,
  owner_id binary(16) NOT NULL,
  type varchar(255) NOT NULL,
  text text DEFAULT NULL,
  medias json DEFAULT NULL,
  metadata json DEFAULT NULL,
  tm_create datetime(6) NOT NULL DEFAULT current_timestamp(6),
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_chat_id (chat_id),
  KEY idx_parent_id (parent_id),
  KEY idx_customer_id (customer_id),
  KEY idx_owner (owner_type, owner_id),
  CONSTRAINT talk_messages_ibfk_1 FOREIGN KEY (chat_id) REFERENCES talk_chats (id) ON DELETE CASCADE,
  CONSTRAINT talk_messages_ibfk_2 FOREIGN KEY (parent_id) REFERENCES talk_messages (id) ON DELETE SET NULL
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
