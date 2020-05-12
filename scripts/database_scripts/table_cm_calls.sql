create table cm_calls(
  -- identity
  id              binary(16),   -- id
  asterisk_id     varchar(255), -- Asterisk id
  channel_id      varchar(255), -- channel id
  flow_id         binary(16),   -- flow id
  conference_id   binary(16),   -- currently joined conference id
  type            varchar(16),  -- type of call

  -- source/destination
  source        json, -- source's type, target, number, name, ...
  source_target varchar(1024),

  destination         json, -- destination's type, target, number, name, ...
  destination_target  varchar(1024),

  -- info
  status            varchar(255), -- current status of call.
  data              json,         -- additional data. sip headers, and so on...
  action            json,         -- current action
  direction         varchar(16),  -- direction of call. incoming/outgoing
  hangup_by         varchar(16),  -- local/remote/empty for not sure.
  hangup_reason     varchar(16),  -- reason

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --

  tm_progressing  datetime(6), -- progrssing timestamp
  tm_ringing      datetime(6), -- rining timestamp
  tm_hangup       datetime(6), -- end timestamp

  primary key(id)
);

create index idx_cm_calls_flowid on cm_calls(flow_id);
create index idx_cm_calls_create on cm_calls(tm_create);
create index idx_cm_calls_hangup on cm_calls(tm_hangup);
create index idx_cm_calls_source_target on cm_calls(source_target);
create index idx_cm_calls_destination_target on cm_calls(destination_target);
