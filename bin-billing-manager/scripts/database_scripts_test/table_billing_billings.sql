create table billing_billings(
  id            binary(16),
  customer_id   binary(16),
  account_id    binary(16),

  status varchar(32),

  reference_type  varchar(32),
  reference_id    binary(16),

  cost_per_unit float,
  cost_total    float,

  billing_unit_count  float,

  -- timestamps
  tm_billing_start  datetime(6),
  tm_billing_end    datetime(6),

  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_billing_billings_customer_id on billing_billings(customer_id);
create index idx_billing_billings_account_id on billing_billings(account_id);
create index idx_billing_billings_reference_id on billing_billings(reference_id);
create index idx_billing_billings_create on billing_billings(tm_create);
