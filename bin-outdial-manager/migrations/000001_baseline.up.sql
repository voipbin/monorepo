-- Baseline of the tables owned by bin-outdial-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS outdial_outdials (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  campaign_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  data text DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_outdial_outdials_customer_id (customer_id),
  KEY idx_outdial_outdials_campaign_id (campaign_id)
);
CREATE TABLE IF NOT EXISTS outdial_outdialtargetcalls (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  campaign_id binary(16) DEFAULT NULL,
  outdial_id binary(16) DEFAULT NULL,
  outdial_target_id binary(16) DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  reference_type varchar(255) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  status varchar(255) DEFAULT NULL,
  destination json DEFAULT NULL,
  destination_index int(11) DEFAULT NULL,
  try_count int(11) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_outdial_outdialtargetcalls_customer_id (customer_id),
  KEY idx_outdial_outdialtargetcalls_campaign_id (campaign_id),
  KEY idx_outdial_outdialtargetcalls_outdial_target_id (outdial_target_id),
  KEY idx_outdial_outdialtargetcalls_activeflow_id (activeflow_id),
  KEY idx_outdial_outdialtargetcalls_reference_id (reference_id)
);
CREATE TABLE IF NOT EXISTS outdial_outdialtargets (
  id binary(16) NOT NULL,
  outdial_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  data text DEFAULT NULL,
  status varchar(255) DEFAULT NULL,
  destination_0 json DEFAULT NULL,
  destination_1 json DEFAULT NULL,
  destination_2 json DEFAULT NULL,
  destination_3 json DEFAULT NULL,
  destination_4 json DEFAULT NULL,
  try_count_0 int(11) DEFAULT NULL,
  try_count_1 int(11) DEFAULT NULL,
  try_count_2 int(11) DEFAULT NULL,
  try_count_3 int(11) DEFAULT NULL,
  try_count_4 int(11) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_outdial_outdialtargets_outdial_id (outdial_id)
);
