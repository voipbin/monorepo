CREATE TABLE IF NOT EXISTS peer_events (
    timestamp     DateTime64(3),
    customer_id   UUID,
    publisher     LowCardinality(String),
    event_type    LowCardinality(String),
    reference_id  UUID,

    direction     LowCardinality(String),

    peer_type     LowCardinality(String),
    peer_target   String,

    local_type    LowCardinality(String),
    local_target  String,

    data          String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (customer_id, peer_type, peer_target, timestamp)
TTL toDateTime(timestamp) + INTERVAL 1 YEAR;
