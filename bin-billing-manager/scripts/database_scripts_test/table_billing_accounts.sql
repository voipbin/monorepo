create table billing_accounts(
  id            binary(16),
  customer_id   binary(16),

  plan_type varchar(255),

  name    varchar(255),
  detail  text,

  balance_credit bigint default 0,
  balance_token  bigint default 0,

  payment_type      varchar(255),
  payment_method    varchar(255),

  tm_last_topup datetime(6),
  tm_next_topup datetime(6),

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_billing_accounts_customer_id on billing_accounts(customer_id);
create index idx_billing_accounts_create on billing_accounts(tm_create);
