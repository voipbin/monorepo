create table conferencecalls(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id
  conference_id binary(16),

  reference_type    varchar(255),
  reference_id      binary(16),

  status  varchar(255),   -- status

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_conferencecalls_customer_id on conferencecalls(customer_id);
create index idx_conferencecalls_conference_id on conferencecalls(conference_id);
create index idx_conferencecalls_reference_id on conferencecalls(reference_id);
create index idx_conferencecalls_create on conferencecalls(tm_create);
