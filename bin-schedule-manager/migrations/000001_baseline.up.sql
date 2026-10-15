-- Baseline of the tables owned by bin-schedule-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- The table option (including ROW_FORMAT=DYNAMIC) is the one the Alembic stream declared for both tables.
CREATE TABLE IF NOT EXISTS schedule_schedules (
  id BINARY(16) NOT NULL,
  customer_id BINARY(16) NOT NULL, -- nil UUID = platform job
  name VARCHAR(255) NOT NULL,
  detail VARCHAR(1024),
  type VARCHAR(32) NOT NULL, -- 'rpc' (the only value; the enum lives in Go)
  cron VARCHAR(128) NOT NULL, -- 5-field cron expression, evaluated in UTC
  target_queue VARCHAR(255) NOT NULL,
  target_uri VARCHAR(1024) NOT NULL,
  target_method VARCHAR(16) NOT NULL,
  target_data_type VARCHAR(255),
  target_data JSON,
  timeout_ms INT NOT NULL,
  retry_max INT NOT NULL DEFAULT 0,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  tm_next_run DATETIME(6), -- NULL = compute on next scan
  tm_last_run DATETIME(6),
  tm_create DATETIME(6),
  tm_update DATETIME(6),
  tm_delete DATETIME(6), -- NULL = active
  PRIMARY KEY (id),
  INDEX idx_schedule_schedules_customer_name (customer_id, name),
  INDEX idx_schedule_schedules_next_run (enabled, tm_next_run)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS schedule_executions (
  id BINARY(16) NOT NULL,
  customer_id BINARY(16) NOT NULL,
  schedule_id BINARY(16) NOT NULL,
  trigger_type VARCHAR(16) NOT NULL, -- 'cron' / 'manual'
  status VARCHAR(32) NOT NULL, -- 'running' / 'success' / 'failed' / 'abandoned'
  status_code INT,
  error TEXT,
  result MEDIUMTEXT,
  attempt_count INT NOT NULL DEFAULT 0,
  duration_ms INT,
  tm_scheduled DATETIME(6) NOT NULL,
  tm_deadline DATETIME(6) NOT NULL,
  tm_start DATETIME(6),
  tm_end DATETIME(6),
  tm_create DATETIME(6),
  tm_update DATETIME(6),
  tm_delete DATETIME(6),
  PRIMARY KEY (id),
  INDEX idx_schedule_executions_schedule (schedule_id, tm_create),
  INDEX idx_schedule_executions_reap (status, tm_deadline),
  INDEX idx_schedule_executions_tm_create (tm_create),
  UNIQUE KEY uq_schedule_executions_schedule_scheduled (schedule_id, trigger_type, tm_scheduled)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
