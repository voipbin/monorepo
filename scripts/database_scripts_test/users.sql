create table users(
  -- identity
  id            integer primary key,  -- id
  
  username      varchar(255), -- username
  password_hash varchar(255), -- password hash

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6)
);

create index idx_users_username on users(username);
