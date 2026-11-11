-- push_kind_prefs holds the notification kinds an agent switched off. A row means "off": there is no
-- enabled column, a kind that has no row is on, and switching a kind on again deletes the row.
CREATE TABLE IF NOT EXISTS push_kind_prefs (
  agent_id binary(16) NOT NULL,
  kind varchar(32) NOT NULL,
  customer_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL,
  PRIMARY KEY (agent_id, kind),
  KEY idx_push_kind_prefs_customer_id (customer_id)
);
