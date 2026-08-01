create table schedule_executions(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- copied from schedule
  schedule_id   binary(16),   -- schedule id

  -- run info
  trigger_type  varchar(16),  -- cron / manual
  status        varchar(32),  -- running / success / failed / abandoned
  status_code   integer,      -- RPC response status code (0 if transport error)
  error         text,         -- error string on failure/abandonment
  result        text,         -- RPC response body on success (MEDIUMTEXT in MySQL)

  attempt_count integer not null default 0,   -- 1 + retries actually used
  duration_ms   integer,                      -- total wall time of the run

  -- run timestamps
  tm_scheduled  datetime(6),  -- the tm_next_run this row consumed (idempotency key part)
  tm_deadline   datetime(6),  -- reap deadline, computed in Go at INSERT
  tm_start      datetime(6),
  tm_end        datetime(6),

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),  -- convention only; never set by the service

  primary key(id)
);

create index idx_schedule_executions_schedule on schedule_executions(schedule_id, tm_create);
create index idx_schedule_executions_reap on schedule_executions(status, tm_deadline);
create index idx_schedule_executions_tm_create on schedule_executions(tm_create);

-- hard DB-level backstop against double-fire (design §4.2)
create unique index uq_schedule_executions_schedule_scheduled on schedule_executions(schedule_id, trigger_type, tm_scheduled);
