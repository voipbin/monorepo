create table scheduler_schedules(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id. nil uuid = platform job

  -- basic info
  name    varchar(255),   -- stable handle for seeds/doctor
  detail  varchar(1024),  -- human description

  type  varchar(32),    -- schedule execution type: rpc
  cron  varchar(128),   -- 5-field cron expression, evaluated in UTC

  -- dispatch target
  target_queue      varchar(255),   -- e.g. bin-manager.number-manager.request
  target_uri        varchar(1024),  -- e.g. /v1/numbers/renew
  target_method     varchar(16),    -- POST/GET/PUT/DELETE
  target_data_type  varchar(255),   -- application/json
  target_data       json,           -- request payload

  timeout_ms  integer,                        -- RPC timeout for the dispatch
  retry_max   integer not null default 0,     -- in-run immediate retries on failure
  enabled     boolean not null default 1,

  -- run bookkeeping
  tm_next_run datetime(6),  -- NULL = compute on next scan
  tm_last_run datetime(6),  -- last successful claim time (fire time, not completion)

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_scheduler_schedules_customer_name on scheduler_schedules(customer_id, name);
create index idx_scheduler_schedules_next_run on scheduler_schedules(enabled, tm_next_run);
