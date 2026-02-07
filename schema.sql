-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               10.4.20-MariaDB - mariadb.org binary distribution
-- Server OS:                    Win64
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
CREATE DATABASE IF NOT EXISTS `archeryhub` /*!40100 DEFAULT CHARACTER SET latin1 */;
USE `archeryhub`;

-- Dumping structure for table archeryhub.activity_logs
CREATE TABLE IF NOT EXISTS `activity_logs` (
  `id` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `action` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_type` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `entity_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `description` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ip_address` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_event` (`event_id`),
  KEY `idx_action` (`action`),
  KEY `idx_created` (`created_at`),
  CONSTRAINT `activity_logs_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `archers` (`uuid`) ON DELETE SET NULL,
  CONSTRAINT `activity_logs_ibfk_2` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.archers
CREATE TABLE IF NOT EXISTS `archers` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `id` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `username` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `google_id` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `avatar_url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `bio` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `password` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `full_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `nickname` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `date_of_birth` date DEFAULT NULL,
  `gender` enum('male','female') COLLATE utf8mb4_unicode_ci DEFAULT 'male',
  `phone` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `address` mediumtext COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `city` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `school` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `bow_type` enum('recurve','compound','barebow','traditional') COLLATE utf8mb4_unicode_ci DEFAULT 'recurve',
  `club_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` enum('active','inactive','suspended') COLLATE utf8mb4_unicode_ci DEFAULT 'active',
  `is_verified` tinyint(1) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `page_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`page_settings`)),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_archers_email` (`email`),
  UNIQUE KEY `idx_archers_slug` (`username`),
  UNIQUE KEY `custom_id` (`id`),
  KEY `idx_club_id` (`club_id`),
  KEY `idx_bow_type` (`bow_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.assignment_history
