create table queue_queuecalls(
  -- identity
  id                binary(16),   -- id
  customer_id       binary(16),   -- owner's id
  queue_id          binary(16),   -- queue id

  reference_type          varchar(255), -- reference's type
  reference_id            binary(16),   -- reference's id
  reference_activeflow_id binary(16),   -- reference's activeflow id

  forward_action_id binary(16),   -- action id for forward.
  confbridge_id     binary(16),

  source            json,
  routing_method    varchar(255),
  tag_ids           json,

  status            varchar(255), --
  service_agent_id  binary(16),   --

  timeout_wait      integer,  --
  timeout_service   integer,  --

  duration_waiting  integer,
  duration_service  integer,

  tm_create   datetime(6),
  tm_service  datetime(6),
  tm_update   datetime(6),
  tm_end      datetime(6),
  tm_delete   datetime(6),

  primary key(id)
);

create index idx_queue_queuecalls_customer_id on queue_queuecalls(customer_id);
create index idx_queue_queuecalls_queue_id on queue_queuecalls(queue_id);
create index idx_queue_queuecalls_reference_id on queue_queuecalls(reference_id);
create index idx_queue_queuecalls_reference_activeflow_id on queue_queuecalls(reference_activeflow_id);
create index idx_queue_queuecalls_service_agent_id on queue_queuecalls(service_agent_id);
