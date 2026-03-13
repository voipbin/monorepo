ALTER TABLE events
ADD COLUMN IF NOT EXISTS activeflow_id String
MATERIALIZED if(event_type LIKE 'activeflow_%', JSONExtractString(data, 'id'), JSONExtractString(data, 'activeflow_id'));
