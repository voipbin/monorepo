create table contact_manager_contact(
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

create index idx_contact_manager_contact_customer_id on contact_manager_contact(customer_id);
create index idx_contact_manager_contact_external_id on contact_manager_contact(external_id);

create table contact_manager_phone_number(
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

create index idx_contact_manager_phone_number_contact_id on contact_manager_phone_number(contact_id);
create index idx_contact_manager_phone_number_number_e164 on contact_manager_phone_number(number_e164);

create table contact_manager_email(
  id            binary(16),
  customer_id   binary(16),
  contact_id    binary(16),

  address       varchar(255),
  type          varchar(20),
  is_primary    boolean,

  tm_create datetime(6),

  primary key(id)
);

create index idx_contact_manager_email_contact_id on contact_manager_email(contact_id);
create index idx_contact_manager_email_address on contact_manager_email(address);

create table contact_manager_tag_assignment(
  contact_id    binary(16),
  tag_id        binary(16),

  tm_create datetime(6),

  primary key(contact_id, tag_id)
);
