CREATE TABLE ai_ai_prompt_histories (
  id          BINARY(16)  NOT NULL,
  customer_id BINARY(16)  NOT NULL,
  ai_id       BINARY(16)  NOT NULL,
  prompt      LONGTEXT    NOT NULL DEFAULT '',
  tm_create   DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  INDEX idx_ai_ai_prompt_histories_ai_id_tm_create (ai_id, tm_create),
  INDEX idx_ai_ai_prompt_histories_customer_id (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
