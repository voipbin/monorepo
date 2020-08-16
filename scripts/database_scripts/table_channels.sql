create table channels(
  -- identity
  id          varchar(255), -- channel id
  asterisk_id varchar(255), -- Asterisk id
  name        varchar(255), -- channel name
  tech        varchar(255), -- channel driver. pjsip, snoop, sip, ...
  transport   varchar(255), -- transport type. udp, tcp, tls, wss, ...

  -- src/dst
  src_name    varchar(255), -- source name
  src_number  varchar(255), -- source number
  dst_name    varchar(255), -- destination name
  dst_number  varchar(255), -- destination number

  -- info
  state     varchar(255), -- current state.
  data      json,         -- additional data. sip headers, and so on...
  stasis    varchar(255), -- stasis application name.
  bridge_id varchar(255), -- bridge id

  dial_result       varchar(255), -- dial result. answer, busy, cancel, ...
  hangup_cause      int,          -- hangup cause code.

  direction varchar(255), -- channel's direction. incoming, outgoing

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --

  tm_answer datetime(6),  -- answer timestamp
  tm_ringing datetime(6), -- rining timestamp
  tm_end datetime(6),     -- end timestamp

  primary key(id)
);

create index idx_channels_create on channels(tm_create);
create index idx_channels_src_number on channels(src_number);
create index idx_channels_dst_number on channels(dst_number);
