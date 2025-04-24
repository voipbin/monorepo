create table queue_queues(
  -- identity
  id                binary(16),       -- id
  customer_id       binary(16),       -- owner's id

  -- basic info
  name            varchar(255),
  detail          text,

  routing_method  varchar(16),
  tag_ids         json,

  -- execute
  execute varchar(255),

  -- wait/service info
  wait_flow_id            binary(16),
  wait_queue_call_ids     json,
  wait_timeout            integer,  -- wait timeout(ms)
  service_queue_call_ids  json,
  service_timeout         integer,  -- service timeout(ms)

  total_incoming_count    integer,  -- total incoming count
  total_serviced_count    integer,  -- total serviced count
  total_abandoned_count   integer,  -- total abandoned count

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_queue_queues_customer_id on queue_queues(customer_id);
