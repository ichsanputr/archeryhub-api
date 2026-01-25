-- --------------------------------------------------------
-- Host:                         103.63.24.40
-- Server version:               10.3.39-MariaDB-0ubuntu0.20.04.2 - Ubuntu 20.04
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


-- Dumping database structure for furnifilux_dev
CREATE DATABASE IF NOT EXISTS `furnifilux_dev` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci */;
USE `furnifilux_dev`;

-- Dumping structure for table furnifilux_dev.akun
CREATE TABLE IF NOT EXISTS `akun` (
  `uuid` char(50) DEFAULT NULL,
  `id` char(50) DEFAULT NULL,
  `name` char(50) DEFAULT NULL,
  `description` char(50) DEFAULT NULL,
  `balance` int(11) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.akun: ~0 rows (approximately)
INSERT INTO `akun` (`uuid`, `id`, `name`, `description`, `balance`, `created_at`, `updated_at`) VALUES
	('35e10da3-5e3e-4d94-a235-1600119e3dc3', 'AKUN#1000', 'BRI UTAMA', NULL, 30955000, '2026-01-24 03:50:04', '2026-01-24 04:09:53');

-- Dumping structure for table furnifilux_dev.finance_options
CREATE TABLE IF NOT EXISTS `finance_options` (
  `uuid` char(50) NOT NULL,
  `category` enum('vendor','klien','proyek','lokasi','akun') NOT NULL,
  `value` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_finance_option` (`category`,`value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.finance_options: ~13 rows (approximately)
INSERT INTO `finance_options` (`uuid`, `category`, `value`, `created_at`) VALUES
	('2466ed69-1b16-4418-9cde-93ba83441bf9', 'vendor', 'HAndra / Marketing', '2025-12-16 06:50:50'),
	('2897d67b-39b9-4880-a392-5e8b71f4c9e9', 'vendor', 'a', '2025-12-16 08:32:37'),
	('550e8400-e29b-41d4-a716-446655443001', 'vendor', 'Vendor A', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443002', 'vendor', 'Vendor B', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443003', 'klien', 'Client A', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443004', 'klien', 'Client B', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443005', 'proyek', 'Proyek Alpha', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443006', 'proyek', 'Proyek Beta', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443007', 'lokasi', 'Jakarta', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443008', 'lokasi', 'Bandung', '2025-10-27 04:24:08'),
	('550e8400-e29b-41d4-a716-446655443009', 'lokasi', 'Jogja', '2025-10-27 04:24:08'),
	('5c1b9a6a-032c-449b-8fb8-4bb2de5e9a88', 'vendor', 'Handra Marketing', '2025-12-16 06:50:56'),
	('63ecd5fa-bd50-4dc3-8029-90a48f850cb1', 'lokasi', 'Bali', '2025-12-16 07:21:22'),
	('eb6c2d46-1d00-4157-a986-2413ff8602e9', 'lokasi', 'Maluku', '2026-01-24 03:51:03'),
	('fb19fd07-5e3d-4761-ae1d-c4ad866fa842', 'vendor', 'Vendor C', '2026-01-24 03:50:55');

-- Dumping structure for table furnifilux_dev.finance_transactions
CREATE TABLE IF NOT EXISTS `finance_transactions` (
  `uuid` char(50) NOT NULL,
  `no` varchar(255) DEFAULT NULL,
  `lead_pay_id` char(50) DEFAULT NULL,
  `lead_proyek_id` char(50) DEFAULT NULL,
  `tanggal` date DEFAULT NULL,
  `jatuh_tempo` date DEFAULT NULL,
  `jenis_transaksi` enum('Debit','Kredit') DEFAULT NULL,
  `kategori_utama` varchar(255) DEFAULT NULL,
  `sub_kategori` varchar(255) DEFAULT NULL,
  `keterangan` text DEFAULT NULL,
  `nominal` int(11) DEFAULT NULL,
  `akun_id` varchar(255) DEFAULT NULL,
  `vendor` varchar(255) DEFAULT NULL,
  `klien` varchar(255) DEFAULT NULL,
  `lokasi` varchar(255) DEFAULT NULL,
  `bulan` varchar(255) DEFAULT NULL,
  `kategori_transaksi` enum('Pendapatan','Pengeluaran','Investasi','Pendanaan') DEFAULT NULL,
  `kategori_arus_kas` enum('Operasional','Investasi','Pendanaan') DEFAULT NULL,
  `kategori_aktivitas` enum('Pendapatan','Proyek','Non Proyek','Marketing','Payroll','Maintenance','Refund','Aset','Modal','Pinjaman','Dividen') DEFAULT NULL,
  `arah_uang` enum('Masuk','Keluar') DEFAULT NULL,
  `debit` int(11) DEFAULT NULL,
  `kredit` int(11) DEFAULT NULL,
  `img_pembayaran` text DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_finance_tanggal` (`tanggal`),
  KEY `idx_finance_kategori_utama` (`kategori_utama`),
  KEY `idx_finance_jenis_transaksi` (`jenis_transaksi`),
  KEY `idx_finance_akun` (`akun_id`) USING BTREE,
  KEY `lead_pay_id` (`lead_pay_id`),
  KEY `lead_proyek_id` (`lead_proyek_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.finance_transactions: ~2 rows (approximately)
INSERT INTO `finance_transactions` (`uuid`, `no`, `lead_pay_id`, `lead_proyek_id`, `tanggal`, `jatuh_tempo`, `jenis_transaksi`, `kategori_utama`, `sub_kategori`, `keterangan`, `nominal`, `akun_id`, `vendor`, `klien`, `lokasi`, `bulan`, `kategori_transaksi`, `kategori_arus_kas`, `kategori_aktivitas`, `arah_uang`, `debit`, `kredit`, `img_pembayaran`, `created_at`, `updated_at`) VALUES
	('136da3fe-2598-471d-98e6-40e6c4f56752', 'FINANCE#1000', NULL, 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', '2026-01-24', '2026-01-25', 'Debit', 'Proyek', 'Pengiriman', 'oke', 45000, '35e10da3-5e3e-4d94-a235-1600119e3dc3', 'Vendor A', 'Arif Budi Setiawan', 'Bandung', 'Januari 2026', 'Pengeluaran', 'Operasional', 'Proyek', 'Keluar', 45000, NULL, 'http://localhost:8002/media/public/oAxRld_20251217_130548.png', '2026-01-24 04:02:36', '2026-01-24 04:02:36'),
	('f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'FINANCE#1001', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', NULL, '2026-01-24', '2026-01-25', 'Kredit', 'Pendapatan', 'Pembayaran Client', NULL, 30000000, '35e10da3-5e3e-4d94-a235-1600119e3dc3', NULL, 'Arif Budi Setiawan', 'Bandung', 'Januari 2026', 'Pendapatan', 'Operasional', 'Pendapatan', 'Masuk', NULL, 30000000, 'http://localhost:8002/media/public/oAxRld_20251217_130548.png', '2026-01-24 04:09:49', '2026-01-24 05:11:24');

-- Dumping structure for table furnifilux_dev.finance_transaction_items
CREATE TABLE IF NOT EXISTS `finance_transaction_items` (
  `uuid` char(50) NOT NULL,
  `transaction_id` char(50) NOT NULL COMMENT 'Foreign key to finance_transactions.uuid',
  `item_name` varchar(255) NOT NULL COMMENT 'Description/name of the item',
  `quantity` int(11) NOT NULL DEFAULT 1 COMMENT 'Quantity of this item',
  `unit_price` int(11) NOT NULL DEFAULT 0 COMMENT 'Price per unit',
  `subtotal` int(11) NOT NULL DEFAULT 0 COMMENT 'Calculated: quantity * unit_price',
  `notes` text DEFAULT NULL COMMENT 'Optional notes for this line item',
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_transaction_items_transaction_id` (`transaction_id`),
  KEY `idx_finance_transaction_items_created_at` (`created_at`),
  CONSTRAINT `fk_transaction_items_transaction` FOREIGN KEY (`transaction_id`) REFERENCES `finance_transactions` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.finance_transaction_items: ~0 rows (approximately)
INSERT INTO `finance_transaction_items` (`uuid`, `transaction_id`, `item_name`, `quantity`, `unit_price`, `subtotal`, `notes`, `created_at`, `updated_at`) VALUES
	('0b1019c7-98a4-437d-abf2-bb56c6e4cbb0', '136da3fe-2598-471d-98e6-40e6c4f56752', 'Pengiriman cat 2 kali', 1, 20000, 20000, '', '2026-01-24 04:02:36', '2026-01-24 04:02:36'),
	('9d8d9123-2e4b-4cdf-b58a-2be389b9d08b', 'f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'Pembayaran pertama', 1, 25000000, 25000000, '', '2026-01-24 04:09:49', '2026-01-24 04:09:49'),
	('be834a4a-2b5d-4b8d-b56e-fd376896db1d', 'f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'Pembayaran #2', 1, 5000000, 5000000, '', '2026-01-24 04:09:49', '2026-01-24 04:09:49'),
	('d174243d-3823-4190-8c29-ec4f6e5aad29', '136da3fe-2598-471d-98e6-40e6c4f56752', 'Pengiriman barang semen', 1, 25000, 25000, '', '2026-01-24 04:02:36', '2026-01-24 04:02:36');

-- Dumping structure for table furnifilux_dev.leads
CREATE TABLE IF NOT EXISTS `leads` (
  `uuid` char(50) NOT NULL,
  `id` varchar(255) DEFAULT NULL,
  `tgl` date DEFAULT NULL,
  `jam_masuk` int(11) DEFAULT NULL,
  `nama_klien` varchar(255) DEFAULT NULL,
  `no_tlp` varchar(50) DEFAULT NULL,
  `alamat` text DEFAULT NULL,
  `kota` varchar(255) DEFAULT NULL,
  `produk` varchar(255) DEFAULT NULL,
  `kategori_produk` varchar(255) DEFAULT NULL,
  `status` enum('New Leads','Follow Up','Survey','Desain','Penawaran','Menunggu Keputusan','Closing','Gagal Closing','No Respon','Lainnya (Non-Leads)','Selesai') DEFAULT NULL,
  `kategori_leads` enum('Belum Teridentifikasi','Customer Leads','Non Leads') DEFAULT NULL,
  `status_kategorisasi` varchar(255) DEFAULT NULL,
  `sumber_leads` varchar(255) DEFAULT NULL,
  `cs` varchar(255) DEFAULT NULL,
  `tgl_closing` date DEFAULT NULL,
  `tgl_selesai` date DEFAULT NULL,
  `keterangan` text DEFAULT NULL,
  `tgl_fu1` date DEFAULT NULL,
  `ket_fu1` text DEFAULT NULL,
  `tgl_fu2` date DEFAULT NULL,
  `ket_fu2` text DEFAULT NULL,
  `tgl_fu3` date DEFAULT NULL,
  `ket_fu3` text DEFAULT NULL,
  `catatan` longtext DEFAULT NULL,
  `harga_jual` int(11) DEFAULT NULL,
  `perkiraan_hpp` int(11) DEFAULT NULL,
  `nilai_pipeline` int(11) DEFAULT NULL,
  `jumlah_pembayaran` int(11) DEFAULT NULL,
  `hpp_aktual` int(11) DEFAULT NULL,
  `omset` int(11) DEFAULT NULL,
  `sisa_pembayaran` int(11) DEFAULT NULL,
  `margin_aktual` int(11) DEFAULT NULL,
  `hari_sejak_fu` int(11) DEFAULT NULL,
  `total_fu` int(11) DEFAULT NULL,
  `durasi_hingga_closing` int(11) DEFAULT NULL,
  `umur_lead` int(11) DEFAULT NULL,
  `img_desain` text DEFAULT NULL,
  `img_survey` text DEFAULT NULL,
  `img_penawaran` text DEFAULT NULL,
  `img_closing` text DEFAULT NULL,
  `img_selesai` text DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_leads_status` (`status`),
  KEY `idx_leads_kota` (`kota`),
  KEY `idx_leads_cs` (`cs`),
  KEY `idx_leads_tgl` (`tgl`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.leads: ~2 rows (approximately)
INSERT INTO `leads` (`uuid`, `id`, `tgl`, `jam_masuk`, `nama_klien`, `no_tlp`, `alamat`, `kota`, `produk`, `kategori_produk`, `status`, `kategori_leads`, `status_kategorisasi`, `sumber_leads`, `cs`, `tgl_closing`, `tgl_selesai`, `keterangan`, `tgl_fu1`, `ket_fu1`, `tgl_fu2`, `ket_fu2`, `tgl_fu3`, `ket_fu3`, `catatan`, `harga_jual`, `perkiraan_hpp`, `nilai_pipeline`, `jumlah_pembayaran`, `hpp_aktual`, `omset`, `sisa_pembayaran`, `margin_aktual`, `hari_sejak_fu`, `total_fu`, `durasi_hingga_closing`, `umur_lead`, `img_desain`, `img_survey`, `img_penawaran`, `img_closing`, `img_selesai`, `created_at`, `updated_at`) VALUES
	('2982d9cf-8417-4c70-8aa0-1834c419042a', 'LEADS#1000', '2026-01-24', 3, 'Muhammad Ichsan', '081325184866', 'Jalan mawar\nSelomartani', 'Maluku Utara', 'Mebel Furniture', 'Exterior Luar', 'New Leads', 'Customer Leads', 'Active', 'Twitter', 'CS G', NULL, NULL, 'Ini Leads Baru', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 0, NULL, NULL, NULL, NULL, NULL, '2026-01-24 03:07:22', '2026-01-24 03:07:22'),
	('d4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'LEADS#1001', '2026-01-23', 5, 'Arif Budi Setiawan', '081238664316', 'Jalan mawar\nSelomartani 2', 'Balikpapan', 'Mebel Luar', 'Exterior', 'Closing', 'Customer Leads', 'Closed', 'Instagram', 'CS B', '2026-01-25', NULL, 'baru mebel luar', '2026-01-24', 'sudah di follow up pertama x', NULL, NULL, NULL, NULL, NULL, 45000000, 35000000, 0, 30000000, 45000, 45000000, 15000000, 44955000, 0, 1, 2, 1, NULL, 'http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg', NULL, 'http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg', NULL, '2026-01-24 03:08:09', '2026-01-24 04:09:52');

-- Dumping structure for table furnifilux_dev.leads_history_edit
CREATE TABLE IF NOT EXISTS `leads_history_edit` (
  `uuid` char(50) NOT NULL,
  `lead_uuid` char(50) NOT NULL,
  `user_id` char(50) NOT NULL,
  `field_name` varchar(255) NOT NULL,
  `old_value` text DEFAULT NULL,
  `new_value` text DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_history_lead` (`lead_uuid`),
  KEY `idx_history_user` (`user_id`),
  CONSTRAINT `fk_history_lead` FOREIGN KEY (`lead_uuid`) REFERENCES `leads` (`uuid`) ON DELETE CASCADE,
  CONSTRAINT `fk_history_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.leads_history_edit: ~0 rows (approximately)
INSERT INTO `leads_history_edit` (`uuid`, `lead_uuid`, `user_id`, `field_name`, `old_value`, `new_value`, `created_at`) VALUES
	('09a26897-d63c-441f-933d-983f31e360b6', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'nama_klien', 'Arif Budi', 'Arif Budi Setiawan', '2026-01-24 03:36:02'),
	('0d11d953-a0e6-41f5-8ecc-a3376e819733', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'durasi_hingga_closing', NULL, '2', '2026-01-24 03:57:53'),
	('2a4b4912-e06c-42b5-a984-aed2018cba9e', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'margin_aktual', NULL, 'Rp 45,000,000.00', '2026-01-24 03:57:53'),
	('2e40fb9c-a9c1-4213-8926-bac5571a3763', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'status', 'Survey', 'Closing', '2026-01-24 03:57:54'),
	('402c9df8-cf2f-4200-ad54-2f0337670c0a', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'status_kategorisasi', 'Active', 'Closed', '2026-01-24 03:57:54'),
	('832034d9-daca-421a-b101-e908d8167960', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'sisa_pembayaran', NULL, 'Rp 45,000,000.00', '2026-01-24 03:57:54'),
	('a7bce619-f426-48d7-967d-f5579effa2a7', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'perkiraan_hpp', NULL, 'Rp 35,000,000.00', '2026-01-24 03:57:53'),
	('ac9a208d-2c21-498f-86fd-360c60ccc4ea', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'alamat', 'Jalan mawar\nSelomartani', 'Jalan mawar\nSelomartani 2', '2026-01-24 03:36:02'),
	('b5c3df03-2ea6-4cbd-95d8-300db60924ee', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'harga_jual', NULL, 'Rp 45,000,000.00', '2026-01-24 03:57:53'),
	('d63ee9f5-8898-4a88-b9ae-9f5e9c49ecd8', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'nilai_pipeline', NULL, 'Rp 0.00', '2026-01-24 03:57:53'),
	('e049682a-c686-483d-ad4c-a09295b16d12', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'omset', NULL, 'Rp 45,000,000.00', '2026-01-24 03:57:53'),
	('ec1a825c-9d73-421c-9803-36305c1e34ba', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'tgl_closing', NULL, '25 Januari 2026', '2026-01-24 03:57:54'),
	('fc191f56-0cb8-433f-94a2-37fd3420a014', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'img_closing', NULL, 'http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg', '2026-01-24 03:57:53'),
	('fc2c3a2b-b750-486c-9667-40db3a8c5261', 'd4a2352f-b2e5-4830-a6b5-6b94a95bae2d', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'no_tlp', '081238664317', '081238664316', '2026-01-24 03:36:03');

-- Dumping structure for table furnifilux_dev.leads_options
CREATE TABLE IF NOT EXISTS `leads_options` (
  `uuid` char(50) NOT NULL,
  `category` enum('kota','kategori_produk','sumber_leads','cs') NOT NULL,
  `value` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_option` (`category`,`value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.leads_options: ~55 rows (approximately)
INSERT INTO `leads_options` (`uuid`, `category`, `value`, `created_at`) VALUES
	('0731e47d-178b-4cc7-abb3-ca7a47984550', 'kategori_produk', 'sdsd', '2025-10-30 07:58:01'),
	('0ba13490-d8d6-4016-a68a-fe0b75fb95c9', 'kota', 'Makasar', '2025-12-16 05:16:02'),
	('0ed7681c-988e-4ce8-8d7f-75023c8e1537', 'kategori_produk', 'Oke', '2025-10-28 08:10:00'),
	('156cef6b-b302-11f0-8cf4-00163e93e03c', 'cs', 'CS D', '2025-10-27 07:56:37'),
	('156d431e-b302-11f0-8cf4-00163e93e03c', 'cs', 'CS E', '2025-10-27 07:56:37'),
	('156d4474-b302-11f0-8cf4-00163e93e03c', 'cs', 'CS F', '2025-10-27 07:56:37'),
	('156d44dd-b302-11f0-8cf4-00163e93e03c', 'cs', 'Customer Service 1', '2025-10-27 07:56:37'),
	('156d4773-b302-11f0-8cf4-00163e93e03c', 'cs', 'Customer Service 2', '2025-10-27 07:56:37'),
	('156d4878-b302-11f0-8cf4-00163e93e03c', 'cs', 'Admin', '2025-10-27 07:56:37'),
	('37a33ea3-960d-4ccb-bc78-9b07990ac963', 'kategori_produk', 'Exterior Luar', '2026-01-24 03:06:35'),
	('49c0cad2-5e83-417b-8eb7-a3bfe5bc54c2', 'kota', 'Papua', '2025-10-28 08:09:42'),
	('4b5f7a9e-bfe5-47b7-bbea-536ed5b68807', 'sumber_leads', 'Twitter', '2026-01-24 03:06:45'),
	('550e8400-e29b-41d4-a716-446655442001', 'kota', 'Jakarta', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442002', 'kota', 'Bogor', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442003', 'kota', 'Depok', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442004', 'kota', 'Tangerang', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442005', 'kota', 'Bekasi', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442006', 'kota', 'Bandung', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442007', 'kota', 'Jogja', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442008', 'kategori_produk', 'Desain', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442009', 'kategori_produk', 'Kitchen Set', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442010', 'kategori_produk', 'Interior', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442011', 'kategori_produk', 'Exterior', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442012', 'kategori_produk', 'Lainnya', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442013', 'sumber_leads', 'Google Ads', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442014', 'sumber_leads', 'Instagram', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442015', 'sumber_leads', 'Facebook', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442016', 'sumber_leads', 'Referral', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442017', 'sumber_leads', 'Tokopedia', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442018', 'sumber_leads', 'Shopee', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442019', 'sumber_leads', 'Website', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442020', 'cs', 'CS A', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442021', 'cs', 'CS B', '2025-10-27 04:24:06'),
	('550e8400-e29b-41d4-a716-446655442022', 'cs', 'CS C', '2025-10-27 04:24:06'),
	('56b3d280-b302-11f0-8cf4-00163e93e03c', 'kota', 'Surabaya', '2025-10-27 07:58:27'),
	('56b570c3-b302-11f0-8cf4-00163e93e03c', 'kota', 'Medan', '2025-10-27 07:58:27'),
	('56b572ce-b302-11f0-8cf4-00163e93e03c', 'kota', 'Semarang', '2025-10-27 07:58:27'),
	('56b57335-b302-11f0-8cf4-00163e93e03c', 'kota', 'Makassar', '2025-10-27 07:58:27'),
	('56b57385-b302-11f0-8cf4-00163e93e03c', 'kota', 'Palembang', '2025-10-27 07:58:27'),
	('56b5a409-b302-11f0-8cf4-00163e93e03c', 'kota', 'Denpasar', '2025-10-27 07:58:27'),
	('56b5a5b8-b302-11f0-8cf4-00163e93e03c', 'kota', 'Batam', '2025-10-27 07:58:27'),
	('56b5a644-b302-11f0-8cf4-00163e93e03c', 'kota', 'Pekanbaru', '2025-10-27 07:58:27'),
	('56b5a6bf-b302-11f0-8cf4-00163e93e03c', 'kota', 'Surakarta', '2025-10-27 07:58:27'),
	('56b5a73f-b302-11f0-8cf4-00163e93e03c', 'kota', 'Malang', '2025-10-27 07:58:27'),
	('56b5a828-b302-11f0-8cf4-00163e93e03c', 'kota', 'Yogyakarta', '2025-10-27 07:58:27'),
	('56b5a8b7-b302-11f0-8cf4-00163e93e03c', 'kota', 'Padang', '2025-10-27 07:58:27'),
	('56b5a90a-b302-11f0-8cf4-00163e93e03c', 'kota', 'Bandar Lampung', '2025-10-27 07:58:27'),
	('56b5a994-b302-11f0-8cf4-00163e93e03c', 'kota', 'Pontianak', '2025-10-27 07:58:27'),
	('56b5a9e1-b302-11f0-8cf4-00163e93e03c', 'kota', 'Balikpapan', '2025-10-27 07:58:27'),
	('56b5aa24-b302-11f0-8cf4-00163e93e03c', 'kota', 'Manado', '2025-10-27 07:58:27'),
	('56b5aa6b-b302-11f0-8cf4-00163e93e03c', 'kota', 'Cirebon', '2025-10-27 07:58:27'),
	('56b5aaad-b302-11f0-8cf4-00163e93e03c', 'kota', 'Sukabumi', '2025-10-27 07:58:27'),
	('56b5aae8-b302-11f0-8cf4-00163e93e03c', 'kota', 'Tasikmalaya', '2025-10-27 07:58:27'),
	('56b5ac2e-b302-11f0-8cf4-00163e93e03c', 'kota', 'Binjai', '2025-10-27 07:58:27'),
	('6c554b76-20ec-44f7-b956-a49b2ab675f0', 'cs', 'Customer Service 3', '2025-12-16 04:42:31'),
	('8c68f293-1f14-4412-9345-7fe62e90cc9b', 'cs', 'CS G', '2026-01-24 03:07:07'),
	('c2bb96a5-c0c8-4b75-aa59-027870b5f1de', 'kota', 'Bali', '2025-10-30 04:16:46'),
	('f39415c5-42ca-4754-8f4f-80cce7c74bfb', 'kota', 'Maluku Utara', '2026-01-24 03:06:08');

-- Dumping structure for table furnifilux_dev.log
CREATE TABLE IF NOT EXISTS `log` (
  `uuid` char(50) NOT NULL,
  `message` text NOT NULL,
  `kind` varchar(255) DEFAULT NULL,
  `user_id` char(50) NOT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_log_user` (`user_id`),
  KEY `idx_log_kind` (`kind`),
  KEY `idx_log_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.log: ~5 rows (approximately)
INSERT INTO `log` (`uuid`, `message`, `kind`, `user_id`, `created_at`) VALUES
	('0d933d3b-cbeb-4702-9fa3-10a3a24a71c2', 'Mengubah lead: Arif Budi Setiawan (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 03:36:02'),
	('169d69a7-531b-4e80-8ba1-0e1c75decfea', 'Mengubah lead: Arif Budi (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 03:10:46'),
	('170d6121-07c2-4f1b-90f8-d8ecb919702a', 'Menambahkan opsi lokasi: Maluku', 'Finance Options', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 03:51:04'),
	('1911f1ed-e3aa-4c86-a13f-c01a9af40f7c', 'Menambahkan akun baru: BRI UTAMA', 'akun', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 03:50:05'),
	('2938bf7c-a48d-4ec7-975d-b77d0131d57a', 'Mengubah lead: Arif Budi (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 03:19:25'),
	('45a1eca3-e415-4308-acb9-46cd08f39f90', 'Menambahkan lead baru: Muhammad Ichsan (LEADS#1000)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 03:07:22'),
	('6d737149-927d-4d21-b274-0a56f1fd8425', 'Mengubah lead: Arif Budi (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 03:20:14'),
	('84c080b5-1251-43a6-b388-b311dece3960', 'Menambahkan lead baru: Arif Budi (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 03:08:09'),
	('c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Mengubah lead: Arif Budi Setiawan (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 03:57:51'),
	('d3801f4e-a6c6-4145-8745-41418c810963', 'Menambahkan opsi vendor: Vendor C', 'Finance Options', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 03:50:55'),
	('dfd8286d-8404-426a-9794-f2952a3370be', 'Menambahkan transaksi keuangan dengan nomor <b>FINANCE#1001</b> (Pendapatan - Rp 30000000)', 'Finance', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 04:09:49'),
	('f363c1be-bc96-4dda-809c-22c257de6692', 'Menambahkan transaksi keuangan dengan nomor <b>FINANCE#1000</b> (Proyek - Rp 45000)', 'Finance', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 04:02:36');

-- Dumping structure for table furnifilux_dev.log_item
CREATE TABLE IF NOT EXISTS `log_item` (
  `uuid` char(50) NOT NULL,
  `log_id` char(50) NOT NULL,
  `message` text NOT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_log_item_log_id` (`log_id`),
  CONSTRAINT `fk_log_item_log` FOREIGN KEY (`log_id`) REFERENCES `log` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.log_item: ~34 rows (approximately)
INSERT INTO `log_item` (`uuid`, `log_id`, `message`, `created_at`) VALUES
	('00361588-73c2-4b38-982c-5bea34c7dc63', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Kategori Arus Kas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Operasional</span>', '2026-01-24 04:02:36'),
	('0079c341-e090-45ce-90be-1fb5f6adaafa', '169d69a7-531b-4e80-8ba1-0e1c75decfea', 'Data \'Status\' berubah dari <span class=\'font-semibold\'>New Leads</span> ke <span class=\'font-semibold\'>Follow Up</span>', '2026-01-24 03:10:47'),
	('014161e7-a473-45c1-a749-84c7fc01ade2', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Umur Lead\' ditambahkan dengan nilai <span class=\'font-semibold\'>1</span>', '2026-01-24 03:08:11'),
	('02f092c2-d6a5-466f-abae-bf10213dce6f', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Alamat\' ditambahkan dengan nilai <span class=\'font-semibold\'>Jalan mawar\nSelomartani</span>', '2026-01-24 03:08:09'),
	('03023fe0-e46b-4b6b-8961-faad892afff7', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Status\' ditambahkan dengan nilai <span class=\'font-semibold\'>New Leads</span>', '2026-01-24 03:08:10'),
	('04360088-182c-41f2-b02a-499bf0f3d4c0', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Bulan\' ditambahkan dengan nilai <span class=\'font-semibold\'>Januari 2026</span>', '2026-01-24 04:09:50'),
	('0601dc2b-7d74-4bd5-badd-23b274475ba4', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Keterangan\' ditambahkan dengan nilai <span class=\'font-semibold\'>Ini Leads Baru</span>', '2026-01-24 03:07:23'),
	('1004e14c-5d96-4fc4-b071-e0ad25da20fb', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Kategori Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-24 04:09:50'),
	('17c8948e-6ab7-408c-904c-9b6710194860', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Jam Masuk\' ditambahkan dengan nilai <span class=\'font-semibold\'>3</span>', '2026-01-24 03:07:22'),
	('1a75eff5-469f-48ed-9fd0-00f1d87a260b', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Umur Lead\' ditambahkan dengan nilai <span class=\'font-semibold\'>0</span>', '2026-01-24 03:07:23'),
	('25bf5a07-39a2-4207-9dc5-9425940fa86c', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Perkiraan HPP\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 35,000,000.00</span>', '2026-01-24 03:57:52'),
	('26142e6a-c712-45bf-9079-a2d1dfcd561d', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Status\' ditambahkan dengan nilai <span class=\'font-semibold\'>New Leads</span>', '2026-01-24 03:07:23'),
	('2a507f46-fb61-4a5e-9151-93d366ee45ef', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Arif Budi Setiawan</span>', '2026-01-24 04:09:50'),
	('310a9c33-2adc-469a-951e-9bb8a4cbaeb9', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Kategori Arus Kas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Operasional</span>', '2026-01-24 04:09:50'),
	('3d5b5377-670f-422b-a108-45eb07e51c38', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'No Telp\' ditambahkan dengan nilai <span class=\'font-semibold\'>081238664315</span>', '2026-01-24 03:08:10'),
	('3f263fb4-3885-44cf-9ce9-80d942d1637e', '2938bf7c-a48d-4ec7-975d-b77d0131d57a', 'Data \'No Telp\' berubah dari <span class=\'font-semibold\'>081238664315</span> ke <span class=\'font-semibold\'>081238664317</span>', '2026-01-24 03:19:25'),
	('4108fd39-a76e-4a3b-8ace-69576bfee73e', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Kota\' ditambahkan dengan nilai <span class=\'font-semibold\'>Balikpapan</span>', '2026-01-24 03:08:10'),
	('44c97ff1-9c49-4efb-ba4e-325064d927b7', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Sumber Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Instagram</span>', '2026-01-24 03:08:10'),
	('4733d3b1-5521-484c-b073-ce330e035ca4', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Item <b>Pembayaran pertama</b> ditambahkan: Qty 1, Harga Rp 25,000,000.00, Catatan: -', '2026-01-24 04:09:51'),
	('47fa67b2-0d16-4b93-a815-0dd7bd2c56da', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Kategori Aktivitas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Proyek</span>', '2026-01-24 04:02:36'),
	('48b69c90-18e1-468c-943e-1a95fbaa0873', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Sub Kategori\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pembayaran Client</span>', '2026-01-24 04:09:51'),
	('5105b3fe-2a25-40bd-a634-f8c4e725f134', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Kategori Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pengeluaran</span>', '2026-01-24 04:02:36'),
	('514328f8-c031-48a8-a1db-6086df4c70d8', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Kategori Utama\' ditambahkan dengan nilai <span class=\'font-semibold\'>Proyek</span>', '2026-01-24 04:02:36'),
	('53f82598-7098-49f7-b962-35579b2f2c5f', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Alamat\' ditambahkan dengan nilai <span class=\'font-semibold\'>Jalan mawar\nSelomartani</span>', '2026-01-24 03:07:22'),
	('54121b09-a216-462a-b25e-2b963ef30211', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Jatuh Tempo\' ditambahkan dengan nilai <span class=\'font-semibold\'>25 Januari 2026</span>', '2026-01-24 04:02:36'),
	('5698d85a-eed3-4eec-93aa-7947afad200c', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Lokasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Bandung</span>', '2026-01-24 04:02:37'),
	('5b96749f-277f-4090-80a1-7ea40ea4f2c7', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>23 Januari 2026</span>', '2026-01-24 03:08:10'),
	('5c75735d-5c61-4aae-a8e8-679a1ed7439d', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Tanggal Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>25 Januari 2026</span>', '2026-01-24 03:57:53'),
	('5ea29d27-a69e-4964-a8bc-bf4b9b5e1326', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Durasi Hari Hingga Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>2</span>', '2026-01-24 03:57:52'),
	('61122717-ec65-43d7-a328-cec137f2492c', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Status Kategorisasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Active</span>', '2026-01-24 03:08:10'),
	('6118adf1-9816-4357-b355-9a135676069b', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Kategori Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Exterior Luar</span>', '2026-01-24 03:07:23'),
	('6279b59c-f88d-40fa-a897-29335f5d8c2d', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Status Kategorisasi\' berubah dari <span class=\'font-semibold\'>Active</span> ke <span class=\'font-semibold\'>Closed</span>', '2026-01-24 03:57:53'),
	('6547a3c3-d588-48eb-b85e-43f5558a95f2', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>24 Januari 2026</span>', '2026-01-24 04:09:51'),
	('668aaa2d-4347-45cc-81e2-399cefe86e2e', '6d737149-927d-4d21-b274-0a56f1fd8425', 'Data \'Status\' berubah dari <span class=\'font-semibold\'>Follow Up</span> ke <span class=\'font-semibold\'>Survey</span>', '2026-01-24 03:20:14'),
	('672a187b-a12d-4250-a071-5855a56eff57', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Jam Masuk\' ditambahkan dengan nilai <span class=\'font-semibold\'>5</span>', '2026-01-24 03:08:09'),
	('698facc2-1ff2-4a0c-8204-98f3e82a0ddf', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Keterangan\' ditambahkan dengan nilai <span class=\'font-semibold\'>oke</span>', '2026-01-24 04:02:37'),
	('6a383e04-959e-4a28-af5f-d9a379af378b', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Status Kategorisasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Active</span>', '2026-01-24 03:07:23'),
	('6e88f5eb-74da-405a-bc20-92248cbc46e7', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Jatuh Tempo\' ditambahkan dengan nilai <span class=\'font-semibold\'>25 Januari 2026</span>', '2026-01-24 04:09:50'),
	('6ef4cfce-c2a0-43d4-909b-2bd4dc84c91a', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Akun ID\' ditambahkan dengan nilai <span class=\'font-semibold\'>BRI UTAMA</span>', '2026-01-24 04:02:36'),
	('6f581ad4-7699-4c99-92de-f2528990fd18', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Sub Kategori\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pengiriman</span>', '2026-01-24 04:02:37'),
	('71c3df7a-453c-4f9e-8fdc-f0f973345262', '169d69a7-531b-4e80-8ba1-0e1c75decfea', 'Data \'Hari Sejak FU Terakhir\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>0</span>', '2026-01-24 03:10:46'),
	('71d383be-e672-4627-a920-a03c63414fcb', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Kategori Utama\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-24 04:09:50'),
	('7208cb45-a691-4a2d-ae8c-6f928e2e68c4', '169d69a7-531b-4e80-8ba1-0e1c75decfea', 'Data \'Total Follow Up\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>1</span>', '2026-01-24 03:10:47'),
	('7d161a66-1e1c-40d5-a6b3-ef5c78d46a84', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Akun ID\' ditambahkan dengan nilai <span class=\'font-semibold\'>BRI UTAMA</span>', '2026-01-24 04:09:50'),
	('7ff12b3c-f20b-48d3-a56a-10cf84d35fdb', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Status\' berubah dari <span class=\'font-semibold\'>Survey</span> ke <span class=\'font-semibold\'>Closing</span>', '2026-01-24 03:57:52'),
	('8004607d-34c3-4d42-8537-88b7cb3038b7', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Sisa Pembayaran\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 45,000,000.00</span>', '2026-01-24 03:57:52'),
	('8843c0bd-4b99-457b-a889-91545f51471f', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Nominal\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 30,000,000.00</span>', '2026-01-24 04:09:51'),
	('8aecfe7e-d027-4182-8975-6203e7d20e7b', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Jenis Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Debit</span>', '2026-01-24 04:02:36'),
	('8b27576d-e01e-45ca-a256-07a2c6e71ca0', '0d933d3b-cbeb-4702-9fa3-10a3a24a71c2', 'Data \'Alamat\' berubah dari <span class=\'font-semibold\'>Jalan mawar\nSelomartani</span> ke <span class=\'font-semibold\'>Jalan mawar\nSelomartani 2</span>', '2026-01-24 03:36:02'),
	('8c440999-b72d-44b5-abea-0f81aee6ec41', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Debit\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 45,000.00</span>', '2026-01-24 04:02:36'),
	('8f8ee90a-9095-4410-894c-ad9cf75c8baf', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Nama Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Arif Budi</span>', '2026-01-24 03:08:10'),
	('9271dd7c-2e5a-4406-9587-5aa79acf2caf', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Item <b>Pengiriman barang semen</b> ditambahkan: Qty 1, Harga Rp 25,000.00, Catatan: -', '2026-01-24 04:02:37'),
	('952a5966-5a2b-4a5f-82e9-c3c24d6c093e', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Nominal\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 45,000.00</span>', '2026-01-24 04:02:37'),
	('96e25e2d-d0d1-4469-8141-d0322acb8a6c', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Foto Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg</span>', '2026-01-24 03:57:52'),
	('9cff7d00-b1b3-41b8-929f-24c23c7d9b57', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Sumber Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Twitter</span>', '2026-01-24 03:07:23'),
	('9dda2710-7438-4b2d-aa05-ec0a47173d7b', '6d737149-927d-4d21-b274-0a56f1fd8425', 'Data \'Foto Survey\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg</span>', '2026-01-24 03:20:14'),
	('a1381103-eb15-46d4-bb34-28f6789b6f56', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Kota\' ditambahkan dengan nilai <span class=\'font-semibold\'>Maluku Utara</span>', '2026-01-24 03:07:23'),
	('a3f9a561-d161-454d-b605-b3346cf03ed9', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Omset\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 45,000,000.00</span>', '2026-01-24 03:57:52'),
	('a70b89c2-9003-46b7-b0f7-c29eb48eb337', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Vendor\' ditambahkan dengan nilai <span class=\'font-semibold\'>Vendor A</span>', '2026-01-24 04:02:37'),
	('ae0fe8ea-2f49-44a6-a016-f8c5a75e7df8', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'No Telp\' ditambahkan dengan nilai <span class=\'font-semibold\'>081325184866</span>', '2026-01-24 03:07:23'),
	('ae51185e-2137-4ec6-9b71-1a08b38bd567', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Kredit\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 30,000,000.00</span>', '2026-01-24 04:09:51'),
	('af181ca4-eb69-4f14-a20d-32c01562fe30', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Item <b>Pembayaran #2</b> ditambahkan: Qty 1, Harga Rp 5,000,000.00, Catatan: -', '2026-01-24 04:09:52'),
	('b3223b90-65b6-47f5-9e9c-8981fa7234d7', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Kategori Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Customer Leads</span>', '2026-01-24 03:07:22'),
	('b6dfeb20-60f9-4e94-bec0-509dfa656953', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Keterangan\' ditambahkan dengan nilai <span class=\'font-semibold\'>baru mebel luar</span>', '2026-01-24 03:08:10'),
	('ba722e06-542e-4e84-8a5c-315a151ff672', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Bulan\' ditambahkan dengan nilai <span class=\'font-semibold\'>Januari 2026</span>', '2026-01-24 04:02:36'),
	('ba909e4b-d481-4643-a181-e6ed953f80d7', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Mebel Luar</span>', '2026-01-24 03:08:10'),
	('bf7f3b6e-4f0f-419f-bc2c-308db3a38a3c', '0d933d3b-cbeb-4702-9fa3-10a3a24a71c2', 'Data \'No Telp\' berubah dari <span class=\'font-semibold\'>081238664317</span> ke <span class=\'font-semibold\'>081238664316</span>', '2026-01-24 03:36:02'),
	('c0a8b310-5216-4933-9d39-230aa5a745fe', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Mebel Furniture</span>', '2026-01-24 03:07:23'),
	('c50a683f-cbaa-44d9-a7c5-ae36dfbeb492', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Item <b>Pengiriman cat 2 kali</b> ditambahkan: Qty 1, Harga Rp 20,000.00, Catatan: -', '2026-01-24 04:02:37'),
	('c56a0e4b-5c75-4f02-b400-e498c30bc555', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Arif Budi Setiawan</span>', '2026-01-24 04:02:37'),
	('caada27a-d7e5-47aa-b826-78909ccd0821', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Kategori Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Exterior</span>', '2026-01-24 03:08:10'),
	('ccfbe7e7-ab02-42a3-b5f6-d570c9b8d8c1', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Lokasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Bandung</span>', '2026-01-24 04:09:51'),
	('cd23ff70-5f5f-4e61-a6e3-59557f47dfbf', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Harga Jual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 45,000,000.00</span>', '2026-01-24 03:57:52'),
	('cfda3250-b748-4a6f-8f42-3775ea055bfb', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'Kategori Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Customer Leads</span>', '2026-01-24 03:08:10'),
	('d02d1197-04e6-4020-92ad-65e2623dd99c', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Nilai Pipeline\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 0.00</span>', '2026-01-24 03:57:52'),
	('d2129973-e6c7-42dd-ac79-0d08225eb828', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'CS\' ditambahkan dengan nilai <span class=\'font-semibold\'>CS G</span>', '2026-01-24 03:07:22'),
	('d3b430ed-a6ff-48f5-b4ec-423ca8dbb2df', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Nama Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Muhammad Ichsan</span>', '2026-01-24 03:07:23'),
	('d4aa48d9-87aa-4ea0-a1fb-a4e395834527', '84c080b5-1251-43a6-b388-b311dece3960', 'Data \'CS\' ditambahkan dengan nilai <span class=\'font-semibold\'>CS B</span>', '2026-01-24 03:08:09'),
	('d9b279f6-2224-426b-8463-7c01b9a4c379', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Arah Uang\' ditambahkan dengan nilai <span class=\'font-semibold\'>Masuk</span>', '2026-01-24 04:09:50'),
	('ddb6b29a-97d1-4dd4-906a-f1fd63708285', '169d69a7-531b-4e80-8ba1-0e1c75decfea', 'Data \'HPP Aktual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 0.00</span>', '2026-01-24 03:10:46'),
	('de87349b-efcc-43ef-9bba-814fe2dcc947', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Kategori Aktivitas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-24 04:09:50'),
	('dec67f3d-8a0b-400f-a607-8477d14abb27', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Arah Uang\' ditambahkan dengan nilai <span class=\'font-semibold\'>Keluar</span>', '2026-01-24 04:02:36'),
	('e839220b-efc8-46a2-93a9-632962d1864a', 'c17386bf-0154-4f5e-954f-afb9f9d8eab9', 'Data \'Margin Aktual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 45,000,000.00</span>', '2026-01-24 03:57:52'),
	('f6026070-ad73-4e6f-82c3-6a1f9a34f84a', '0d933d3b-cbeb-4702-9fa3-10a3a24a71c2', 'Data \'Nama Klien\' berubah dari <span class=\'font-semibold\'>Arif Budi</span> ke <span class=\'font-semibold\'>Arif Budi Setiawan</span>', '2026-01-24 03:36:02'),
	('f729efdb-efbe-42fa-ab9e-6cdc7a953adb', 'f363c1be-bc96-4dda-809c-22c257de6692', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>24 Januari 2026</span>', '2026-01-24 04:02:37'),
	('f8906a15-a41b-4954-ab71-8b42d23ea613', '45a1eca3-e415-4308-acb9-46cd08f39f90', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>24 Januari 2026</span>', '2026-01-24 03:07:23'),
	('f8ebafbf-9c18-4c8c-b9a3-14475dfd5bb9', 'dfd8286d-8404-426a-9794-f2952a3370be', 'Data \'Jenis Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Kredit</span>', '2026-01-24 04:09:50');

-- Dumping structure for table furnifilux_dev.media
CREATE TABLE IF NOT EXISTS `media` (
  `uuid` char(50) NOT NULL,
  `user_id` char(50) NOT NULL,
  `url` varchar(500) NOT NULL,
  `kind` varchar(50) DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_media_user` (`user_id`),
  KEY `idx_media_created` (`created_at`),
  CONSTRAINT `fk_media_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`uuid`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.media: ~7 rows (approximately)
INSERT INTO `media` (`uuid`, `user_id`, `url`, `kind`, `created_at`) VALUES
	('1641aa3a-2ac1-44b8-b84d-ab99dfb9184e', '550e8400-e29b-41d4-a716-446655440003', 'https://i.pinimg.com/1200x/96/be/d6/96bed63e541a937c3ce6f51850ece087.jpg', 'image', '2025-12-16 05:24:55'),
	('1ce3f52d-bebd-4eff-9fa9-8424e8aa5077', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'http://localhost:8002/media/public/oAxRld_20251217_130548.png', 'image', '2025-12-17 06:05:49'),
	('30d66007-516b-4c68-b35e-baaba9e4d4f5', '550e8400-e29b-41d4-a716-446655440003', 'https://furnivilux.namia.online/api/media/public/lqjuQr_20251216_141714.jpg', 'image', '2025-12-16 07:17:14'),
	('669da5c4-36e6-44a8-a663-c103cf4eb6a4', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg', 'image', '2026-01-24 03:20:04'),
	('78e911a8-5495-4cd0-acbb-a4e2d4439c57', '550e8400-e29b-41d4-a716-446655440003', 'https://furnivilux.namia.online/api/media/public/i4Bjjf_20251216_124310.jpg', 'image', '2025-12-16 05:43:10'),
	('8b2e59f6-f8f2-420e-af73-35d7f6992802', '550e8400-e29b-41d4-a716-446655440003', 'https://i.pinimg.com/1200x/c8/01/7b/c8017b681e08f1c747852dc0c870fd45.jpg', 'image', '2025-12-16 05:24:55'),
	('c0e802c1-4227-4083-9327-3a4e0fa69dbd', '550e8400-e29b-41d4-a716-446655440003', 'http://i.pinimg.com/736x/62/3a/5c/623a5cd339cad02e6bbf0c3421325725.jpg', 'image', '2025-12-16 05:24:55'),
	('c0e802c1-4227-4083-9327-3a4e0fa69dbk', '550e8400-e29b-41d4-a716-446655440003', 'https://i.pinimg.com/1200x/d8/52/48/d85248983d5bb1bc201a1d6c4e692671.jpg', 'image', '2025-12-16 05:24:55');

-- Dumping structure for table furnifilux_dev.number_id
CREATE TABLE IF NOT EXISTS `number_id` (
  `kind` varchar(50) NOT NULL,
  `latest_number` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Dumping data for table furnifilux_dev.number_id: ~3 rows (approximately)
INSERT INTO `number_id` (`kind`, `latest_number`, `created_at`, `updated_at`) VALUES
	('akun', 1001, '2025-11-07 08:30:19', '2026-01-24 03:50:04'),
	('finance_transactions', 1002, '2025-10-27 07:39:40', '2026-01-24 04:09:48'),
	('leads', 1002, '2025-10-27 07:39:40', '2026-01-24 03:08:09');

-- Dumping structure for table furnifilux_dev.roles
CREATE TABLE IF NOT EXISTS `roles` (
  `uuid` char(50) NOT NULL,
  `name` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `status` enum('Active','Inactive') DEFAULT 'Active',
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `pages` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Array of allowed page access',
  `features` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Array of allowed features',
  `menu` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Array of allowed menu items',
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_roles_name` (`name`),
  KEY `idx_roles_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.roles: ~6 rows (approximately)
INSERT INTO `roles` (`uuid`, `name`, `description`, `status`, `created_at`, `updated_at`, `pages`, `features`, `menu`) VALUES
	('23098553-2423-4411-9234-938245023945', 'Owner', 'Business Owner', 'Active', '2025-12-16 09:12:02', '2025-12-17 05:57:07', '["profile","keuangan---daftar","keuangan---detail","leads-list","leads-detail","report-list","report-penjualan","report-laba-rugi","report-transaksi-klien","report-pengeluaran-klien","notifications","leads-add","leads-edit","finance-list","finance-add","finance-detail","finance-edit","accounts-list","accounts-add","accounts-detail","accounts-edit","reports-list","reports-sales","reports-client-transactions","reports-profit-loss","reports-client-expenses","logs-list","logs-detail"]', '["view-finance","view-leads","view-report","add-leads","edit-leads","add-finance","edit-finance","add-accounts","edit-accounts","view-accounts","view-reports","view-logs"]', '["finance","leads","laporan","notifikasi","accounts","reports","logs"]'),
	('550e8400-e29b-41d4-a716-446655440001', 'Admin', 'Full system access', 'Active', '2025-10-27 04:24:09', '2025-12-17 04:18:42', '["profile", "finance-list", "finance-add", "finance-edit", "finance-detail", "accounts-list", "accounts-add", "accounts-edit", "accounts-detail", "leads-list", "leads-add", "leads-edit", "leads-detail", "reports-list", "reports-sales", "reports-profit-loss", "reports-client-transactions", "reports-client-expenses", "logs-list", "logs-detail", "role-permission"]', '["add-finance", "edit-finance", "view-finance", "add-leads", "edit-leads", "view-leads", "add-accounts", "edit-accounts", "view-accounts", "view-reports", "view-logs", "manage-roles"]', '["finance", "accounts", "leads", "reports", "logs", "role-permission"]'),
	('94354894-3531-411a-821f-998845244501', 'Keuangan', 'Finance operations', 'Active', '2025-12-16 09:12:02', '2025-12-17 05:55:28', '["profile","keuangan---daftar","keuangan---tambah","keuangan---edit","keuangan---detail","report-list","report-penjualan","report-laba-rugi","report-transaksi-klien","report-pengeluaran-klien","notifications","finance-list","finance-add","finance-detail","finance-edit","accounts-list","accounts-detail","reports-list","reports-profit-loss","reports-client-expenses","reports-sales","reports-client-transactions"]', '["edit-finance","view-finance","delete-finance-request","view-report","add-finance","view-accounts","view-reports"]', '["finance","laporan","notifikasi","reports","accounts"]'),
	('c2834bba-9588-444a-9524-118464332512', 'CS', 'Customer Service', 'Active', '2025-12-16 09:12:02', '2025-12-17 05:25:50', '["profile","keuangan---daftar","keuangan---detail","leads-list","leads-add","leads-edit","leads-detail","report-list","report-penjualan","report-laba-rugi","report-transaksi-klien","report-pengeluaran-klien","notifications"]', '["add-leads","edit-leads","view-leads","view-report"]', '["leads","laporan","notifikasi"]'),
	('e4321234-1234-4321-1234-554433221100', 'PM', 'Project Manager', 'Active', '2025-12-16 09:12:02', '2025-12-18 03:13:24', '["profile","leads-list","leads-detail","report-list","report-penjualan","report-laba-rugi","report-transaksi-klien","report-pengeluaran-klien","notifications","leads-edit"]', '["view-leads","view-report","edit-leads"]', '["leads","laporan","notifikasi"]'),
	('f81d4fae-7dec-11d0-a765-00a0c91e6bf6', 'Direktur', 'Director', 'Active', '2025-12-16 09:12:02', '2025-12-17 05:54:13', '["profile","keuangan---daftar","keuangan---tambah","keuangan---edit","keuangan---detail","akun---daftar","akun---tambah","akun---edit","akun---detail","leads-list","leads-detail","report-list","report-penjualan","report-laba-rugi","report-transaksi-klien","report-pengeluaran-klien","notifications","log-list","log-detail","finance-list","finance-detail","accounts-list","accounts-detail","reports-list","reports-sales","reports-client-transactions","reports-client-expenses","logs-list","logs-detail","reports-profit-loss"]', '["view-finance","delete-finance","approve-finance","view-leads","delete-leads","approve-leads","add-akun","edit-akun","delete-akun","view-report","view-log","view-accounts","view-reports","view-logs"]', '["finance","akun","leads","laporan","log","notifikasi","accounts","reports","logs"]');

-- Dumping structure for table furnifilux_dev.users
CREATE TABLE IF NOT EXISTS `users` (
  `uuid` char(50) NOT NULL,
  `username` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `role_id` char(50) DEFAULT NULL,
  `status` enum('Active','Inactive') DEFAULT 'Active',
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `idx_users_username` (`username`),
  UNIQUE KEY `idx_users_email` (`email`),
  KEY `idx_users_role` (`role_id`),
  KEY `idx_users_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.users: ~6 rows (approximately)
INSERT INTO `users` (`uuid`, `username`, `email`, `password`, `name`, `role_id`, `status`, `created_at`, `updated_at`) VALUES
	('550e8400-e29b-41d4-a716-446655440003', 'admin', 'admin@furnifilux.com', 'admin123', 'System Administrator', '550e8400-e29b-41d4-a716-446655440001', 'Active', '2025-10-27 04:29:05', '2025-10-27 04:33:39'),
	('a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'finance_user', 'finance@furnifilux.com', 'admin123', 'Staf Keuangan', '94354894-3531-411a-821f-998845244501', 'Active', '2025-12-16 09:15:53', '2025-12-16 09:19:58'),
	('b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'cs_user', 'cs@furnifilux.com', 'admin123', 'Customer Service', 'c2834bba-9588-444a-9524-118464332512', 'Active', '2025-12-16 09:15:53', '2025-12-16 09:19:59'),
	('c3d4e5f6-a1b2-4c3d-0e4f-5a6b7c8d9e0f', 'direktur_user', 'direktur@furnifilux.com', 'admin123', 'Bapak Direktur', 'f81d4fae-7dec-11d0-a765-00a0c91e6bf6', 'Active', '2025-12-16 09:15:53', '2025-12-16 09:20:01'),
	('d4e5f6a1-b2c3-4d4e-1f5a-6b7c8d9e0f1a', 'owner_user', 'owner@furnifilux.com', 'admin123', 'Pemilik Bisnis', '23098553-2423-4411-9234-938245023945', 'Active', '2025-12-16 09:15:53', '2025-12-16 09:20:02'),
	('e5f6a1b2-c3d4-4e5f-2a6b-7c8d9e0f1a2b', 'pm_user', 'pm@furnifilux.com', 'admin123', 'Project Manager', 'e4321234-1234-4321-1234-554433221100', 'Active', '2025-12-16 09:15:53', '2025-12-16 09:20:07');

/*!40103 SET TIME_ZONE=IFNULL(@OLD_TIME_ZONE, 'system') */;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
