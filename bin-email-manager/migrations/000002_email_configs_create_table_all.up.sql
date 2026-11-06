CREATE TABLE IF NOT EXISTS email_configs (
  config_key         varchar(128)    NOT NULL,
  value_type         varchar(16)     NOT NULL,
  value_json         json            DEFAULT NULL,
  secret_ciphertext  varbinary(2048) DEFAULT NULL,
  key_version        int             NOT NULL DEFAULT 0,
  revision           int unsigned    NOT NULL DEFAULT 1,
  updated_by         binary(16)      DEFAULT NULL,
  tm_create          datetime(6)     NOT NULL,
  tm_update          datetime(6)     NOT NULL,
  PRIMARY KEY (config_key),
  CONSTRAINT chk_email_configs_one_of CHECK ((value_json IS NULL) <> (secret_ciphertext IS NULL)),
  CONSTRAINT chk_email_configs_type CHECK (value_type IN ('string','int','bool','duration','enum','secret')),
  CONSTRAINT chk_email_configs_secret_type CHECK ((value_type = 'secret') = (secret_ciphertext IS NOT NULL)),
  CONSTRAINT chk_email_configs_json CHECK (value_json IS NULL OR CAST(JSON_TYPE(value_json) AS BINARY) <> CAST('NULL' AS BINARY)),
  CONSTRAINT chk_email_configs_version CHECK (
    (secret_ciphertext IS NULL AND key_version = 0) OR (secret_ciphertext IS NOT NULL AND key_version > 0))
);
