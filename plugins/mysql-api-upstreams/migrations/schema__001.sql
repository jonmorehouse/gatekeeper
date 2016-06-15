-- upstream table
CREATE TABLE IF NOT EXISTS `upstream` (
	`id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'unique upstream id',
  	`name` varchar(400) COLLATE utf8_unicode_ci NOT NULL COMMENT 'consumer name',
	`created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'utc date the upstream was created',
	`timeout` varchar(100) NOT NULL COMMENT 'string formatted timeout',

  	PRIMARY KEY (`id`),
	UNIQUE KEY `idx_name` (`name`) COMMENT 'unique index ensuring that each upstream name is unique'
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `upstream` AUTO_INCREMENT = 1000;

-- upstream_mapping table
CREATE TABLE IF NOT EXISTS `upstream_mapping` (
	`id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'unique primary key',
	`upstream_id` int(11) NOT NULL COMMENT 'parent upstream ID',
	`mapping` varchar(400) COLLATE utf8_unicode_ci NOT NULL COMMENT 'hostname or prefix',
	`mapping_type` ENUM('prefix', 'hostname', 'protocol'),

	PRIMARY KEY (`id`),
	UNIQUE KEY `idx_mapping` (`mapping`, `mapping_type`) COMMENT 'composite index ensuring that no mapping/mapping-type combination exists'
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `upstream_mapping` AUTO_INCREMENT = 1000;

-- backend table
CREATE TABLE IF NOT EXISTS `backend` (	
	`id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'unique backend id', 
	`upstream_id` int(11) NOT NULL COMMENT 'parent upstream id',
	`healthcheck` varchar(400) COLLATE utf8_unicode_ci NOT NULL COMMENT 'healthcheck',
        `address` varchar(400) COLLATE utf8_unicode_ci NOT NULL COMMENT 'address',
	`created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'utc date the upstream was created',
        
        PRIMARY KEY (`id`),
	UNIQUE KEY `idx_upstream_backend` (`address`, `upstream_id`) COMMENT 'unique index ensuring uniqueness for backends within an upstream'
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `backend` AUTO_INCREMENT = 1000;
