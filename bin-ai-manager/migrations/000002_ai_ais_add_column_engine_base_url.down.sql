-- DOWN-LOSSY: dropping the column discards the stored base URLs
ALTER TABLE ai_ais DROP COLUMN engine_base_url, ALGORITHM=INPLACE, LOCK=NONE;
