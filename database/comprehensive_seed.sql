-- ============================================================
-- COMPREHENSIVE SEED: Archeris Mobile API Audit
-- Phase 0: Full test data for all API phases
-- Run: mysql -u root archeris < comprehensive_seed.sql
-- ============================================================

SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- STEP 1: UPDATE EXISTING ARCHERS — fix bow_type, city, avatar
-- ============================================================

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Sleman',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '01806bbe-b7cc-413c-b7ec-abd69f2775d4'; -- Andi Saputra

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Sleman',
  gender     = 'female',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '54e9646f-c667-4b87-970c-74da698caefb'; -- Siti Rahmawati

UPDATE archers SET
  bow_type   = 'compound',
  city       = 'Yogyakarta',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = 'ba5c9c1e-1bf7-4014-ac88-539cc6947e06'; -- Budi Setiawan

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Bantul',
  gender     = 'female',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '82477f4f-9da9-4ca0-86ad-b77c6e161b64'; -- Citra Lestari

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Sleman',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '0d635052-2486-4fea-8ac0-570246d2d885'; -- Eko Pranoto

UPDATE archers SET
  bow_type   = 'barebow',
  city       = 'Gunungkidul',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = 'daea3f95-fb72-405b-986b-ab69185c5b76'; -- Fajar Ramadhan

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Kulon Progo',
  gender     = 'female',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = 'b6711f75-85c4-4413-af54-f474f9b62635'; -- Gina Permata

UPDATE archers SET
  bow_type   = 'compound',
  city       = 'Sleman',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '4ce93d38-a78d-4a6b-be27-62ba8dccea0e'; -- Hendra Wijaya

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Bantul',
  gender     = 'female',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '74ba7fa6-19ce-41d4-8d6e-e790f822c9ca'; -- Intan Sari

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Yogyakarta',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = 'f0632fb1-60f9-47be-9426-65475b80b766'; -- Joko Firmansyah

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Sleman',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '8e2ed4ef-5768-4a53-a827-33385edadf9b'; -- Faris Aditya Pratama

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Sleman',
  gender     = 'female',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = 'd9ec12bb-0842-4ed9-bda7-7b1596a8b86b'; -- Dewi Anggraini

UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Yogyakarta',
  gender     = 'male',
  avatar_url = CONCAT('https://api.dicebear.com/7.x/bottts/svg?seed=', uuid)
WHERE uuid = '4b075b44-c1d7-4f16-8017-3a34615836f5'; -- Rizky Kurniawan

-- Update existing test archer (stewie4king dev account) if exists
UPDATE archers SET
  bow_type   = 'recurve',
  city       = 'Sleman',
  gender     = 'male',
  avatar_url = 'https://api.dicebear.com/7.x/bottts/svg?seed=stewie4king'
WHERE email = 'stewie4king@gmail.com';

-- ============================================================
-- STEP 2: UPDATE EXISTING ORGANIZER — ensure usable for tests
-- ============================================================

UPDATE organizers SET
  subscription_status = 'active',
  subscription_plan_id = 5,
  avatar_url   = 'https://api.dicebear.com/7.x/initials/svg?seed=ACO',
  banner_url   = 'https://picsum.photos/seed/aco-banner/1200/400',
  status       = 'active'
WHERE uuid = '413db8ca-88fe-436c-82d1-203a48205e46';

-- ============================================================
-- STEP 3: ADD 2 MORE SCOREKEEPERS for the organizer
-- ============================================================

INSERT IGNORE INTO scorekeepers (uuid, organization_uuid, code, name, email, status, token_version, created_at, updated_at)
VALUES
  ('sk-audit-001', '413db8ca-88fe-436c-82d1-203a48205e46', 'SK001', 'Budi Wasit', 'sk001@archeris.test', 'active', 1, NOW(), NOW()),
  ('sk-audit-002', '413db8ca-88fe-436c-82d1-203a48205e46', 'SK002', 'Dewi Scorer', 'sk002@archeris.test', 'active', 1, NOW(), NOW());

-- ============================================================
-- STEP 4: THREE TOURNAMENTS
-- T1: Kejurda DIY 2026    — upcoming, published (main test)
-- T2: POPDA Sleman 2026   — upcoming, published
-- T3: Latber SAC Juli     — past/done, published (scoring data)
-- ============================================================

