CREATE TABLE ps_aors (
  id varchar(40) NOT NULL,
  contact varchar(255) DEFAULT NULL,
  default_expiration int(11) DEFAULT NULL,
  mailboxes varchar(80) DEFAULT NULL,
  max_contacts int(11) DEFAULT NULL,
  minimum_expiration int(11) DEFAULT NULL,
  remove_existing check (remove_existing in('yes','no')) DEFAULT NULL,
  qualify_frequency int(11) DEFAULT NULL,
  authenticate_qualify check (authenticate_qualify in ('yes','no')) DEFAULT NULL,
  maximum_expiration int(11) DEFAULT NULL,
  outbound_proxy varchar(40) DEFAULT NULL,
  support_path check (support_path in ('yes','no')) DEFAULT NULL,
  qualify_timeout float DEFAULT NULL,
  voicemail_extension varchar(40) DEFAULT NULL,
  primary key(id)
);
