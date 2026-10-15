-- Baseline of the tables owned by bin-flow-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS flow_flows (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  type varchar(255) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  actions json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  on_complete_flow_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  direct_id binary(16) DEFAULT NULL,
  direct_hash varchar(255) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_flow_flows_customer_id (customer_id)
);
CREATE TABLE IF NOT EXISTS flow_activeflows (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  status varchar(16) DEFAULT NULL,
  flow_id binary(16) DEFAULT NULL,
  reference_type varchar(255) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  reference_activeflow_id binary(16) DEFAULT NULL,
  stack_map json DEFAULT NULL,
  current_stack_id binary(16) DEFAULT 0x00000000000000000000000000000000,
  current_action json DEFAULT NULL,
  execute_count int(11) DEFAULT NULL,
  forward_stack_id binary(16) DEFAULT 0x00000000000000000000000000000000,
  forward_action_id binary(16) DEFAULT NULL,
  executed_actions json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  on_complete_flow_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  webhook_uri varchar(1024) DEFAULT NULL,
  webhook_method varchar(16) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_flow_activeflows_customer_id (customer_id),
  KEY idx_flow_activeflows_flow_id (flow_id),
  KEY idx_flow_activeflows_reference_id (reference_id)
);
