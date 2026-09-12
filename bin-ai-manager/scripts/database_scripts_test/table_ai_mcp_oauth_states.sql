create table ai_mcp_oauth_states(
  state         varchar(64)  not null,
  customer_id   binary(16)   not null,
  mcp_server_id binary(16),
  vendor        varchar(64)  not null,
  pkce_verifier varchar(128) not null,

  tm_create datetime(6) not null,
  tm_expire datetime(6) not null,

  primary key(state)
);

create index idx_ai_mcp_oauth_states_customer_id on ai_mcp_oauth_states(customer_id);
create index idx_ai_mcp_oauth_states_tm_expire on ai_mcp_oauth_states(tm_expire);
