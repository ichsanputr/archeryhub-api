-- Seed 20 Archers with Complete Data and Register to Indonesian Open Championship 2025
-- Event UUID: f9272ae0-f76f-11f0-87db-c3c8a1ce2650
-- Club UUID: 218e8243-6e03-41b3-a551-65936bd12815

-- Category 1: Recurve Senior Men (0748da0d-f832-11f0-87db-c3c8a1ce2650) - 7 archers
-- Category 2: Recurve Senior Women (0748dc7a-f832-11f0-87db-c3c8a1ce2650) - 7 archers
-- Category 3: Barebow Senior Men (0748ddc4-f832-11f0-87db-c3c8a1ce2650) - 6 archers

-- ============================================
-- CATEGORY 1: Recurve Senior Men (7 archers)
-- ============================================

-- Archer 1
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'budi_santoso',
    'budi-santoso',
    'budi.santoso@example.com',
    'BUD1234',
    'Budi Santoso',
    'Budi',
    '1995-03-15',
    'male',
    'IDN',
    '081234567001',
    'Jl. Merdeka No. 45, Jakarta Pusat',
    'Jakarta',
    'DKI Jakarta',
    '10110',
    'NID320123456789',
    'recurve',
    'right',
    8,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer1_uuid = (SELECT uuid FROM archers WHERE email = 'budi.santoso@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer1_uuid,
    '0748da0d-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 2
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'andi_wijaya',
    'andi-wijaya',
    'andi.wijaya@example.com',
    'AND5678',
    'Andi Wijaya',
    'Andi',
    '1992-07-22',
    'male',
    'IDN',
    '081234567002',
    'Jl. Sudirman No. 78, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12190',
    'NID320123456790',
    'recurve',
    'right',
    10,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer2_uuid = (SELECT uuid FROM archers WHERE email = 'andi.wijaya@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer2_uuid,
    '0748da0d-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 3
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'doni_pratama',
    'doni-pratama',
    'doni.pratama@example.com',
    'DON9012',
    'Doni Pratama',
    'Doni',
    '1990-11-08',
    'male',
    'IDN',
    '081234567003',
    'Jl. Gatot Subroto No. 12, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12930',
    'NID320123456791',
    'recurve',
    'right',
    12,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer3_uuid = (SELECT uuid FROM archers WHERE email = 'doni.pratama@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer3_uuid,
    '0748da0d-f832-11f0-87db-c3c8a1ce2650',
    'belum_lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 4
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'erik_kurniawan',
    'erik-kurniawan',
    'erik.kurniawan@example.com',
    'ERI3456',
    'Erik Kurniawan',
    'Erik',
    '1993-05-18',
    'male',
    'IDN',
    '081234567004',
    'Jl. Thamrin No. 56, Jakarta Pusat',
    'Jakarta',
    'DKI Jakarta',
    '10250',
    'NID320123456792',
    'recurve',
    'right',
    7,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer4_uuid = (SELECT uuid FROM archers WHERE email = 'erik.kurniawan@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer4_uuid,
    '0748da0d-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 5
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'fajar_nugroho',
    'fajar-nugroho',
    'fajar.nugroho2@example.com',
    'FAJ7890',
    'Fajar Nugroho',
    'Fajar',
    '1994-09-25',
    'male',
    'IDN',
    '081234567005',
    'Jl. Kuningan No. 34, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12940',
    'NID320123456793',
    'recurve',
    'left',
    9,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer5_uuid = (SELECT uuid FROM archers WHERE email = 'fajar.nugroho2@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer5_uuid,
    '0748da0d-f832-11f0-87db-c3c8a1ce2650',
    'menunggu_acc',
    'pending',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 6
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'guntur_saputra',
    'guntur-saputra',
    'guntur.saputra@example.com',
    'GUN1235',
    'Guntur Saputra',
    'Guntur',
    '1991-12-30',
    'male',
    'IDN',
    '081234567006',
    'Jl. Senopati No. 67, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12190',
    'NID320123456794',
    'recurve',
    'right',
    11,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer6_uuid = (SELECT uuid FROM archers WHERE email = 'guntur.saputra@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer6_uuid,
    '0748da0d-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 7
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'hendra_maulana',
    'hendra-maulana',
    'hendra.maulana@example.com',
    'HEN5679',
    'Hendra Maulana',
    'Hendra',
    '1996-02-14',
    'male',
    'IDN',
    '081234567007',
    'Jl. Kebayoran Baru No. 89, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12160',
    'NID320123456795',
    'recurve',
    'right',
    6,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer7_uuid = (SELECT uuid FROM archers WHERE email = 'hendra.maulana@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer7_uuid,
    '0748da0d-f832-11f0-87db-c3c8a1ce2650',
    'belum_lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- ============================================
-- CATEGORY 2: Recurve Senior Women (7 archers)
-- ============================================

-- Archer 8
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'sari_dewi',
    'sari-dewi',
    'sari.dewi2@example.com',
    'SAR9013',
    'Sari Dewi Lestari',
    'Sari',
    '1995-04-20',
    'female',
    'IDN',
    '081234567008',
    'Jl. Menteng No. 23, Jakarta Pusat',
    'Jakarta',
    'DKI Jakarta',
    '10310',
    'NID320123456796',
    'recurve',
    'right',
    8,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer8_uuid = (SELECT uuid FROM archers WHERE email = 'sari.dewi2@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer8_uuid,
    '0748dc7a-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 9
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'putri_ayu',
    'putri-ayu',
    'putri.ayu2@example.com',
    'PUT3457',
    'Putri Ayu Sari',
    'Putri',
    '1993-08-12',
    'female',
    'IDN',
    '081234567009',
    'Jl. Kemang No. 45, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12730',
    'NID320123456797',
    'recurve',
    'right',
    10,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer9_uuid = (SELECT uuid FROM archers WHERE email = 'putri.ayu2@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer9_uuid,
    '0748dc7a-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 10
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'rina_wulandari',
    'rina-wulandari',
    'rina.wulandari2@example.com',
    'RIN7891',
    'Rina Wulandari',
    'Rina',
    '1992-06-28',
    'female',
    'IDN',
    '081234567010',
    'Jl. Cikini No. 56, Jakarta Pusat',
    'Jakarta',
    'DKI Jakarta',
    '10330',
    'NID320123456798',
    'recurve',
    'right',
    12,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer10_uuid = (SELECT uuid FROM archers WHERE email = 'rina.wulandari2@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer10_uuid,
    '0748dc7a-f832-11f0-87db-c3c8a1ce2650',
    'belum_lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 11
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'lina_marlina',
    'lina-marlina2',
    'lina.marlina2@example.com',
    'LIN1236',
    'Lina Marlina',
    'Lina',
    '1994-10-05',
    'female',
    'IDN',
    '081234567011',
    'Jl. Blok M No. 78, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12160',
    'NID320123456799',
    'recurve',
    'right',
    7,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer11_uuid = (SELECT uuid FROM archers WHERE email = 'lina.marlina2@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer11_uuid,
    '0748dc7a-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 12
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'maya_sari',
    'maya-sari',
    'maya.sari@example.com',
    'MAY5670',
    'Maya Sari Dewi',
    'Maya',
    '1996-01-15',
    'female',
    'IDN',
    '081234567012',
    'Jl. Pondok Indah No. 12, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12310',
    'NID320123456800',
    'recurve',
    'right',
    9,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer12_uuid = (SELECT uuid FROM archers WHERE email = 'maya.sari@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer12_uuid,
    '0748dc7a-f832-11f0-87db-c3c8a1ce2650',
    'menunggu_acc',
    'pending',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 13
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'nina_kartika',
    'nina-kartika',
    'nina.kartika@example.com',
    'NIN9014',
    'Nina Kartika',
    'Nina',
    '1991-03-22',
    'female',
    'IDN',
    '081234567013',
    'Jl. Kebon Jeruk No. 34, Jakarta Barat',
    'Jakarta',
    'DKI Jakarta',
    '11530',
    'NID320123456801',
    'recurve',
    'left',
    11,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer13_uuid = (SELECT uuid FROM archers WHERE email = 'nina.kartika@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer13_uuid,
    '0748dc7a-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 14
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'sinta_ratna',
    'sinta-ratna',
    'sinta.ratna@example.com',
    'SIN3458',
    'Sinta Ratna Dewi',
    'Sinta',
    '1995-07-18',
    'female',
    'IDN',
    '081234567014',
    'Jl. Tanah Abang No. 67, Jakarta Pusat',
    'Jakarta',
    'DKI Jakarta',
    '10160',
    'NID320123456802',
    'recurve',
    'right',
    6,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer14_uuid = (SELECT uuid FROM archers WHERE email = 'sinta.ratna@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer14_uuid,
    '0748dc7a-f832-11f0-87db-c3c8a1ce2650',
    'belum_lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- ============================================
-- CATEGORY 3: Barebow Senior Men (6 archers)
-- ============================================

-- Archer 15
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'agus_supriyadi',
    'agus-supriyadi',
    'agus.supriyadi@example.com',
    'AGU7892',
    'Agus Supriyadi',
    'Agus',
    '1993-09-10',
    'male',
    'IDN',
    '081234567015',
    'Jl. Cempaka Putih No. 89, Jakarta Pusat',
    'Jakarta',
    'DKI Jakarta',
    '10510',
    'NID320123456803',
    'barebow',
    'right',
    8,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer15_uuid = (SELECT uuid FROM archers WHERE email = 'agus.supriyadi@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer15_uuid,
    '0748ddc4-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 16
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'bambang_sutrisno',
    'bambang-sutrisno',
    'bambang.sutrisno@example.com',
    'BAM1237',
    'Bambang Sutrisno',
    'Bambang',
    '1990-12-25',
    'male',
    'IDN',
    '081234567016',
    'Jl. Pasar Minggu No. 45, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12520',
    'NID320123456804',
    'barebow',
    'right',
    10,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer16_uuid = (SELECT uuid FROM archers WHERE email = 'bambang.sutrisno@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer16_uuid,
    '0748ddc4-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 17
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'cahyo_wibowo',
    'cahyo-wibowo',
    'cahyo.wibowo@example.com',
    'CAH5671',
    'Cahyo Wibowo',
    'Cahyo',
    '1992-04-30',
    'male',
    'IDN',
    '081234567017',
    'Jl. Tebet No. 56, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12810',
    'NID320123456805',
    'barebow',
    'right',
    9,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer17_uuid = (SELECT uuid FROM archers WHERE email = 'cahyo.wibowo@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer17_uuid,
    '0748ddc4-f832-11f0-87db-c3c8a1ce2650',
    'belum_lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 18
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'dani_hermawan',
    'dani-hermawan',
    'dani.hermawan@example.com',
    'DAN9015',
    'Dani Hermawan',
    'Dani',
    '1994-06-08',
    'male',
    'IDN',
    '081234567018',
    'Jl. Cilandak No. 78, Jakarta Selatan',
    'Jakarta',
    'DKI Jakarta',
    '12430',
    'NID320123456806',
    'barebow',
    'left',
    7,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer17_uuid = (SELECT uuid FROM archers WHERE email = 'dani.hermawan@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer17_uuid,
    '0748ddc4-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 19
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'edi_santoso',
    'edi-santoso',
    'edi.santoso@example.com',
    'EDI3459',
    'Edi Santoso',
    'Edi',
    '1991-11-20',
    'male',
    'IDN',
    '081234567019',
    'Jl. Rawamangun No. 12, Jakarta Timur',
    'Jakarta',
    'DKI Jakarta',
    '13220',
    'NID320123456807',
    'barebow',
    'right',
    11,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer19_uuid = (SELECT uuid FROM archers WHERE email = 'edi.santoso@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer19_uuid,
    '0748ddc4-f832-11f0-87db-c3c8a1ce2650',
    'menunggu_acc',
    'pending',
    500000,
    NOW(),
    NOW(),
    NOW()
);

-- Archer 20
INSERT INTO archers (uuid, username, slug, email, athlete_code, full_name, nickname, date_of_birth, gender, country, phone, address, city, province, postal_code, national_id, bow_type, dominant_hand, experience_years, club_id, status, role, password, created_at, updated_at)
VALUES (
    UUID(),
    'firman_ramadhan',
    'firman-ramadhan',
    'firman.ramadhan@example.com',
    'FIR7893',
    'Firman Ramadhan',
    'Firman',
    '1995-05-14',
    'male',
    'IDN',
    '081234567020',
    'Jl. Duren Sawit No. 34, Jakarta Timur',
    'Jakarta',
    'DKI Jakarta',
    '13440',
    'NID320123456808',
    'barebow',
    'right',
    6,
    '218e8243-6e03-41b3-a551-65936bd12815',
    'active',
    'archer',
    '$2a$10$example_hashed_password',
    NOW(),
    NOW()
);

SET @archer20_uuid = (SELECT uuid FROM archers WHERE email = 'firman.ramadhan@example.com' LIMIT 1);

INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, payment_amount, registration_date, created_at, updated_at)
VALUES (
    UUID(),
    'f9272ae0-f76f-11f0-87db-c3c8a1ce2650',
    @archer20_uuid,
    '0748ddc4-f832-11f0-87db-c3c8a1ce2650',
    'lunas',
    'approved',
    500000,
    NOW(),
    NOW(),
    NOW()
);
