create table calls(
  -- identity
  id            binary(16),   -- id
  asterisk_id   varchar(255), -- Asterisk id
  channel_id    varchar(255), -- chcallsannel id

  -- source/destination
  source json, -- source's type, target, number, name, ...
  source_target varchar(1024) generated always as (source->"$.target") stored,

  destination json, -- destination's type, target, number, name, ...
  destination_target varchar(1024) generated always as (destination->"$.target") stored,

  -- info
  status            varchar(255), -- current status of call.
  data              json,         -- additional data. sip headers, and so on...
  direction         varchar(16),  -- direction of call. incoming/outgoing
  service           varchar(16),  -- service
  ended_by          varchar(16),  -- local/remote/empty for not sure.

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --

  tm_answer datetime(6),  -- answer timestamp
  tm_ringing datetime(6), -- rining timestamp
  tm_end datetime(6),     -- end timestamp

  primary key(id)
);

create index idx_calls_create on calls(tm_create);
create index idx_calls_end on calls(tm_end);
create index idx_calls_source_target on calls(source_target);
create index idx_calls_destination_target on calls(destination_target);
