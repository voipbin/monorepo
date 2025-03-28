
CREATE TABLE ai_summaries (
  id            BINARY(16) NOT NULL,
  customer_id   BINARY(16) NOT NULL,

  activeflow_id     BINARY(16) NOT NULL,
  reference_type    VARCHAR(255) NOT NULL,
  reference_id      BINARY(16) NOT NULL,

  status    VARCHAR(16) NOT NULL,
  language  VARCHAR(16) NOT NULL,
  content   TEXT NOT NULL,

  tm_create DATETIME(6),
  tm_update DATETIME(6),
  tm_delete DATETIME(6),

  PRIMARY KEY(id)
);

CREATE INDEX idx_ai_summaries_customer_id ON ai_summaries(customer_id);
CREATE INDEX idx_ai_summaries_activeflow ON ai_summaries(activeflow_id);
CREATE INDEX idx_ai_summaries_reference_id_language ON ai_summaries(reference_id, language);
CREATE INDEX idx_ai_summaries_reference_type ON ai_summaries(reference_type);
CREATE INDEX idx_ai_summaries_reference_id ON ai_summaries(reference_id);
