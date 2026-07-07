-- ============================================================
-- SEED RESULTS DATA
-- Seed qualification scores, elimination matches for existing events
-- ============================================================

-- 1. QUALIFICATION END SCORES FOR LATBER (Session: qs-0001, Event: 9c0d1e2f...)
-- Add scores for participants already registered in the event
-- ============================================================

-- Delete old mock scores first
DELETE FROM qualification_arrow_scores WHERE end_score_uuid IN (SELECT uuid FROM qualification_end_scores WHERE session_uuid = 'qs-0001');
DELETE FROM qualification_end_scores WHERE session_uuid = 'qs-0001';

-- Insert end scores for stewie4king (f748e84a-02d6-4bb5-9a89-059e85bd5c76)
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-stewie-01', 'qs-0001', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 1, 28, 1, 2, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-stewie-02', 'qs-0001', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 2, 27, 0, 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-stewie-03', 'qs-0001', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 3, 29, 1, 3, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-stewie-04', 'qs-0001', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 4, 26, 0, 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-stewie-05', 'qs-0001', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 5, 28, 1, 2, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-stewie-06', 'qs-0001', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 6, 27, 0, 2, DATE_SUB(NOW(), INTERVAL 30 DAY));

-- Insert arrow scores for stewie4king end 1
INSERT INTO qualification_arrow_scores (uuid, end_score_uuid, arrow_number, score, is_x, created_at) VALUES
  ('qas-stewie-01', 'qes-stewie-01', 1, 10, 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qas-stewie-02', 'qes-stewie-01', 2, 9, 0, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qas-stewie-03', 'qes-stewie-01', 3, 9, 0, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qas-stewie-04', 'qes-stewie-01', 4, 10, 0, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qas-stewie-05', 'qes-stewie-01', 5, 8, 0, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qas-stewie-06', 'qes-stewie-01', 6, 10, 0, DATE_SUB(NOW(), INTERVAL 30 DAY));

-- Insert end scores for Fahmi Ahza (44f987e3-de70-41b7-a238-362c93b36b96)
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-fahmi-01', 'qs-0001', '44f987e3-de70-41b7-a238-362c93b36b96', 1, 30, 2, 4, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fahmi-02', 'qs-0001', '44f987e3-de70-41b7-a238-362c93b36b96', 2, 28, 1, 2, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fahmi-03', 'qs-0001', '44f987e3-de70-41b7-a238-362c93b36b96', 3, 29, 1, 3, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fahmi-04', 'qs-0001', '44f987e3-de70-41b7-a238-362c93b36b96', 4, 27, 0, 2, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fahmi-05', 'qs-0001', '44f987e3-de70-41b7-a238-362c93b36b96', 5, 28, 1, 2, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fahmi-06', 'qs-0001', '44f987e3-de70-41b7-a238-362c93b36b96', 6, 30, 2, 4, DATE_SUB(NOW(), INTERVAL 30 DAY));

-- Insert end scores for Fadil Ata (b14417f7-3764-4e58-a198-122801939c21)
INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end, created_at)
VALUES
  ('qes-fadil-01', 'qs-0001', 'b14417f7-3764-4e58-a198-122801939c21', 1, 25, 0, 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fadil-02', 'qs-0001', 'b14417f7-3764-4e58-a198-122801939c21', 2, 26, 0, 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fadil-03', 'qs-0001', 'b14417f7-3764-4e58-a198-122801939c21', 3, 24, 0, 0, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fadil-04', 'qs-0001', 'b14417f7-3764-4e58-a198-122801939c21', 4, 27, 0, 2, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fadil-05', 'qs-0001', 'b14417f7-3764-4e58-a198-122801939c21', 5, 25, 0, 1, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('qes-fadil-06', 'qs-0001', 'b14417f7-3764-4e58-a198-122801939c21', 6, 26, 1, 1, DATE_SUB(NOW(), INTERVAL 30 DAY));

-- 2. ELIMINATION MATCHES FOR LATBER (Bracket: elb-0001)
-- ============================================================

-- We need elimination entries first
INSERT IGNORE INTO elimination_entries (uuid, bracket_uuid, participant_type, participant_uuid, seed, qual_total_score, created_at)
VALUES
  ('ele-0001', 'elb-0001', 'archer', 'f748e84a-02d6-4bb5-9a89-059e85bd5c76', 1, 165, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('ele-0002', 'elb-0001', 'archer', '44f987e3-de70-41b7-a238-362c93b36b96', 2, 172, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('ele-0003', 'elb-0001', 'archer', 'b14417f7-3764-4e58-a198-122801939c21', 3, 153, DATE_SUB(NOW(), INTERVAL 30 DAY)),
  ('ele-0004', 'elb-0001', 'archer', '59eb3258-bc50-45ac-9ddf-4cb6e90174e6', 4, 150, DATE_SUB(NOW(), INTERVAL 30 DAY));

-- Quarterfinal matches
INSERT IGNORE INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, winner_entry_uuid, status, total_score_a, total_score_b, created_at)
VALUES
  ('elm-qf-01', 'QF-01', 'elb-0001', 1, 1, 'ele-0001', 'ele-0004', 'ele-0001', 'finished', 4, 2, DATE_SUB(NOW(), INTERVAL 29 DAY)),
  ('elm-qf-02', 'QF-02', 'elb-0001', 1, 2, 'ele-0002', 'ele-0003', 'ele-0002', 'finished', 4, 1, DATE_SUB(NOW(), INTERVAL 29 DAY));

-- Semifinal match
INSERT IGNORE INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, winner_entry_uuid, status, total_score_a, total_score_b, created_at)
VALUES
  ('elm-sf-01', 'SF-01', 'elb-0001', 2, 1, 'ele-0001', 'ele-0002', 'ele-0002', 'finished', 3, 4, DATE_SUB(NOW(), INTERVAL 28 DAY));

-- Final match
INSERT IGNORE INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, winner_entry_uuid, status, total_score_a, total_score_b, created_at)
VALUES
  ('elm-final-01', 'F-01', 'elb-0001', 3, 1, 'ele-0002', NULL, 'ele-0002', 'finished', 4, 0, DATE_SUB(NOW(), INTERVAL 27 DAY));

-- 3. UPDATE stewie4king's participant in POPDA event with real score
-- ============================================================
UPDATE event_participants SET qual_score = 285, qual_rank = 3 WHERE uuid = 'f748e84a-02d6-4bb5-9a89-059e85bd5c76';

-- 4. ADD notifications about results
-- ============================================================
INSERT IGNORE INTO notifications (user_id, user_role, type, title, message, is_read, created_at) VALUES
  ('11e0974c-a7f6-4b76-811f-5291137f164e', 'archer', 'success', 'Hasil Latihan Bersama Sleman Archery', 'Kualifikasi: 165 poin. Peringkat 1. Lanjut ke semifinal!', 0, DATE_SUB(NOW(), INTERVAL 28 DAY)),
  ('11e0974c-a7f6-4b76-811f-5291137f164e', 'archer', 'info', 'Hasil Eliminasi Latihan Bersama', 'Selamat! Anda juara 1 di kategori Umum Putra.', 0, DATE_SUB(NOW(), INTERVAL 27 DAY));

SELECT 'SEED RESULTS COMPLETE' AS status;
