CREATE TABLE email_emails (
  -- identity
  id          binary(16),
  customer_id binary(16),

  activeflow_id binary(16),

  -- provider info
  provider_type varchar(255),
  provider_reference_id varchar(255),

  -- from/to info
  source        json,
  destinations  json,

  -- message info
  status  varchar(255),
  subject varchar(255),
  content text,

  attachments json,

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_email_emails_customer_id on email_emails(customer_id);
create index idx_email_emails_activeflow_id on email_emails(activeflow_id);
create index idx_email_emails_provider_type on email_emails(provider_type);
create index idx_email_emails_provider_reference_id on email_emails(provider_reference_id);
