-- SQLite-compatible schema for talk_chats table
-- Used for unit testing with in-memory SQLite database

CREATE TABLE talk_chats (
    id              BLOB PRIMARY KEY,
    customer_id     BLOB NOT NULL,
    type            TEXT NOT NULL,
    tm_create       TEXT NOT NULL,
    tm_update       TEXT NOT NULL,
    tm_delete       TEXT
);

CREATE INDEX idx_talk_chats_customer_id ON talk_chats(customer_id);
