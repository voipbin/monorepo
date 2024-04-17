create table conversation_medias(
  -- identity
  id            binary(16),     -- id
  customer_id   binary(16),     -- customer id

  type      varchar(255),
  Filename  varchar(2047),

  -- timestamps
  tm_create   datetime(6),  --
  tm_update   datetime(6),  --
  tm_delete   datetime(6),  --

  primary key(id)
);

create index idx_conversation_medias_customerid on conversation_medias(customer_id);
