CREATE TABLE `cloudyBans_list` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `kind` varchar(16) NOT NULL,
  `player` varchar(16) NOT NULL,
  `uuid` char(36) DEFAULT NULL,
  `moderator` varchar(16) NOT NULL,
  `ts` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `comment` varchar(128) NOT NULL,
  `expires` timestamp NULL DEFAULT NULL,
  `ip` varchar(16) NOT NULL,
  `pardon_moderator` varchar(16) DEFAULT NULL,
  `pardon_comment` varchar(128) DEFAULT NULL,
  `pardon_ts` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `cloudyBans_logins` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `player` varchar(16) NOT NULL,
  `ip` varchar(16) NOT NULL,
  `ts` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `end` timestamp NULL DEFAULT NULL,
  `uuid` varchar(36) DEFAULT NULL,
  `Premium` tinyint(1) DEFAULT NULL,
  `AuthProperties` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `uuid` (`uuid`),
  KEY `player` (`player`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
