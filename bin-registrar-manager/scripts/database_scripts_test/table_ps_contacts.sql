CREATE TABLE ps_contacts (
  id varchar(255) NOT NULL,
  uri varchar(511) DEFAULT NULL,
  expiration_time bigint(20) DEFAULT NULL,
  qualify_frequency int(10) DEFAULT NULL,
  outbound_proxy varchar(40) DEFAULT NULL,
  path text DEFAULT NULL,
  user_agent varchar(255) DEFAULT NULL,
  qualify_timeout float DEFAULT NULL,
  reg_server varchar(255) DEFAULT NULL,
  authenticate_qualify check (authenticate_qualify in ('yes','no')) DEFAULT NULL,
  via_addr varchar(40) DEFAULT NULL,
  via_port int(10) DEFAULT NULL,
  call_id varchar(255) DEFAULT NULL,
  endpoint varchar(40) DEFAULT NULL,
  prune_on_boot check (prune_on_boot in ('yes','no')) DEFAULT NULL,
  primary key(id)
);

create index idx_ps_contacts_endpoint on ps_contacts(endpoint);
