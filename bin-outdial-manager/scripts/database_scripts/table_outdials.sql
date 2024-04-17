create table outdials(
  -- identity
  id          binary(16),
  customer_id binary(16),

  campaign_id binary(16),

  name      varchar(255),
  detail    text,

  data  text,

  -- timestamps
  tm_create datetime(6),  -- create
  tm_update datetime(6),  -- update
  tm_delete datetime(6),  -- delete

  primary key(id)
);

create index idx_outdials_customer_id on outdials(customer_id);
create index idx_outdials_campaign_id on outdials(campaign_id);
