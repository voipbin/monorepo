create table ai_ai_prompt_histories (
  id          binary(16)  not null,
  customer_id binary(16)  not null,
  ai_id       binary(16)  not null,
  prompt      longtext    not null,
  tm_create   datetime(6) not null,
  primary key (id)
);

create index idx_ai_ai_prompt_histories_ai_id_tm_create on ai_ai_prompt_histories(ai_id, tm_create);
create index idx_ai_ai_prompt_histories_customer_id on ai_ai_prompt_histories(customer_id);
