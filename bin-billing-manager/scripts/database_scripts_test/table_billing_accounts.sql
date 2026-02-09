create table billing_accounts(
  id            binary(16),
  customer_id   binary(16),

  type      varchar(255),
  plan_type varchar(255),

  name    varchar(255),
  detail  text,

  balance float,

  payment_type      varchar(255),
  payment_method    varchar(255),

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_billing_accounts_customer_id on billing_accounts(customer_id);
create index idx_billing_accounts_create on billing_accounts(tm_create);
