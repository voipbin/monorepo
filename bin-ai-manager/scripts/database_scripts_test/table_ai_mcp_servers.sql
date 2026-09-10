create table ai_mcp_servers(
  id                binary(16),
  customer_id       binary(16),

  name              varchar(255),
  detail            text,

  url               varchar(2048),
  status            varchar(16),

  auth_type         varchar(16),
  api_key_header    varchar(255),
  secret_ciphertext blob,
  secret_nonce      binary(12),
  key_version       smallint,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_ai_mcp_servers_create on ai_mcp_servers(tm_create);
create index idx_ai_mcp_servers_customer_id on ai_mcp_servers(customer_id);
