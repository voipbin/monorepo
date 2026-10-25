CREATE TABLE IF NOT EXISTS number_providers (
  provider_name varchar(32) NOT NULL,
  enabled tinyint(1) NOT NULL DEFAULT 0,
  priority int NOT NULL DEFAULT 100,
  config json DEFAULT NULL,
  secret_ciphertext varbinary(2048) DEFAULT NULL,
  key_version int NOT NULL DEFAULT 0,
  updated_by binary(16) DEFAULT NULL,
  secret_updated_by binary(16) DEFAULT NULL,
  tm_secret_update datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  PRIMARY KEY (provider_name)
);
