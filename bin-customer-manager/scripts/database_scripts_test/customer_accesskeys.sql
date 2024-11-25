create table customer_accesskeys (
    id          binary(16),
    customer_id binary(16),

    name varchar(255),
    detail text,

    token varchar(1023),

    tm_expire datetime(6), -- Expiry timestamp

    tm_create datetime(6), -- Created timestamp
    tm_update datetime(6), -- Updated timestamp
    tm_delete datetime(6), -- Deleted timestamp

    primary key(id)
);

create index idx_customer_accesskeys_customer_id on customer_accesskeys(customer_id);
create index idx_customer_accesskeys_token on customer_accesskeys(token);