CREATE TABLE IF NOT EXISTS `assignment_history` (
  `uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'archer, match, target',
  `entity_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `previous_assignment` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `new_assignment` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `changed_by` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'user_uuid who made the change',
  `reason` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_ah_event` (`event_uuid`),
  KEY `idx_ah_entity` (`entity_type`,`entity_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

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
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.clubs
CREATE TABLE IF NOT EXISTS `clubs` (
  `uuid` varchar(36) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `slug_changed` tinyint(1) DEFAULT 0,
  `user_id` varchar(36) DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `abbreviation` varchar(20) DEFAULT NULL,
  `description` text DEFAULT NULL,
  `banner_url` varchar(500) DEFAULT NULL,
  `logo_url` varchar(500) DEFAULT NULL,
  `email` varchar(255) NOT NULL,
  `google_id` varchar(100) DEFAULT NULL,
  `avatar_url` varchar(255) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `address` text DEFAULT NULL,
  `city` varchar(100) DEFAULT NULL,
  `province` varchar(100) DEFAULT NULL,
  `postal_code` varchar(10) DEFAULT NULL,
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
  `social_media` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`social_media`)),
  `page_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`page_settings`)),
  `status` enum('active','inactive','suspended') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_clubs_email` (`email`),
  UNIQUE KEY `slug` (`slug`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_organization_id` (`organization_id`),
  KEY `idx_city` (`city`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

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
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.club_members
CREATE TABLE IF NOT EXISTS `club_members` (
  `uuid` varchar(36) NOT NULL,
  `club_id` varchar(36) NOT NULL,
  `archer_id` varchar(36) NOT NULL,
  `status` enum('pending','active','rejected','left') DEFAULT 'pending',
  `joined_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_archer_club` (`archer_id`),
  KEY `idx_club_members_club` (`club_id`),
  KEY `idx_club_members_archer` (`archer_id`),
  KEY `idx_club_members_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.elimination_brackets
CREATE TABLE IF NOT EXISTS `elimination_brackets` (
  `uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `bracket_id` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `category_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `bracket_type` enum('individual','team3','mixed2') COLLATE utf8mb4_unicode_ci NOT NULL,
  `format` enum('recurve_set','compound_total') COLLATE utf8mb4_unicode_ci NOT NULL,
  `bracket_size` int(10) unsigned NOT NULL,
  `ends_per_match` int(11) DEFAULT 5,
  `arrows_per_end` int(11) DEFAULT 3,
  `generated_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `bracket_id` (`bracket_id`),
  KEY `idx_eb_event_category` (`event_uuid`,`category_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.elimination_entries
CREATE TABLE IF NOT EXISTS `elimination_entries` (
  `uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `bracket_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `participant_type` enum('archer','team') COLLATE utf8mb4_unicode_ci NOT NULL,
  `participant_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `seed` int(10) unsigned NOT NULL,
  `qual_total_score` int(10) unsigned DEFAULT NULL,
  `qual_total_x` int(10) unsigned DEFAULT NULL,
  `qual_total_10` int(10) unsigned DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_ee_bracket_seed` (`bracket_uuid`,`seed`),
  UNIQUE KEY `uq_ee_bracket_participant` (`bracket_uuid`,`participant_type`,`participant_uuid`),
  KEY `idx_ee_bracket` (`bracket_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.elimination_matches
CREATE TABLE IF NOT EXISTS `elimination_matches` (
  `uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `bracket_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `round_no` int(10) unsigned NOT NULL,
  `match_no` int(10) unsigned NOT NULL,
  `entry_a_uuid` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `entry_b_uuid` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `winner_entry_uuid` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` enum('pending','in_progress','finished') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `is_bye` tinyint(1) NOT NULL DEFAULT 0,
  `scheduled_at` datetime DEFAULT NULL,
  `target_uuid` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_em_round_match` (`bracket_uuid`,`round_no`,`match_no`),
  KEY `idx_em_bracket_round` (`bracket_uuid`,`round_no`),
  KEY `idx_elim_matches_target_uuid` (`target_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.elimination_match_arrow_scores
CREATE TABLE IF NOT EXISTS `elimination_match_arrow_scores` (
  `uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `match_end_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `arrow_no` int(10) unsigned NOT NULL,
  `score` tinyint(3) unsigned NOT NULL,
  `is_x` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_emas_end_arrow` (`match_end_uuid`,`arrow_no`),
  KEY `idx_emas_end` (`match_end_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.elimination_match_ends
CREATE TABLE IF NOT EXISTS `elimination_match_ends` (
  `uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `match_uuid` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `end_no` int(10) unsigned NOT NULL,
  `side` enum('A','B') COLLATE utf8mb4_unicode_ci NOT NULL,
  `end_total` int(10) unsigned NOT NULL DEFAULT 0,
  `x_count` int(10) unsigned NOT NULL DEFAULT 0,
  `ten_count` int(10) unsigned NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_eme_match_end_side` (`match_uuid`,`end_no`,`side`),
  KEY `idx_eme_match` (`match_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.events
CREATE TABLE IF NOT EXISTS `events` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `slug` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `code` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `short_name` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `venue` varchar(200) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `gmaps_link` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(200) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `address` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `city` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `start_date` datetime DEFAULT NULL,
  `end_date` datetime DEFAULT NULL,
  `registration_deadline` datetime DEFAULT NULL,
  `description` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `banner_url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `logo_url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `num_distances` tinyint(4) DEFAULT 1,
  `num_sessions` tinyint(4) DEFAULT 1,
  `entry_fee` decimal(10,2) DEFAULT 0.00,
  `status` enum('draft','active') COLLATE utf8mb4_unicode_ci DEFAULT 'draft',
  `organizer_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `total_prize` decimal(15,2) DEFAULT 0.00,
  `technical_guidebook_url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `page_settings` longtext COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `faq` longtext COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `whatsapp_number` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `venue_type` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location_type` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `slug` (`slug`),
  KEY `idx_code` (`code`),
  KEY `idx_status` (`status`),
  KEY `idx_start_date` (`start_date`),
  KEY `idx_organizer` (`organizer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.event_categories
CREATE TABLE IF NOT EXISTS `event_categories` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_id` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `division_uuid` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `category_uuid` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_type_uuid` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `gender_division_uuid` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `max_participants` int(11) DEFAULT NULL,
  `status` enum('active','inactive') COLLATE utf8mb4_unicode_ci DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `fk_event_categories_event` (`event_id`),
  CONSTRAINT `fk_event_categories_event` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.event_category_refs
CREATE TABLE IF NOT EXISTS `event_category_refs` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `bow_type_id` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `age_group_id` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('active','inactive') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.event_images
CREATE TABLE IF NOT EXISTS `event_images` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_id` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `url` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL,
  `caption` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `alt_text` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `display_order` int(11) DEFAULT 0,
  `is_primary` tinyint(1) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_event_images_event` (`event_id`),
  KEY `idx_event_images_order` (`event_id`,`display_order`),
  CONSTRAINT `fk_event_images_event` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.event_participants
CREATE TABLE IF NOT EXISTS `event_participants` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_id` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `archer_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `category_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `payment_amount` decimal(10,2) DEFAULT 0.00,
  `payment_proof_urls` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `payment_status` enum('menunggu_acc','belum_lunas','lunas') COLLATE utf8mb4_unicode_ci DEFAULT 'belum_lunas',
  `status` enum('Ditolak','Menunggu Acc','Terdaftar') COLLATE utf8mb4_unicode_ci DEFAULT 'Menunggu Acc',
  `target_name` varchar(10) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `back_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `qr_raw` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_reregistration_at` timestamp NULL DEFAULT NULL,
  `registration_date` timestamp NOT NULL DEFAULT current_timestamp(),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `event_id` (`event_id`),
  KEY `event_category_id` (`category_id`),
  KEY `athlete_id` (`archer_id`),
  CONSTRAINT `event_participants_ibfk_1` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`),
  CONSTRAINT `event_participants_ibfk_3` FOREIGN KEY (`archer_id`) REFERENCES `archers` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.event_schedule
CREATE TABLE IF NOT EXISTS `event_schedule` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_id` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `start_time` datetime NOT NULL,
  `end_time` datetime DEFAULT NULL,
  `day_order` tinyint(4) DEFAULT NULL,
  `sort_order` int(11) DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_event_schedule_event_id` (`event_id`),
  CONSTRAINT `fk_event_schedule_event` FOREIGN KEY (`event_id`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.event_targets
CREATE TABLE IF NOT EXISTS `event_targets` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_event_target_name` (`event_uuid`,`target_name`),
  KEY `idx_event_uuid` (`event_uuid`),
  CONSTRAINT `event_targets_ibfk_1` FOREIGN KEY (`event_uuid`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Master data for physical targets available in an event';

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.media
CREATE TABLE IF NOT EXISTS `media` (
  `uuid` varchar(36) NOT NULL,
  `user_id` varchar(36) NOT NULL,
  `user_type` varchar(50) NOT NULL,
  `url` text NOT NULL,
  `caption` varchar(255) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `mime_type` varchar(100) NOT NULL DEFAULT 'image/jpeg',
  `size` bigint(20) NOT NULL DEFAULT 0,
  PRIMARY KEY (`uuid`),
  KEY `idx_user` (`user_id`,`user_type`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

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
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.notifications
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(36) NOT NULL,
  `user_role` enum('club','organization','archer','admin') NOT NULL,
  `type` enum('success','warning','danger','info','default') DEFAULT 'default',
  `title` varchar(255) NOT NULL,
  `message` text NOT NULL,
  `link` varchar(500) DEFAULT NULL,
  `is_read` tinyint(1) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_user_role` (`user_role`),
  KEY `idx_is_read` (`is_read`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

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
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

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
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.organizations
CREATE TABLE IF NOT EXISTS `organizations` (
  `uuid` varchar(36) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `user_id` varchar(36) DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `acronym` varchar(20) DEFAULT NULL,
  `description` text DEFAULT NULL,
  `vision` text DEFAULT NULL,
  `mission` text DEFAULT NULL,
  `history` text DEFAULT NULL,
  `website` varchar(255) DEFAULT NULL,
  `email` varchar(255) NOT NULL,
  `google_id` varchar(100) DEFAULT NULL,
  `avatar_url` varchar(255) DEFAULT NULL,
  `banner_url` varchar(255) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `whatsapp_no` varchar(20) DEFAULT NULL,
  `address` text DEFAULT NULL,
  `city` varchar(100) DEFAULT NULL,
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
  `social_media` longtext DEFAULT NULL,
  `verification_status` enum('pending','verified','rejected') DEFAULT 'pending',
  `status` enum('active','inactive','suspended') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `page_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`page_settings`)),
  `faq` longtext DEFAULT NULL,
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_orgs_email` (`email`),
  UNIQUE KEY `idx_orgs_username` (`slug`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_verification_status` (`verification_status`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

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
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.products
CREATE TABLE IF NOT EXISTS `products` (
  `uuid` varchar(36) NOT NULL,
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
  KEY `idx_products_status` (`status`),
  KEY `idx_products_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.qualification_arrow_scores
CREATE TABLE IF NOT EXISTS `qualification_arrow_scores` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `end_score_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `arrow_number` int(10) unsigned NOT NULL,
  `score` tinyint(3) unsigned NOT NULL,
  `is_x` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_qas_end_arrow` (`end_score_uuid`,`arrow_number`),
  KEY `idx_qas_end_score` (`end_score_uuid`),
  KEY `idx_qas_score` (`score`),
  CONSTRAINT `fk_qas_end_score` FOREIGN KEY (`end_score_uuid`) REFERENCES `qualification_end_scores` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.qualification_end_scores
CREATE TABLE IF NOT EXISTS `qualification_end_scores` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `session_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `archer_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `end_number` int(10) unsigned NOT NULL,
  `total_score_end` int(10) unsigned NOT NULL DEFAULT 0,
  `x_count_end` int(10) unsigned NOT NULL DEFAULT 0,
  `ten_count_end` int(10) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_qes_session_archer_end` (`session_uuid`,`archer_uuid`,`end_number`),
  KEY `idx_qes_archer` (`archer_uuid`),
  KEY `idx_qes_session` (`session_uuid`),
  KEY `idx_qes_end_number` (`end_number`),
  CONSTRAINT `fk_qes_archer` FOREIGN KEY (`archer_uuid`) REFERENCES `archers` (`uuid`) ON DELETE CASCADE,
  CONSTRAINT `fk_qes_session` FOREIGN KEY (`session_uuid`) REFERENCES `qualification_sessions` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.qualification_sessions
CREATE TABLE IF NOT EXISTS `qualification_sessions` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `session_code` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `session_date` date DEFAULT NULL,
  `name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_time` datetime DEFAULT NULL,
  `end_time` datetime DEFAULT NULL,
  `total_ends` int(10) unsigned NOT NULL DEFAULT 12,
  `arrows_per_end` int(10) unsigned NOT NULL DEFAULT 6,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_qs_event_name` (`event_uuid`,`name`),
  UNIQUE KEY `session_code` (`session_code`),
  KEY `idx_qs_event` (`event_uuid`),
  CONSTRAINT `fk_qs_event` FOREIGN KEY (`event_uuid`) REFERENCES `events` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.qualification_target_assignments
CREATE TABLE IF NOT EXISTS `qualification_target_assignments` (
  `uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `session_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `archer_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_uuid` varchar(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_position` enum('A','B','C','D') COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_by` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `uq_qta_session_archer` (`session_uuid`,`archer_uuid`),
  UNIQUE KEY `uq_qta_session_target_pos` (`session_uuid`,`target_uuid`,`target_position`),
  KEY `fk_qta_target` (`target_uuid`),
  KEY `idx_qta_session_target` (`session_uuid`,`target_uuid`),
  KEY `idx_qta_archer` (`archer_uuid`),
  CONSTRAINT `fk_qta_archer` FOREIGN KEY (`archer_uuid`) REFERENCES `archers` (`uuid`) ON DELETE CASCADE,
  CONSTRAINT `fk_qta_session` FOREIGN KEY (`session_uuid`) REFERENCES `qualification_sessions` (`uuid`) ON DELETE CASCADE,
  CONSTRAINT `fk_qta_target` FOREIGN KEY (`target_uuid`) REFERENCES `event_targets` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.ref_age_groups
CREATE TABLE IF NOT EXISTS `ref_age_groups` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.ref_bow_types
CREATE TABLE IF NOT EXISTS `ref_bow_types` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.ref_disciplines
CREATE TABLE IF NOT EXISTS `ref_disciplines` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.ref_event_types
CREATE TABLE IF NOT EXISTS `ref_event_types` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.ref_gender_divisions
CREATE TABLE IF NOT EXISTS `ref_gender_divisions` (
  `uuid` varchar(36) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `code` (`code`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.sellers
CREATE TABLE IF NOT EXISTS `sellers` (
  `uuid` varchar(36) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `google_id` varchar(100) DEFAULT NULL,
  `user_id` varchar(36) DEFAULT NULL,
  `store_name` varchar(255) NOT NULL,
  `password` varchar(255) DEFAULT NULL,
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
  `page_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`page_settings`)),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `username` (`slug`),
  KEY `idx_sellers_status` (`status`),
  KEY `idx_sellers_verified` (`is_verified`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.teams
CREATE TABLE IF NOT EXISTS `teams` (
  `uuid` varchar(36) NOT NULL,
  `tournament_id` varchar(36) NOT NULL,
  `event_id` varchar(36) NOT NULL,
  `team_name` varchar(100) NOT NULL,
  `team_rank` int(11) DEFAULT NULL,
  `total_score` int(11) DEFAULT 0,
  `total_x_count` int(11) DEFAULT 0,
  `status` enum('active','eliminated','qualified') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_tournament` (`tournament_id`),
  KEY `idx_event` (`event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

-- Dumping structure for table archeryhub.team_members
CREATE TABLE IF NOT EXISTS `team_members` (
  `uuid` varchar(36) NOT NULL,
  `team_id` varchar(36) NOT NULL,
  `participant_id` varchar(36) NOT NULL,
  `member_order` int(11) NOT NULL,
  `total_score` int(11) DEFAULT 0,
  `total_x_count` int(11) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_team` (`team_id`),
  KEY `idx_participant` (`participant_id`),
  CONSTRAINT `fk_team` FOREIGN KEY (`team_id`) REFERENCES `teams` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- Data exporting was unselected.

/*!40103 SET TIME_ZONE=IFNULL(@OLD_TIME_ZONE, 'system') */;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
