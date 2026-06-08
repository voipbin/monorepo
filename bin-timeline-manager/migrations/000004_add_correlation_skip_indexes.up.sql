-- Backfill MATERIALIZED columns on existing parts. Columns added in 000002/000003
-- only auto-compute for rows inserted after the ALTER; pre-existing parts may have
-- empty resource_id / activeflow_id. MATERIALIZE COLUMN rewrites old parts so that
-- historical events become correlatable.
ALTER TABLE events MATERIALIZE COLUMN resource_id;
ALTER TABLE events MATERIALIZE COLUMN activeflow_id;

-- Data-skipping indexes for point lookups on resource_id / activeflow_id.
-- The table is ORDER BY (event_type, timestamp), so filtering by these columns
-- alone is a full scan without an index.
ALTER TABLE events ADD INDEX IF NOT EXISTS idx_resource_id   resource_id   TYPE bloom_filter GRANULARITY 1;
ALTER TABLE events ADD INDEX IF NOT EXISTS idx_activeflow_id activeflow_id TYPE bloom_filter GRANULARITY 1;

-- Build the indexes over existing parts.
ALTER TABLE events MATERIALIZE INDEX idx_resource_id;
ALTER TABLE events MATERIALIZE INDEX idx_activeflow_id;
