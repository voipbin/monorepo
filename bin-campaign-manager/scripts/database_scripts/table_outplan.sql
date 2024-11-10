
create table campaign_outplans(
  -- identity
  id          binary(16),
  customer_id binary(16),

  name    varchar(255),
  detail  text,

  source  json,

  dial_timeout  integer,
  try_interval  integer,
  max_try_count_0 integer,
  max_try_count_1 integer,
  max_try_count_2 integer,
  max_try_count_3 integer,
  max_try_count_4 integer,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_campaign_outplans_customer_id on campaign_outplans(customer_id);
