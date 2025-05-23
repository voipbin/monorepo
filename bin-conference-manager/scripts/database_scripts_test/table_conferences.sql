create table conference_conferences(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id
  type          varchar(255), -- type
  flow_id       binary(16),   -- flow id
  confbridge_id binary(16),   -- confbridge id

  -- info
  status  varchar(255),   -- status
  name    varchar(255),   -- conference's name
  detail  text,           -- conference's detail description
  data    json,           -- additional data
  timeout int,            -- timeout. second

  pre_flow_id  binary(16),   -- pre flow id
  post_flow_id binary(16),   -- post flow id

  conferencecall_ids json, -- conferencecalls in the conference

  -- record info
  recording_id   binary(16),  -- current record id
  recording_ids  json,        -- record ids

  transcribe_id  binary(16),  -- current transcribe id
  transcribe_ids json,        -- transcribe ids

  -- timestamps
  tm_end    datetime(6),  --
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_conference_conferences_create on conference_conferences(tm_create);
create index idx_conference_conferences_customer_id on conference_conferences(customer_id);
create index idx_conference_conferences_flow_id on conference_conferences(flow_id);
create index idx_conference_conferences_confbridge_id on conference_conferences(confbridge_id);
