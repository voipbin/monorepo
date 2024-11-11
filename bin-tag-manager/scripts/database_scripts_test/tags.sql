create table tag_tags(
  -- identity
  id            binary(16),  -- id
  customer_id   binary(16),

  -- basic info
  name        varchar(255),
  detail      text,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_tag_tags_customer_id on tag_tags(customer_id);
create index idx_tag_tags_name on tag_tags(name);
