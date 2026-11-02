-- DOWN-LOSSY: dropping the column discards the recorded dispositions
ALTER TABLE contact_cases DROP COLUMN disposition, ALGORITHM=INPLACE, LOCK=NONE;
