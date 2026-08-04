create table call_outbound_configs(
  -- identity
  -- NOTE: id and customer_id are BINARY(16) in MySQL (migration dd500d4e8cb7,
  -- 2026-05-07, which converted them from VARCHAR(36)). The ,uuid read path
  -- (copyUUID -> uuid.FromBytes) requires exactly 16 bytes, so tests must write
  -- genuine 16-byte binary values here, not 36-char strings.
  id          binary(16),   -- outbound config id
  customer_id binary(16),   -- customer id

  name    varchar(255), -- name
  detail  text,         -- detail

  destination_whitelist json,         -- ISO 3166 alpha-2 lowercase country codes
  codecs                varchar(255), -- comma-separated; empty = server default

  default_outgoing_source_number_id binary(16), -- default outgoing source number id

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create unique index uq_call_outbound_configs_customer_id on call_outbound_configs(customer_id);
create index idx_call_outbound_configs_tm_create on call_outbound_configs(tm_create);
