-- Baseline of the tables owned by bin-conference-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the tables
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS conference_conferencecalls (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  conference_id binary(16) DEFAULT NULL,
  reference_type varchar(255) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  status varchar(255) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_conference_conferencecalls_customer_id (customer_id),
  KEY idx_conference_conferencecalls_conference_id (conference_id),
  KEY idx_conference_conferencecalls_reference_id (reference_id),
  KEY idx_conference_conferencecalls_create (tm_create),
  KEY conference_conferencecalls_activeflow_id (activeflow_id)
);

CREATE TABLE IF NOT EXISTS conference_conferences (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  type varchar(255) DEFAULT NULL,
  flow_id binary(16) DEFAULT NULL,
  confbridge_id binary(16) DEFAULT NULL,
  status varchar(255) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  data json DEFAULT NULL,
  timeout int DEFAULT NULL,
  pre_flow_id binary(16) DEFAULT NULL,
  post_flow_id binary(16) DEFAULT NULL,
  conferencecall_ids json DEFAULT NULL,
  recording_id binary(16) DEFAULT NULL,
  recording_ids json DEFAULT NULL,
  transcribe_id binary(16) DEFAULT '',
  transcribe_ids json DEFAULT NULL,
  tm_end datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  direct_id binary(16) DEFAULT NULL,
  direct_hash varchar(255) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_conference_conferences_create (tm_create),
  KEY idx_conference_conferences_customer_id (customer_id),
  KEY idx_conference_conferences_flow_id (flow_id),
  KEY idx_conference_conferences_confbridge_id (confbridge_id)
);
