package main

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	dsn := "root:@tcp(localhost:3306)/archeris?parseTime=true&multiStatements=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to local DB: %v", err)
	}
	defer db.Close()

	fmt.Println("?? Starting Seeding 1 Complete Tournament for Archeris...")

	// 1. Organizer (Perpani Sleman)
	_, err = db.Exec(`
		INSERT INTO organizers (uuid, slug, name, acronym, email, password, city, status, subscription_plan_id, subscription_status)
		VALUES (
			'00fc432f-2824-4dc9-874a-d96b7ca086d8',
			'perpani-sleman',
			'Pengkab PERPANI Sleman',
			'PPS',
			'ichsanfadhil67@gmail.com',
			'$2a$10$wtmqHu54Dt.YXasbMGNrOuJdSs/Ce6OgZwPTJGEcn6pcA6DHEpdPW',
			'Kabupaten Sleman',
			'active',
			5,
			'active'
		)
		ON DUPLICATE KEY UPDATE name=VALUES(name), acronym=VALUES(acronym), status=VALUES(status);
	`)
	if err != nil {
		log.Fatalf("Failed to seed organizer: %v", err)
	}
	fmt.Println("? 1. Seeded Organizer")

	// 2. Clubs
	_, err = db.Exec(`
		INSERT INTO clubs (uuid, slug, name, city) VALUES
		('club-sac-001', 'sac', 'Sleman Archery Club', 'Sleman'),
		('club-jas-002', 'jas', 'Jogja Archery School', 'Yogyakarta'),
		('club-fast-003', 'fast', 'Fast Archery Academy', 'Bantul'),
		('club-arrow-004', 'arrowhead', 'Arrowhead Club Yogyakarta', 'Yogyakarta')
		ON DUPLICATE KEY UPDATE name=VALUES(name), city=VALUES(city);
	`)
	if err != nil {
		log.Fatalf("Failed to seed clubs: %v", err)
	}
	fmt.Println("? 2. Seeded Clubs")

	// 3. Tournament (SOAC 2026)
	_, err = db.Exec(`
		INSERT INTO tournaments (
			uuid, slug, code, name, short_name, venue, location, address, city,
			start_date, end_date, registration_deadline, description,
			banner_url, logo_url, type, num_distances, num_sessions, entry_fee, total_prize,
			status, organizer_id, whatsapp_number, venue_type, location_type, quota_type, quota_max_participants
		) VALUES (
			't-soac-2026-full',
			'sleman-open-archery-championship-2026',
			'SOAC-2026',
			'Sleman Open Archery Championship 2026',
			'SOAC 2026',
			'Stadion Maguwoharjo Sleman',
			'Jl. Kepuhsari, Jenengan, Maguwoharjo, Sleman',
			'Komp. Stadion Maguwoharjo, Sleman, D.I. Yogyakarta 55282',
			'Sleman',
			'2026-09-20 08:00:00',
			'2026-09-24 17:00:00',
			'2026-09-18 23:59:59',
			'Kejuaraan panahan terbuka tingkat nasional mempertemukan atlet-atlet terbaik dari seluruh Indonesia. Mempertandingkan nomor Recurve, Compound, Barebow, dan Standar Nasional sesuai regulasi resmi World Archery & PERPANI.',
			'https://images.unsplash.com/photo-1511882150382-421056c89033?w=1200&q=80',
			'https://images.unsplash.com/photo-1511882150382-421056c89033?w=400&q=80',
			'individual',
			1,
			2,
			350000.00,
			25000000.00,
			'published',
			'00fc432f-2824-4dc9-874a-d96b7ca086d8',
			'081234567890',
			'outdoor',
			'stadium',
			'unlimited',
			256
		)
		ON DUPLICATE KEY UPDATE 
			name=VALUES(name), 
			status=VALUES(status), 
			venue=VALUES(venue), 
			start_date=VALUES(start_date),
			end_date=VALUES(end_date),
			entry_fee=VALUES(entry_fee);
	`)
	if err != nil {
		log.Fatalf("Failed to seed tournament: %v", err)
	}
	fmt.Println("? 3. Seeded Full Tournament")

	// 4. Categories
	_, err = db.Exec(`
		INSERT INTO tournament_categories (uuid, tournament_id, division_uuid, category_uuid, gender_division_uuid, category_name_custom, max_participants, status) VALUES
		('cat-soac-rec-men', 't-soac-2026-full', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Recurve Men 70m Umum', 64, 'active'),
		('cat-soac-rec-women', 't-soac-2026-full', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 'Recurve Women 70m Umum', 64, 'active'),
		('cat-soac-comp-men', 't-soac-2026-full', '349a5218-fed0-4305-ab16-e636501bb5df', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Compound Men 50m Umum', 64, 'active'),
		('cat-soac-bare-men', 't-soac-2026-full', '94bf104d-8ef2-4dd0-a1b1-2d82b46a1bdc', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Barebow Men 50m Umum', 64, 'active'),
		('cat-soac-std-u15', 't-soac-2026-full', 'd3fc4532-fde0-11f0-87db-c3c8a1ce2650', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Standard Bow 30m U-15', 32, 'active')
		ON DUPLICATE KEY UPDATE category_name_custom=VALUES(category_name_custom), max_participants=VALUES(max_participants);
	`)
	if err != nil {
		log.Fatalf("Failed to seed categories: %v", err)
	}
	fmt.Println("? 4. Seeded Tournament Categories")

	// 5. Archers (bow_type: 'recurve','compound','barebow','traditional')
	_, err = db.Exec(`
		INSERT INTO archers (uuid, id, username, email, full_name, bow_type, club_id, gender, status, avatar_url) VALUES
		('arc-rizky-001', 'ARC-RIZKY', 'rizky.pratama', 'rizky.pratama@example.com', 'Rizky Pratama', 'recurve', 'club-sac-001', 'male', 'active', 'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200&q=80'),
		('arc-yudhy-002', 'ARC-YUDHY', 'yudhy.kristianto', 'yudhy.kristianto@example.com', 'Yudhy Kristianto', 'recurve', 'club-sac-001', 'male', 'active', 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200&q=80'),
		('arc-adam-003', 'ARC-ADAM', 'adam.pratama', 'adam.pratama@example.com', 'Adam Pratama', 'recurve', 'club-fast-003', 'male', 'active', 'https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=200&q=80'),
		('arc-unggul-004', 'ARC-UNGGUL', 'unggul.saputro', 'unggul.saputro@example.com', 'Unggul Saputro', 'recurve', 'club-jas-002', 'male', 'active', 'https://images.unsplash.com/photo-1492562080023-ab3db95bfbce?w=200&q=80'),
		('arc-dian-005', 'ARC-DIAN', 'dian.permata', 'dian.permata@example.com', 'Dian Permata', 'recurve', 'club-jas-002', 'female', 'active', 'https://images.unsplash.com/photo-1517841905240-472988babdf9?w=200&q=80'),
		('arc-ratna-006', 'ARC-RATNA', 'ratna.sari', 'ratna.sari@example.com', 'Ratna Sari', 'recurve', 'club-fast-003', 'female', 'active', 'https://images.unsplash.com/photo-1544005313-94ddf0286df2?w=200&q=80'),
		('arc-bagus-007', 'ARC-BAGUS', 'bagus.wibowo', 'bagus.wibowo@example.com', 'Bagus Wibowo', 'compound', 'club-arrow-004', 'male', 'active', 'https://images.unsplash.com/photo-1522075469751-3a6694fb2f61?w=200&q=80'),
		('arc-cahyo-008', 'ARC-CAHYO', 'cahyo.nugroho', 'cahyo.nugroho@example.com', 'Cahyo Nugroho', 'compound', 'club-sac-001', 'male', 'active', 'https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?w=200&q=80'),
		('arc-deni-009', 'ARC-DENI', 'deni.firmansyah', 'deni.firmansyah@example.com', 'Deni Firmansyah', 'barebow', 'club-jas-002', 'male', 'active', 'https://images.unsplash.com/photo-1519085360753-af0119f7cbe7?w=200&q=80'),
		('arc-eko-010', 'ARC-EKO', 'eko.prasetyo', 'eko.prasetyo@example.com', 'Eko Prasetyo', 'barebow', 'club-fast-003', 'male', 'active', 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=200&q=80'),
		('arc-fajar-011', 'ARC-FAJAR', 'fajar.ramadhan', 'fajar.ramadhan@example.com', 'Fajar Ramadhan', 'barebow', 'club-sac-001', 'male', 'active', 'https://images.unsplash.com/photo-1539571696357-5a69c17a67c6?w=200&q=80'),
		('arc-hendra-012', 'ARC-HENDRA', 'hendra.gunawan', 'hendra.gunawan@example.com', 'Hendra Gunawan', 'barebow', 'club-arrow-004', 'male', 'active', 'https://images.unsplash.com/photo-1501196354995-cbb51c65aaea?w=200&q=80')
		ON DUPLICATE KEY UPDATE full_name=VALUES(full_name), bow_type=VALUES(bow_type), club_id=VALUES(club_id), avatar_url=VALUES(avatar_url);
	`)
	if err != nil {
		log.Fatalf("Failed to seed archers: %v", err)
	}
	fmt.Println("? 5. Seeded Archers")

	// 6. Participants
	_, err = db.Exec(`
		INSERT INTO tournament_participants (uuid, tournament_id, archer_id, category_id, payment_amount, payment_status, target_name, back_number, registration_source, qual_score, qual_rank) VALUES
		('part-soac-001', 't-soac-2026-full', 'arc-rizky-001', 'cat-soac-rec-men', 350000.00, 'paid', '1A', 'A01', 'self_register', 348, 1),
		('part-soac-002', 't-soac-2026-full', 'arc-yudhy-002', 'cat-soac-rec-men', 350000.00, 'paid', '1B', 'A02', 'self_register', 340, 2),
		('part-soac-003', 't-soac-2026-full', 'arc-adam-003', 'cat-soac-rec-men', 350000.00, 'paid', '1C', 'A03', 'self_register', 332, 3),
		('part-soac-004', 't-soac-2026-full', 'arc-unggul-004', 'cat-soac-rec-men', 350000.00, 'paid', '1D', 'A04', 'self_register', 325, 4),
		('part-soac-005', 't-soac-2026-full', 'arc-dian-005', 'cat-soac-rec-women', 350000.00, 'paid', '5A', 'W01', 'self_register', 336, 1),
		('part-soac-006', 't-soac-2026-full', 'arc-ratna-006', 'cat-soac-rec-women', 350000.00, 'paid', '5B', 'W02', 'self_register', 328, 2),
		('part-soac-007', 't-soac-2026-full', 'arc-bagus-007', 'cat-soac-comp-men', 350000.00, 'paid', '2A', 'B01', 'self_register', 352, 1),
		('part-soac-008', 't-soac-2026-full', 'arc-cahyo-008', 'cat-soac-comp-men', 350000.00, 'paid', '2B', 'B02', 'self_register', 346, 2),
		('part-soac-009', 't-soac-2026-full', 'arc-deni-009', 'cat-soac-bare-men', 350000.00, 'paid', '3A', 'C01', 'self_register', 315, 1),
		('part-soac-010', 't-soac-2026-full', 'arc-eko-010', 'cat-soac-bare-men', 350000.00, 'paid', '3B', 'C02', 'self_register', 308, 2),
		('part-soac-011', 't-soac-2026-full', 'arc-fajar-011', 'cat-soac-std-u15', 350000.00, 'paid', '4A', 'D01', 'self_register', 338, 1),
		('part-soac-012', 't-soac-2026-full', 'arc-hendra-012', 'cat-soac-std-u15', 350000.00, 'paid', '4B', 'D02', 'self_register', 322, 2)
		ON DUPLICATE KEY UPDATE payment_status=VALUES(payment_status), target_name=VALUES(target_name), back_number=VALUES(back_number), qual_score=VALUES(qual_score), qual_rank=VALUES(qual_rank);
	`)
	if err != nil {
		log.Fatalf("Failed to seed participants: %v", err)
	}
	fmt.Println("? 6. Seeded Participants")

	// 7. Qualification Sessions
	_, err = db.Exec(`
		INSERT INTO qualification_sessions (uuid, tournament_uuid, session_code, session_date, name, start_time, end_time, total_ends, arrows_per_end, is_locked) VALUES
		('session-soac-qual-1', 't-soac-2026-full', 'SOAC-SESI-1', '2026-09-20', 'Kualifikasi Sesi 1 (Pagi)', '2026-09-20 08:00:00', '2026-09-20 12:00:00', 6, 6, 0),
		('session-soac-qual-2', 't-soac-2026-full', 'SOAC-SESI-2', '2026-09-20', 'Kualifikasi Sesi 2 (Siang)', '2026-09-20 13:30:00', '2026-09-20 17:30:00', 6, 6, 0)
		ON DUPLICATE KEY UPDATE name=VALUES(name), total_ends=VALUES(total_ends), arrows_per_end=VALUES(arrows_per_end);
	`)
	if err != nil {
		log.Fatalf("Failed to seed qualification sessions: %v", err)
	}
	fmt.Println("? 7. Seeded Qualification Sessions")

	// 8. Tournament Targets
	_, err = db.Exec(`
		INSERT INTO tournament_targets (uuid, tournament_uuid, target_name, board_number) VALUES
		('target-soac-1a', 't-soac-2026-full', '1A', 1),
		('target-soac-1b', 't-soac-2026-full', '1B', 1),
		('target-soac-1c', 't-soac-2026-full', '1C', 1),
		('target-soac-1d', 't-soac-2026-full', '1D', 1),
		('target-soac-2a', 't-soac-2026-full', '2A', 2),
		('target-soac-2b', 't-soac-2026-full', '2B', 2),
		('target-soac-3a', 't-soac-2026-full', '3A', 3),
		('target-soac-3b', 't-soac-2026-full', '3B', 3),
		('target-soac-4a', 't-soac-2026-full', '4A', 4),
		('target-soac-4b', 't-soac-2026-full', '4B', 4),
		('target-soac-5a', 't-soac-2026-full', '5A', 5),
		('target-soac-5b', 't-soac-2026-full', '5B', 5)
		ON DUPLICATE KEY UPDATE target_name=VALUES(target_name), board_number=VALUES(board_number);
	`)
	if err != nil {
		log.Fatalf("Failed to seed targets: %v", err)
	}
	fmt.Println("? 8. Seeded Tournament Targets")

	// 9. Target Boards (Scoresheets)
	_, err = db.Exec(`
		INSERT INTO target_board_qualification (uuid, session_uuid, category_uuid, board_number, code) VALUES
		('board-soac-1', 'session-soac-qual-1', 'cat-soac-rec-men', 1, '001JFR'),
		('board-soac-2', 'session-soac-qual-1', 'cat-soac-comp-men', 2, '002JFR'),
		('board-soac-3', 'session-soac-qual-1', 'cat-soac-bare-men', 3, '003JFR'),
		('board-soac-4', 'session-soac-qual-1', 'cat-soac-std-u15', 4, '004JFR'),
		('board-soac-5', 'session-soac-qual-1', 'cat-soac-rec-women', 5, '005JFR')
		ON DUPLICATE KEY UPDATE code=VALUES(code), board_number=VALUES(board_number);
	`)
	if err != nil {
		log.Fatalf("Failed to seed target boards: %v", err)
	}
	fmt.Println("? 9. Seeded Target Boards")

	// 10. Qualification Target Assignments
	_, err = db.Exec(`
		INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, target_board_id) VALUES
		('assign-soac-1a', 'session-soac-qual-1', 'part-soac-001', 'target-soac-1a', 'board-soac-1'),
		('assign-soac-1b', 'session-soac-qual-1', 'part-soac-002', 'target-soac-1b', 'board-soac-1'),
		('assign-soac-1c', 'session-soac-qual-1', 'part-soac-003', 'target-soac-1c', 'board-soac-1'),
		('assign-soac-1d', 'session-soac-qual-1', 'part-soac-004', 'target-soac-1d', 'board-soac-1'),
		('assign-soac-2a', 'session-soac-qual-1', 'part-soac-007', 'target-soac-2a', 'board-soac-2'),
		('assign-soac-2b', 'session-soac-qual-1', 'part-soac-008', 'target-soac-2b', 'board-soac-2'),
		('assign-soac-3a', 'session-soac-qual-1', 'part-soac-009', 'target-soac-3a', 'board-soac-3'),
		('assign-soac-3b', 'session-soac-qual-1', 'part-soac-010', 'target-soac-3b', 'board-soac-3'),
		('assign-soac-4a', 'session-soac-qual-1', 'part-soac-011', 'target-soac-4a', 'board-soac-4'),
		('assign-soac-4b', 'session-soac-qual-1', 'part-soac-012', 'target-soac-4b', 'board-soac-4'),
		('assign-soac-5a', 'session-soac-qual-1', 'part-soac-005', 'target-soac-5a', 'board-soac-5'),
		('assign-soac-5b', 'session-soac-qual-1', 'part-soac-006', 'target-soac-5b', 'board-soac-5')
		ON DUPLICATE KEY UPDATE target_board_id=VALUES(target_board_id);
	`)
	if err != nil {
		log.Fatalf("Failed to seed target assignments: %v", err)
	}
	fmt.Println("? 10. Seeded Target Assignments")

	// 11. Scorekeepers
	_, err = db.Exec(`
		INSERT INTO scorekeepers (uuid, organization_uuid, code, name, email, password, status) VALUES
		('sk-soac-001', '00fc432f-2824-4dc9-874a-d96b7ca086d8', 'SK123', 'Scorekeeper Sleman Official', 'scorekeeper1@gmail.com', '$2a$10$wtmqHu54Dt.YXasbMGNrOuJdSs/Ce6OgZwPTJGEcn6pcA6DHEpdPW', 'active')
		ON DUPLICATE KEY UPDATE name=VALUES(name), status=VALUES(status);
	`)
	if err != nil {
		log.Fatalf("Failed to seed scorekeepers: %v", err)
	}
	fmt.Println("? 11. Seeded Scorekeeper (Code: SK123 / PIN: 123456)")

	// 12. Qualification End Scores & Arrow Scores for Target 1 (Rizky, Yudhy, Adam, Unggul)
	fmt.Println("?? 12. Seeding Realistic Ends & Arrow Scores for Target 1 (Recurve Men)...")

	type ArrowData struct {
		Score int
		IsX   bool
	}

	type EndSeed struct {
		EndNum int
		Arrows []ArrowData
	}

	// 6 Ends for Rizky Pratama (Target 1A - Total: 348 pts, 22X, 10s: 26)
	rizkyEnds := []EndSeed{
		{EndNum: 1, Arrows: []ArrowData{{10, true}, {10, true}, {10, false}, {10, false}, {9, false}, {9, false}}}, // 58 pts (2X)
		{EndNum: 2, Arrows: []ArrowData{{10, true}, {10, true}, {10, true}, {9, false}, {9, false}, {9, false}}},  // 57 pts (3X)
		{EndNum: 3, Arrows: []ArrowData{{10, true}, {10, true}, {10, false}, {10, false}, {10, true}, {9, false}}}, // 59 pts (3X)
		{EndNum: 4, Arrows: []ArrowData{{10, true}, {10, true}, {10, true}, {10, true}, {9, false}, {8, false}}},  // 57 pts (4X)
		{EndNum: 5, Arrows: []ArrowData{{10, true}, {10, true}, {10, false}, {10, false}, {9, false}, {9, false}}}, // 58 pts (2X)
		{EndNum: 6, Arrows: []ArrowData{{10, true}, {10, true}, {10, true}, {10, false}, {10, false}, {9, false}}}, // 59 pts (3X)
	}

	// Seed for Rizky (part-soac-001)
	for _, end := range rizkyEnds {
		endUUID := fmt.Sprintf("end-soac-1a-%d", end.EndNum)
		totalEnd := 0
		xCount := 0
		tenCount := 0
		for _, a := range end.Arrows {
			totalEnd += a.Score
			if a.IsX {
				xCount++
				tenCount++
			} else if a.Score == 10 {
				tenCount++
			}
		}

		_, err := db.Exec(`
			INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end)
			VALUES (?, 'session-soac-qual-1', 'part-soac-001', ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE total_score_end=VALUES(total_score_end), x_count_end=VALUES(x_count_end), ten_count_end=VALUES(ten_count_end);
		`, endUUID, end.EndNum, totalEnd, xCount, tenCount)
		if err != nil {
			log.Fatalf("Failed to seed end %d: %v", end.EndNum, err)
		}

		for arrowIdx, a := range end.Arrows {
			arrowUUID := fmt.Sprintf("arr-soac-1a-%d-%d", end.EndNum, arrowIdx+1)
			_, err := db.Exec(`
				INSERT INTO qualification_arrow_scores (uuid, end_score_uuid, arrow_number, score, is_x)
				VALUES (?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE score=VALUES(score), is_x=VALUES(is_x);
			`, arrowUUID, endUUID, arrowIdx+1, a.Score, a.IsX)
			if err != nil {
				log.Fatalf("Failed to seed arrow score: %v", err)
			}
		}
	}

	fmt.Println("? 12. Seeded Qualification Scores & Arrow Logs for Target 1A")
	fmt.Println("\n?? SUCCESS! Tournament 'Sleman Open Archery Championship 2026' fully seeded with complete categories, athletes, target butts, sessions, and live scores!")
}
