# embroidery structure

# assets and goods records assumes all items are owned by a normal user
# to simplify the database scheme. However, the store might own some items.
# This is resolved at application level.
# For this, the store itself has a client (client_id: "store")
# That "leaks" data to other users. Special care should be taken not to destroy
# or alter information on it while operating on data for other users.

CREATE TABLE `address` (
  `address_id` char(36) NOT NULL,
  `client_id` char(36) NOT NULL DEFAULT '',
  `name` varchar(50) NOT NULL DEFAULT '',
  `address_line1` varchar(100) NOT NULL DEFAULT '',
  `address_line2` varchar(100) NOT NULL DEFAULT '',
  `city` varchar(50) NOT NULL DEFAULT '',
  `state` varchar(50) NOT NULL DEFAULT '',
  `country` varchar(50) NOT NULL DEFAULT '',
  `zip_code` int(11) NOT NULL,
  `phone` varchar(30) NOT NULL DEFAULT '',
  `status` enum('ACTIVE','REMOVED') NOT NULL DEFAULT 'ACTIVE',
  PRIMARY KEY (`address_id`),
  KEY `client_id` (`client_id`),
  CONSTRAINT `address_fk_clients_client_id` FOREIGN KEY (`address_id`) REFERENCES `clients` (`client_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `asset` (
  `asset_id` char(36) NOT NULL,
  `client_id` char(36) NOT NULL DEFAULT '',
  `filepath` varchar(100) NOT NULL DEFAULT '',
  `status` enum('ACTIVE','REMOVED') NOT NULL,
  `original_filepath` varchar(100) NOT NULL DEFAULT '',
  `received_date` datetime NOT NULL,
  PRIMARY KEY (`asset_id`),
  KEY `client_id` (`client_id`),
  KEY `filepath` (`filepath`),
  KEY `status` (`status`),
  CONSTRAINT `asset_fk_clients_client_id` FOREIGN KEY (`client_id`) REFERENCES `clients` (`client_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `authentication` (
  `employee_id` char(36) NOT NULL,
  `email` varchar(254) NOT NULL DEFAULT '',
  `password` char(32) NOT NULL DEFAULT '',
  `access_level` enum('OWNER','EMPLOYEE') NOT NULL,
  PRIMARY KEY (`employee_id`),
  KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `clients` (
  `client_id` char(36) NOT NULL,
  `first_name` varchar(60) NOT NULL DEFAULT '',
  `last_name` varchar(60) NOT NULL DEFAULT '',
  `email` varchar(254) NOT NULL DEFAULT '',
  `status` enum('ACTIVE','ARCHIVED') NOT NULL,
  PRIMARY KEY (`client_id`),
  KEY `name` (`first_name`,`last_name`),
  KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `goodsRecords` (
  `good_id` char(36) NOT NULL,
  `job_id` char(36) NOT NULL DEFAULT '',
  `employee_id` char(36) NOT NULL DEFAULT '',
  `owner_id` char(36) NOT NULL DEFAULT '',
  `type` enum('TOWEL','LINE','SHIRT','UNIFORM','OTHER') NOT NULL,
  `amount` bigint(20) NOT NULL,
  `unit` enum('MM','SQUARE_CM','ML','UNITS') NOT NULL,
  `notes` text NOT NULL,
  `date` datetime NOT NULL,
  `status` enum('ACQUIRED','IN_STOCK','IN_USE','MISSING','DECOMMISSIONED') NOT NULL,
  PRIMARY KEY (`good_id`),
  KEY `job_id` (`job_id`),
  KEY `employee_id` (`employee_id`),
  KEY `owner_id` (`owner_id`),
  KEY `type` (`type`),
  CONSTRAINT `goodsRecords_fk_authentication_employee_id` FOREIGN KEY (`employee_id`) REFERENCES `authentication` (`employee_id`),
  CONSTRAINT `goodsRecords_fk_clients_client_id` FOREIGN KEY (`owner_id`) REFERENCES `clients` (`client_id`),
  CONSTRAINT `goodsRecords_fk_jobs_job_id` FOREIGN KEY (`job_id`) REFERENCES `job` (`job_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `job` (
  `job_id` char(36) NOT NULL,
  `order_id` char(36) NOT NULL DEFAULT '',
  `client_id` char(36) NOT NULL DEFAULT '',
  `asset_id` char(36) NOT NULL DEFAULT '',
  `status` enum('CREATED','QUEUE','IN_PROGRESS','CANCELED','DONE') NOT NULL,
  `type` enum('M1','M2') NOT NULL,
  `amount` int(11) NOT NULL,
  `price` bigint(20) NOT NULL,
  `start_time` datetime NOT NULL,
  `end_time` datetime NOT NULL,
  `complexity` bigint(20) NOT NULL,
  PRIMARY KEY (`job_id`),
  KEY `order_id` (`order_id`),
  KEY `client_id` (`client_id`),
  KEY `asset_id` (`asset_id`),
  KEY `status` (`status`),
  CONSTRAINT `job_fk_asset_assets_id` FOREIGN KEY (`asset_id`) REFERENCES `asset` (`asset_id`),
  CONSTRAINT `job_fk_clients_client_id` FOREIGN KEY (`client_id`) REFERENCES `clients` (`client_id`),
  CONSTRAINT `job_fk_order_order_id` FOREIGN KEY (`order_id`) REFERENCES `order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `order` (
  `order_id` char(36) NOT NULL,
  `client_id` char(36) NOT NULL DEFAULT '',
  `client_address_id` char(36) NOT NULL DEFAULT '',
  `open_time` datetime NOT NULL,
  `close_time` datetime NOT NULL,
  `status` enum('OPEN','WAITING_FOR_PAYMENT','STAND_BY','QUEUE','IN_PROGRESS','CANCELED','DONE') NOT NULL,
  `price_total` bigint(20) NOT NULL,
  PRIMARY KEY (`order_id`),
  KEY `client_id` (`client_id`),
  KEY `client_addres_id` (`client_address_id`),
  KEY `open_time` (`open_time`),
  CONSTRAINT `order_fk_address_address_id` FOREIGN KEY (`client_address_id`) REFERENCES `address` (`address_id`),
  CONSTRAINT `order_fk_clients_client_id` FOREIGN KEY (`client_id`) REFERENCES `clients` (`client_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `orderPayment` (
  `payment_id` char(36) NOT NULL,
  `client_id` char(36) NOT NULL DEFAULT '',
  `order_id` char(36) NOT NULL DEFAULT '',
  `price_total` bigint(20) NOT NULL,
  `provider` enum('CASH_FLOW','CREDIT_CARD','DEBIT_CARD','MONEY_TRANSFER') NOT NULL,
  `time` datetime NOT NULL,
  PRIMARY KEY (`payment_id`),
  KEY `client_id` (`client_id`),
  KEY `order_id` (`order_id`),
  CONSTRAINT `orderPayment_fk_clients_client_id` FOREIGN KEY (`client_id`) REFERENCES `clients` (`client_id`),
  CONSTRAINT `orderPayment_fk_order_order_id` FOREIGN KEY (`order_id`) REFERENCES `order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
