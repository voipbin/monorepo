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

create table contact_addresses(
  id            binary(16),
  customer_id   binary(16),
  contact_id    binary(16),

  type          varchar(255),
  target        varchar(255),
  target_name   varchar(255),
  name          varchar(255),
  detail        varchar(255),
  is_primary    boolean,

  -- STORED generated column mirroring production: one-primary-per-contact.
  -- SQLite supports generated columns (3.31+), enabling the same
  -- UNIQUE(customer_id, primary_contact_uk) one-primary invariant in tests.
  primary_contact_uk binary(16) generated always as (case when is_primary=1 then contact_id else null end) stored,

  tm_create datetime(6),
  tm_update datetime(6),

  primary key(id)
);

create unique index idx_contact_addresses_cust_type_target on contact_addresses(customer_id, type, target);
create unique index idx_contact_addresses_cust_primary on contact_addresses(customer_id, primary_contact_uk);
create index idx_contact_addresses_contact_id on contact_addresses(contact_id);
create index idx_contact_addresses_lookup on contact_addresses(customer_id, type, target);

create table contact_tag_assignments(
  contact_id    binary(16),
  tag_id        binary(16),

  tm_create datetime(6),

  primary key(contact_id, tag_id)
);

create table contact_interactions(
  id             binary(16)    not null,
  customer_id    binary(16)    not null,
  direction      varchar(255)  not null default '',
  peer_type      varchar(255)  not null default '',
  peer_target    varchar(255)  not null default '',
  local_type     varchar(255)  not null default '',
  local_target   varchar(255)  not null default '',
  reference_type varchar(255)  not null default '',
  reference_id   binary(16)    not null,
  tm_interaction datetime(6),
  tm_create      datetime(6),
  primary key(id)
);
create unique index idx_contact_interactions_idem on contact_interactions(reference_type, reference_id, peer_target);
create index idx_contact_interactions_peer on contact_interactions(customer_id, peer_type, peer_target);
create index idx_contact_interactions_cursor on contact_interactions(customer_id, tm_create);

create table contact_resolutions (
  id                binary(16)    not null,
  customer_id       binary(16)    not null,
  contact_id        binary(16)    not null,
  interaction_id    binary(16)    not null,
  resolution_type   varchar(255)  not null default '',
  resolved_by_type  varchar(255)  not null default '',
  resolved_by_id    binary(16)    not null,
  tm_create         datetime(6),
  tm_update         datetime(6),
  tm_delete         datetime(6),
  primary key(id)
);

create index idx_contact_resolutions_contact_interaction
  on contact_resolutions(customer_id, contact_id, tm_delete);
create index idx_contact_resolutions_interaction
  on contact_resolutions(customer_id, interaction_id, tm_delete);
