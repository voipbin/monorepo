create table queuecallreferences(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),
  type          varchar(255),

  current_queuecall_id  binary(16),
  queuecall_ids         json,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);
