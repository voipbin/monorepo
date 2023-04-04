create table channels(
  -- identity
  id          varchar(255), -- channel id
  asterisk_id varchar(255), -- Asterisk id
  name        varchar(255), -- channel name
  type        varchar(255), -- channel's type. this context sets by call/conference handler.
  tech        varchar(255), -- channel driver. pjsip, snoop, sip, ...

  -- sip info
  sip_call_id     varchar(255), -- sip call id.
  sip_transport   varchar(255), -- sip transport type. udp, tcp, tls, wss, ...

  -- src/dst
  src_name    varchar(255), -- source name
  src_number  varchar(255), -- source number
  dst_name    varchar(255), -- destination name
  dst_number  varchar(255), -- destination number

  -- info
  state         varchar(255), -- current state.
  data          json,         -- additional data. sip headers, and so on...
  stasis_name   varchar(255), -- stasis application name.
  stasis_data   json,         -- stasis application data.
  bridge_id     varchar(255), -- bridge id
  playback_id   varchar(255), -- playback id

  dial_result       varchar(255), -- dial result. answer, busy, cancel, ...
  hangup_cause      int,          -- hangup cause code.

  direction       varchar(255),   -- channel's direction. incoming, outgoing
  mute_direction  varchar(16),    --

  -- timestamps
  tm_create datetime(6),  -- created timestamp
  tm_update datetime(6),  -- last updated timestamp
  tm_delete datetime(6),  -- destroyed timestamp

  tm_answer datetime(6),  -- answer timestamp
  tm_ringing datetime(6), -- rining timestamp
  tm_end datetime(6),     -- end timestamp

  primary key(id)
);

create index idx_channels_create on channels(tm_create);
create index idx_channels_src_number on channels(src_number);
create index idx_channels_dst_number on channels(dst_number);
create index idx_channels_sip_call_id on channels(sip_call_id);
