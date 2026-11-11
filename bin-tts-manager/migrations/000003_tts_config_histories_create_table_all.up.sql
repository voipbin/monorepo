CREATE TABLE IF NOT EXISTS tts_config_histories (
  id               binary(16)   NOT NULL,
  config_key       varchar(128) NOT NULL,
  action           varchar(16)  NOT NULL,
  is_secret        tinyint(1)   NOT NULL,
  old_value_json   json         DEFAULT NULL,
  new_value_json   json         DEFAULT NULL,
  old_key_version  int          NOT NULL DEFAULT 0,
  new_key_version  int          NOT NULL DEFAULT 0,
  actor_id         binary(16)   NOT NULL,
  source           varchar(16)  NOT NULL,
  tm_create        datetime(6)  NOT NULL,
  PRIMARY KEY (id),
  KEY idx_tts_config_histories_key (config_key, tm_create),
  KEY idx_tts_config_histories_create (tm_create),
  CONSTRAINT chk_tts_config_histories_secret CHECK (is_secret = 0 OR (old_value_json IS NULL AND new_value_json IS NULL)),
  CONSTRAINT chk_tts_config_histories_action CHECK (action IN ('set','clear')),
  CONSTRAINT chk_tts_config_histories_source CHECK (source IN ('cli','api'))
);
