CREATE TABLE conference_conferencecalls(
  -- identity
  id            BINARY(16),   -- id
  customer_id   BINARY(16),   -- customer id

  activeflow_id BINARY(16),   -- activeflow id
  conference_id BINARY(16),

  reference_type    VARCHAR(255),
  reference_id      BINARY(16),

  status  VARCHAR(255),   -- status

  -- timestamps
  tm_create DATETIME(6),  --
  tm_update DATETIME(6),  --
  tm_delete DATETIME(6),  --

  PRIMARY KEY(id)
);

CREATE INDEX idx_conference_conferencecalls_customer_id ON conference_conferencecalls(customer_id);
CREATE INDEX idx_conference_conferencecalls_conference_id ON conference_conferencecalls(conference_id);
CREATE INDEX idx_conference_conferencecalls_reference_id ON conference_conferencecalls(reference_id);
CREATE INDEX idx_conference_conferencecalls_create ON conference_conferencecalls(tm_create);
CREATE INDEX idx_conference_conferencecalls_activeflow_id ON conference_conferencecalls(activeflow_id);
