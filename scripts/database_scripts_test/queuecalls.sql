create table queuecalls(
  -- identity
  id                binary(16),   -- id
  user_id           integer,      -- user's id

  queue_id          binary(16),   -- queue id
  reference_type    varchar(255), -- reference's type
  reference_id      binary(16),   -- reference's id
  forward_action_id binary(16),   -- action id for forward.
  exit_action_id    binary(16),   -- action id for queue exit.
  confbridge_id     binary(16),

  webhook_uri     varchar(1023),
  webhook_method  varchar(255),

  source            json,
  routing_method    varchar(255),
  tag_ids           json,

  status            varchar(255), --
  service_agent_id  binary(16),   --

  timeout_wait      integer,  --
  timeout_service   integer,  --

  tm_create   datetime(6),
  tm_service  datetime(6),
  tm_update   datetime(6),
  tm_delete   datetime(6),

  primary key(id)
);

create index idx_queuecalls_userid on queuecalls(user_id);
create index idx_queuecalls_queueid on queuecalls(queue_id);
create index idx_queuecalls_referenceid on queuecalls(reference_id);
create index idx_queuecalls_serviceagentid on queuecalls(service_agent_id);
