-- SQLite-compatible schema for talk_participants table

CREATE TABLE talk_participants (
    id              BLOB PRIMARY KEY,
    chat_id         BLOB NOT NULL,
    owner_type      TEXT NOT NULL,
    owner_id        BLOB NOT NULL,
    tm_joined       TEXT NOT NULL
);

CREATE INDEX idx_talk_participants_chat_id ON talk_participants(chat_id);
CREATE INDEX idx_talk_participants_owner ON talk_participants(owner_type, owner_id);
CREATE UNIQUE INDEX unique_participant ON talk_participants(chat_id, owner_type, owner_id);
