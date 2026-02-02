CREATE TABLE IF NOT EXISTS events (
    timestamp DateTime64(3),
    event_type LowCardinality(String),
    publisher LowCardinality(String),
    data_type LowCardinality(String),
    data String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (event_type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 1 YEAR;
