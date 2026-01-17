-- SQLite-compatible schema for chat_messages table

CREATE TABLE chat_messages (
    id              BLOB PRIMARY KEY,
    customer_id     BLOB NOT NULL,
    chat_id         BLOB NOT NULL,
    parent_id       BLOB,
    owner_type      TEXT NOT NULL,
    owner_id        BLOB NOT NULL,
    type            TEXT NOT NULL,
    text            TEXT,
    medias          TEXT,
    metadata        TEXT,
    tm_create       TEXT NOT NULL,
    tm_update       TEXT NOT NULL,
    tm_delete       TEXT
);

CREATE INDEX idx_chat_messages_chat_id ON chat_messages(chat_id);
CREATE INDEX idx_chat_messages_parent_id ON chat_messages(parent_id);
CREATE INDEX idx_chat_messages_customer_id ON chat_messages(customer_id);
CREATE INDEX idx_chat_messages_owner ON chat_messages(owner_type, owner_id);
