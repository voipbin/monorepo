-- Baseline of the tables owned by bin-push-manager.
-- push_devices holds the push tokens of the agents' devices. A token is a credential, so rows are
-- deleted for real (no tm_delete), and the unique key on the token lets a token belong to one agent only.
-- No table options are given on purpose: the database defaults are inherited.
CREATE TABLE IF NOT EXISTS push_devices (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  agent_id binary(16) NOT NULL,
  platform varchar(16) NOT NULL,
  token varbinary(512) NOT NULL,
  tm_create datetime(6) NOT NULL,
  tm_update datetime(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_push_devices_token (token),
  KEY idx_push_devices_customer_id (customer_id),
  KEY idx_push_devices_agent_id (agent_id)
);
