-- SQLite-compatible schema for chat_participants table

CREATE TABLE chat_participants (
    id              BLOB PRIMARY KEY,
    customer_id     BLOB NOT NULL,
    chat_id         BLOB NOT NULL,
    owner_type      TEXT NOT NULL,
    owner_id        BLOB NOT NULL,
    tm_joined       TEXT NOT NULL
);

CREATE INDEX idx_chat_participants_chat_id ON chat_participants(chat_id);
CREATE INDEX idx_chat_participants_owner ON chat_participants(owner_type, owner_id);
CREATE INDEX idx_chat_participants_customer_id ON chat_participants(customer_id);
CREATE UNIQUE INDEX unique_participant ON chat_participants(chat_id, owner_type, owner_id);
