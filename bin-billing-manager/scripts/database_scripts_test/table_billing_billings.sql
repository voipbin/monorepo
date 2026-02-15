create table billing_billings(
  id            binary(16),
  customer_id   binary(16),
  account_id    binary(16),

  transaction_type varchar(32),
  status           varchar(32),

  reference_type  varchar(32),
  reference_id    binary(16),

  cost_type             varchar(64),
  usage_duration        integer default 0,
  billable_units        integer default 0,

  rate_token_per_unit   bigint default 0,
  rate_credit_per_unit  bigint default 0,

  amount_token          bigint default 0,
  amount_credit         bigint default 0,

  balance_token_snapshot  bigint default 0,
  balance_credit_snapshot bigint default 0,

  idempotency_key binary(16),

  -- timestamps
  tm_billing_start  datetime(6),
  tm_billing_end    datetime(6),

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_billing_billings_customer_id on billing_billings(customer_id);
create index idx_billing_billings_account_id on billing_billings(account_id);
create index idx_billing_billings_reference_id on billing_billings(reference_id);
create unique index idx_billings_ref_type_id_active on billing_billings(reference_type, reference_id, tm_delete);
create index idx_billing_billings_create on billing_billings(tm_create);
