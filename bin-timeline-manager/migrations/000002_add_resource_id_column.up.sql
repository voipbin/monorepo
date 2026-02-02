ALTER TABLE events
ADD COLUMN resource_id String MATERIALIZED JSONExtractString(data, 'id');
