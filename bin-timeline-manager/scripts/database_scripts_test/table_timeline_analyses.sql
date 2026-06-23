
CREATE TABLE timeline_analyses (
  id            binary(16)   NOT NULL,
  customer_id   binary(16)   NOT NULL,
  activeflow_id binary(16)   NOT NULL,

  status        varchar(32)  NOT NULL,
  result        json,
  model         varchar(255) NOT NULL DEFAULT '',
  error         text,

  tm_create datetime(6) NOT NULL,
  tm_update datetime(6),

  PRIMARY KEY(id)
);

CREATE UNIQUE INDEX uq_timeline_analyses_activeflow ON timeline_analyses(activeflow_id);
CREATE INDEX idx_timeline_analyses_customer_create ON timeline_analyses(customer_id, tm_create);
