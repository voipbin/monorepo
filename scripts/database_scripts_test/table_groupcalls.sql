create table groupcalls(
  -- identity
  id                binary(16),   -- id
  customer_id       binary(16),   -- customer id

  status    varchar(16),
  flow_id   binary(16),

  source        json,
  destinations  json,

  master_call_id        binary(16),
  master_groupcall_id   binary(16),

  ring_method   varchar(255),
  answer_method varchar(255),

  answer_call_id    binary(16),
  call_ids          json,

  answer_groupcall_id    binary(16),
  groupcall_ids          json,

  call_count        integer,
  groupcall_count   integer,
  dial_index        integer,

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_groupcalls_customer_id on groupcalls(customer_id);
