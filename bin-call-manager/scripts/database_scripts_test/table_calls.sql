create table calls(
  -- identity
  id                binary(16),   -- id
  customer_id       binary(16),   -- customer id

  channel_id        varchar(255), -- channel id
  bridge_id         varchar(255), -- call bridge id
  flow_id           binary(16),   -- flow id
  active_flow_id    binary(16),   -- active_flow id
  confbridge_id     binary(16),   -- currently joined confbridge id
  type              varchar(16),  -- type of call

  -- etc info
  master_call_id    binary(16),   -- master call id
  chained_call_ids  json,         -- chained call ids

  -- recording info
  recording_id  binary(16),     -- current recording id
  recording_ids json,           -- recording ids

  -- external media info
  external_media_id binary(16), -- external media id

  groupcall_id binary(16),

  -- source/destination
  source        json, -- source's type, target, number, name, ...
  source_target varchar(1024),

  destination         json, -- destination's type, target, number, name, ...
  destination_target  varchar(1024),

  -- info
  status            varchar(255),   -- current status of call.
  data              json,           -- additional data. sip headers, and so on...
  action            json,           -- current action
  action_next_hold  boolean,        -- hold the action next
  direction         varchar(16),    -- direction of call. incoming/outgoing
  mute_direction    varchar(16),    -- mute direction

  hangup_by         varchar(16),    -- local/remote/empty for not sure.
  hangup_reason     varchar(16),    -- reason

  -- dialroute
  dialroute_id  binary(16),
  dialroutes    json,

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  tm_progressing  datetime(6), -- progrssing timestamp
  tm_ringing      datetime(6), -- rining timestamp
  tm_hangup       datetime(6), -- hangup timestamp

  primary key(id)
);

create index idx_calls_channelid on calls(channel_id);
create index idx_calls_flowid on calls(flow_id);
create index idx_calls_create on calls(tm_create);
create index idx_calls_hangup on calls(tm_hangup);
create index idx_calls_source_target on calls(source_target);
create index idx_calls_destination_target on calls(destination_target);
create index idx_calls_customer_id on calls(customer_id);
create index idx_calls_external_media_id on calls(external_media_id);
create index idx_calls_groupcall_id on calls(groupcall_id);
