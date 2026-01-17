-- SQLite-compatible schema for chat_chats table
-- Used for unit testing with in-memory SQLite database

CREATE TABLE chat_chats (
    id              BLOB PRIMARY KEY,
    customer_id     BLOB NOT NULL,
    type            TEXT NOT NULL,
    tm_create       TEXT NOT NULL,
    tm_update       TEXT NOT NULL,
    tm_delete       TEXT
);

CREATE INDEX idx_chat_chats_customer_id ON chat_chats(customer_id);