INSERT IGNORE INTO tournaments (uuid, slug, code, name, short_name, venue, location, address, city, start_date, end_date, registration_deadline, description, banner_url, logo_url, type, entry_fee, status, organizer_id, total_prize, venue_type, visibility, quota_type, quota_max_participants, created_at, updated_at)
VALUES
-- T1: Kejurda DIY 2026
(
  'tourn-kejurda-diy-2026',
  'kejurda-diy-2026',
  'KJD-2026',
  'Kejuaraan Daerah DIY 2026',
  'Kejurda DIY',
  'Stadion Maguwoharjo',
  'Sleman, DIY',
  'Jl. Maguwoharjo No. 1, Sleman, Yogyakarta',
  'Sleman',
  '2026-11-15 08:00:00',
  '2026-11-17 17:00:00',
  '2026-11-10 23:59:59',
  '<p>Kejuaraan Daerah Istimewa Yogyakarta 2026. Kompetisi resmi antar kabupaten/kota se-DIY. Memperebutkan medali emas, perak, dan perunggu serta tiket ke Porda DIY.</p>',
  'https://picsum.photos/seed/kejurda-diy-2026/1200/600',
  'https://picsum.photos/seed/kejurda-diy-logo/400/400',
  'individual',
  200000.00,
  'published',
  '413db8ca-88fe-436c-82d1-203a48205e46',
  15000000.00,
  'outdoor',
  'listed',
  'standard',
  200,
  NOW(), NOW()
),
-- T2: POPDA Sleman 2026
(
  'tourn-popda-sleman-2026',
  'popda-sleman-2026',
  'PPD-2026',
  'Seleksi POPDA Kab. Sleman 2026',
  'POPDA Sleman',
  'GOR Sleman',
  'Sleman, DIY',
  'Jl. Stadion No. 1, Beran, Sleman',
  'Sleman',
  '2026-10-10 08:00:00',
  '2026-10-12 17:00:00',
  '2026-10-05 23:59:59',
  '<p>Seleksi atlet panahan untuk POPDA Kabupaten Sleman tahun 2026. Terbuka untuk kategori U-15 dan U-18 domisili Kabupaten Sleman.</p>',
  'https://picsum.photos/seed/popda-sleman-2026/1200/600',
  'https://picsum.photos/seed/popda-sleman-logo/400/400',
  'individual',
  150000.00,
  'published',
  '413db8ca-88fe-436c-82d1-203a48205e46',
  5000000.00,
  'outdoor',
  'listed',
  'free',
  50,
  NOW(), NOW()
),
-- T3: Latber SAC (past — scoring data already here)
(
  'tourn-latber-sac-2026',
  'latber-sac-juli-2026',
  'LAT-2026',
  'Latihan Bersama SAC Juli 2026',
  'Latber SAC',
  'Lapangan Tridadi',
  'Sleman, DIY',
  'Jl. Tridadi No. 10, Sleman',
  'Sleman',
  '2026-07-20 08:00:00',
  '2026-07-20 17:00:00',
  '2026-07-18 23:59:59',
  '<p>Latihan bersama antar klub panahan se-Kabupaten Sleman. Cocok untuk atlet U-15 dan Umum yang ingin mengasah kemampuan sebelum kompetisi resmi.</p>',
  'https://picsum.photos/seed/latber-sac-2026/1200/600',
  'https://picsum.photos/seed/latber-sac-logo/400/400',
  'individual',
  50000.00,
  'published',
  '413db8ca-88fe-436c-82d1-203a48205e46',
  2000000.00,
  'outdoor',
  'listed',
  'free',
  30,
  NOW(), NOW()
);

-- ============================================================
-- STEP 5: TOURNAMENT IMAGES
-- ============================================================

INSERT IGNORE INTO tournament_images (uuid, tournament_id, url, caption, display_order, is_primary, created_at, updated_at)
VALUES
  ('timg-kjd-01', 'tourn-kejurda-diy-2026', 'https://picsum.photos/seed/kjd-gallery-1/800/600', 'Suasana Lapangan Maguwoharjo', 1, 1, NOW(), NOW()),
  ('timg-kjd-02', 'tourn-kejurda-diy-2026', 'https://picsum.photos/seed/kjd-gallery-2/800/600', 'Para Peserta Kualifikasi', 2, 0, NOW(), NOW()),
  ('timg-kjd-03', 'tourn-kejurda-diy-2026', 'https://picsum.photos/seed/kjd-gallery-3/800/600', 'Upacara Pembukaan', 3, 0, NOW(), NOW()),
  ('timg-ppd-01', 'tourn-popda-sleman-2026', 'https://picsum.photos/seed/ppd-gallery-1/800/600', 'GOR Sleman dari Udara', 1, 1, NOW(), NOW()),
  ('timg-lat-01', 'tourn-latber-sac-2026',  'https://picsum.photos/seed/lat-gallery-1/800/600', 'Atlet Latber SAC', 1, 1, NOW(), NOW());

-- ============================================================
-- STEP 6: TOURNAMENT SCHEDULE ITEMS
-- ============================================================

INSERT IGNORE INTO tournament_schedule_items (uuid, tournament_id, title, description, start_time, end_time, day_number, sort_order, activity_type, created_at, updated_at)
VALUES
-- Kejurda DIY T1
('tsched-kjd-01', 'tourn-kejurda-diy-2026', 'Technical Meeting', 'Pertemuan teknis, penjelasan peraturan, dan inspeksi peralatan', '2026-11-15 08:00:00', '2026-11-15 10:00:00', 1, 1, 'technical_meeting', NOW(), NOW()),
('tsched-kjd-02', 'tourn-kejurda-diy-2026', 'Kualifikasi Recurve & Compound', 'Babak kualifikasi semua divisi', '2026-11-15 10:30:00', '2026-11-15 17:00:00', 1, 2, 'qualification', NOW(), NOW()),
('tsched-kjd-03', 'tourn-kejurda-diy-2026', 'Kualifikasi Barebow', 'Babak kualifikasi divisi Barebow', '2026-11-16 08:00:00', '2026-11-16 14:00:00', 2, 3, 'qualification', NOW(), NOW()),
('tsched-kjd-04', 'tourn-kejurda-diy-2026', 'Eliminasi 1/8 & 1/4 Final', 'Babak gugur 16 besar dan 8 besar', '2026-11-16 14:30:00', '2026-11-16 18:00:00', 2, 4, 'elimination', NOW(), NOW()),
('tsched-kjd-05', 'tourn-kejurda-diy-2026', 'Semifinal & Final', 'Babak semifinal, final perunggu, dan final emas', '2026-11-17 09:00:00', '2026-11-17 15:00:00', 3, 5, 'final', NOW(), NOW()),
('tsched-kjd-06', 'tourn-kejurda-diy-2026', 'Upacara Penutupan & Pembagian Hadiah', 'Ceremoni medali dan pembagian sertifikat', '2026-11-17 15:30:00', '2026-11-17 17:00:00', 3, 6, 'award_ceremony', NOW(), NOW()),
-- POPDA T2
('tsched-ppd-01', 'tourn-popda-sleman-2026', 'Technical Meeting', 'Briefing teknis dan registrasi ulang', '2026-10-10 07:30:00', '2026-10-10 09:00:00', 1, 1, 'technical_meeting', NOW(), NOW()),
('tsched-ppd-02', 'tourn-popda-sleman-2026', 'Kualifikasi U-15', 'Babak kualifikasi kategori U-15 Putra & Putri', '2026-10-10 09:30:00', '2026-10-10 15:00:00', 1, 2, 'qualification', NOW(), NOW()),
('tsched-ppd-03', 'tourn-popda-sleman-2026', 'Kualifikasi U-18', 'Babak kualifikasi kategori U-18 Putra & Putri', '2026-10-11 08:00:00', '2026-10-11 14:00:00', 2, 3, 'qualification', NOW(), NOW()),
('tsched-ppd-04', 'tourn-popda-sleman-2026', 'Eliminasi Final', 'Babak final semua kategori', '2026-10-12 08:00:00', '2026-10-12 15:00:00', 3, 4, 'final', NOW(), NOW());

