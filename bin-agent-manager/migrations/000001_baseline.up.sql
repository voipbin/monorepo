-- Baseline of the tables owned by bin-agent-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS agent_addresses (
  id binary(16) NOT NULL,
  agent_id binary(16) DEFAULT NULL,
  customer_id binary(16) DEFAULT NULL,
  type varchar(255) DEFAULT NULL,
  target varchar(255) DEFAULT NULL,
  target_name varchar(255) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text,
  idx int DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_agent_addresses_owner (customer_id, type, target),
  KEY idx_agent_addresses_agent_id (agent_id)
);

CREATE TABLE IF NOT EXISTS agent_agents (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  username varchar(255) DEFAULT NULL,
  password_hash varchar(255) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text,
  ring_method varchar(255) DEFAULT NULL,
  status varchar(255) DEFAULT NULL,
  permission int DEFAULT NULL,
  tag_ids json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  direct_id binary(16) DEFAULT NULL,
  direct_hash varchar(255) DEFAULT NULL,
  reserve_reference_type varchar(255) DEFAULT NULL,
  reserve_reference_id binary(16) DEFAULT NULL,
  tm_reserve datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_agent_agents_customer_id (customer_id),
  KEY idx_agent_agents_username (username)
);
