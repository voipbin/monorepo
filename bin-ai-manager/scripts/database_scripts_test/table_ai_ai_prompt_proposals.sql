CREATE TABLE ai_ai_prompt_proposals (
  id                         binary(16) NOT NULL,
  customer_id                binary(16) NOT NULL,
  ai_id                      binary(16) NOT NULL,

  audit_ids                  json         NOT NULL,
  basis_prompt_history_id    binary(16)   NOT NULL,

  original_prompt            text,
  proposed_prompt            text,
  rationale                  text,

  status                     varchar(32)  NOT NULL DEFAULT 'progressing',
  error                      varchar(128) NOT NULL DEFAULT '',
  applied_prompt_history_id  binary(16)   NOT NULL DEFAULT 0x00000000000000000000000000000000,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  PRIMARY KEY(id)
);

CREATE INDEX idx_ai_ai_prompt_proposals_customer_id ON ai_ai_prompt_proposals(customer_id);
CREATE INDEX idx_ai_ai_prompt_proposals_ai_id       ON ai_ai_prompt_proposals(ai_id);
CREATE INDEX idx_ai_ai_prompt_proposals_tm_create   ON ai_ai_prompt_proposals(tm_create);
