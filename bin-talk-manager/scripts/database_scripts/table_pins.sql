-- SQLite-compatible schema for talk_pins table
-- Used for unit testing with in-memory SQLite database
-- Hand maintained copy of migrations/000002_talk_pins_create_table.up.sql

CREATE TABLE talk_pins (
    id              BLOB PRIMARY KEY,
    customer_id     BLOB NOT NULL,
    owner_type      TEXT NOT NULL,
    owner_id        BLOB NOT NULL,
    chat_id         BLOB NOT NULL,
    target_type     TEXT NOT NULL,
    target_id       BLOB NOT NULL,
    tm_create       TEXT NOT NULL
);

CREATE UNIQUE INDEX uq_talk_pins_owner_target ON talk_pins(owner_type, owner_id, target_type, target_id);
CREATE INDEX idx_talk_pins_owner_create ON talk_pins(customer_id, owner_type, owner_id, tm_create);
CREATE INDEX idx_talk_pins_chat_id ON talk_pins(chat_id);
