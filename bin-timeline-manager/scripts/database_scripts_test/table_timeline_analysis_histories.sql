
CREATE TABLE timeline_analysis_histories (
  id            binary(16)   NOT NULL,
  analysis_id   binary(16)   NOT NULL,
  customer_id   binary(16)   NOT NULL,
  activeflow_id binary(16)   NOT NULL,

  status        varchar(32)  NOT NULL,
  result        json,
  model         varchar(255) NOT NULL DEFAULT '',
  error         text,
  reason        varchar(32)  NOT NULL,

  tm_create datetime(6) NOT NULL,

  PRIMARY KEY(id)
);

CREATE INDEX idx_timeline_analysis_histories_activeflow ON timeline_analysis_histories(activeflow_id, tm_create);
CREATE INDEX idx_timeline_analysis_histories_analysis ON timeline_analysis_histories(analysis_id, tm_create);
