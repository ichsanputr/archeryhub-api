-- ============================================================
-- SEED DATABASE: Archeris
-- ============================================================

-- 1. UPDATE ALL IMAGE URLs TO PICSUM
-- ============================================================

UPDATE archers SET avatar_url = CONCAT('https://picsum.photos/seed/', uuid, '/200/200') WHERE avatar_url IS NULL OR avatar_url = '';
UPDATE archers SET banner_url = CONCAT('https://picsum.photos/seed/', uuid, '-banner/1200/400') WHERE banner_url IS NULL OR banner_url = '';

UPDATE clubs SET logo_url = CONCAT('https://picsum.photos/seed/', COALESCE(slug, uuid), '/200/200') WHERE logo_url IS NULL OR logo_url = '';

UPDATE news SET image_url = CONCAT('https://picsum.photos/seed/', COALESCE(slug, uuid), '/800/400');

UPDATE media SET url = CONCAT('https://picsum.photos/seed/', uuid, '/800/600');

-- Update stewie4king avatar
UPDATE archers SET avatar_url = 'https://picsum.photos/seed/stewie4king/200/200', banner_url = 'https://picsum.photos/seed/stewie4king-banner/1200/400' WHERE email = 'stewie4king@gmail.com';

-- 2. INSERT ORGANIZER (if not exists)
-- ============================================================

INSERT IGNORE INTO organizers (uuid, slug, name, acronym, email, password, city, status, subscription_plan_id, subscription_status, created_at, updated_at)
VALUES (
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  'perpani-sleman',
  'Perpani Sleman',
  'PPS',
  'ichsanfadhil67@gmail.com',
  '123456',
  'Sleman',
  'active',
  5,
  'active',
  NOW(),
  NOW()
);

-- 3. INSERT EVENTS
-- ============================================================

-- Event 1: Seleksi POPDA Kab. Sleman 2026 (upcoming)
INSERT IGNORE INTO events (uuid, slug, code, name, short_name, venue, location, city, start_date, end_date, registration_deadline, description, banner_url, logo_url, type, entry_fee, status, organizer_id, total_prize, created_at, updated_at)
VALUES (
  '7247378c-b3cb-46d7-9ea3-78526733e7a7',
  'seleksi-popda-kabsleman-2026',
  'POPDA-2026',
  'Seleksi POPDA Kab. Sleman 2026',
  'POPDA Sleman',
  'GOR Sleman',
  'Jl. Stadion No. 1, Sleman',
  'Sleman',
  '2026-08-15 08:00:00',
  '2026-08-17 17:00:00',
  '2026-08-10 23:59:59',
  'Seleksi atlet panahan untuk POPDA Kabupaten Sleman tahun 2026. Terbuka untuk umum kategori U-15 dan U-18.',
  'https://picsum.photos/seed/popda-sleman-2026/1200/600',
  'https://picsum.photos/seed/popda-sleman-logo/400/400',
  'individual',
  150000.00,
  'published',
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  5000000.00,
  NOW(),
  NOW()
);

-- Event 2: Kejurda DIY 2026 (upcoming)
INSERT IGNORE INTO events (uuid, slug, code, name, short_name, venue, location, city, start_date, end_date, registration_deadline, description, banner_url, logo_url, type, entry_fee, status, organizer_id, total_prize, created_at, updated_at)
VALUES (
  '8b9c0d1e-2f3a-4b5c-6d7e-8f9a0b1c2d3e',
  'kejurda-diy-2026',
  'KEJURDA-2026',
  'Kejuaraan Daerah DIY 2026',
  'Kejurda DIY',
  'Stadion Maguwoharjo',
  'Jl. Maguwoharjo No. 1, Sleman',
  'Sleman',
  '2026-09-20 08:00:00',
  '2026-09-23 17:00:00',
  '2026-09-15 23:59:59',
  'Kejuaraan Daerah Istimewa Yogyakarta tahun 2026. Memperebutkan medali emas, perak, dan perunggu.',
  'https://picsum.photos/seed/kejurda-diy-2026/1200/600',
  'https://picsum.photos/seed/kejurda-diy-logo/400/400',
  'individual',
  200000.00,
  'published',
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  10000000.00,
  NOW(),
  NOW()
);

