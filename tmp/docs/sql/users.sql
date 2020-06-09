CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL,
  `username` varchar(50) NOT NULL,
  `password` varchar(50) NOT NULL,
  PRIMARY KEY (`id`)
);

INSERT INTO `users` (`id`, `username`, `password`) VALUES ('1', 'test', 'test123');
