create table transcribes(
  -- identity
  id            binary(16),   -- id
  user_id       integer,      -- user id
  type          varchar(16),  -- type of transcribe
  reference_id  binary(16),   -- call/conference/recording's id
  host_id       binary(16),   -- host id

  language          varchar(16),    -- BCP47 type's language code. en-US
  webhook_uri       varchar(1023),  -- webhook uri
  webhook_method    varchar(16),    -- webhook method

  transcripts json, -- transcripts

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_transcribles_reference_id on channels(reference_id);
create index idx_transcribles_user_id on channels(user_id);
