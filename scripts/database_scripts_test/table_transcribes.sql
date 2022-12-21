create table transcribes(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id
  type          varchar(16),  -- type of transcribe
  reference_id  binary(16),   -- call/conference/recording's id

  host_id       binary(16),   -- host id
  language      varchar(16),  -- BCP47 type's language code. en-US
  direction     varchar(255),

  transcripts json, -- transcripts

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_transcribles_reference_id on transcribes(reference_id);
create index idx_transcribles_customerid on transcribes(customer_id);
