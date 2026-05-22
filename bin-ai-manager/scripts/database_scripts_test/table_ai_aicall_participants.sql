create table ai_aicall_participants (
  ai_id      binary(16) not null,
  aicall_id  binary(16) not null,
  tm_create  datetime(6),
  primary key (ai_id, aicall_id)
);

create index idx_ai_aicall_participants_aicall_id on ai_aicall_participants(aicall_id);
