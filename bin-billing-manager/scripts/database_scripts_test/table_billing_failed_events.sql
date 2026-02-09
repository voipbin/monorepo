create table billing_failed_events(
  id              binary(16) not null,
  event_type      varchar(255) not null,
  event_publisher varchar(255) not null,
  event_data      text not null,
  error_message   text not null,
  retry_count     integer not null default 0,
  max_retries     integer not null default 5,
  next_retry_at   datetime(6) not null,
  status          varchar(32) not null default 'pending',

  -- timestamps
  tm_create datetime(6) not null,
  tm_update datetime(6) not null,

  primary key(id)
);

create index idx_failed_events_status_retry on billing_failed_events(status, next_retry_at);
