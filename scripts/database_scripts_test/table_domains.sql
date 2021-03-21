create table domains(
  -- identity
  id        binary(16), -- id
  user_id   integer,      -- user id

  name    varchar(255),
  detail  varchar(255),

  domain_name varchar(255),

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_domains_user_id on domains(user_id);
create index idx_domains_domain_name on domains(domain_name);
