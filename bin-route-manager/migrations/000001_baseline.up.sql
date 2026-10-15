-- Baseline of the tables owned by bin-route-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS route_providercalls (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  provider_id binary(16) DEFAULT NULL,
  flow_id binary(16) DEFAULT NULL,
  source json DEFAULT NULL,
  destinations json DEFAULT NULL,
  anonymous varchar(16) DEFAULT NULL,
  call_ids json DEFAULT NULL,
  groupcall_ids json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_route_providercalls_customer_id (customer_id),
  KEY idx_route_providercalls_provider_id (provider_id),
  KEY idx_route_providercalls_tm_delete_tm_create (tm_delete, tm_create)
);
CREATE TABLE IF NOT EXISTS route_providers (
  id binary(16) NOT NULL,
  type varchar(255) DEFAULT NULL,
  hostname varchar(255) DEFAULT NULL,
  tech_prefix varchar(255) DEFAULT NULL,
  tech_postfix varchar(255) DEFAULT NULL,
  tech_headers json DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  health_status varchar(64) NOT NULL DEFAULT 'unknown',
  health_checked_at datetime(6) DEFAULT NULL,
  metadata json DEFAULT NULL,
  codecs varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS route_routes (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  provider_id binary(16) DEFAULT NULL,
  priority int DEFAULT NULL,
  target varchar(255) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_route_routes_customer_id (customer_id),
  KEY idx_route_routes_provider_id (provider_id)
);
