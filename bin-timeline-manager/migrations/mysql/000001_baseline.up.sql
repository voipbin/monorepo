-- Baseline of the MySQL tables owned by bin-timeline-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- The table option pins the character set and the collation, so the result does not depend on the
-- default of the server it runs on.
CREATE TABLE IF NOT EXISTS timeline_analyses (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  activeflow_id binary(16) NOT NULL,
  status varchar(32) NOT NULL,
  result json DEFAULT NULL,
  model varchar(255) NOT NULL DEFAULT '',
  error text DEFAULT NULL,
  tm_create datetime(6) NOT NULL,
  tm_update datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_timeline_analyses_activeflow (activeflow_id),
  KEY idx_timeline_analyses_customer_create (customer_id, tm_create)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
