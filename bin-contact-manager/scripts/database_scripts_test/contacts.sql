create table contact_contacts(
  -- identity
  id            binary(16),
  customer_id   binary(16),

  -- basic info
  first_name    varchar(255),
  last_name     varchar(255),
  display_name  varchar(255),
  company       varchar(255),
  job_title     varchar(255),

  -- tracking
  source        varchar(50),
  external_id   varchar(255),
  notes         text,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_contact_contacts_customer_id on contact_contacts(customer_id);
create index idx_contact_contacts_external_id on contact_contacts(external_id);

create table contact_phone_numbers(
  id            binary(16),
  customer_id   binary(16),
  contact_id    binary(16),

  number        varchar(50),
  number_e164   varchar(20),
  type          varchar(20),
  is_primary    boolean,

  tm_create datetime(6),

  primary key(id)
);

create index idx_contact_phone_numbers_contact_id on contact_phone_numbers(contact_id);
create index idx_contact_phone_numbers_number_e164 on contact_phone_numbers(number_e164);

create table contact_emails(
  id            binary(16),
  customer_id   binary(16),
  contact_id    binary(16),

  address       varchar(255),
  type          varchar(20),
  is_primary    boolean,

  tm_create datetime(6),

  primary key(id)
);

create index idx_contact_emails_contact_id on contact_emails(contact_id);
create index idx_contact_emails_address on contact_emails(address);

create table contact_tag_assignments(
  contact_id    binary(16),
  tag_id        binary(16),

  tm_create datetime(6),

  primary key(contact_id, tag_id)
);
