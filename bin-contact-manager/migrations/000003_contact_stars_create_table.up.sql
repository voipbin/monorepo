CREATE TABLE IF NOT EXISTS contact_stars (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  agent_id binary(16) NOT NULL,
  contact_id binary(16) NOT NULL,
  tm_create datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_contact_stars_agent_contact (agent_id, contact_id),
  KEY idx_contact_stars_agent_create (customer_id, agent_id, tm_create)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
