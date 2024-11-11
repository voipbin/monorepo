create table transcribe_transcripts(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id
  transcribe_id binary(16),

  direction varchar(16),
  message   text,

  -- timestamps
  tm_transcript datetime(6),

  tm_create     datetime(6),  --
  tm_delete     datatime(6),  --

  primary key(id)
);

create index idx_transcribe_transcripts_customer_id on transcribe_transcripts(customer_id);
create index idx_transcribe_transcripts_transcribe_id on transcribe_transcripts(transcribe_id);
