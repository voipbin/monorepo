-- Baseline of the tables owned by bin-registrar-manager (BIN database only; the asterisk ps_* tables are not part of it).
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS registrar_customer_domains (
  customer_id binary(16) NOT NULL,
  domain_label varchar(64) NOT NULL,
  realm varchar(255) NOT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  PRIMARY KEY (customer_id),
  UNIQUE KEY ux_registrar_customer_domains_domain_label (domain_label),
  UNIQUE KEY ux_registrar_customer_domains_realm (realm)
);
CREATE TABLE IF NOT EXISTS registrar_extensions (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail varchar(255) DEFAULT NULL,
  endpoint_id varchar(255) DEFAULT NULL,
  aor_id varchar(255) DEFAULT NULL,
  auth_id varchar(255) DEFAULT NULL,
  extension varchar(255) DEFAULT NULL,
  domain_name varchar(255) DEFAULT NULL,
  realm varchar(255) DEFAULT NULL,
  username varchar(255) DEFAULT NULL,
  password varchar(255) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  direct_id binary(16) DEFAULT NULL,
  direct_hash varchar(255) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_registrar_extensions_customer_id (customer_id),
  KEY idx_registrar_extensions_extension (extension),
  KEY idx_registrar_extensions_domain_name (domain_name),
  KEY idx_registrar_extensions_username (username),
  KEY idx_registrar_extensions_realm (realm)
);
CREATE TABLE IF NOT EXISTS registrar_sip_auths (
  id binary(16) NOT NULL,
  reference_type varchar(255) DEFAULT NULL,
  auth_types json DEFAULT NULL,
  realm varchar(255) DEFAULT NULL,
  username varchar(255) DEFAULT NULL,
  password varchar(255) DEFAULT NULL,
  allowed_ips json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_registrar_sip_auths_realm (realm)
);
CREATE TABLE IF NOT EXISTS registrar_trunks (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail varchar(255) DEFAULT NULL,
  domain_name varchar(255) DEFAULT NULL,
  auth_types json DEFAULT NULL,
  realm varchar(255) DEFAULT NULL,
  username varchar(255) DEFAULT NULL,
  password varchar(255) DEFAULT NULL,
  allowed_ips json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_registrar_trunks_customer_id (customer_id),
  KEY idx_registrar_trunks_domain_name (domain_name),
  KEY idx_registrar_trunks_realm (realm)
);
