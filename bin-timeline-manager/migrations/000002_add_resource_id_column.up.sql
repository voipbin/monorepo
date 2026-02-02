ALTER TABLE events
ADD COLUMN IF NOT EXISTS resource_id String MATERIALIZED JSONExtractString(data, 'id');
