create table registrar_customer_domains (
  customer_id   binary(16), -- owner's id
  domain_label  varchar(64),
  realm         varchar(255),

  -- timestamps (no tm_delete: hard delete on customer_deleted)
  tm_create datetime(6),
  tm_update datetime(6),

  primary key(customer_id)
);

create unique index ux_registrar_customer_domains_domain_label on registrar_customer_domains(domain_label);
create unique index ux_registrar_customer_domains_realm on registrar_customer_domains(realm);
