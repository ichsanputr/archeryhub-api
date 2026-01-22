-- --------------------------------------------------------
-- Host:                         151.243.222.93
-- Server version:               10.5.29-MariaDB-0+deb11u1 - Debian 11
-- Server OS:                    debian-linux-gnu
-- HeidiSQL Version:             12.8.0.6908
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;


-- Dumping database structure for archeryhub
CREATE DATABASE IF NOT EXISTS `archeryhub` /*!40100 DEFAULT CHARACTER SET latin1 COLLATE latin1_swedish_ci */;
USE `archeryhub`;

-- Dumping structure for table archeryhub.activity_logs
CREATE TABLE IF NOT EXISTS `activity_logs` (
  `id` varchar(36) NOT NULL,
  `user_id` varchar(36) DEFAULT NULL,
  `event_id` varchar(36) DEFAULT NULL,
  `action` varchar(50) NOT NULL,
  `entity_type` varchar(50) DEFAULT NULL,
  `entity_id` varchar(36) DEFAULT NULL,
  `description` text DEFAULT NULL,
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` text DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_event` (`event_id`),
  KEY `idx_action` (`action`),
  KEY `idx_created` (`created_at`),
  CONSTRAINT `activity_logs_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `archers` (`uuid`) ON DELETE SET NULL,
  CONSTRAINT `activity_logs_ibfk_2` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.activity_logs: ~13 rows (approximately)
INSERT INTO `activity_logs` (`id`, `user_id`, `event_id`, `action`, `entity_type`, `entity_id`, `description`, `ip_address`, `user_agent`, `created_at`) VALUES
	('43604df6-be63-4d9d-9582-68ff5244c44d', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', NULL, 'user_registered', 'archer', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', 'User registered via Google: iniasya1@gmail.com', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 05:43:26'),
	('5cb72b52-ea05-4b53-86d4-fc318198813b', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', NULL, 'user_logged_in', 'archer', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', 'User logged in via Google', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 05:43:26'),
	('62e47146-36fb-49a6-a1be-14742e72abcd', '8d22cac0-cee6-42db-888f-b634c8e921b5', NULL, 'user_logged_in', 'archer', '8d22cac0-cee6-42db-888f-b634c8e921b5', 'User logged in via Google', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 05:49:11'),
	('79165a39-eff2-44da-b33f-8567bc19e53e', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', NULL, 'user_logged_in', 'archer', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', 'User logged in: iniasya1', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36', '2026-01-22 02:42:35'),
	('7ec79d29-5854-4f40-bdbf-15ff792b1660', NULL, NULL, 'user_logged_in', 'archer', '835f1598-4551-425d-b82f-af097ce0f56c', 'User logged in via Google', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 04:16:00'),
	('7fcd45c9-474c-41fc-80f6-4b959918fb7e', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', NULL, 'user_logged_in', 'archer', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', 'User logged in via Google', '140.213.190.104', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36', '2026-01-21 10:05:56'),
	('a2c3e20b-cdb3-46d4-b177-3a3bd4817c9a', NULL, NULL, 'user_logged_in', 'archer', '835f1598-4551-425d-b82f-af097ce0f56c', 'User logged in via Google', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 04:14:49'),
	('a515591c-968c-4f92-9882-a20f7061c919', '8d22cac0-cee6-42db-888f-b634c8e921b5', NULL, 'user_logged_in', 'archer', '8d22cac0-cee6-42db-888f-b634c8e921b5', 'User logged in via Google', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 06:59:18'),
	('bc0acec6-a497-412f-a0c1-a4df6ff5ee5d', NULL, NULL, 'user_registered', 'archer', '835f1598-4551-425d-b82f-af097ce0f56c', 'User registered via Google: ichsanfadhil67@gmail.com', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 04:14:49'),
	('ce47853a-390f-4a3c-ae55-d7a8069dde8e', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', NULL, 'user_logged_in', 'archer', '0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', 'User logged in: iniasya1', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36', '2026-01-22 01:47:52'),
	('d5067f2d-87bd-4f24-a053-130278bccf95', NULL, NULL, 'user_logged_in', 'archer', '3aea01f4-7795-4413-b9e1-bf497b5c822a', 'User logged in via Google', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 04:18:51'),
	('dfb9075f-c7be-4b3b-accb-56d31f1b53cf', NULL, NULL, 'user_registered', 'archer', '3aea01f4-7795-4413-b9e1-bf497b5c822a', 'User registered via Google: ngekode24@gmail.com', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 04:18:51'),
	('e38f5b37-a4f7-4809-a884-8516b664e864', '8d22cac0-cee6-42db-888f-b634c8e921b5', NULL, 'user_registered', 'archer', '8d22cac0-cee6-42db-888f-b634c8e921b5', 'User registered via Google: rekusameno@gmail.com', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 05:42:29'),
	('ed1b8b8c-91a2-4932-a139-cbff9c301a68', '8d22cac0-cee6-42db-888f-b634c8e921b5', NULL, 'user_logged_in', 'archer', '8d22cac0-cee6-42db-888f-b634c8e921b5', 'User logged in via Google', '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36', '2026-01-21 05:42:29');

-- Dumping structure for table archeryhub.archers
CREATE TABLE IF NOT EXISTS `archers` (
  `uuid` varchar(36) NOT NULL,
  `username` varchar(50) DEFAULT NULL,
  `slug` varchar(100) DEFAULT NULL,
  `email` varchar(100) DEFAULT NULL,
  `athlete_code` varchar(20) DEFAULT NULL,
  `google_id` varchar(100) DEFAULT NULL,
  `avatar_url` varchar(255) DEFAULT NULL,
  `bio` text DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `role` enum('archer','admin','organization','club') DEFAULT 'archer',
  `user_id` varchar(36) DEFAULT NULL,
  `full_name` varchar(255) NOT NULL,
  `nickname` varchar(100) DEFAULT NULL,
  `date_of_birth` date DEFAULT NULL,
  `gender` enum('male','female') DEFAULT 'male',
  `country` varchar(3) DEFAULT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `address` mediumtext DEFAULT NULL,
  `city` varchar(100) DEFAULT NULL,
  `province` varchar(100) DEFAULT NULL,
  `postal_code` varchar(10) DEFAULT NULL,
  `national_id` varchar(50) DEFAULT NULL,
  `bow_type` enum('recurve','compound','barebow','traditional') DEFAULT 'recurve',
  `dominant_hand` enum('left','right') DEFAULT 'right',
  `experience_years` int(11) DEFAULT 0,
  `club_id` varchar(36) DEFAULT NULL,
  `current_ranking` int(11) DEFAULT NULL,
  `best_score` int(11) DEFAULT NULL,
  `emergency_contact_name` varchar(255) DEFAULT NULL,
  `emergency_contact_phone` varchar(20) DEFAULT NULL,
  `medical_conditions` mediumtext DEFAULT NULL,
  `achievements` mediumtext DEFAULT NULL,
  `status` enum('active','inactive','suspended') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_archers_email` (`email`),
  UNIQUE KEY `idx_archers_username` (`username`),
  UNIQUE KEY `athlete_code` (`athlete_code`),
  UNIQUE KEY `idx_archers_slug` (`slug`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_club_id` (`club_id`),
  KEY `idx_bow_type` (`bow_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.archers: ~2 rows (approximately)
INSERT INTO `archers` (`uuid`, `username`, `slug`, `email`, `athlete_code`, `google_id`, `avatar_url`, `bio`, `password`, `role`, `user_id`, `full_name`, `nickname`, `date_of_birth`, `gender`, `country`, `phone`, `address`, `city`, `province`, `postal_code`, `national_id`, `bow_type`, `dominant_hand`, `experience_years`, `club_id`, `current_ranking`, `best_score`, `emergency_contact_name`, `emergency_contact_phone`, `medical_conditions`, `achievements`, `status`, `created_at`, `updated_at`) VALUES
	('0da70d11-8fb7-4bf4-8ae4-97caadfb6ed2', 'iniasya1', 'muhammad-ichsanul-fadhil', 'iniasya1@gmail.com', NULL, '110028875946359193685', 'https://lh3.googleusercontent.com/a/ACg8ocJVsmM5eGXrPB2h8KdD2EdmoEIdnWPzj-BXz-Lxk_vXba255A=s96-c', NULL, '12345', 'archer', NULL, 'Muhammad Ichsanul Fadhil', NULL, NULL, 'male', NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'recurve', 'right', 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'active', '2026-01-21 05:43:26', '2026-01-22 01:55:38'),
	('8d22cac0-cee6-42db-888f-b634c8e921b5', 'rekusameno', 'reku-sameno', 'rekusameno@gmail.com', NULL, '107117052357653799197', 'https://lh3.googleusercontent.com/a/ACg8ocJ_9YGUmQr5Y_tAuSYGo5lFPrXvsm7iLf-JpsWm3WJdEc4MC8o=s96-c', NULL, '12345', 'archer', NULL, 'reku sameno', NULL, NULL, 'male', NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'recurve', 'right', 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'active', '2026-01-21 05:42:29', '2026-01-22 01:55:38');

-- Dumping structure for table archeryhub.archer_profile
CREATE TABLE IF NOT EXISTS `archer_profile` (
  `id` varchar(36) NOT NULL,
  `archer_id` varchar(36) NOT NULL,
  `section_type` varchar(50) NOT NULL COMMENT 'Type of section: bio, achievements, stats, gallery, contact, social, etc.',
  `section_order` int(11) DEFAULT 0 COMMENT 'Order of display on profile page',
  `title` varchar(255) DEFAULT NULL COMMENT 'Optional section title',
  `content` text DEFAULT NULL COMMENT 'Section content (JSON or text depending on section_type)',
  `is_visible` tinyint(1) DEFAULT 1 COMMENT 'Whether section is visible on public profile',
  `metadata` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Additional section metadata (e.g., gallery images, social links)' CHECK (json_valid(`metadata`)),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_archer_profile_archer` (`archer_id`),
  KEY `idx_archer_profile_order` (`archer_id`,`section_order`),
  KEY `idx_archer_profile_type` (`section_type`),
  CONSTRAINT `fk_archer_profile_archer` FOREIGN KEY (`archer_id`) REFERENCES `archers` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.archer_profile: ~0 rows (approximately)

-- Dumping structure for table archeryhub.assignment_history
CREATE TABLE IF NOT EXISTS `assignment_history` (
  `uuid` char(36) NOT NULL,
  `event_uuid` char(36) NOT NULL,
  `entity_type` varchar(50) NOT NULL COMMENT 'archer, match, target',
  `entity_uuid` char(36) NOT NULL,
  `previous_assignment` text DEFAULT NULL,
  `new_assignment` text DEFAULT NULL,
  `changed_by` char(36) DEFAULT NULL COMMENT 'user_uuid who made the change',
  `reason` varchar(255) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_ah_event` (`event_uuid`),
  KEY `idx_ah_entity` (`entity_type`,`entity_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.assignment_history: ~0 rows (approximately)

-- Dumping structure for table archeryhub.cart_items
CREATE TABLE IF NOT EXISTS `cart_items` (
  `uuid` varchar(36) NOT NULL,
  `user_id` varchar(36) NOT NULL,
  `product_id` varchar(36) NOT NULL,
  `quantity` int(11) NOT NULL DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_user_product` (`user_id`,`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.cart_items: ~0 rows (approximately)

-- Dumping structure for table archeryhub.clubs
CREATE TABLE IF NOT EXISTS `clubs` (
  `uuid` varchar(36) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `username` varchar(50) DEFAULT NULL,
  `user_id` varchar(36) DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `abbreviation` varchar(20) DEFAULT NULL,
  `description` text DEFAULT NULL,
  `banner_url` varchar(500) DEFAULT NULL,
  `email` varchar(255) NOT NULL,
  `google_id` varchar(100) DEFAULT NULL,
  `avatar_url` varchar(255) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `role` enum('club','admin') DEFAULT 'club',
  `phone` varchar(20) DEFAULT NULL,
  `address` text DEFAULT NULL,
  `city` varchar(100) DEFAULT NULL,
  `province` varchar(100) DEFAULT NULL,
  `postal_code` varchar(10) DEFAULT NULL,
  `latitude` decimal(10,8) DEFAULT NULL,
  `longitude` decimal(11,8) DEFAULT NULL,
  `established_date` date DEFAULT NULL,
  `registration_number` varchar(100) DEFAULT NULL,
  `organization_id` varchar(36) DEFAULT NULL,
  `head_coach_name` varchar(255) DEFAULT NULL,
  `head_coach_phone` varchar(20) DEFAULT NULL,
  `training_schedule` text DEFAULT NULL,
  `facilities` text DEFAULT NULL,
  `membership_fee` decimal(12,2) DEFAULT NULL,
  `website` varchar(255) DEFAULT NULL,
  `social_facebook` varchar(255) DEFAULT NULL,
  `social_instagram` varchar(255) DEFAULT NULL,
  `member_count` int(11) DEFAULT 0,
  `verification_status` enum('pending','verified','rejected') DEFAULT 'pending',
  `status` enum('active','inactive','suspended') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_clubs_email` (`email`),
  UNIQUE KEY `idx_clubs_username` (`username`),
  UNIQUE KEY `slug` (`slug`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_organization_id` (`organization_id`),
  KEY `idx_city` (`city`),
  KEY `idx_verification_status` (`verification_status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.clubs: ~1 rows (approximately)
INSERT INTO `clubs` (`uuid`, `slug`, `username`, `user_id`, `name`, `abbreviation`, `description`, `banner_url`, `email`, `google_id`, `avatar_url`, `password`, `role`, `phone`, `address`, `city`, `province`, `postal_code`, `latitude`, `longitude`, `established_date`, `registration_number`, `organization_id`, `head_coach_name`, `head_coach_phone`, `training_schedule`, `facilities`, `membership_fee`, `website`, `social_facebook`, `social_instagram`, `member_count`, `verification_status`, `status`, `created_at`, `updated_at`) VALUES
	('218e8243-6e03-41b3-a551-65936bd12815', 'akun-ngekode', 'ngekode24', NULL, 'Akun Ngekode', NULL, NULL, NULL, 'ngekode24@gmail.com', '108209379527075657295', 'https://lh3.googleusercontent.com/a/ACg8ocLa2E_ZRpeQvZmfuQtHiVy_7UlKQuSuEAPwXwT7AE2OEyIYIg=s96-c', '123456', 'club', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 0, 'pending', 'active', '2026-01-21 05:42:59', '2026-01-22 01:35:21');

-- Dumping structure for table archeryhub.club_invitations
CREATE TABLE IF NOT EXISTS `club_invitations` (
  `uuid` varchar(36) NOT NULL,
  `club_id` varchar(36) NOT NULL,
  `email` varchar(255) NOT NULL,
  `invited_by` varchar(36) NOT NULL,
  `status` enum('pending','accepted','expired','cancelled') DEFAULT 'pending',
  `token` varchar(100) DEFAULT NULL,
  `message` text DEFAULT NULL,
  `expires_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `accepted_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `token` (`token`),
  KEY `idx_invitations_club` (`club_id`),
  KEY `idx_invitations_email` (`email`),
  KEY `idx_invitations_token` (`token`),
  KEY `idx_invitations_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.club_invitations: ~0 rows (approximately)

-- Dumping structure for table archeryhub.club_members
CREATE TABLE IF NOT EXISTS `club_members` (
  `uuid` varchar(36) NOT NULL,
  `club_id` varchar(36) NOT NULL,
  `archer_id` varchar(36) NOT NULL,
  `status` enum('pending','active','rejected','left') DEFAULT 'pending',
  `role` enum('member','coach','admin') DEFAULT 'member',
  `joined_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_archer_club` (`archer_id`),
  KEY `idx_club_members_club` (`club_id`),
  KEY `idx_club_members_archer` (`archer_id`),
  KEY `idx_club_members_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.club_members: ~0 rows (approximately)

-- Dumping structure for table archeryhub.events
CREATE TABLE IF NOT EXISTS `events` (
  `uuid` varchar(36) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `code` varchar(20) NOT NULL,
  `name` varchar(200) NOT NULL,
  `short_name` varchar(100) DEFAULT NULL,
  `venue` varchar(200) DEFAULT NULL,
  `gmaps_link` varchar(255) DEFAULT NULL,
  `location` varchar(200) DEFAULT NULL,
  `country` varchar(3) DEFAULT NULL,
  `latitude` decimal(10,8) DEFAULT NULL,
  `longitude` decimal(11,8) DEFAULT NULL,
  `start_date` datetime DEFAULT NULL,
  `end_date` datetime DEFAULT NULL,
  `registration_deadline` datetime DEFAULT NULL,
  `description` text DEFAULT NULL,
  `banner_url` varchar(255) DEFAULT NULL,
  `logo_url` varchar(255) DEFAULT NULL,
  `type` varchar(36) DEFAULT NULL,
  `num_distances` tinyint(4) DEFAULT 1,
  `num_sessions` tinyint(4) DEFAULT 1,
  `entry_fee` decimal(10,2) DEFAULT 0.00,
  `max_participants` int(11) DEFAULT NULL,
  `status` enum('draft','published','ongoing','completed','archived') DEFAULT 'draft',
  `organizer_id` varchar(36) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `slug` (`slug`),
  KEY `idx_code` (`code`),
  KEY `idx_status` (`status`),
  KEY `idx_start_date` (`start_date`),
  KEY `idx_organizer` (`organizer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.events: ~1 rows (approximately)
INSERT INTO `events` (`uuid`, `slug`, `code`, `name`, `short_name`, `venue`, `gmaps_link`, `location`, `country`, `latitude`, `longitude`, `start_date`, `end_date`, `registration_deadline`, `description`, `banner_url`, `logo_url`, `type`, `num_distances`, `num_sessions`, `entry_fee`, `max_participants`, `status`, `organizer_id`, `created_at`, `updated_at`) VALUES
	('3f8ea75d-09de-4448-aac2-7a50853e9ff9', 'asjaks', 'ASJMMG', 'asjaks', NULL, 'akjskas', 'ihk', NULL, NULL, NULL, NULL, '2026-01-16 05:20:00', '2026-01-28 06:17:00', '2026-01-06 06:17:00', '', NULL, NULL, '331b52a7-812d-4dde-aeaf-978e79bf293a', 1, NULL, 350000.00, NULL, 'draft', '5fff777a-b9aa-417c-9046-00585f5806b0', '2026-01-20 19:17:24', '2026-01-21 08:43:09');

-- Dumping structure for table archeryhub.event_categories
CREATE TABLE IF NOT EXISTS `event_categories` (
  `uuid` varchar(36) NOT NULL,
  `event_id` varchar(36) NOT NULL,
  `division_uuid` varchar(36) DEFAULT NULL,
  `category_uuid` varchar(36) DEFAULT NULL,
  `max_participants` int(11) DEFAULT NULL,
  `status` enum('active','inactive') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `fk_event_categories_event` (`event_id`),
  CONSTRAINT `fk_event_categories_event` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.event_categories: ~0 rows (approximately)

-- Dumping structure for table archeryhub.event_images
CREATE TABLE IF NOT EXISTS `event_images` (
  `uuid` varchar(36) NOT NULL,
  `event_id` varchar(36) NOT NULL,
  `url` varchar(500) NOT NULL,
  `caption` varchar(255) DEFAULT NULL,
  `alt_text` varchar(255) DEFAULT NULL,
  `display_order` int(11) DEFAULT 0,
  `is_primary` tinyint(1) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_event_images_event` (`event_id`),
  KEY `idx_event_images_order` (`event_id`,`display_order`),
  CONSTRAINT `fk_event_images_event` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.event_images: ~1 rows (approximately)
INSERT INTO `event_images` (`uuid`, `event_id`, `url`, `caption`, `alt_text`, `display_order`, `is_primary`, `created_at`) VALUES
	('f0ff9e65-3d32-4131-935d-176665e66745', '3f8ea75d-09de-4448-aac2-7a50853e9ff9', 'http://localhost:8001/media/asas-1ac50ec4.png', 'asas-1ac50ec4.png', NULL, 0, 1, '2026-01-21 02:17:24');

-- Dumping structure for table archeryhub.event_participants
CREATE TABLE IF NOT EXISTS `event_participants` (
  `uuid` varchar(36) NOT NULL,
  `event_id` varchar(36) NOT NULL,
  `archer_id` varchar(36) NOT NULL,
  `category_id` varchar(36) NOT NULL,
  `payment_amount` decimal(10,2) DEFAULT 0.00,
  `payment_status` enum('pending','paid','failed','refunded') DEFAULT 'pending',
  `accreditation_status` enum('pending','approved','rejected') DEFAULT 'pending',
  `back_number` varchar(10) DEFAULT NULL,
  `target_number` varchar(10) DEFAULT NULL,
  `session` int(11) DEFAULT NULL,
  `registration_date` timestamp NOT NULL DEFAULT current_timestamp(),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `event_id` (`event_id`),
  KEY `event_category_id` (`category_id`),
  KEY `athlete_id` (`archer_id`),
  CONSTRAINT `event_participants_ibfk_1` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`),
  CONSTRAINT `event_participants_ibfk_2` FOREIGN KEY (`category_id`) REFERENCES `event_categories` (`uuid`),
  CONSTRAINT `event_participants_ibfk_3` FOREIGN KEY (`archer_id`) REFERENCES `archers` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.event_participants: ~0 rows (approximately)

-- Dumping structure for table archeryhub.matches
CREATE TABLE IF NOT EXISTS `matches` (
  `uuid` char(36) NOT NULL,
  `event_category_uuid` char(36) NOT NULL,
  `round_name` varchar(50) NOT NULL COMMENT 'e.g., 1/32, 1/16, Quarter-final, Semi-final, Final',
  `match_order` int(11) NOT NULL DEFAULT 1,
  `status` varchar(20) NOT NULL DEFAULT 'scheduled' COMMENT 'scheduled, live, completed, bye',
  `winner_uuid` char(36) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_matches_event_category` (`event_category_uuid`),
  KEY `idx_matches_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.matches: ~0 rows (approximately)

-- Dumping structure for table archeryhub.match_end_scores
CREATE TABLE IF NOT EXISTS `match_end_scores` (
  `uuid` char(36) NOT NULL,
  `match_participant_uuid` char(36) NOT NULL,
  `end_number` int(11) NOT NULL,
  `arrow_scores` varchar(255) DEFAULT NULL COMMENT 'Comma separated values: 10,9,X,...',
  `end_total` int(11) NOT NULL DEFAULT 0,
  `end_10_count` int(11) NOT NULL DEFAULT 0,
  `end_x_count` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uk_mes_end` (`match_participant_uuid`,`end_number`),
  KEY `idx_mes_participant` (`match_participant_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.match_end_scores: ~0 rows (approximately)

-- Dumping structure for table archeryhub.match_participants
CREATE TABLE IF NOT EXISTS `match_participants` (
  `uuid` char(36) NOT NULL,
  `match_uuid` char(36) NOT NULL,
  `archer_uuid` char(36) DEFAULT NULL COMMENT 'NULL for bye or TBD',
  `seed` int(11) DEFAULT NULL,
  `score` int(11) DEFAULT 0,
  `result` varchar(10) DEFAULT NULL COMMENT 'win, loss',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_mp_match` (`match_uuid`),
  KEY `idx_mp_archer` (`archer_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.match_participants: ~0 rows (approximately)

-- Dumping structure for table archeryhub.match_target_assignments
CREATE TABLE IF NOT EXISTS `match_target_assignments` (
  `uuid` char(36) NOT NULL,
  `match_uuid` char(36) NOT NULL,
  `target_number` int(11) NOT NULL,
  `target_position` varchar(5) DEFAULT NULL COMMENT 'A, B, or empty for full match on target',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uk_mta_target` (`match_uuid`,`target_number`,`target_position`),
  KEY `idx_mta_match` (`match_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.match_target_assignments: ~0 rows (approximately)

-- Dumping structure for table archeryhub.news
CREATE TABLE IF NOT EXISTS `news` (
  `uuid` varchar(36) NOT NULL,
  `organization_id` varchar(36) DEFAULT NULL,
  `club_id` varchar(36) DEFAULT NULL,
  `title` varchar(500) NOT NULL,
  `slug` varchar(500) DEFAULT NULL,
  `excerpt` text DEFAULT NULL,
  `content` longtext DEFAULT NULL,
  `image_url` varchar(500) DEFAULT NULL,
  `category` enum('event','pengumuman','prestasi','lainnya') DEFAULT 'pengumuman',
  `status` enum('draft','published') DEFAULT 'draft',
  `views` int(11) DEFAULT 0,
  `author_name` varchar(255) DEFAULT NULL,
  `author_id` varchar(36) DEFAULT NULL,
  `meta_title` varchar(255) DEFAULT NULL,
  `meta_description` text DEFAULT NULL,
  `published_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `slug` (`slug`),
  KEY `idx_news_org` (`organization_id`),
  KEY `idx_news_club` (`club_id`),
  KEY `idx_news_status` (`status`),
  KEY `idx_news_category` (`category`),
  KEY `idx_news_published` (`published_at`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.news: ~0 rows (approximately)

-- Dumping structure for table archeryhub.orders
CREATE TABLE IF NOT EXISTS `orders` (
  `uuid` varchar(36) NOT NULL,
  `seller_id` varchar(36) NOT NULL,
  `buyer_id` varchar(36) NOT NULL,
  `total_amount` decimal(12,2) NOT NULL,
  `status` enum('pending','processing','shipped','done','cancelled') DEFAULT 'pending',
  `payment_status` enum('unpaid','paid','expired','failed') DEFAULT 'unpaid',
  `shipping_address` text DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_seller_id` (`seller_id`),
  KEY `idx_buyer_id` (`buyer_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.orders: ~0 rows (approximately)

-- Dumping structure for table archeryhub.order_items
CREATE TABLE IF NOT EXISTS `order_items` (
  `uuid` varchar(36) NOT NULL,
  `order_id` varchar(36) NOT NULL,
  `product_id` varchar(36) NOT NULL,
  `quantity` int(11) NOT NULL,
  `price` decimal(12,2) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_product_id` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.order_items: ~0 rows (approximately)

-- Dumping structure for table archeryhub.organizations
CREATE TABLE IF NOT EXISTS `organizations` (
  `uuid` varchar(36) NOT NULL,
  `username` varchar(50) DEFAULT NULL,
  `user_id` varchar(36) DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `acronym` varchar(20) DEFAULT NULL,
  `type` enum('federation','association','committee','sponsor','other') DEFAULT 'association',
  `description` text DEFAULT NULL,
  `website` varchar(255) DEFAULT NULL,
  `email` varchar(255) NOT NULL,
  `google_id` varchar(100) DEFAULT NULL,
  `avatar_url` varchar(255) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `role` enum('organization','admin') DEFAULT 'organization',
  `phone` varchar(20) DEFAULT NULL,
  `address` text DEFAULT NULL,
  `city` varchar(100) DEFAULT NULL,
  `province` varchar(100) DEFAULT NULL,
  `postal_code` varchar(10) DEFAULT NULL,
  `country` varchar(100) DEFAULT 'Indonesia',
  `registration_number` varchar(100) DEFAULT NULL,
  `established_date` date DEFAULT NULL,
  `contact_person_name` varchar(255) DEFAULT NULL,
  `contact_person_email` varchar(255) DEFAULT NULL,
  `contact_person_phone` varchar(20) DEFAULT NULL,
  `social_facebook` varchar(255) DEFAULT NULL,
  `social_instagram` varchar(255) DEFAULT NULL,
  `social_twitter` varchar(255) DEFAULT NULL,
  `verification_status` enum('pending','verified','rejected') DEFAULT 'pending',
  `status` enum('active','inactive','suspended') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_orgs_email` (`email`),
  UNIQUE KEY `idx_orgs_username` (`username`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_type` (`type`),
  KEY `idx_verification_status` (`verification_status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.organizations: ~1 rows (approximately)
INSERT INTO `organizations` (`uuid`, `username`, `user_id`, `name`, `acronym`, `type`, `description`, `website`, `email`, `google_id`, `avatar_url`, `password`, `role`, `phone`, `address`, `city`, `province`, `postal_code`, `country`, `registration_number`, `established_date`, `contact_person_name`, `contact_person_email`, `contact_person_phone`, `social_facebook`, `social_instagram`, `social_twitter`, `verification_status`, `status`, `created_at`, `updated_at`) VALUES
	('a1fdd1c4-632a-44d9-9be4-c96461e4530e', 'ichsanfadhil67', NULL, 'Muhammad Ichsan', NULL, 'association', NULL, NULL, 'ichsanfadhil67@gmail.com', '101125385602255730839', 'https://lh3.googleusercontent.com/a/ACg8ocJgdxl4O25bbwOOWJE0qkbdINIPRk1FkLDoc8s5E1IrbNarK06x=s96-c', '12345', 'organization', NULL, NULL, NULL, NULL, NULL, 'Indonesia', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'pending', 'active', '2026-01-21 04:26:33', '2026-01-22 03:03:01');

-- Dumping structure for table archeryhub.payment_transactions
CREATE TABLE IF NOT EXISTS `payment_transactions` (
  `uuid` varchar(36) NOT NULL,
  `reference` varchar(100) NOT NULL,
  `tripay_reference` varchar(100) DEFAULT NULL,
  `user_id` varchar(36) NOT NULL,
  `event_id` varchar(36) DEFAULT NULL,
  `registration_id` varchar(36) DEFAULT NULL,
  `amount` decimal(12,2) NOT NULL,
  `fee_amount` decimal(12,2) DEFAULT 0.00,
  `total_amount` decimal(12,2) NOT NULL,
  `payment_method` varchar(50) DEFAULT NULL,
  `payment_channel` varchar(50) DEFAULT NULL,
  `va_number` varchar(100) DEFAULT NULL,
  `qr_url` text DEFAULT NULL,
  `checkout_url` text DEFAULT NULL,
  `pay_code` varchar(100) DEFAULT NULL,
  `instructions` text DEFAULT NULL,
  `status` enum('pending','paid','expired','failed','refunded') DEFAULT 'pending',
  `paid_at` timestamp NULL DEFAULT NULL,
  `expired_at` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `callback_data` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`callback_data`)),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `reference` (`reference`),
  KEY `idx_reference` (`reference`),
  KEY `idx_tripay_reference` (`tripay_reference`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_tournament_id` (`event_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.payment_transactions: ~1 rows (approximately)
INSERT INTO `payment_transactions` (`uuid`, `reference`, `tripay_reference`, `user_id`, `event_id`, `registration_id`, `amount`, `fee_amount`, `total_amount`, `payment_method`, `payment_channel`, `va_number`, `qr_url`, `checkout_url`, `pay_code`, `instructions`, `status`, `paid_at`, `expired_at`, `callback_data`, `created_at`, `updated_at`) VALUES
	('8308081e-21f9-4b21-97a9-a2102dbcd123', 'PAY-2bd8c757-1d4', 'DEV-T32426331453OJEYP', '5fff777a-b9aa-417c-9046-00585f5806b0', 'ed94f043-d822-4446-a189-67f423e8d41b', NULL, 50000.00, 0.00, 50000.00, 'QRIS2', NULL, NULL, 'https://tripay.co.id/qr/DEV-T32426331453OJEYP', 'https://tripay.co.id/checkout/DEV-T32426331453OJEYP', NULL, NULL, 'pending', NULL, '2026-01-20 02:04:23', NULL, '2026-01-19 09:04:23', '2026-01-19 09:04:23');

-- Dumping structure for table archeryhub.products
CREATE TABLE IF NOT EXISTS `products` (
  `uuid` varchar(36) NOT NULL,
  `organization_id` varchar(36) DEFAULT NULL,
  `club_id` varchar(36) DEFAULT NULL,
  `seller_id` varchar(36) DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `description` text DEFAULT NULL,
  `price` decimal(12,2) NOT NULL,
  `sale_price` decimal(12,2) DEFAULT NULL,
  `category` enum('equipment','apparel','accessories','training','other') DEFAULT 'other',
  `stock` int(11) DEFAULT 0,
  `status` enum('draft','active','sold_out','archived') DEFAULT 'draft',
  `image_url` varchar(500) DEFAULT NULL,
  `images` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`images`)),
  `specifications` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`specifications`)),
  `views` int(11) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `slug` (`slug`),
  KEY `idx_products_org` (`organization_id`),
  KEY `idx_products_club` (`club_id`),
  KEY `idx_products_status` (`status`),
  KEY `idx_products_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.products: ~3 rows (approximately)
INSERT INTO `products` (`uuid`, `organization_id`, `club_id`, `seller_id`, `name`, `slug`, `description`, `price`, `sale_price`, `category`, `stock`, `status`, `image_url`, `images`, `specifications`, `views`, `created_at`, `updated_at`) VALUES
	('f38b1a3c-f73e-11f0-87db-c3c8a1ce2650', NULL, NULL, 'eb0fb0ce-f73e-11f0-87db-c3c8a1ce2650', 'Win&Win Wiawis ATF-DX Riser', 'wiawis-atf-dx', 'High-end recurve riser with excellent balance.', 12500000.00, 11800000.00, 'equipment', 5, 'active', 'https://images.unsplash.com/photo-1511082782071-4a07a1c37deb?w=800', NULL, NULL, 120, '2026-01-22 03:03:42', '2026-01-22 03:03:42'),
	('f38b1d51-f73e-11f0-87db-c3c8a1ce2650', NULL, NULL, 'eb0fb0ce-f73e-11f0-87db-c3c8a1ce2650', 'Easton X10 Arrows (12pcs)', 'easton-x10-12', 'The gold standard for Olympic recurve and compound.', 6500000.00, NULL, 'equipment', 10, 'active', 'https://images.unsplash.com/photo-1541534741688-6078c64b52d3?w=800', NULL, NULL, 85, '2026-01-22 03:03:42', '2026-01-22 03:03:42'),
	('f38b1e5f-f73e-11f0-87db-c3c8a1ce2650', NULL, NULL, 'eb0fb0ce-f73e-11f0-87db-c3c8a1ce2650', 'Avalon Tec One Quiver', 'avalon-tec-one-quiver', 'Durable and stylish field quiver.', 450000.00, 380000.00, 'accessories', 20, 'active', 'https://images.unsplash.com/photo-1590483734724-383b9f449303?w=800', NULL, NULL, 45, '2026-01-22 03:03:42', '2026-01-22 03:03:42');

-- Dumping structure for table archeryhub.qualification_assignments
CREATE TABLE IF NOT EXISTS `qualification_assignments` (
  `uuid` varchar(36) NOT NULL,
  `session_uuid` varchar(36) NOT NULL,
  `participant_uuid` varchar(36) NOT NULL,
  `target_number` int(11) NOT NULL,
  `target_position` enum('A','B','C','D') NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `session_uuid` (`session_uuid`,`target_number`,`target_position`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.qualification_assignments: ~0 rows (approximately)

-- Dumping structure for table archeryhub.qualification_end_scores
CREATE TABLE IF NOT EXISTS `qualification_end_scores` (
  `uuid` varchar(36) NOT NULL,
  `assignment_uuid` varchar(36) NOT NULL,
  `end_number` int(11) NOT NULL,
  `arrow_1` varchar(2) DEFAULT NULL,
  `arrow_2` varchar(2) DEFAULT NULL,
  `arrow_3` varchar(2) DEFAULT NULL,
  `arrow_4` varchar(2) DEFAULT NULL,
  `arrow_5` varchar(2) DEFAULT NULL,
  `arrow_6` varchar(2) DEFAULT NULL,
  `end_total` int(11) DEFAULT 0,
  `end_x_count` int(11) DEFAULT 0,
  `end_10_count` int(11) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `assignment_uuid` (`assignment_uuid`,`end_number`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.qualification_end_scores: ~0 rows (approximately)

-- Dumping structure for table archeryhub.qualification_sessions
CREATE TABLE IF NOT EXISTS `qualification_sessions` (
  `uuid` varchar(36) NOT NULL,
  `event_category_uuid` varchar(36) NOT NULL,
  `session_name` varchar(100) NOT NULL,
  `session_order` int(11) DEFAULT 1,
  `start_time` datetime DEFAULT NULL,
  `status` enum('draft','ongoing','completed') DEFAULT 'draft',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.qualification_sessions: ~0 rows (approximately)

-- Dumping structure for table archeryhub.ref_age_groups
CREATE TABLE IF NOT EXISTS `ref_age_groups` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.ref_age_groups: ~6 rows (approximately)
INSERT INTO `ref_age_groups` (`uuid`, `code`, `name`, `created_at`) VALUES
	('1a7dc374-3016-49bf-ac56-9dededae7116', 'u10', 'U-10', '2026-01-21 01:49:51'),
	('55c3beff-8d11-4c93-8a5a-25eff347af4a', 'u13', 'U-13', '2026-01-21 01:49:51'),
	('d0ac100f-3a30-46bd-a986-17eec4274c86', 'master', 'Master', '2026-01-21 01:49:51'),
	('f0ab19a5-efe2-4ceb-b177-57966249af04', 'u18', 'U-18', '2026-01-21 01:49:51'),
	('f235b870-724b-44ac-8683-b665df0c0548', 'senior', 'Senior', '2026-01-21 01:49:51'),
	('f879b964-0929-45d3-9a6a-7d80f3cf708f', 'u15', 'U-15', '2026-01-21 01:49:51');

-- Dumping structure for table archeryhub.ref_bow_types
CREATE TABLE IF NOT EXISTS `ref_bow_types` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.ref_bow_types: ~4 rows (approximately)
INSERT INTO `ref_bow_types` (`uuid`, `code`, `name`, `created_at`) VALUES
	('349a5218-fed0-4305-ab16-e636501bb5df', 'compound', 'Compound', '2026-01-21 01:49:49'),
	('5c95a503-4fd4-465f-9dd3-b08568181792', 'recurve', 'Recurve', '2026-01-21 01:49:49'),
	('8718ac12-a02a-4c38-aedd-5721c83c6513', 'traditional', 'Traditional', '2026-01-21 01:49:49'),
	('94bf104d-8ef2-4dd0-a1b1-2d82b46a1bdc', 'barebow', 'Barebow', '2026-01-21 01:49:49');

-- Dumping structure for table archeryhub.ref_disciplines
CREATE TABLE IF NOT EXISTS `ref_disciplines` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.ref_disciplines: ~4 rows (approximately)
INSERT INTO `ref_disciplines` (`uuid`, `code`, `name`, `created_at`) VALUES
	('331b52a7-812d-4dde-aeaf-978e79bf293a', 'target_outdoor', 'Target Outdoor', '2026-01-21 01:49:48'),
	('33502405-e5ff-4725-a882-d279666fe35c', 'field', 'Field', '2026-01-21 01:49:48'),
	('35db84b0-ef7e-4617-924a-be4531b9833e', '3d', '3D', '2026-01-21 01:49:48'),
	('47a8f20c-0502-4b0b-a754-4bbc47d430f0', 'target_indoor', 'Target Indoor', '2026-01-21 01:49:48');

-- Dumping structure for table archeryhub.ref_event_types
CREATE TABLE IF NOT EXISTS `ref_event_types` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.ref_event_types: ~3 rows (approximately)
INSERT INTO `ref_event_types` (`uuid`, `code`, `name`, `created_at`) VALUES
	('160f7979-706c-41bb-ba42-111186ad31ab', 'mixed_team', 'Mixed Team', '2026-01-21 01:49:50'),
	('3bfbc4ad-afb2-44b2-a686-8d5fd46e5e2f', 'team', 'Team', '2026-01-21 01:49:50'),
	('da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'individual', 'Individual', '2026-01-21 01:49:50');

-- Dumping structure for table archeryhub.ref_gender_divisions
CREATE TABLE IF NOT EXISTS `ref_gender_divisions` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.ref_gender_divisions: ~3 rows (approximately)
INSERT INTO `ref_gender_divisions` (`uuid`, `code`, `name`, `created_at`) VALUES
	('afbded2f-705c-480f-84d2-6962bcb4b2ef', 'women', 'Women', '2026-01-21 01:49:51'),
	('d60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'men', 'Men', '2026-01-21 01:49:51'),
	('e6d15459-d6d0-405e-aa8c-b297c9d1b2f3', 'open', 'Open', '2026-01-21 01:49:51');

-- Dumping structure for table archeryhub.sellers
CREATE TABLE IF NOT EXISTS `sellers` (
  `uuid` varchar(36) NOT NULL,
  `username` varchar(50) DEFAULT NULL,
  `google_id` varchar(100) DEFAULT NULL,
  `user_id` varchar(36) DEFAULT NULL,
  `store_name` varchar(255) NOT NULL,
  `store_slug` varchar(255) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `role` enum('seller','admin') DEFAULT 'seller',
  `description` text DEFAULT NULL,
  `avatar_url` varchar(500) DEFAULT NULL,
  `banner_url` varchar(500) DEFAULT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `address` text DEFAULT NULL,
  `city` varchar(100) DEFAULT NULL,
  `province` varchar(100) DEFAULT NULL,
  `is_verified` tinyint(1) DEFAULT 0,
  `rating` decimal(3,2) DEFAULT 0.00,
  `total_sales` int(11) DEFAULT 0,
  `status` enum('pending','active','suspended') DEFAULT 'pending',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `store_slug` (`store_slug`),
  UNIQUE KEY `username` (`username`),
  KEY `idx_sellers_status` (`status`),
  KEY `idx_sellers_verified` (`is_verified`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.sellers: ~1 rows (approximately)
INSERT INTO `sellers` (`uuid`, `username`, `google_id`, `user_id`, `store_name`, `store_slug`, `password`, `role`, `description`, `avatar_url`, `banner_url`, `phone`, `email`, `address`, `city`, `province`, `is_verified`, `rating`, `total_sales`, `status`, `created_at`, `updated_at`) VALUES
	('eb0fb0ce-f73e-11f0-87db-c3c8a1ce2650', 'garuda_archery', NULL, NULL, 'Garuda Archery Store', 'garuda-archery', '12345', 'seller', NULL, NULL, NULL, NULL, 'seller@example.com', NULL, NULL, NULL, 1, 0.00, 0, 'active', '2026-01-22 03:03:28', '2026-01-22 03:06:25');

-- Dumping structure for table archeryhub.seller_profiles
CREATE TABLE IF NOT EXISTS `seller_profiles` (
  `uuid` varchar(36) NOT NULL,
  `seller_id` varchar(36) NOT NULL,
  `sections` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`sections`)),
  `catalog_config` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`catalog_config`)),
  `theme_color` varchar(20) DEFAULT NULL,
  `banner_text` text DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `seller_id` (`seller_id`),
  CONSTRAINT `seller_profiles_ibfk_1` FOREIGN KEY (`seller_id`) REFERENCES `sellers` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci;

-- Dumping data for table archeryhub.seller_profiles: ~0 rows (approximately)

-- Dumping structure for table archeryhub.teams
CREATE TABLE IF NOT EXISTS `teams` (
  `uuid` varchar(36) NOT NULL,
  `event_id` varchar(36) NOT NULL,
  `category_id` varchar(36) NOT NULL,
  `team_name` varchar(100) NOT NULL,
  `country_code` varchar(10) NOT NULL,
  `country_name` varchar(100) DEFAULT NULL,
  `team_rank` int(11) DEFAULT NULL,
  `total_score` int(11) DEFAULT 0,
  `total_x_count` int(11) DEFAULT 0,
  `status` enum('active','eliminated','qualified') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_teams_tournament` (`event_id`),
  KEY `idx_teams_event` (`category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.teams: ~0 rows (approximately)

-- Dumping structure for table archeryhub.team_members
CREATE TABLE IF NOT EXISTS `team_members` (
  `uuid` varchar(36) NOT NULL,
  `team_id` varchar(36) NOT NULL,
  `participant_id` varchar(36) NOT NULL,
  `member_order` int(11) NOT NULL,
  `is_substitute` tinyint(1) DEFAULT 0,
  `total_score` int(11) DEFAULT 0,
  `total_x_count` int(11) DEFAULT 0,
  PRIMARY KEY (`uuid`),
  KEY `idx_team_members_team` (`team_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table archeryhub.team_members: ~0 rows (approximately)

/*!40103 SET TIME_ZONE=IFNULL(@OLD_TIME_ZONE, 'system') */;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
