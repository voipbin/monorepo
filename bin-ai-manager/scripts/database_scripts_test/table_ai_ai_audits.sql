
CREATE TABLE ai_ai_audits (
  id                binary(16) NOT NULL,
  customer_id       binary(16) NOT NULL,

  aicall_id         binary(16) NOT NULL,
  ai_id             binary(16) NOT NULL,
  prompt_history_id binary(16) NOT NULL,

  status        varchar(16)  NOT NULL DEFAULT 'progressing',
  overall_score int,
  evaluation    text,
  language      varchar(16)  NOT NULL DEFAULT '',
  error         varchar(255) NOT NULL DEFAULT '',

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  PRIMARY KEY(id)
);

CREATE INDEX idx_ai_ai_audits_customer_id ON ai_ai_audits(customer_id);
CREATE INDEX idx_ai_ai_audits_aicall_id   ON ai_ai_audits(aicall_id);
CREATE INDEX idx_ai_ai_audits_ai_id       ON ai_ai_audits(ai_id);
CREATE UNIQUE INDEX idx_ai_ai_audits_aicall_ai ON ai_ai_audits(aicall_id, ai_id);
CREATE INDEX idx_ai_ai_audits_tm_create   ON ai_ai_audits(tm_create);
