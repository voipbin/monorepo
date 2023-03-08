create table groupcalls(
  -- identity
  id                binary(16),   -- id
  customer_id       binary(16),   -- customer id

  source        json,
  destinations  json,

  ring_method   varchar(255),
  answer_method varchar(255),

  answer_call_id    binary(16),
  call_ids          json,

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_groupcalls_customer_id on groupcalls(customer_id);