-- Event 3: Latihan Bersama Sleman Archery (past)
INSERT IGNORE INTO events (uuid, slug, code, name, short_name, venue, location, city, start_date, end_date, registration_deadline, description, banner_url, logo_url, type, entry_fee, status, organizer_id, total_prize, created_at, updated_at)
VALUES (
  '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f',
  'latihan-bersama-sleman-archery',
  'LATIH-2026',
  'Latihan Bersama Sleman Archery Club',
  'Latber SAC',
  'Lapangan Tridadi',
  'Jl. Tridadi No. 10, Sleman',
  'Sleman',
  '2026-05-10 08:00:00',
  '2026-05-10 17:00:00',
  '2026-05-08 23:59:59',
  'Latihan bersama antar klub panahan se-Kabupaten Sleman. Cocok untuk persiapan POPDA.',
  'https://picsum.photos/seed/latber-sleman/1200/600',
  'https://picsum.photos/seed/latber-sleman-logo/400/400',
  'individual',
  50000.00,
  'published',
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  1000000.00,
  NOW(),
  NOW()
);

-- 4. INSERT EVENT CATEGORIES
-- ============================================================

-- Categories for POPDA (Event 1)
INSERT IGNORE INTO event_categories (uuid, event_id, division_uuid, category_uuid, event_type_uuid, gender_division_uuid, max_participants, status)
VALUES
  ('13817e7f-694a-47d7-add7-e1fd38511e1d', '7247378c-b3cb-46d7-9ea3-78526733e7a7', '33502405-e5ff-4725-a882-d279666fe35c', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 50, 'active'),
  ('248b8f80-5a6b-4c8e-add8-e2fd48622e2e', '7247378c-b3cb-46d7-9ea3-78526733e7a7', '33502405-e5ff-4725-a882-d279666fe35c', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 50, 'active'),
  ('359c9f91-6b7c-4d9f-bee9-e3fd49733f3f', '7247378c-b3cb-46d7-9ea3-78526733e7a7', '33502405-e5ff-4725-a882-d279666fe35c', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 50, 'active');

-- Categories for Kejurda DIY (Event 2)
INSERT IGNORE INTO event_categories (uuid, event_id, division_uuid, category_uuid, event_type_uuid, gender_division_uuid, max_participants, status)
VALUES
  ('46ad0a02-7c8d-4e0a-ff0a-f4fe50844a4a', '8b9c0d1e-2f3a-4b5c-6d7e-8f9a0b1c2d3e', '331b52a7-812d-4dde-aeaf-978e79bf293a', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 40, 'active'),
  ('57be1b13-8d9e-4f1b-001b-f5af61955b5b', '8b9c0d1e-2f3a-4b5c-6d7e-8f9a0b1c2d3e', '331b52a7-812d-4dde-aeaf-978e79bf293a', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 40, 'active'),
  ('68cf2c24-9eaf-4a2c-112c-f6b630726c6c', '8b9c0d1e-2f3a-4b5c-6d7e-8f9a0b1c2d3e', '331b52a7-812d-4dde-aeaf-978e79bf293a', 'f0ab19a5-efe2-4ceb-b177-57966249af04', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 40, 'active'),
  ('79d03d35-afb0-4b3d-223d-f7c741837d7d', '8b9c0d1e-2f3a-4b5c-6d7e-8f9a0b1c2d3e', '331b52a7-812d-4dde-aeaf-978e79bf293a', 'f0ab19a5-efe2-4ceb-b177-57966249af04', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 40, 'active');

