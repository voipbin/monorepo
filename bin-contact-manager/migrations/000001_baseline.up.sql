-- Baseline of the tables owned by bin-contact-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- The table options are the ones Alembic declared for each table; JSON columns are written as json.
CREATE TABLE IF NOT EXISTS contact_activities (
  id binary(16) NOT NULL,
  contact_id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  `type` varchar(20) NOT NULL,
  direction varchar(20) NOT NULL,
  reference_id binary(16) DEFAULT NULL,
  summary varchar(500) DEFAULT '',
  duration bigint(20) DEFAULT 0,
  tm_activity datetime(6) NOT NULL,
  tm_create datetime(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_contact_activities_contact_time (contact_id, tm_activity DESC),
  KEY idx_contact_activities_customer_time (customer_id, tm_activity DESC),
  KEY idx_contact_activities_reference (reference_id)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE IF NOT EXISTS contact_address_ownership_periods (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  contact_id binary(16) NOT NULL,
  `type` varchar(255) NOT NULL DEFAULT '',
  target varchar(255) NOT NULL DEFAULT '',
  valid_from datetime(6) DEFAULT NULL,
  valid_to datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  open_period_uk binary(32) GENERATED ALWAYS AS (
    IF(valid_to IS NULL, UNHEX(SHA2(CONCAT_WS('|', customer_id, `type`, target), 256)), NULL)
  ) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY idx_ownership_periods_open (open_period_uk),
  KEY idx_ownership_periods_contact (customer_id, contact_id),
  KEY idx_ownership_periods_lookup (customer_id, `type`, target, valid_from, valid_to)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
CREATE TABLE IF NOT EXISTS contact_addresses (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  contact_id binary(16) DEFAULT NULL,
  `type` varchar(255) NOT NULL DEFAULT '',
  target varchar(255) NOT NULL DEFAULT '',
  target_name varchar(255) NOT NULL DEFAULT '',
  name varchar(255) NOT NULL DEFAULT '',
  detail varchar(255) NOT NULL DEFAULT '',
  is_primary tinyint(1) NOT NULL DEFAULT 0,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  primary_contact_uk binary(16) GENERATED ALWAYS AS (IF(is_primary = 1, contact_id, NULL)) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY idx_contact_addresses_identifier (customer_id, `type`, target),
  UNIQUE KEY idx_contact_addresses_primary (customer_id, primary_contact_uk),
  KEY idx_contact_addresses_contact_type (contact_id, `type`),
  CONSTRAINT chk_contact_addresses_primary_resolved CHECK (is_primary = 0 OR contact_id IS NOT NULL)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
CREATE TABLE IF NOT EXISTS contact_case_notes (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  case_id binary(16) NOT NULL,
  author_type varchar(32) NOT NULL DEFAULT '',
  author_id binary(16) DEFAULT NULL,
  `text` text NOT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_contact_case_notes_case_id (case_id, tm_delete)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
CREATE TABLE IF NOT EXISTS contact_cases (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  peer json NOT NULL,
  `local` json NULL,
  reference_type varchar(255) NOT NULL DEFAULT '',
  reference_id varchar(255) NOT NULL DEFAULT '',
  name varchar(255) NOT NULL DEFAULT '',
  detail text DEFAULT NULL,
  contact_id binary(16) DEFAULT NULL,
  owner_type varchar(255) DEFAULT NULL,
  owner_id binary(16) DEFAULT NULL,
  status varchar(32) NOT NULL DEFAULT 'open',
  opened_at datetime(6) DEFAULT NULL,
  closed_at datetime(6) DEFAULT NULL,
  closed_reason varchar(32) DEFAULT NULL,
  closed_by_type varchar(32) DEFAULT NULL,
  closed_by_id binary(16) DEFAULT NULL,
  previous_case_id binary(16) DEFAULT NULL,
  tag_ids json NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  peer_type varchar(255) GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type'))) STORED,
  peer_target varchar(255) GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.target'))) STORED,
  local_type varchar(255) GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(`local`, '$.type'))) STORED,
  local_target varchar(255) GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(`local`, '$.target'))) STORED,
  open_peer_uk binary(32) GENERATED ALWAYS AS (
    IF(status = 'open', UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)), NULL)
  ) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY uq_case_open_peer (open_peer_uk),
  KEY idx_case_unresolved (customer_id, status, contact_id),
  KEY idx_case_owner (customer_id, owner_type, owner_id),
  KEY idx_case_customer_reftype (customer_id, reference_type),
  KEY idx_case_customer_reference_id (customer_id, reference_id),
  CONSTRAINT chk_cases_peer_type_nn CHECK (peer_type IS NOT NULL),
  CONSTRAINT chk_cases_peer_target_nn CHECK (peer_target IS NOT NULL)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
CREATE TABLE IF NOT EXISTS contact_contacts (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  first_name varchar(100) DEFAULT '',
  last_name varchar(100) DEFAULT '',
  display_name varchar(200) DEFAULT '',
  company varchar(200) DEFAULT '',
  job_title varchar(100) DEFAULT '',
  source varchar(50) DEFAULT 'manual',
  external_id varchar(255) DEFAULT '',
  notes text DEFAULT NULL,
  total_calls bigint(20) DEFAULT 0,
  total_messages bigint(20) DEFAULT 0,
  last_contact_at datetime(6) DEFAULT '9999-01-01 00:00:00.000000',
  tm_create datetime(6) NOT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_contact_contacts_customer (customer_id),
  KEY idx_contact_contacts_customer_name (customer_id, display_name),
  KEY idx_contact_contacts_customer_external (customer_id, external_id),
  KEY idx_contact_contacts_tm_create (tm_create),
  FULLTEXT KEY idx_contact_contacts_search (first_name, last_name, display_name, company)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE IF NOT EXISTS contact_list_members (
  list_id binary(16) NOT NULL,
  contact_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL,
  PRIMARY KEY (list_id, contact_id),
  KEY idx_contact_list_members_contact (contact_id)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE IF NOT EXISTS contact_lists (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  name varchar(100) NOT NULL,
  detail text DEFAULT NULL,
  `type` varchar(20) DEFAULT 'static',
  `query` text DEFAULT NULL,
  contact_count bigint(20) DEFAULT 0,
  tm_create datetime(6) NOT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_contact_lists_customer (customer_id),
  KEY idx_contact_lists_tm_create (tm_create)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE IF NOT EXISTS contact_tag_assignments (
  contact_id binary(16) NOT NULL,
  tag_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL,
  PRIMARY KEY (contact_id, tag_id),
  KEY idx_contact_tag_assignments_tag (tag_id)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
