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
create index idx_contact_addresses_contact_type on contact_addresses(contact_id, type);
create index idx_contact_addresses_lookup on contact_addresses(customer_id, type, target);

create table contact_tag_assignments(
  contact_id    binary(16),
  tag_id        binary(16),

  tm_create datetime(6),

  primary key(contact_id, tag_id)
);

-- contact_interactions mirrors migration
-- 2026-07-22-case-interaction-peer-local-address-json-design.md §3.2:
-- peer/local are now JSON columns holding a full commonaddress.Address
-- (Type/Target/...); peer_type/peer_target/local_type/local_target are
-- STORED generated columns derived from that JSON, kept ONLY for
-- indexing/filtering -- there is no corresponding Go struct field for
-- them anymore (see interaction.Interaction, which has Peer/Local
-- commonaddress.Address fields, not flat Peer/Local Type/Target
-- fields). SQLite's json_extract/json1 stands in for MySQL's
-- JSON_UNQUOTE(JSON_EXTRACT(...)) here.
create table contact_interactions(
  id             binary(16)    not null,
  customer_id    binary(16)    not null,
  direction      varchar(255)  not null default '',
  peer           json          not null,
  local          json,
  peer_type      varchar(255)  generated always as (json_extract(peer, '$.type')) stored not null,
  peer_target    varchar(255)  generated always as (json_extract(peer, '$.target')) stored not null,
  local_type     varchar(255)  generated always as (json_extract(local, '$.type')) stored,
  local_target   varchar(255)  generated always as (json_extract(local, '$.target')) stored,
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
  interaction_id    binary(16),
  case_id           binary(16),
  resolution_type   varchar(255)  not null default '',
  resolved_by_type  varchar(255)  not null default '',
  resolved_by_id    binary(16)    not null,
  tm_create         datetime(6),
  tm_update         datetime(6),
  tm_delete         datetime(6),

  -- Mirrors migration 99e7e955a149's case_positive_uk (design §3.3): at
  -- most one active case-level positive Resolution per case. SQLite has
  -- no IF()/SHA2; CASE WHEN over the raw case_id is equivalent here (no
  -- truncation risk, same rationale as open_peer_uk above).
  case_positive_uk binary(16) generated always as (
    case when resolution_type = 'positive' and interaction_id is null and tm_delete is null
         then case_id else null end
  ) stored,

  primary key(id)
);

create unique index uq_resolution_case_positive on contact_resolutions(case_positive_uk);
create index idx_contact_resolutions_contact_interaction
  on contact_resolutions(customer_id, contact_id, tm_delete);
create index idx_contact_resolutions_interaction
  on contact_resolutions(customer_id, interaction_id, tm_delete);
create index idx_contact_resolutions_case
  on contact_resolutions(customer_id, case_id, tm_delete);

-- contact_cases mirrors migration f718e26f2c44 (design §3.1). SQLite has no
-- SHA2/UNHEXFn; the production open_peer_uk hash-digest technique is
-- replaced here with a plain STORED generated column over the same
-- (customer_id, peer_type, peer_target, reference_type) tuple -- SQLite's
-- test rows are small and never approach the truncation risk that
-- motivated the hash in production (see f718e26f2c44's docstring). Same
-- "value only when status='open', NULL otherwise" partial-unique
-- invariant either way.
-- contact_cases mirrors migration
-- 2026-07-22-case-interaction-peer-local-address-json-design.md §3.1:
-- peer/local are now JSON columns holding a full commonaddress.Address;
-- peer_type/peer_target/local_type/local_target are STORED generated
-- columns derived from that JSON, kept ONLY for indexing/filtering --
-- there is no corresponding Go struct field for them anymore (see
-- kase.Case, which has Peer/Local commonaddress.Address fields, not
-- flat PeerType/PeerTarget/LocalType/LocalTarget fields).
create table contact_cases (
  id                binary(16)    not null,
  customer_id       binary(16)    not null,

  peer              json          not null,
  local             json          not null,
  peer_type         varchar(255)  generated always as (json_extract(peer, '$.type')) stored not null,
  peer_target       varchar(255)  generated always as (json_extract(peer, '$.target')) stored not null,
  local_type        varchar(255)  generated always as (json_extract(local, '$.type')) stored,
  local_target      varchar(255)  generated always as (json_extract(local, '$.target')) stored,
  reference_type    varchar(255)  not null default '',

  name              varchar(255)  not null default '',
  detail            text,

  contact_id        binary(16),

  owner_type        varchar(255),
  owner_id          binary(16),

  status            varchar(32)   not null default 'open',
  opened_at         datetime(6),
  closed_at         datetime(6),
  closed_reason     varchar(32),
  closed_by_type    varchar(32),
  closed_by_id      binary(16),

  previous_case_id  binary(16),

  tag_ids           json,

  tm_create         datetime(6),
  tm_update         datetime(6),

  open_peer_uk varchar(600) generated always as (
    case when status = 'open' then customer_id || '|' || peer_type || '|' || peer_target || '|' || reference_type else null end
  ) stored,

  primary key(id)
);

create unique index uq_case_open_peer on contact_cases(open_peer_uk);
create index idx_case_unresolved on contact_cases(customer_id, status, contact_id);
create index idx_case_owner on contact_cases(customer_id, owner_type, owner_id);
create index idx_case_customer_reftype on contact_cases(customer_id, reference_type);

-- contact_case_notes mirrors migration 437ca5f2e4ee (design §3.5).
-- Physically separate from contact_interactions -- never surfaced in any
-- customer-facing webhook or API response.
create table contact_case_notes (
  id          binary(16)   not null,
  customer_id binary(16)   not null,
  case_id     binary(16)   not null,

  author_type varchar(32)  not null default '',
  author_id   binary(16),

  text        text         not null,

  tm_create   datetime(6),
  tm_update   datetime(6),
  tm_delete   datetime(6),

  primary key(id)
);

create index idx_contact_case_notes_case_id on contact_case_notes(case_id, tm_delete);

-- contact_case_tag_assignments (junction table) removed -- superseded by
-- contact_cases.tag_ids (design VOIP-1254, migration c2e53c0f6453),
-- mirroring bin-queue-manager's Queue.TagIDs storage exactly.

-- contact_address_ownership_periods mirrors migration
-- 2d8f0ea90565_contact_address_ownership_periods_ (design §3.1/§9,
-- Phase 1). SQLite has no SHA2/UNHEX -- the production open_period_uk
-- hash-digest technique is replaced here with a plain STORED generated
-- column over the same (customer_id, type, target) tuple, same
-- "value only when valid_to IS NULL (open), NULL otherwise"
-- partial-unique invariant as production, direct lift of contact_cases'
-- open_peer_uk test-schema pattern above.
create table contact_address_ownership_periods (
  id          binary(16)    not null,
  customer_id binary(16)    not null,
  contact_id  binary(16)    not null,

  type        varchar(255)  not null default '',
  target      varchar(255)  not null default '',

  valid_from  datetime(6),
  valid_to    datetime(6),

  open_period_uk varchar(600) generated always as (
    case when valid_to is null then customer_id || '|' || type || '|' || target else null end
  ) stored,

  tm_create   datetime(6),
  tm_update   datetime(6),

  primary key(id)
);

create unique index idx_ownership_periods_open on contact_address_ownership_periods(open_period_uk);
create index idx_ownership_periods_contact on contact_address_ownership_periods(customer_id, contact_id);
create index idx_ownership_periods_lookup on contact_address_ownership_periods(customer_id, type, target, valid_from, valid_to);
