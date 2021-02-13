CREATE TABLE `ps_auths` (
  `id` varchar(40) NOT NULL,
--   `auth_type` enum('md5','userpass','google_oauth') DEFAULT NULL,
  `auth_type` check(auth_type in('md5','userpass','google_oauth')) DEFAULT NULL,
  `nonce_lifetime` int(11) DEFAULT NULL,
  `md5_cred` varchar(40) DEFAULT NULL,
  `password` varchar(80) DEFAULT NULL,
  `realm` varchar(40) DEFAULT NULL,
  `username` varchar(40) DEFAULT NULL,
  `refresh_token` varchar(255) DEFAULT NULL,
  `oauth_clientid` varchar(255) DEFAULT NULL,
  `oauth_secret` varchar(255) DEFAULT NULL,
  primary key (`id`)
)
