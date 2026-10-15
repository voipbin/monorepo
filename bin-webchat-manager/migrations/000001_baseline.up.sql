-- Baseline of the tables owned by bin-webchat-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS webchat_widgets (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  name varchar(255) DEFAULT NULL,
  status varchar(16) NOT NULL,
  direct_id binary(16) DEFAULT NULL,
  direct_hash varchar(255) DEFAULT NULL,
  session_flow_id binary(16) DEFAULT NULL,
  message_flow_id binary(16) DEFAULT NULL,
  session_idle_timeout int DEFAULT NULL,
  theme_config json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_webchat_widgets_direct_id (direct_id),
  KEY idx_webchat_widgets_customer_id_tm_create (customer_id, tm_create),
  KEY idx_webchat_widgets_customer_id_tm_delete (customer_id, tm_delete)
);

CREATE TABLE IF NOT EXISTS webchat_sessions (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  widget_id binary(16) NOT NULL,
  status varchar(16) NOT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  tm_last_activity datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_end datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  page_url varchar(2048) DEFAULT NULL,
  referrer varchar(2048) DEFAULT NULL,
  peer json DEFAULT NULL,
  `local` json DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_webchat_sessions_customer_id_tm_create (customer_id, tm_create),
  KEY idx_webchat_sessions_customer_id_tm_delete (customer_id, tm_delete),
  KEY idx_webchat_sessions_widget_id_status (widget_id, status),
  KEY idx_webchat_sessions_widget_id_tm_create (widget_id, tm_create),
  KEY idx_webchat_sessions_status_tm_last_activity (status, tm_last_activity)
);

CREATE TABLE IF NOT EXISTS webchat_messages (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  widget_id binary(16) NOT NULL,
  session_id binary(16) NOT NULL,
  direction varchar(16) NOT NULL,
  status varchar(16) NOT NULL,
  `text` varchar(4000) DEFAULT NULL,
  sender_id binary(16) DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_webchat_messages_session_id_tm_create (session_id, tm_create),
  KEY idx_webchat_messages_customer_id_tm_create (customer_id, tm_create),
  KEY idx_webchat_messages_widget_id_tm_create (widget_id, tm_create)
);
