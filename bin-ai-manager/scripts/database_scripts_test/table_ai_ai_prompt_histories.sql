CREATE TABLE ai_ai_prompt_histories (
  id            BINARY(16) NOT NULL,
  customer_id   BINARY(16) NOT NULL,

  ai_id         BINARY(16) NOT NULL,
  prompt        TEXT NOT NULL,

  tm_create DATETIME(6),

  PRIMARY KEY(id)
);

CREATE INDEX idx_ai_prompt_histories_ai_id ON ai_ai_prompt_histories(ai_id);
CREATE INDEX idx_ai_prompt_histories_tm_create ON ai_ai_prompt_histories(tm_create);
