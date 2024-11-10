create table transcribe_transcribes(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id
  type          varchar(16),  -- type of transcribe

  reference_type  varchar(16),
  reference_id    binary(16),   -- call/conference/recording's id

  status        varchar(16),
  host_id       binary(16),   -- host id
  language      varchar(16),  -- BCP47 type's language code. en-US
  direction     varchar(255),

  streaming_ids json,

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_transcribe_transcribes_reference_id on transcribe_transcribes(reference_id);
create index idx_transcribe_transcribes_customerid on transcribe_transcribes(customer_id);
