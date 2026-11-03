-- SQLite-compatible schema for talk_mentions table

CREATE TABLE talk_mentions (
    id              BLOB PRIMARY KEY,
    customer_id     BLOB NOT NULL,
    chat_id         BLOB NOT NULL,
    message_id      BLOB NOT NULL,
    agent_id        BLOB NOT NULL,
    tm_create       TEXT NOT NULL
);

CREATE UNIQUE INDEX uq_talk_mentions_message_agent ON talk_mentions(message_id, agent_id);
CREATE INDEX idx_talk_mentions_agent_create ON talk_mentions(customer_id, agent_id, tm_create);
CREATE INDEX idx_talk_mentions_chat_id ON talk_mentions(chat_id);