-- Categories for Latber (Event 3)
INSERT IGNORE INTO event_categories (uuid, event_id, division_uuid, category_uuid, event_type_uuid, gender_division_uuid, max_participants, status)
VALUES
  ('8ae14e46-afb0-4c3e-334e-f8d852948e8e', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', '33502405-e5ff-4725-a882-d279666fe35c', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 30, 'active'),
  ('9bf25f57-bf0c-4d4f-445f-f9e96395a9af', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', '33502405-e5ff-4725-a882-d279666fe35c', 'f235b870-724b-44ac-8683-b665df0c0548', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 30, 'active');

-- 5. INSERT EVENT IMAGES
-- ============================================================

INSERT IGNORE INTO event_images (uuid, event_id, url, caption, display_order, is_primary)
VALUES
  ('img-0001-0001', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'https://picsum.photos/seed/popda-banner/1200/600', 'Banner POPDA Sleman 2026', 0, 1),
  ('img-0001-0002', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'https://picsum.photos/seed/popda-gallery-1/800/600', 'Suasana Seleksi POPDA', 1, 0),
  ('img-0001-0003', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'https://picsum.photos/seed/popda-gallery-2/800/600', 'Para Peserta', 2, 0);

-- 6. INSERT EVENT SCHEDULES
-- ============================================================

INSERT IGNORE INTO event_schedule (uuid, event_id, title, description, start_time, end_time, day_order, sort_order)
VALUES
  ('sch-0001-001', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'Technical Meeting', 'Pertemuan teknis dan briefing juri', '2026-08-15 08:00:00', '2026-08-15 10:00:00', 1, 1),
  ('sch-0001-002', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'Kualifikasi U-15 Putra', 'Babak kualifikasi kategori U-15 Putra', '2026-08-15 10:30:00', '2026-08-15 16:00:00', 1, 2),
  ('sch-0001-003', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'Kualifikasi U-15 Putri', 'Babak kualifikasi kategori U-15 Putri', '2026-08-16 08:00:00', '2026-08-16 14:00:00', 2, 3),
  ('sch-0001-004', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'Eliminasi & Final', 'Babak eliminasi dan final semua kategori', '2026-08-17 08:00:00', '2026-08-17 17:00:00', 3, 4);

-- 7. INSERT SELLER
-- ============================================================

INSERT IGNORE INTO sellers (uuid, slug, store_name, email, password, phone, city, description, status, is_verified, created_at, updated_at)
VALUES (
  'sel-0001-0000-0000-0000-000000000001',
  'panahan-store-1',
  'Panahan Store 1',
  'seller@panahan.com',
  '12345',
  '081234567891',
  'Sleman',
  'Toko perlengkapan panahan terlengkap di Yogyakarta. Menyediakan busur, anak panah, aksesoris, dan perlengkapan panahan lainnya.',
  'active',
  1,
  NOW(),
  NOW()
);

-- 8. INSERT PRODUCTS
-- ============================================================

INSERT IGNORE INTO products (uuid, seller_id, name, slug, description, price, category, stock, status, image_url, specifications, created_at, updated_at)
VALUES
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Hoyt Formula Xi Riser', 'hoyt-formula-xi-riser', 'Riser aluminium premium untuk recurve bow.', 4500000.00, 'equipment', 10, 'active', 'https://picsum.photos/seed/hoyt-riser/600/600', '{"brand": "Hoyt", "material": "Aluminium", "weight": "1250g", "color": ["Black", "Silver", "Blue"]}', NOW(), NOW()),
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Easton X10 Arrows', 'easton-x10-arrows', 'Anak panah karbon premium untuk kompetisi.', 3200000.00, 'equipment', 20, 'active', 'https://picsum.photos/seed/easton-x10/600/600', '{"brand": "Easton", "material": "Carbon", "length": "70cm", "sizes": ["450", "500", "550"]}', NOW(), NOW()),
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Shibuya Ultima Sight', 'shibuya-ultima-sight', 'Alat bidik presisi tinggi untuk recurve.', 1850000.00, 'accessories', 15, 'active', 'https://picsum.photos/seed/shibuya-sight/600/600', '{"brand": "Shibuya", "material": "Stainless Steel", "color": ["Black", "Red"]}', NOW(), NOW()),
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Cartel Stabilizer Set', 'cartel-stabilizer-set', 'Set stabilizer panjang 30 inci untuk recurve.', 975000.00, 'accessories', 25, 'active', 'https://picsum.photos/seed/cartel-stabilizer/600/600', '{"brand": "Cartel", "length": "30 inch", "color": ["Black"]}', NOW(), NOW()),
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Win & Win Finger Tab', 'win-win-finger-tab', 'Pelindung jari kulit sapi premium.', 350000.00, 'accessories', 30, 'active', 'https://picsum.photos/seed/winwin-tab/600/600', '{"brand": "Win & Win", "material": "Leather", "sizes": ["S", "M", "L", "XL"]}', NOW(), NOW()),
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Kaos Archery Hub Official', 'kaos-archery-hub-official', 'Kaos official Archery Hub, bahan cotton combed 30s.', 120000.00, 'apparel', 50, 'active', 'https://picsum.photos/seed/ah-merch/600/600', '{"material": "Cotton Combed 30s", "sizes": ["S", "M", "L", "XL", "XXL"], "color": ["Navy", "Black", "White"]}', NOW(), NOW()),
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Beiter Nock Set', 'beiter-nock-set', 'Set nock plastik presisi untuk panah karbon.', 85000.00, 'accessories', 100, 'active', 'https://picsum.photos/seed/beiter-nock/600/600', '{"brand": "Beiter", "material": "Plastic", "sizes": ["Small", "Medium", "Large"]}', NOW(), NOW()),
  (UUID(), 'sel-0001-0000-0000-0000-000000000001', 'Buku Panduan Panahan Dasar', 'buku-panduan-panahan-dasar', 'Buku panduan teknik dasar panahan untuk pemula.', 75000.00, 'training', 40, 'active', 'https://picsum.photos/seed/buku-panahan/600/600', '{"author": "Tim Archery Hub", "pages": 120, "language": "Indonesia"}', NOW(), NOW());

-- 9. INSERT ADDITIONAL ARCHERS WITH REALISTIC DATA
-- ============================================================

-- Update existing archers with proper data
UPDATE archers SET avatar_url = CONCAT('https://picsum.photos/seed/', uuid, '/200/200'), city = 'Sleman' WHERE city IS NULL OR city = '';

-- 10. MARK EXISTING PARTICIPANTS WITH PROPER EVENT/CATEGORY REFERENCES
-- ============================================================
-- The event_participants already exist for event 7247378c-b3cb-46d7-9ea3-78526733e7a7
-- with category 13817e7f-694a-47d7-add7-e1fd38511e1d which is now properly inserted above.

-- Update stewie4king's participant to use picsum QR
UPDATE event_participants SET qr_raw = CONCAT('AH-', uuid) WHERE qr_raw IS NULL;

-- 11. INSERT PAYMENT TRANSACTIONS for stewie4king
-- ============================================================

INSERT IGNORE INTO payment_transactions (uuid, reference, tripay_reference, user_id, event_id, registration_id, amount, fee_amount, total_amount, payment_method, payment_channel, status, paid_at, created_at, updated_at)
VALUES
  (UUID(), CONCAT('PAY-POPDA-', UUID()), NULL, '11e0974c-a7f6-4b76-811f-5291137f164e', '7247378c-b3cb-46d7-9ea3-78526733e7a7', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 150000.00, 4500.00, 154500.00, 'BCA Virtual Account', 'BCAVA', 'paid', NOW(), DATE_SUB(NOW(), INTERVAL 7 DAY), NOW()),
  (UUID(), CONCAT('PAY-KEJURDA-', UUID()), NULL, '11e0974c-a7f6-4b76-811f-5291137f164e', '8b9c0d1e-2f3a-4b5c-6d7e-8f9a0b1c2d3e', NULL, 200000.00, 6000.00, 206000.00, 'GoPay', 'GOPAY', 'pending', NULL, NOW(), NOW()),
  (UUID(), CONCAT('PAY-LATBER-', UUID()), NULL, '11e0974c-a7f6-4b76-811f-5291137f164e', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', NULL, 50000.00, 0.00, 50000.00, 'QRIS', 'QRIS', 'paid', DATE_SUB(NOW(), INTERVAL 30 DAY), DATE_SUB(NOW(), INTERVAL 30 DAY), NOW());

-- 12. INSERT QUALIFICATION SESSION (for Latber - past event)
-- ============================================================

INSERT IGNORE INTO qualification_sessions (uuid, event_uuid, session_code, session_date, name, start_time, end_time, total_ends, arrows_per_end, created_at)
VALUES
  ('qs-0001', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', 'LATBER-Q-01', DATE_SUB(CURDATE(), INTERVAL 30 DAY), 'Sesi Kualifikasi Pagi', DATE_SUB(NOW(), INTERVAL 30 DAY), DATE_SUB(NOW(), INTERVAL 29 DAY), 6, 6, DATE_SUB(NOW(), INTERVAL 30 DAY));

INSERT IGNORE INTO qualification_session_categories (session_uuid, category_uuid)
VALUES
  ('qs-0001', '8ae14e46-afb0-4c3e-334e-f8d852948e8e'),
  ('qs-0001', '9bf25f57-bf0c-4d4f-445f-f9e96395a9af');

-- 13. INSERT TARGET BOARDS (for session)
-- ============================================================

INSERT IGNORE INTO event_targets (uuid, event_uuid, target_name, board_number, created_at)
VALUES
  ('tgt-0001', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', '1A', 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('tgt-0002', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', '1B', 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('tgt-0003', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', '2A', 2, DATE_SUB(NOW(), INTERVAL 30 DAY));

INSERT IGNORE INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, created_at)
VALUES
  ('qta-0001', 'qs-0001', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 'tgt-0001', DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qta-0002', 'qs-0001', '44f987e3-de70-41b7-a238-362c93b36b96', 'tgt-0002', DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qta-0003', 'qs-0001', 'b14417f7-3764-4e58-a198-122801939c21', 'tgt-0003', DATE_SUB(NOW(), INTERVAL 30 DAY));

-- 14. INSERT ELIMINATION BRACKET
-- ============================================================

INSERT IGNORE INTO elimination_brackets (uuid, bracket_id, event_uuid, category_uuid, bracket_type, format, bracket_size, status, created_at)
VALUES
  ('elb-0001', 'ELB-LATBER-001', '9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f', '8ae14e46-afb0-4c3e-334e-f8d852948e8e', 'individual', 'recurve_set', 8, 'completed', DATE_SUB(NOW(), INTERVAL 30 DAY));

-- 15. UPDATE PLAN.MD COMPLETION STATUS
-- ============================================================
-- All seeding completed successfully.

SELECT 'SEEDING COMPLETE' AS status;
