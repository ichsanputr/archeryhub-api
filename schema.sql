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

-- Dumping data for table furnifilux_dev.akun: ~2 rows (approximately)
INSERT INTO `akun` (`uuid`, `id`, `name`, `description`, `balance`, `created_at`, `updated_at`) VALUES
	('bd51dedb-9bbd-4f9f-8213-9f055de6f7b7', 'AKUN#1000', 'BRI', NULL, 7600000, '2026-01-24 05:11:31', '2026-01-26 07:01:06'),
	('2dec47ee-2721-4740-9989-73c47a56788c', 'AKUN#1001', 'BCA', 'Akun BCA', 200000, '2026-01-26 01:34:44', '2026-01-26 01:34:44');

-- Dumping structure for table furnifilux_dev.finance_history_edit
CREATE TABLE IF NOT EXISTS `finance_history_edit` (
  `uuid` char(50) NOT NULL,
  `transaction_uuid` char(50) NOT NULL,
  `user_id` char(50) NOT NULL,
  `field_name` varchar(255) NOT NULL,
  `old_value` text DEFAULT NULL,
  `new_value` text DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  KEY `idx_history_transaction` (`transaction_uuid`),
  KEY `idx_history_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.finance_history_edit: ~5 rows (approximately)
INSERT INTO `finance_history_edit` (`uuid`, `transaction_uuid`, `user_id`, `field_name`, `old_value`, `new_value`, `created_at`) VALUES
	('3facae82-c857-43ac-8f0c-f09b606bbc6a', 'f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'keterangan', NULL, 'oke 1', '2026-01-24 04:38:13'),
	('45e41956-62df-42a2-9e17-c89bfc26ef27', 'f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'Item Pembayaran #1: Harga', 'Rp 25,000,000.00', 'Rp 20,000,000.00', '2026-01-24 04:42:14'),
	('78835b48-73b2-4100-93a3-98e17d0082b9', 'f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'kredit', 'Rp 30,000,000.00', 'Rp 25,000,000.00', '2026-01-24 04:42:14'),
	('f533aca9-2768-4eee-bb28-3d4bdae22ba0', 'f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'nominal', 'Rp 30,000,000.00', 'Rp 25,000,000.00', '2026-01-24 04:42:14'),
	('f57cebf2-211e-4659-8d97-3f5213f80abe', 'f6ddf578-f1fc-4f68-9353-e26acaf56e9f', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'Item Pembayaran pertama #1: Nama', 'Pembayaran pertama #1', 'Pembayaran #1', '2026-01-24 04:41:40');

-- Dumping structure for table furnifilux_dev.finance_options
CREATE TABLE IF NOT EXISTS `finance_options` (
  `uuid` char(50) NOT NULL,
  `category` enum('vendor','klien','proyek','lokasi','akun') NOT NULL,
  `value` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`uuid`),
  UNIQUE KEY `unique_finance_option` (`category`,`value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dumping data for table furnifilux_dev.finance_options: ~15 rows (approximately)
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

-- Dumping data for table furnifilux_dev.finance_transactions: ~3 rows (approximately)
INSERT INTO `finance_transactions` (`uuid`, `no`, `lead_pay_id`, `lead_proyek_id`, `tanggal`, `jatuh_tempo`, `jenis_transaksi`, `kategori_utama`, `sub_kategori`, `keterangan`, `nominal`, `akun_id`, `vendor`, `klien`, `lokasi`, `bulan`, `kategori_transaksi`, `kategori_arus_kas`, `kategori_aktivitas`, `arah_uang`, `debit`, `kredit`, `img_pembayaran`, `created_at`, `updated_at`) VALUES
	('2b5c5a9b-fdf5-47f6-bf81-d56e65e57d2c', 'FINANCE#1000', NULL, '982244ed-d91c-4fc5-b153-65b70a4c5891', '2026-01-24', '2026-01-29', 'Debit', 'Proyek', 'Material', NULL, 400000, 'bd51dedb-9bbd-4f9f-8213-9f055de6f7b7', 'HAndra / Marketing', 'Agung Firman', 'Jogja', 'Januari 2026', 'Pengeluaran', 'Operasional', 'Proyek', 'Keluar', 400000, NULL, 'https://furnivilux.namia.online/api/media/public/1gaUPY_20260124_121158.png', '2026-01-24 05:12:10', '2026-01-24 05:12:10'),
	('40061e55-4444-4f99-94f7-c8334ebd63e0', 'FINANCE#1002', 'bb6596ac-af30-4d38-9b93-5835de41755e', NULL, '2026-01-26', NULL, 'Kredit', 'Pendapatan', 'Pembayaran Client', NULL, 5000000, 'bd51dedb-9bbd-4f9f-8213-9f055de6f7b7', NULL, 'Muhammad Ichsan', NULL, 'Januari 2026', 'Pendapatan', 'Operasional', 'Pendapatan', 'Masuk', NULL, 5000000, 'https://furnivilux.namia.online/api/media/public/1gaUPY_20260124_121158.png', '2026-01-26 07:01:06', '2026-01-26 07:01:06'),
	('ee1f8edc-debd-4788-bc74-acd4a714b57e', 'FINANCE#1001', '982244ed-d91c-4fc5-b153-65b70a4c5891', NULL, '2026-01-24', '2026-01-27', 'Kredit', 'Pendapatan', 'Pembayaran Client', NULL, 2000000, 'bd51dedb-9bbd-4f9f-8213-9f055de6f7b7', NULL, 'Agung Firman', 'Jakarta', 'Januari 2026', 'Pendapatan', 'Operasional', 'Pendapatan', 'Masuk', NULL, 2000000, 'https://furnivilux.namia.online/api/media/public/1gaUPY_20260124_121158.png', '2026-01-24 05:12:59', '2026-01-24 05:12:59');

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

-- Dumping data for table furnifilux_dev.finance_transaction_items: ~3 rows (approximately)
INSERT INTO `finance_transaction_items` (`uuid`, `transaction_id`, `item_name`, `quantity`, `unit_price`, `subtotal`, `notes`, `created_at`, `updated_at`) VALUES
	('15d9d2c6-0004-459f-8cdb-30154261a769', 'ee1f8edc-debd-4788-bc74-acd4a714b57e', 'pembayaran #1', 1, 2000000, 2000000, '', '2026-01-24 05:12:59', '2026-01-24 05:12:59'),
	('65edeac6-c914-48d2-a747-f651ac11ecc2', '2b5c5a9b-fdf5-47f6-bf81-d56e65e57d2c', 'Kayu, Mebel', 1, 400000, 400000, '', '2026-01-24 05:12:10', '2026-01-24 05:12:10'),
	('db3f3875-2845-4792-a6fd-f149a944b9b6', '40061e55-4444-4f99-94f7-c8334ebd63e0', 'pembayaran dp #1', 1, 5000000, 5000000, '', '2026-01-26 07:01:06', '2026-01-26 07:01:06');

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
	('982244ed-d91c-4fc5-b153-65b70a4c5891', 'LEADS#1001', '2026-01-25', 4, 'Agung Firman', '0895417205060', 'Demangan, Selomartani, Kalasan, Sleman\nSelomartani', 'Bandar Lampung', 'Kayu Furni', 'Exterior Luar', 'Closing', 'Customer Leads', 'Closed', 'Instagram', 'CS B', '2026-01-25', NULL, NULL, '2026-01-26', 'Oke', NULL, NULL, NULL, NULL, NULL, 2500000, 2000000, 0, 2000000, 400000, 2500000, 500000, 2100000, -2, 1, 0, -1, NULL, NULL, NULL, 'https://furnivilux.namia.online/api/media/public/i4Bjjf_20251216_124310.jpg', NULL, '2026-01-24 04:56:35', '2026-01-24 05:12:59'),
	('bb6596ac-af30-4d38-9b93-5835de41755e', 'LEADS#1000', '2026-01-24', 4, 'Muhammad Ichsan', '081238664315', 'Jalan mawar\nSelomartani', 'Bandar Lampung', 'Mebel Bawah', 'Desain', 'Closing', 'Customer Leads', 'Closed', 'Instagram', 'CS B', '2026-01-26', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 80000000, 50000000, 0, 5000000, 0, 80000000, 75000000, 80000000, NULL, NULL, 2, 2, NULL, NULL, NULL, 'http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg', NULL, '2026-01-24 04:55:07', '2026-01-26 07:01:06');

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

-- Dumping data for table furnifilux_dev.leads_history_edit: ~30 rows (approximately)
INSERT INTO `leads_history_edit` (`uuid`, `lead_uuid`, `user_id`, `field_name`, `old_value`, `new_value`, `created_at`) VALUES
	('0bf16549-f0a3-489d-b8c0-cb7b9cd0ccc6', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'perkiraan_hpp', NULL, 'Rp 50,000,000.00', '2026-01-26 06:58:06'),
	('0d35ed62-8c5c-4348-bf1d-bbf301759931', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'harga_jual', NULL, 'Rp 80,000,000.00', '2026-01-26 06:58:06'),
	('0e49971d-a94e-4fbb-bba5-fe8a6a60a721', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'status', 'Follow Up', 'Closing', '2026-01-24 05:04:20'),
	('104e5e03-86ea-44ff-8cb2-6b62a469eccf', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'tgl_closing', NULL, '26 Januari 2026', '2026-01-26 06:58:06'),
	('26600666-8842-4388-92b3-d1eda30cb41e', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'omset', NULL, 'Rp 2,500,000.00', '2026-01-24 05:04:20'),
	('281fad70-5abe-4b6d-8b24-fa6546c4e704', '982244ed-d91c-4fc5-b153-65b70a4c5891', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'hpp_aktual', NULL, 'Rp 0.00', '2026-01-24 04:57:10'),
	('2d519115-9067-4048-87e4-0a6717902ec0', '982244ed-d91c-4fc5-b153-65b70a4c5891', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'tgl_fu1', NULL, '26 Januari 2026', '2026-01-24 04:57:10'),
	('2ed57615-15c9-4a0e-a794-1733ed8787f8', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'margin_aktual', NULL, 'Rp 2,500,000.00', '2026-01-24 05:04:20'),
	('35241cba-2609-4cec-a8c8-4fa2c0235529', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'durasi_hingga_closing', NULL, '0', '2026-01-24 05:04:20'),
	('43247989-4105-4929-a050-0580e2d2e214', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'tgl_closing', NULL, '25 Januari 2026', '2026-01-24 05:04:20'),
	('5f12c406-862a-412c-bee1-0d250a31092d', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'status_kategorisasi', 'Active', 'Closed', '2026-01-26 06:58:06'),
	('60fbce26-ad18-44ce-bc4b-fec701e658fa', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'status', 'New Leads', 'Closing', '2026-01-26 06:58:06'),
	('69b35390-a168-4906-94d2-5f847c182f5e', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'hpp_aktual', NULL, 'Rp 0.00', '2026-01-26 06:58:06'),
	('8113045e-7a1f-4918-91a6-1350653516e6', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'img_closing', NULL, 'http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg', '2026-01-26 06:58:06'),
	('833faf20-c4e3-4935-9ace-c26c83c38050', '982244ed-d91c-4fc5-b153-65b70a4c5891', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'ket_fu1', NULL, 'Oke', '2026-01-24 04:57:10'),
	('9a025bd8-fc6f-4e36-af1f-652cbc2b8259', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'perkiraan_hpp', NULL, 'Rp 2,000,000.00', '2026-01-24 05:04:20'),
	('9aec4101-fb1f-4cdb-bc4c-c37190dadeaf', '982244ed-d91c-4fc5-b153-65b70a4c5891', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'status', 'New Leads', 'Follow Up', '2026-01-24 04:57:10'),
	('9e52ed9e-8377-427e-80d8-f7d1a461251d', '982244ed-d91c-4fc5-b153-65b70a4c5891', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'total_fu', NULL, '1', '2026-01-24 04:57:10'),
	('a1efeca3-a46c-4056-85f1-39d9e03469c7', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'durasi_hingga_closing', NULL, '2', '2026-01-26 06:58:06'),
	('a2b535b2-3364-4938-8f99-3c0027ea2d28', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'img_closing', NULL, 'https://furnivilux.namia.online/api/media/public/i4Bjjf_20251216_124310.jpg', '2026-01-24 05:04:20'),
	('ae135cee-ce6e-4db2-8a63-f9496e611aa8', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'umur_lead', '0', '2', '2026-01-26 06:58:06'),
	('b2b4ddfd-5ea4-4a90-9e82-c43383da5c03', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'nilai_pipeline', NULL, 'Rp 0.00', '2026-01-24 05:04:20'),
	('c02ab50a-c2fa-41ab-9c2e-fa874c345865', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'harga_jual', NULL, 'Rp 2,500,000.00', '2026-01-24 05:04:20'),
	('d1c033a7-2c56-4ec0-ab90-beda905cc85d', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'nilai_pipeline', NULL, 'Rp 0.00', '2026-01-26 06:58:06'),
	('dcd020e0-4875-48c7-bcb4-9544f169e63d', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'sisa_pembayaran', NULL, 'Rp 80,000,000.00', '2026-01-26 06:58:06'),
	('dd917e37-56cf-45bf-8ea0-bb122301982b', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'omset', NULL, 'Rp 80,000,000.00', '2026-01-26 06:58:06'),
	('ec41e61b-7aa4-4611-ade9-6acc40f53cce', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'sisa_pembayaran', NULL, 'Rp 2,500,000.00', '2026-01-24 05:04:20'),
	('f6e99d23-5744-4782-a828-14f1eb50b0e2', '982244ed-d91c-4fc5-b153-65b70a4c5891', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'hari_sejak_fu', NULL, '-2', '2026-01-24 04:57:10'),
	('fe6d3484-3e37-48e5-8e51-6cda5b2c7cd0', '982244ed-d91c-4fc5-b153-65b70a4c5891', '550e8400-e29b-41d4-a716-446655440003', 'status_kategorisasi', 'Active', 'Closed', '2026-01-24 05:04:20'),
	('ff186619-89bd-47ce-afed-15917834e2bd', 'bb6596ac-af30-4d38-9b93-5835de41755e', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'margin_aktual', NULL, 'Rp 80,000,000.00', '2026-01-26 06:58:06');

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

-- Dumping data for table furnifilux_dev.log: ~10 rows (approximately)
INSERT INTO `log` (`uuid`, `message`, `kind`, `user_id`, `created_at`) VALUES
	('1e14c25d-1f9b-43fc-a6b9-4197bb41af14', 'Menambahkan akun baru: BCA', 'akun', '550e8400-e29b-41d4-a716-446655440003', '2026-01-26 01:34:44'),
	('2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Menambahkan lead baru: Agung Firman (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 04:56:35'),
	('2d1d4307-54e4-4116-a245-255a9b40e000', 'Mengubah lead: Agung Firman (LEADS#1001)', 'leads', '550e8400-e29b-41d4-a716-446655440003', '2026-01-24 05:04:20'),
	('7455a9ce-4c51-4488-8817-b290001b83dc', 'Mengubah lead: Agung Firman (LEADS#1001)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 04:57:10'),
	('7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Menambahkan transaksi keuangan dengan nomor <b>FINANCE#1000</b> (Proyek - Rp 400000)', 'Finance', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 05:12:10'),
	('78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Menambahkan lead baru: Muhammad Ichsan (LEADS#1000)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-24 04:55:07'),
	('ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Mengubah lead: Muhammad Ichsan (LEADS#1000)', 'leads', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', '2026-01-26 06:58:06'),
	('b16cab7d-87f5-4912-b7ad-dc58353e32ca', 'Menambahkan akun baru: BRI', 'akun', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 05:11:31'),
	('bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Menambahkan transaksi keuangan dengan nomor <b>FINANCE#1002</b> (Pendapatan - Rp 5000000)', 'Finance', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-26 07:01:06'),
	('fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Menambahkan transaksi keuangan dengan nomor <b>FINANCE#1001</b> (Pendapatan - Rp 2000000)', 'Finance', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', '2026-01-24 05:12:59');

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

-- Dumping data for table furnifilux_dev.log_item: ~105 rows (approximately)
INSERT INTO `log_item` (`uuid`, `log_id`, `message`, `created_at`) VALUES
	('024a7ef5-1fe7-49b2-a014-ec5ca906c30e', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Tanggal Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>26 Januari 2026</span>', '2026-01-26 06:58:06'),
	('032410e5-23a8-47ca-8202-8d75ff3f7196', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Bulan\' ditambahkan dengan nilai <span class=\'font-semibold\'>Januari 2026</span>', '2026-01-24 05:12:59'),
	('08974e13-232b-4d3f-9adc-8175025ce1e0', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'CS\' ditambahkan dengan nilai <span class=\'font-semibold\'>CS B</span>', '2026-01-24 04:56:35'),
	('08ac1c40-7414-40d6-a8f6-3baca8d90fb4', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Durasi Hari Hingga Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>0</span>', '2026-01-24 05:04:20'),
	('0948b8a6-86c2-4abf-af3b-786e3d3ec707', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'No Telp\' ditambahkan dengan nilai <span class=\'font-semibold\'>081238664315</span>', '2026-01-24 04:55:08'),
	('0bcda492-6bdf-4f45-995c-21f60540fd3f', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Kategori Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Desain</span>', '2026-01-24 04:55:08'),
	('14d95bfd-af7c-4c10-a48c-0008a7f31d5b', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Kategori Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Customer Leads</span>', '2026-01-24 04:55:08'),
	('1e1fcf2a-75a9-4544-a989-43319426b5bb', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Lokasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Jogja</span>', '2026-01-24 05:12:10'),
	('227b5497-44cd-4838-bf1f-5f4148c86b11', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Bulan\' ditambahkan dengan nilai <span class=\'font-semibold\'>Januari 2026</span>', '2026-01-24 05:12:10'),
	('25efeb35-8ce3-4418-a80c-0645a274c8d8', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Mebel Bawah</span>', '2026-01-24 04:55:08'),
	('2ae8322f-9f15-4185-9c3d-5e2af3d920a2', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Kategori Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Exterior Luar</span>', '2026-01-24 04:56:35'),
	('2b2e7c25-ba23-4c50-b3f3-f8edca6ce84b', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Kredit\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 5,000,000.00</span>', '2026-01-26 07:01:06'),
	('2cae444a-0f15-414c-ba0d-8d6ba4460b1e', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Margin Aktual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 80,000,000.00</span>', '2026-01-26 06:58:06'),
	('2d63fc06-7e37-4ebf-b8c4-0bdfc56c01f9', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Akun ID\' ditambahkan dengan nilai <span class=\'font-semibold\'>BRI</span>', '2026-01-24 05:12:10'),
	('2f1de728-e127-4cc3-9722-7b1c87dce27c', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Status Kategorisasi\' berubah dari <span class=\'font-semibold\'>Active</span> ke <span class=\'font-semibold\'>Closed</span>', '2026-01-26 06:58:06'),
	('30768762-88ae-4d28-a8a5-55dcdc305443', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Jatuh Tempo\' ditambahkan dengan nilai <span class=\'font-semibold\'>29 Januari 2026</span>', '2026-01-24 05:12:10'),
	('31e3d80b-79cb-40b5-8ded-59e555188428', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'CS\' ditambahkan dengan nilai <span class=\'font-semibold\'>CS B</span>', '2026-01-24 04:55:08'),
	('3366e531-a377-4984-8b1c-34dcfdd2ff39', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>24 Januari 2026</span>', '2026-01-24 05:12:10'),
	('33ed7c1e-15a0-46aa-a70f-d6b13199e8cf', '7455a9ce-4c51-4488-8817-b290001b83dc', 'Data \'Tanggal FU 1\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>26 Januari 2026</span>', '2026-01-24 04:57:10'),
	('34a803f5-99a6-49ba-9432-628a7bf10199', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>24 Januari 2026</span>', '2026-01-24 05:12:59'),
	('3d3b4dbe-e1ea-4845-b1b5-495f3300450f', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Kategori Utama\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-24 05:12:59'),
	('43664ef8-f868-423a-bd5a-353655e651d0', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Kategori Utama\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-26 07:01:06'),
	('458a0840-6433-4dca-ac71-06bfa5f15cf2', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Jenis Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Kredit</span>', '2026-01-24 05:12:59'),
	('495ca51e-cdf9-479d-8690-cac3ff7f6ad5', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Perkiraan HPP\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 50,000,000.00</span>', '2026-01-26 06:58:06'),
	('4ac16b05-7a7a-4bcb-821a-2a4617bfccd6', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Nama Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Muhammad Ichsan</span>', '2026-01-24 04:55:08'),
	('4f896a82-e34e-44ba-a30e-f475e630e363', '7455a9ce-4c51-4488-8817-b290001b83dc', 'Data \'Keterangan FU 1\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Oke</span>', '2026-01-24 04:57:10'),
	('5076864d-74ad-40ce-8788-2ceb8416c0c7', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Agung Firman</span>', '2026-01-24 05:12:10'),
	('51f74596-5bae-4737-a716-d880148a0255', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Nominal\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 5,000,000.00</span>', '2026-01-26 07:01:06'),
	('58dc3d94-b519-4cbf-ad11-dbca87992e64', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Harga Jual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 80,000,000.00</span>', '2026-01-26 06:58:06'),
	('5ad69976-b593-4b46-b2af-13265f54ef31', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Status Kategorisasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Active</span>', '2026-01-24 04:55:08'),
	('5edaf0b2-3b4e-499d-a80a-2f47de3f11ae', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Agung Firman</span>', '2026-01-24 05:12:59'),
	('621ef719-bc07-49a5-ae12-e0dc984314c1', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Kategori Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pengeluaran</span>', '2026-01-24 05:12:10'),
	('626e4ef8-259b-4e0d-997c-23113f2338b9', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Umur Lead\' berubah dari <span class=\'font-semibold\'>0</span> ke <span class=\'font-semibold\'>2</span>', '2026-01-26 06:58:06'),
	('6b61b327-8f87-49f4-a5a7-f667feb9988b', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Nominal\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 400,000.00</span>', '2026-01-24 05:12:10'),
	('6ba03cfc-3eee-488d-b93c-250d35865e0b', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Status\' ditambahkan dengan nilai <span class=\'font-semibold\'>New Leads</span>', '2026-01-24 04:55:08'),
	('6d0ead06-2fcc-477a-bb24-c4c2f48261d1', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'HPP Aktual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 0.00</span>', '2026-01-26 06:58:06'),
	('6ecf95c6-2602-43dc-806c-76692cb3a2f4', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Item <b>Kayu, Mebel</b> ditambahkan: Qty 1, Harga Rp 400,000.00, Catatan: -', '2026-01-24 05:12:10'),
	('6fb023f6-2f65-45b9-a8bf-1309220a86c2', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Status Kategorisasi\' berubah dari <span class=\'font-semibold\'>Active</span> ke <span class=\'font-semibold\'>Closed</span>', '2026-01-24 05:04:20'),
	('738593dd-2f41-4962-9278-1b9bf6afc7f8', '7455a9ce-4c51-4488-8817-b290001b83dc', 'Data \'Hari Sejak FU Terakhir\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>-2</span>', '2026-01-24 04:57:10'),
	('73c99d76-da91-45ca-ad5d-c53589089824', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Omset\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 2,500,000.00</span>', '2026-01-24 05:04:20'),
	('777e9719-214f-4e0d-81dd-9b15ac527698', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Kategori Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-26 07:01:06'),
	('79267db5-2178-4028-810e-892032ff6245', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Vendor\' ditambahkan dengan nilai <span class=\'font-semibold\'>HAndra / Marketing</span>', '2026-01-24 05:12:10'),
	('7a23e47d-801f-4f46-849f-aa12ba24f9f3', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Muhammad Ichsan</span>', '2026-01-26 07:01:06'),
	('7a7862ed-16f8-4d46-a4c2-5ff03cdccd90', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Kategori Utama\' ditambahkan dengan nilai <span class=\'font-semibold\'>Proyek</span>', '2026-01-24 05:12:10'),
	('7c2681d5-cae9-472e-8f06-56fed70147ea', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Margin Aktual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 2,500,000.00</span>', '2026-01-24 05:04:20'),
	('7c5041ca-6d81-4032-8897-e6cb4f24e56b', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Kategori Aktivitas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-24 05:12:59'),
	('7c9f8d70-6fc6-477b-8843-12abcfcdfc75', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Perkiraan HPP\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 2,000,000.00</span>', '2026-01-24 05:04:20'),
	('8510acd0-0d49-4390-b169-54bbeb1081be', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Item <b>pembayaran #1</b> ditambahkan: Qty 1, Harga Rp 2,000,000.00, Catatan: -', '2026-01-24 05:12:59'),
	('86a1a2c7-ea3f-4884-a3b3-228d7cdffb3a', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Nominal\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 2,000,000.00</span>', '2026-01-24 05:12:59'),
	('89f27214-6580-43b2-9d71-4eec32212ff5', '7455a9ce-4c51-4488-8817-b290001b83dc', 'Data \'Status\' berubah dari <span class=\'font-semibold\'>New Leads</span> ke <span class=\'font-semibold\'>Follow Up</span>', '2026-01-24 04:57:10'),
	('8db61857-b886-489c-828d-7625673a0ade', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Sumber Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Instagram</span>', '2026-01-24 04:55:08'),
	('909cc16a-bf80-4b70-a040-709e7d6e7225', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Kategori Arus Kas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Operasional</span>', '2026-01-26 07:01:06'),
	('92a72d08-8f69-48c4-a554-68c28b94176c', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Status Kategorisasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Active</span>', '2026-01-24 04:56:35'),
	('92b72c66-358e-47d2-bc03-a264b70f900e', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Sisa Pembayaran\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 2,500,000.00</span>', '2026-01-24 05:04:20'),
	('93a6bbbe-62d3-4515-9507-1f21a2d49773', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Item <b>pembayaran dp #1</b> ditambahkan: Qty 1, Harga Rp 5,000,000.00, Catatan: -', '2026-01-26 07:01:06'),
	('94088637-38a6-4246-b7cb-b0b1888cddee', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Lokasi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Jakarta</span>', '2026-01-24 05:12:59'),
	('97e4e900-9e2a-440c-bf2b-33780a012246', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Foto Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg</span>', '2026-01-26 06:58:06'),
	('9924b38c-0979-47d1-bff5-7cdfd0dfe363', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Arah Uang\' ditambahkan dengan nilai <span class=\'font-semibold\'>Masuk</span>', '2026-01-24 05:12:59'),
	('9b5416a9-0124-49bb-9093-bb7df14de561', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'No Telp\' ditambahkan dengan nilai <span class=\'font-semibold\'>0895417205060</span>', '2026-01-24 04:56:35'),
	('a2640fd5-f5ae-4987-b68e-31ef5e1db0ff', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Umur Lead\' ditambahkan dengan nilai <span class=\'font-semibold\'>-1</span>', '2026-01-24 04:56:35'),
	('a4efe483-b387-42a0-800f-afa26d4b5b94', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Akun ID\' ditambahkan dengan nilai <span class=\'font-semibold\'>BRI</span>', '2026-01-26 07:01:06'),
	('a704a0ef-403d-4a73-baa0-3b58249c145d', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Akun ID\' ditambahkan dengan nilai <span class=\'font-semibold\'>BRI</span>', '2026-01-24 05:12:59'),
	('a783983f-3436-4e2b-bbc6-132a03bae9d2', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Sumber Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Instagram</span>', '2026-01-24 04:56:35'),
	('ab3e1505-67bb-426b-8c83-2d7925928279', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Sub Kategori\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pembayaran Client</span>', '2026-01-26 07:01:06'),
	('abba93a2-cfb7-40e4-bf2d-3848047a7b4f', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Status\' berubah dari <span class=\'font-semibold\'>New Leads</span> ke <span class=\'font-semibold\'>Closing</span>', '2026-01-26 06:58:06'),
	('ac2dc068-b18f-4a6c-843b-4b6a0a7459e3', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Kategori Aktivitas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-26 07:01:06'),
	('ae0979ad-3a2e-46ab-a40e-6285365d30d1', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Alamat\' ditambahkan dengan nilai <span class=\'font-semibold\'>Demangan, Selomartani, Kalasan, Sleman\nSelomartani</span>', '2026-01-24 04:56:35'),
	('af8c085a-123d-4595-b1f0-3759946f83d6', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Jenis Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Debit</span>', '2026-01-24 05:12:10'),
	('b131a750-ee8a-4398-91ad-aa4684df268c', '7455a9ce-4c51-4488-8817-b290001b83dc', 'Data \'HPP Aktual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 0.00</span>', '2026-01-24 04:57:10'),
	('b4cd35a1-6d2e-46bc-b070-204c1ff56366', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>25 Januari 2026</span>', '2026-01-24 04:56:35'),
	('b6653acb-2559-49b6-9fed-e02bb4416b68', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Debit\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 400,000.00</span>', '2026-01-24 05:12:10'),
	('b6979665-3fb7-45c2-947e-0868545ea662', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Jam Masuk\' ditambahkan dengan nilai <span class=\'font-semibold\'>4</span>', '2026-01-24 04:55:08'),
	('b992665a-1b7e-4337-9fbe-3ed8c8e49238', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Umur Lead\' ditambahkan dengan nilai <span class=\'font-semibold\'>0</span>', '2026-01-24 04:55:08'),
	('bce58380-1469-43f2-9ee5-b61e3e4c625c', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Nama Klien\' ditambahkan dengan nilai <span class=\'font-semibold\'>Agung Firman</span>', '2026-01-24 04:56:35'),
	('bd67b3e1-cf5c-467f-9c2b-43d5932ea37e', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Arah Uang\' ditambahkan dengan nilai <span class=\'font-semibold\'>Keluar</span>', '2026-01-24 05:12:10'),
	('c34927e6-94d3-40e9-bfe0-26aa1657e24a', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Jam Masuk\' ditambahkan dengan nilai <span class=\'font-semibold\'>4</span>', '2026-01-24 04:56:35'),
	('c6937d6d-775d-4555-bc22-6468f3fb03cb', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Kredit\' ditambahkan dengan nilai <span class=\'font-semibold\'>Rp 2,000,000.00</span>', '2026-01-24 05:12:59'),
	('c723de4a-e7db-4d7d-832e-e07c72b736ac', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Tanggal Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>25 Januari 2026</span>', '2026-01-24 05:04:20'),
	('c896bf78-a717-4ea0-bdbc-bfb3593f6406', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Kategori Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pendapatan</span>', '2026-01-24 05:12:59'),
	('cc662afa-1374-443b-9521-26ef4730956e', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Produk\' ditambahkan dengan nilai <span class=\'font-semibold\'>Kayu Furni</span>', '2026-01-24 04:56:35'),
	('ccac4745-5948-4903-af23-fb157baee4ab', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Kota\' ditambahkan dengan nilai <span class=\'font-semibold\'>Bandar Lampung</span>', '2026-01-24 04:55:08'),
	('d101ce28-b6f9-4121-80c2-185499e17816', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Sub Kategori\' ditambahkan dengan nilai <span class=\'font-semibold\'>Pembayaran Client</span>', '2026-01-24 05:12:59'),
	('d322ba07-6770-4e50-8331-7900e9e5ccf9', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Sisa Pembayaran\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 80,000,000.00</span>', '2026-01-26 06:58:06'),
	('d6f583bd-4609-47e5-81a5-ee4cf465ba2c', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Status\' berubah dari <span class=\'font-semibold\'>Follow Up</span> ke <span class=\'font-semibold\'>Closing</span>', '2026-01-24 05:04:20'),
	('d74c8ed8-bb23-48e7-9f22-6eaa40bb2e20', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Kategori Arus Kas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Operasional</span>', '2026-01-24 05:12:59'),
	('d772f077-e2c0-43b1-a2ee-5043090a9cc1', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>26 Januari 2026</span>', '2026-01-26 07:01:06'),
	('d8cb7140-dce0-446c-8161-b1d5fe138e17', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Tanggal\' ditambahkan dengan nilai <span class=\'font-semibold\'>24 Januari 2026</span>', '2026-01-24 04:55:08'),
	('da9203d0-b43f-4b9f-b5a1-0e8558b2b3a0', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Kota\' ditambahkan dengan nilai <span class=\'font-semibold\'>Bandar Lampung</span>', '2026-01-24 04:56:35'),
	('dc01d6da-8c3f-485e-92eb-01891caeb527', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Nilai Pipeline\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 0.00</span>', '2026-01-24 05:04:20'),
	('de0e7ce1-7393-42ef-8276-7a5671947415', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Kategori Arus Kas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Operasional</span>', '2026-01-24 05:12:10'),
	('e09e2cdd-2df2-4f8f-bbf9-b78e5ac8f99e', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Bulan\' ditambahkan dengan nilai <span class=\'font-semibold\'>Januari 2026</span>', '2026-01-26 07:01:06'),
	('e1914950-ddd9-4342-b5e6-815dd25f0ffb', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Arah Uang\' ditambahkan dengan nilai <span class=\'font-semibold\'>Masuk</span>', '2026-01-26 07:01:06'),
	('ea967920-213c-4faa-9a4d-8dd899460c60', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Foto Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>https://furnivilux.namia.online/api/media/public/i4Bjjf_20251216_124310.jpg</span>', '2026-01-24 05:04:20'),
	('ec9d53ae-32da-457a-ba36-dcfaed921b4c', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Omset\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 80,000,000.00</span>', '2026-01-26 06:58:06'),
	('ed886847-3689-4274-aa46-4167e9491534', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Status\' ditambahkan dengan nilai <span class=\'font-semibold\'>New Leads</span>', '2026-01-24 04:56:35'),
	('ede27a1c-ca31-4ec1-8b5d-c7069b11678d', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Nilai Pipeline\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 0.00</span>', '2026-01-26 06:58:06'),
	('ee75be11-f064-4e8b-8e7a-37e9dc03e858', 'fbf4a649-2b4f-4e4a-b577-8d259c9d3136', 'Data \'Jatuh Tempo\' ditambahkan dengan nilai <span class=\'font-semibold\'>27 Januari 2026</span>', '2026-01-24 05:12:59'),
	('eff6dd8b-6ae1-4e20-9809-f949cf8d311a', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Kategori Aktivitas\' ditambahkan dengan nilai <span class=\'font-semibold\'>Proyek</span>', '2026-01-24 05:12:10'),
	('f087a0de-65d5-4b6c-a4f5-21d1ea4d4b74', 'ab0955fe-c417-4ac7-b424-099d4d0e0deb', 'Data \'Durasi Hari Hingga Closing\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>2</span>', '2026-01-26 06:58:06'),
	('f0c0ab3a-2f13-4d84-bd5d-89bce2473d37', '7455a9ce-4c51-4488-8817-b290001b83dc', 'Data \'Total Follow Up\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>1</span>', '2026-01-24 04:57:10'),
	('f6204de2-6cff-4dcf-9637-09aa24ae3071', '2af4e380-1816-4ffc-b649-2abf3d8b132c', 'Data \'Kategori Leads\' ditambahkan dengan nilai <span class=\'font-semibold\'>Customer Leads</span>', '2026-01-24 04:56:35'),
	('f6be9e31-baaf-4ffc-9e14-757d87f83c5d', 'bf1b7636-7293-4803-a7f3-cc1e3c58a806', 'Data \'Jenis Transaksi\' ditambahkan dengan nilai <span class=\'font-semibold\'>Kredit</span>', '2026-01-26 07:01:06'),
	('f7317fe0-5464-434c-9990-529cf4c56967', '7790c297-8201-45ba-8f33-aeb8c821eb8b', 'Data \'Sub Kategori\' ditambahkan dengan nilai <span class=\'font-semibold\'>Material</span>', '2026-01-24 05:12:10'),
	('fa0d3308-222f-4fd4-82f1-80bad9d8a600', '2d1d4307-54e4-4116-a245-255a9b40e000', 'Data \'Harga Jual\' berubah dari <span class=\'font-semibold\'>-</span> ke <span class=\'font-semibold\'>Rp 2,500,000.00</span>', '2026-01-24 05:04:20'),
	('fc558829-47a9-4822-be0d-1d055b8dda26', '78ffe166-2ff5-4c9e-83da-3c7e249e4092', 'Data \'Alamat\' ditambahkan dengan nilai <span class=\'font-semibold\'>Jalan mawar\nSelomartani</span>', '2026-01-24 04:55:08');

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

-- Dumping data for table furnifilux_dev.media: ~9 rows (approximately)
INSERT INTO `media` (`uuid`, `user_id`, `url`, `kind`, `created_at`) VALUES
	('1641aa3a-2ac1-44b8-b84d-ab99dfb9184e', '550e8400-e29b-41d4-a716-446655440003', 'https://i.pinimg.com/1200x/96/be/d6/96bed63e541a937c3ce6f51850ece087.jpg', 'image', '2025-12-16 05:24:55'),
	('1ce3f52d-bebd-4eff-9fa9-8424e8aa5077', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'http://localhost:8002/media/public/oAxRld_20251217_130548.png', 'image', '2025-12-17 06:05:49'),
	('30d66007-516b-4c68-b35e-baaba9e4d4f5', '550e8400-e29b-41d4-a716-446655440003', 'https://furnivilux.namia.online/api/media/public/lqjuQr_20251216_141714.jpg', 'image', '2025-12-16 07:17:14'),
	('669da5c4-36e6-44a8-a663-c103cf4eb6a4', 'b2c3d4e5-f6a1-4b2c-9d3e-4f5a6b7c8d9e', 'http://localhost:8002/media/public/wYoEU8_20260124_102001.jpeg', 'image', '2026-01-24 03:20:04'),
	('78e911a8-5495-4cd0-acbb-a4e2d4439c57', '550e8400-e29b-41d4-a716-446655440003', 'https://furnivilux.namia.online/api/media/public/i4Bjjf_20251216_124310.jpg', 'image', '2025-12-16 05:43:10'),
	('82dc5ceb-6f5c-48f6-8fe9-9f6781707018', 'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d', 'https://furnivilux.namia.online/api/media/public/1gaUPY_20260124_121158.png', 'image', '2026-01-24 05:11:58'),
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
	('akun', 1002, '2025-11-07 08:30:19', '2026-01-26 01:34:44'),
	('finance_transactions', 1003, '2025-10-27 07:39:40', '2026-01-26 07:01:06'),
	('leads', 1002, '2025-10-27 07:39:40', '2026-01-24 04:56:35');

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
	('550e8400-e29b-41d4-a716-446655440003', 'admin', 'admin@furnifilux.com', 'admin123', 'System Administrator 1', '550e8400-e29b-41d4-a716-446655440001', 'Active', '2025-10-27 04:29:05', '2026-01-26 02:44:25'),
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
