create table ai_ai_prompt_proposals (
  id                         binary(16) not null,
  customer_id                binary(16) not null,
  ai_id                      binary(16) not null,

  audit_ids                  json         not null,
  basis_prompt_history_id    binary(16)   not null,

  original_prompt            text,
  proposed_prompt            text,
  rationale                  text,

  status                     varchar(32)  not null default 'progressing',
  error                      varchar(128) not null default '',
  applied_prompt_history_id  binary(16)   not null default 0x00000000000000000000000000000000,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_ai_ai_prompt_proposals_customer_id on ai_ai_prompt_proposals(customer_id);
create index idx_ai_ai_prompt_proposals_ai_id       on ai_ai_prompt_proposals(ai_id);
create index idx_ai_ai_prompt_proposals_tm_create   on ai_ai_prompt_proposals(tm_create);