-- ============================================================
-- STEP 7: TOURNAMENT CATEGORIES
-- ref_bow_types: Recurve=5c95a503, Compound=349a5218, Barebow=94bf104d
-- ref_age_groups: U-15=f879b964, U-18=f0ab19a5, Umum=f235b870
-- ref_gender_divisions: Men=d60f4939, Women=afbded2f, Mixed=f7a787a7
-- ref_tournament_types: Individual=da2740c8, Team=3bfbc4ad, MixedTeam=160f7979
-- ============================================================

INSERT IGNORE INTO tournament_categories (uuid, tournament_id, division_uuid, category_uuid, tournament_type_uuid, gender_division_uuid, max_participants, status, created_at, updated_at)
VALUES
-- Kejurda DIY — 8 categories
('tcat-kjd-01', 'tourn-kejurda-diy-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 40, 'active', NOW(), NOW()), -- Recurve U-15 Putra
('tcat-kjd-02', 'tourn-kejurda-diy-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 40, 'active', NOW(), NOW()), -- Recurve U-15 Putri
('tcat-kjd-03', 'tourn-kejurda-diy-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f0ab19a5-efe2-4ceb-b177-57966249af04', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 40, 'active', NOW(), NOW()), -- Recurve U-18 Putra
('tcat-kjd-04', 'tourn-kejurda-diy-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f0ab19a5-efe2-4ceb-b177-57966249af04', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 40, 'active', NOW(), NOW()), -- Recurve U-18 Putri
('tcat-kjd-05', 'tourn-kejurda-diy-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 40, 'active', NOW(), NOW()), -- Recurve Umum Putra
('tcat-kjd-06', 'tourn-kejurda-diy-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 40, 'active', NOW(), NOW()), -- Recurve Umum Putri
('tcat-kjd-07', 'tourn-kejurda-diy-2026', '349a5218-fed0-4305-ab16-e636501bb5df', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 30, 'active', NOW(), NOW()), -- Compound Umum Putra
('tcat-kjd-08', 'tourn-kejurda-diy-2026', '94bf104d-8ef2-4dd0-a1b1-2d82b46a1bdc', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 30, 'active', NOW(), NOW()), -- Barebow Umum Putra

-- POPDA Sleman — 4 categories
('tcat-ppd-01', 'tourn-popda-sleman-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 25, 'active', NOW(), NOW()), -- Recurve U-15 Putra
('tcat-ppd-02', 'tourn-popda-sleman-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 25, 'active', NOW(), NOW()), -- Recurve U-15 Putri
('tcat-ppd-03', 'tourn-popda-sleman-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f0ab19a5-efe2-4ceb-b177-57966249af04', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 25, 'active', NOW(), NOW()), -- Recurve U-18 Putra
('tcat-ppd-04', 'tourn-popda-sleman-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f0ab19a5-efe2-4ceb-b177-57966249af04', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 25, 'active', NOW(), NOW()), -- Recurve U-18 Putri

-- Latber SAC — 2 categories (for scoring test)
('tcat-lat-01', 'tourn-latber-sac-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 20, 'active', NOW(), NOW()), -- Recurve Umum Putra
('tcat-lat-02', 'tourn-latber-sac-2026', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 20, 'active', NOW(), NOW()); -- Recurve Umum Putri

-- ============================================================
-- STEP 8: TOURNAMENT PARTICIPANTS — 15+ archers across tournaments
-- Reuse existing archer UUIDs from the database
-- ============================================================

-- Kejurda DIY T1 — 10 participants
INSERT IGNORE INTO tournament_participants (uuid, tournament_id, archer_id, category_id, payment_amount, payment_status, qr_raw, registration_source, registration_date, created_at, updated_at)
VALUES
('tp-kjd-01', 'tourn-kejurda-diy-2026', '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'tcat-kjd-05', 200000.00, 'paid',   'AH-KJD-01', 'self_register', NOW(), NOW(), NOW()), -- Andi Saputra → Recurve Umum Putra
('tp-kjd-02', 'tourn-kejurda-diy-2026', '54e9646f-c667-4b87-970c-74da698caefb', 'tcat-kjd-06', 200000.00, 'paid',   'AH-KJD-02', 'self_register', NOW(), NOW(), NOW()), -- Siti Rahmawati → Recurve Umum Putri
('tp-kjd-03', 'tourn-kejurda-diy-2026', 'ba5c9c1e-1bf7-4014-ac88-539cc6947e06', 'tcat-kjd-07', 200000.00, 'paid',   'AH-KJD-03', 'self_register', NOW(), NOW(), NOW()), -- Budi Setiawan → Compound Umum Putra
('tp-kjd-04', 'tourn-kejurda-diy-2026', '82477f4f-9da9-4ca0-86ad-b77c6e161b64', 'tcat-kjd-06', 200000.00, 'paid',   'AH-KJD-04', 'self_register', NOW(), NOW(), NOW()), -- Citra Lestari → Recurve Umum Putri
('tp-kjd-05', 'tourn-kejurda-diy-2026', '0d635052-2486-4fea-8ac0-570246d2d885', 'tcat-kjd-05', 200000.00, 'paid',   'AH-KJD-05', 'self_register', NOW(), NOW(), NOW()), -- Eko Pranoto → Recurve Umum Putra
('tp-kjd-06', 'tourn-kejurda-diy-2026', 'daea3f95-fb72-405b-986b-ab69185c5b76', 'tcat-kjd-08', 200000.00, 'paid',   'AH-KJD-06', 'self_register', NOW(), NOW(), NOW()), -- Fajar Ramadhan → Barebow Umum Putra
('tp-kjd-07', 'tourn-kejurda-diy-2026', 'b6711f75-85c4-4413-af54-f474f9b62635', 'tcat-kjd-06', 200000.00, 'pending','AH-KJD-07', 'self_register', NOW(), NOW(), NOW()), -- Gina Permata → pending
('tp-kjd-08', 'tourn-kejurda-diy-2026', '4ce93d38-a78d-4a6b-be27-62ba8dccea0e', 'tcat-kjd-07', 200000.00, 'paid',   'AH-KJD-08', 'self_register', NOW(), NOW(), NOW()), -- Hendra Wijaya → Compound Umum Putra
('tp-kjd-09', 'tourn-kejurda-diy-2026', '74ba7fa6-19ce-41d4-8d6e-e790f822c9ca', 'tcat-kjd-06', 200000.00, 'pending','AH-KJD-09', 'self_register', NOW(), NOW(), NOW()), -- Intan Sari → pending
('tp-kjd-10', 'tourn-kejurda-diy-2026', 'f0632fb1-60f9-47be-9426-65475b80b766', 'tcat-kjd-05', 200000.00, 'paid',   'AH-KJD-10', 'self_register', NOW(), NOW(), NOW()), -- Joko Firmansyah → paid

-- POPDA Sleman T2 — 6 participants (U-15 & U-18)
('tp-ppd-01', 'tourn-popda-sleman-2026', '8e2ed4ef-5768-4a53-a827-33385edadf9b', 'tcat-ppd-01', 150000.00, 'paid',   'AH-PPD-01', 'self_register', NOW(), NOW(), NOW()), -- Faris Aditya → U-15 Putra
('tp-ppd-02', 'tourn-popda-sleman-2026', 'd9ec12bb-0842-4ed9-bda7-7b1596a8b86b', 'tcat-ppd-02', 150000.00, 'paid',   'AH-PPD-02', 'self_register', NOW(), NOW(), NOW()), -- Dewi Anggraini → U-15 Putri
('tp-ppd-03', 'tourn-popda-sleman-2026', '4b075b44-c1d7-4f16-8017-3a34615836f5', 'tcat-ppd-03', 150000.00, 'paid',   'AH-PPD-03', 'self_register', NOW(), NOW(), NOW()), -- Rizky Kurniawan → U-18 Putra
('tp-ppd-04', 'tourn-popda-sleman-2026', '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'tcat-ppd-01', 150000.00, 'paid',   'AH-PPD-04', 'self_register', NOW(), NOW(), NOW()), -- Andi Saputra → U-15 Putra
('tp-ppd-05', 'tourn-popda-sleman-2026', '54e9646f-c667-4b87-970c-74da698caefb', 'tcat-ppd-02', 150000.00, 'pending','AH-PPD-05', 'self_register', NOW(), NOW(), NOW()), -- Siti Rahmawati → pending
('tp-ppd-06', 'tourn-popda-sleman-2026', '0d635052-2486-4fea-8ac0-570246d2d885', 'tcat-ppd-03', 150000.00, 'paid',   'AH-PPD-06', 'self_register', NOW(), NOW(), NOW()), -- Eko Pranoto → U-18 Putra

-- Latber SAC T3 — 6 participants (scoring data)
('tp-lat-01', 'tourn-latber-sac-2026', '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'tcat-lat-01', 50000.00, 'paid', 'AH-LAT-01', 'self_register', DATE_SUB(NOW(), INTERVAL 70 DAY), DATE_SUB(NOW(), INTERVAL 70 DAY), NOW()), -- Andi
('tp-lat-02', 'tourn-latber-sac-2026', '54e9646f-c667-4b87-970c-74da698caefb', 'tcat-lat-02', 50000.00, 'paid', 'AH-LAT-02', 'self_register', DATE_SUB(NOW(), INTERVAL 70 DAY), DATE_SUB(NOW(), INTERVAL 70 DAY), NOW()), -- Siti
('tp-lat-03', 'tourn-latber-sac-2026', '0d635052-2486-4fea-8ac0-570246d2d885', 'tcat-lat-01', 50000.00, 'paid', 'AH-LAT-03', 'self_register', DATE_SUB(NOW(), INTERVAL 70 DAY), DATE_SUB(NOW(), INTERVAL 70 DAY), NOW()), -- Eko
('tp-lat-04', 'tourn-latber-sac-2026', 'daea3f95-fb72-405b-986b-ab69185c5b76', 'tcat-lat-01', 50000.00, 'paid', 'AH-LAT-04', 'self_register', DATE_SUB(NOW(), INTERVAL 70 DAY), DATE_SUB(NOW(), INTERVAL 70 DAY), NOW()), -- Fajar
('tp-lat-05', 'tourn-latber-sac-2026', '82477f4f-9da9-4ca0-86ad-b77c6e161b64', 'tcat-lat-02', 50000.00, 'paid', 'AH-LAT-05', 'self_register', DATE_SUB(NOW(), INTERVAL 70 DAY), DATE_SUB(NOW(), INTERVAL 70 DAY), NOW()), -- Citra
('tp-lat-06', 'tourn-latber-sac-2026', 'b6711f75-85c4-4413-af54-f474f9b62635', 'tcat-lat-02', 50000.00, 'paid', 'AH-LAT-06', 'self_register', DATE_SUB(NOW(), INTERVAL 70 DAY), DATE_SUB(NOW(), INTERVAL 70 DAY), NOW()); -- Gina

-- Set last_reregistration_at for some (checked in)
UPDATE tournament_participants SET last_reregistration_at = DATE_SUB(NOW(), INTERVAL 65 DAY)
WHERE uuid IN ('tp-lat-01','tp-lat-02','tp-lat-03','tp-lat-04');

-- ============================================================
-- STEP 9: PAYMENT TRANSACTIONS
-- ============================================================

INSERT IGNORE INTO payment_transactions (uuid, reference, user_id, tournament_id, registration_id, amount, fee_amount, total_amount, payment_method, payment_channel, status, paid_at, created_at, updated_at)
VALUES
('ptx-kjd-01', 'REF-KJD-2026-001', '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'tourn-kejurda-diy-2026', 'tp-kjd-01', 200000.00, 6000.00, 206000.00, 'BCA Virtual Account', 'BCAVA', 'paid',    DATE_SUB(NOW(), INTERVAL 15 DAY), DATE_SUB(NOW(), INTERVAL 16 DAY), NOW()),
('ptx-kjd-02', 'REF-KJD-2026-002', '54e9646f-c667-4b87-970c-74da698caefb', 'tourn-kejurda-diy-2026', 'tp-kjd-02', 200000.00, 6000.00, 206000.00, 'GoPay',              'GOPAY',  'paid',    DATE_SUB(NOW(), INTERVAL 14 DAY), DATE_SUB(NOW(), INTERVAL 15 DAY), NOW()),
('ptx-kjd-03', 'REF-KJD-2026-003', 'ba5c9c1e-1bf7-4014-ac88-539cc6947e06', 'tourn-kejurda-diy-2026', 'tp-kjd-03', 200000.00, 6000.00, 206000.00, 'QRIS',               'QRIS',   'paid',    DATE_SUB(NOW(), INTERVAL 13 DAY), DATE_SUB(NOW(), INTERVAL 14 DAY), NOW()),
('ptx-kjd-07', 'REF-KJD-2026-007', 'b6711f75-85c4-4413-af54-f474f9b62635', 'tourn-kejurda-diy-2026', 'tp-kjd-07', 200000.00, 6000.00, 206000.00, 'BRI Virtual Account','BRIVA',  'pending', NULL,                             DATE_SUB(NOW(), INTERVAL 2 DAY),  NOW()),
('ptx-ppd-01', 'REF-PPD-2026-001', '8e2ed4ef-5768-4a53-a827-33385edadf9b', 'tourn-popda-sleman-2026', 'tp-ppd-01', 150000.00, 4500.00, 154500.00, 'Dana',               'DANA',   'paid',    DATE_SUB(NOW(), INTERVAL 20 DAY), DATE_SUB(NOW(), INTERVAL 21 DAY), NOW()),
('ptx-ppd-02', 'REF-PPD-2026-002', 'd9ec12bb-0842-4ed9-bda7-7b1596a8b86b', 'tourn-popda-sleman-2026', 'tp-ppd-02', 150000.00, 4500.00, 154500.00, 'OVO',                'OVO',    'paid',    DATE_SUB(NOW(), INTERVAL 19 DAY), DATE_SUB(NOW(), INTERVAL 20 DAY), NOW()),
('ptx-ppd-05', 'REF-PPD-2026-005', '54e9646f-c667-4b87-970c-74da698caefb', 'tourn-popda-sleman-2026', 'tp-ppd-05', 150000.00, 4500.00, 154500.00, 'Mandiri Virtual',    'MANDIRIVA','pending', NULL,                           DATE_SUB(NOW(), INTERVAL 1 DAY),  NOW()),
('ptx-lat-01', 'REF-LAT-2026-001', '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'tourn-latber-sac-2026',   'tp-lat-01', 50000.00,  0.00,   50000.00,  'Transfer Bank',      'MANUAL', 'paid',    DATE_SUB(NOW(), INTERVAL 72 DAY), DATE_SUB(NOW(), INTERVAL 72 DAY), NOW());

-- Update participant.payment_id
UPDATE tournament_participants SET payment_id = 'ptx-kjd-01' WHERE uuid = 'tp-kjd-01';
UPDATE tournament_participants SET payment_id = 'ptx-kjd-02' WHERE uuid = 'tp-kjd-02';
UPDATE tournament_participants SET payment_id = 'ptx-kjd-03' WHERE uuid = 'tp-kjd-03';
UPDATE tournament_participants SET payment_id = 'ptx-ppd-01' WHERE uuid = 'tp-ppd-01';
UPDATE tournament_participants SET payment_id = 'ptx-lat-01' WHERE uuid = 'tp-lat-01';

-- ============================================================
-- STEP 10: TOURNAMENT TARGETS (for Latber — scoring test)
-- ============================================================

INSERT IGNORE INTO tournament_targets (uuid, tournament_uuid, target_name, board_number, created_at, updated_at)
VALUES
  ('ttgt-lat-01', 'tourn-latber-sac-2026', '1A', 1, NOW(), NOW()),
  ('ttgt-lat-02', 'tourn-latber-sac-2026', '1B', 1, NOW(), NOW()),
  ('ttgt-lat-03', 'tourn-latber-sac-2026', '1C', 1, NOW(), NOW()),
  ('ttgt-lat-04', 'tourn-latber-sac-2026', '2A', 2, NOW(), NOW()),
  ('ttgt-lat-05', 'tourn-latber-sac-2026', '2B', 2, NOW(), NOW()),
  ('ttgt-lat-06', 'tourn-latber-sac-2026', '2C', 2, NOW(), NOW());

-- ============================================================
-- STEP 11: QUALIFICATION SESSION for Latber (T3)
-- ============================================================

INSERT IGNORE INTO qualification_sessions (uuid, tournament_uuid, session_code, session_date, name, start_time, end_time, total_ends, arrows_per_end, is_locked, created_at, updated_at)
VALUES
  ('qsess-lat-01', 'tourn-latber-sac-2026', 'LAT-Q-01', DATE_SUB(CURDATE(), INTERVAL 67 DAY), 'Sesi Kualifikasi Pagi', DATE_SUB(NOW(), INTERVAL 67 DAY), DATE_SUB(NOW(), INTERVAL 66 DAY), 6, 6, 1, DATE_SUB(NOW(), INTERVAL 70 DAY), NOW());

INSERT IGNORE INTO qualification_session_categories (session_uuid, category_uuid)
VALUES
  ('qsess-lat-01', 'tcat-lat-01'),
  ('qsess-lat-01', 'tcat-lat-02');

-- ============================================================
-- STEP 12: TARGET ASSIGNMENTS (qualification)
-- ============================================================

INSERT IGNORE INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, created_at, updated_at)
VALUES
  ('qta-lat-01', 'qsess-lat-01', 'tp-lat-01', 'ttgt-lat-01', NOW(), NOW()), -- Andi → 1A
  ('qta-lat-02', 'qsess-lat-01', 'tp-lat-02', 'ttgt-lat-02', NOW(), NOW()), -- Siti → 1B
  ('qta-lat-03', 'qsess-lat-01', 'tp-lat-03', 'ttgt-lat-03', NOW(), NOW()), -- Eko  → 1C
  ('qta-lat-04', 'qsess-lat-01', 'tp-lat-04', 'ttgt-lat-04', NOW(), NOW()), -- Fajar → 2A
  ('qta-lat-05', 'qsess-lat-01', 'tp-lat-05', 'ttgt-lat-05', NOW(), NOW()), -- Citra → 2B
  ('qta-lat-06', 'qsess-lat-01', 'tp-lat-06', 'ttgt-lat-06', NOW(), NOW()); -- Gina  → 2C

-- ============================================================
-- STEP 13: QUALIFICATION END SCORES — 6 ends × 6 archers
-- (All 6 ends with arrow-by-arrow for archer 1 & 2)
-- ============================================================

-- Andi Saputra (tp-lat-01) — Total: 165 (28+27+28+27+28+27)
DELETE FROM qualification_arrow_scores WHERE end_score_uuid LIKE 'qes-lat-andi%';
DELETE FROM qualification_end_scores WHERE uuid LIKE 'qes-lat-andi%';

INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-lat-andi-01', 'qsess-lat-01', 'tp-lat-01', 1, 28, 1, 2, DATE_SUB(NOW(), INTERVAL 67 DAY)),
  ('qes-lat-andi-02', 'qsess-lat-01', 'tp-lat-01', 2, 27, 0, 1, DATE_SUB(NOW(), INTERVAL 67 DAY)),
  ('qes-lat-andi-03', 'qsess-lat-01', 'tp-lat-01', 3, 28, 1, 2, DATE_SUB(NOW(), INTERVAL 67 DAY)),
  ('qes-lat-andi-04', 'qsess-lat-01', 'tp-lat-01', 4, 27, 0, 1, DATE_SUB(NOW(), INTERVAL 67 DAY)),
  ('qes-lat-andi-05', 'qsess-lat-01', 'tp-lat-01', 5, 28, 1, 2, DATE_SUB(NOW(), INTERVAL 67 DAY)),
  ('qes-lat-andi-06', 'qsess-lat-01', 'tp-lat-01', 6, 27, 0, 1, DATE_SUB(NOW(), INTERVAL 67 DAY));

INSERT INTO qualification_arrow_scores (uuid, end_score_uuid, arrow_number, score, is_x, created_at) VALUES
  ('qas-andi-01-1','qes-lat-andi-01',1,10,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-01-2','qes-lat-andi-01',2,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-01-3','qes-lat-andi-01',3,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-01-4','qes-lat-andi-01',4,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-01-5','qes-lat-andi-01',5,8,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-01-6','qes-lat-andi-01',6,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-02-1','qes-lat-andi-02',1,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-02-2','qes-lat-andi-02',2,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-02-3','qes-lat-andi-02',3,8,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-02-4','qes-lat-andi-02',4,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-02-5','qes-lat-andi-02',5,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-02-6','qes-lat-andi-02',6,8,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-03-1','qes-lat-andi-03',1,10,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-03-2','qes-lat-andi-03',2,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-03-3','qes-lat-andi-03',3,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-03-4','qes-lat-andi-03',4,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-03-5','qes-lat-andi-03',5,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-03-6','qes-lat-andi-03',6,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-04-1','qes-lat-andi-04',1,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-04-2','qes-lat-andi-04',2,8,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-04-3','qes-lat-andi-04',3,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-04-4','qes-lat-andi-04',4,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-04-5','qes-lat-andi-04',5,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-04-6','qes-lat-andi-04',6,8,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-05-1','qes-lat-andi-05',1,10,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-05-2','qes-lat-andi-05',2,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-05-3','qes-lat-andi-05',3,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-05-4','qes-lat-andi-05',4,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-05-5','qes-lat-andi-05',5,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-05-6','qes-lat-andi-05',6,8,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-06-1','qes-lat-andi-06',1,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-06-2','qes-lat-andi-06',2,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-06-3','qes-lat-andi-06',3,10,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-06-4','qes-lat-andi-06',4,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-06-5','qes-lat-andi-06',5,9,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qas-andi-06-6','qes-lat-andi-06',6,8,0,DATE_SUB(NOW(),INTERVAL 67 DAY));

-- Eko Pranoto (tp-lat-03) — Total: 158
DELETE FROM qualification_end_scores WHERE uuid LIKE 'qes-lat-eko%';
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-lat-eko-01','qsess-lat-01','tp-lat-03',1,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-eko-02','qsess-lat-01','tp-lat-03',2,27,0,2,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-eko-03','qsess-lat-01','tp-lat-03',3,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-eko-04','qsess-lat-01','tp-lat-03',4,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-eko-05','qsess-lat-01','tp-lat-03',5,27,0,2,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-eko-06','qsess-lat-01','tp-lat-03',6,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY));

-- Fajar Ramadhan (tp-lat-04) — Total: 152
DELETE FROM qualification_end_scores WHERE uuid LIKE 'qes-lat-fajar%';
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-lat-fajar-01','qsess-lat-01','tp-lat-04',1,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-fajar-02','qsess-lat-01','tp-lat-04',2,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-fajar-03','qsess-lat-01','tp-lat-04',3,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-fajar-04','qsess-lat-01','tp-lat-04',4,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-fajar-05','qsess-lat-01','tp-lat-04',5,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-fajar-06','qsess-lat-01','tp-lat-04',6,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY));

-- Siti Rahmawati (tp-lat-02) — Total: 160 (Putri, scored same day)
DELETE FROM qualification_end_scores WHERE uuid LIKE 'qes-lat-siti%';
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-lat-siti-01','qsess-lat-01','tp-lat-02',1,27,1,2,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-siti-02','qsess-lat-01','tp-lat-02',2,27,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-siti-03','qsess-lat-01','tp-lat-02',3,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-siti-04','qsess-lat-01','tp-lat-02',4,27,1,2,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-siti-05','qsess-lat-01','tp-lat-02',5,27,0,2,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-siti-06','qsess-lat-01','tp-lat-02',6,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY));

-- Citra Lestari (tp-lat-05) — Total: 155
DELETE FROM qualification_end_scores WHERE uuid LIKE 'qes-lat-citra%';
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-lat-citra-01','qsess-lat-01','tp-lat-05',1,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-citra-02','qsess-lat-01','tp-lat-05',2,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-citra-03','qsess-lat-01','tp-lat-05',3,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-citra-04','qsess-lat-01','tp-lat-05',4,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-citra-05','qsess-lat-01','tp-lat-05',5,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-citra-06','qsess-lat-01','tp-lat-05',6,26,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY));

-- Gina Permata (tp-lat-06) — Total: 148
DELETE FROM qualification_end_scores WHERE uuid LIKE 'qes-lat-gina%';
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-lat-gina-01','qsess-lat-01','tp-lat-06',1,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-gina-02','qsess-lat-01','tp-lat-06',2,24,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-gina-03','qsess-lat-01','tp-lat-06',3,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-gina-04','qsess-lat-01','tp-lat-06',4,24,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-gina-05','qsess-lat-01','tp-lat-06',5,25,0,0,DATE_SUB(NOW(),INTERVAL 67 DAY)),
  ('qes-lat-gina-06','qsess-lat-01','tp-lat-06',6,25,0,1,DATE_SUB(NOW(),INTERVAL 67 DAY));

-- Update qual_score & qual_rank on participants
UPDATE tournament_participants SET qual_score = 165, qual_rank = 1 WHERE uuid = 'tp-lat-01'; -- Andi
UPDATE tournament_participants SET qual_score = 160, qual_rank = 1 WHERE uuid = 'tp-lat-02'; -- Siti (rank 1 in Putri)
UPDATE tournament_participants SET qual_score = 158, qual_rank = 2 WHERE uuid = 'tp-lat-03'; -- Eko
UPDATE tournament_participants SET qual_score = 152, qual_rank = 3 WHERE uuid = 'tp-lat-04'; -- Fajar
UPDATE tournament_participants SET qual_score = 155, qual_rank = 2 WHERE uuid = 'tp-lat-05'; -- Citra
UPDATE tournament_participants SET qual_score = 148, qual_rank = 3 WHERE uuid = 'tp-lat-06'; -- Gina

-- ============================================================
-- STEP 14: ELIMINATION BRACKET for Latber (Recurve Umum Putra)
-- ============================================================

INSERT IGNORE INTO elimination_brackets (uuid, bracket_id, tournament_uuid, category_uuid, bracket_type, format, bracket_size, status, created_at, updated_at)
VALUES
  ('elb-lat-01', 'ELB-LAT-001', 'tourn-latber-sac-2026', 'tcat-lat-01', 'individual', 'set_system', 4, 'completed', DATE_SUB(NOW(), INTERVAL 66 DAY), NOW());

INSERT IGNORE INTO elimination_entries (uuid, bracket_uuid, participant_type, participant_uuid, seed, qual_total_score, created_at)
VALUES
  ('ele-lat-01', 'elb-lat-01', 'participant', 'tp-lat-01', 1, 165, DATE_SUB(NOW(), INTERVAL 66 DAY)),
  ('ele-lat-02', 'elb-lat-01', 'participant', 'tp-lat-03', 2, 158, DATE_SUB(NOW(), INTERVAL 66 DAY)),
  ('ele-lat-03', 'elb-lat-01', 'participant', 'tp-lat-04', 3, 152, DATE_SUB(NOW(), INTERVAL 66 DAY)),
  ('ele-lat-xx', 'elb-lat-01', 'participant', 'tp-lat-01', 4, 0,   DATE_SUB(NOW(), INTERVAL 66 DAY)); -- bye

-- Semifinal 1: Andi (seed 1) vs Fajar (seed 3) → Andi menang 6-2
INSERT IGNORE INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, winner_entry_uuid, status, total_score_a, total_score_b, created_at, updated_at)
VALUES
  ('elm-lat-sf1', 'SF-1', 'elb-lat-01', 1, 1, 'ele-lat-01', 'ele-lat-03', 'ele-lat-01', 'finished', 6, 2, DATE_SUB(NOW(), INTERVAL 66 DAY), NOW()),
  ('elm-lat-sf2', 'SF-2', 'elb-lat-01', 1, 2, 'ele-lat-02', 'ele-lat-xx', 'ele-lat-02', 'finished', 6, 0, DATE_SUB(NOW(), INTERVAL 66 DAY), NOW()),
  ('elm-lat-gf',  'GF-1', 'elb-lat-01', 2, 1, 'ele-lat-01', 'ele-lat-02', 'ele-lat-01', 'finished', 6, 4, DATE_SUB(NOW(), INTERVAL 65 DAY), NOW());

-- ============================================================
-- STEP 15: BROADCASTS for organizer
-- ============================================================

INSERT IGNORE INTO broadcasts (uuid, tournament_id, organizer_id, title, content, type, status, created_at, updated_at)
VALUES
  ('bcast-kjd-01', 'tourn-kejurda-diy-2026', '413db8ca-88fe-436c-82d1-203a48205e46', 'Informasi Penting: Technical Meeting', 'Technical meeting akan dilaksanakan pada 15 November 2026 pukul 08.00 WIB di Aula GOR Sleman. Semua peserta wajib hadir.', 'announcement', 'published', DATE_SUB(NOW(), INTERVAL 5 DAY), NOW()),
  ('bcast-kjd-02', 'tourn-kejurda-diy-2026', '413db8ca-88fe-436c-82d1-203a48205e46', 'Reminder: Batas Pembayaran Pendaftaran', 'Batas akhir pembayaran pendaftaran adalah 10 November 2026. Segera selesaikan pembayaran agar slot terjamin.', 'reminder', 'published', DATE_SUB(NOW(), INTERVAL 3 DAY), NOW()),
  ('bcast-ppd-01', 'tourn-popda-sleman-2026', '413db8ca-88fe-436c-82d1-203a48205e46', 'Pembukaan Pendaftaran POPDA Sleman 2026', 'Pendaftaran POPDA Sleman 2026 resmi dibuka! Daftarkan dirimu sekarang sebelum kuota habis.', 'announcement', 'published', DATE_SUB(NOW(), INTERVAL 10 DAY), NOW());

-- ============================================================
-- STEP 16: ARCHER CERTIFICATES (for Latber — past tournament)
-- ============================================================

INSERT IGNORE INTO archer_certificates (uuid, tournament_id, archer_id, registration_id, certificate_no, issue_date, pdf_url, created_at)
VALUES
  ('cert-lat-01', 'tourn-latber-sac-2026', '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'tp-lat-01', 'CERT-LAT-2026-001', DATE_SUB(NOW(), INTERVAL 60 DAY), 'https://picsum.photos/seed/cert-lat-2026/800/600', DATE_SUB(NOW(), INTERVAL 60 DAY)),
  ('cert-lat-02', 'tourn-latber-sac-2026', '0d635052-2486-4fea-8ac0-570246d2d885', 'tp-lat-03', 'CERT-LAT-2026-002', DATE_SUB(NOW(), INTERVAL 60 DAY), 'https://picsum.photos/seed/cert-lat-2026-2/800/600', DATE_SUB(NOW(), INTERVAL 60 DAY)),
  ('cert-lat-03', 'tourn-latber-sac-2026', '54e9646f-c667-4b87-970c-74da698caefb', 'tp-lat-02', 'CERT-LAT-2026-003', DATE_SUB(NOW(), INTERVAL 60 DAY), 'https://picsum.photos/seed/cert-lat-2026-3/800/600', DATE_SUB(NOW(), INTERVAL 60 DAY));

-- ============================================================
-- STEP 17: NOTIFICATIONS
-- ============================================================

INSERT IGNORE INTO notifications (uuid, user_id, user_role, type, title, message, is_read, created_at)
VALUES
  (UUID(), '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'archer', 'success', 'Pembayaran Dikonfirmasi', 'Pembayaran Kejurda DIY 2026 kamu sudah dikonfirmasi. Selamat bertanding!', 0, DATE_SUB(NOW(), INTERVAL 15 DAY)),
  (UUID(), '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'archer', 'info',    'Technical Meeting Kejurda DIY', 'Technical meeting akan dilakukan 15 Nov 2026 pukul 08.00 WIB.', 0, DATE_SUB(NOW(), INTERVAL 5 DAY)),
  (UUID(), '01806bbe-b7cc-413c-b7ec-abd69f2775d4', 'archer', 'success', 'Sertifikat Tersedia', 'Sertifikat Latber SAC Juli 2026 kamu sudah bisa diunduh.', 1, DATE_SUB(NOW(), INTERVAL 60 DAY)),
  (UUID(), '54e9646f-c667-4b87-970c-74da698caefb', 'archer', 'warning', 'Pembayaran Belum Dikonfirmasi', 'Pembayaran POPDA Sleman 2026 kamu belum dikonfirmasi. Cek status pembayaran.', 0, DATE_SUB(NOW(), INTERVAL 1 DAY)),
  (UUID(), '413db8ca-88fe-436c-82d1-203a48205e46', 'organizer', 'info', 'Peserta Baru: Kejurda DIY', '10 peserta baru telah mendaftar untuk Kejurda DIY 2026.', 0, DATE_SUB(NOW(), INTERVAL 10 DAY));

SET FOREIGN_KEY_CHECKS = 1;

SELECT 'COMPREHENSIVE SEED COMPLETE' AS status;
SELECT COUNT(*) AS archer_count FROM archers;
SELECT COUNT(*) AS tournament_count FROM tournaments;
SELECT COUNT(*) AS participant_count FROM tournament_participants;
SELECT COUNT(*) AS payment_count FROM payment_transactions;
SELECT COUNT(*) AS end_score_count FROM qualification_end_scores;
SELECT COUNT(*) AS elim_match_count FROM elimination_matches;
SELECT COUNT(*) AS cert_count FROM archer_certificates;
