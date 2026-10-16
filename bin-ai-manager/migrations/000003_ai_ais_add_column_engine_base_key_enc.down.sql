-- DOWN-LOSSY: dropping the column discards the stored encrypted keys
ALTER TABLE ai_ais DROP COLUMN engine_base_key_enc, ALGORITHM=INPLACE, LOCK=NONE;
