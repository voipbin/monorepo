create table outdial_outdials(
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

create index idx_outdial_outdials_customer_id on outdial_outdials(customer_id);
create index idx_outdial_outdials_campaign_id on outdial_outdials(campaign_id);
