package main

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	dsn := "root:@tcp(localhost:3306)/archeris_dev?parseTime=true&multiStatements=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to local DB: %v", err)
	}
	defer db.Close()

	fmt.Println("🏹 Starting Seeding Scoring & Tournament Data for archeris_dev...")

	// 1. Clubs
	_, err = db.Exec(`
		INSERT INTO clubs (uuid, slug, name, city) VALUES
		('club-jas-001', 'jas', 'Jogja Archery School', 'Yogyakarta'),
		('club-sac-002', 'sac', 'Sleman Archery Club', 'Sleman'),
		('club-fast-003', 'fast', 'Fast Archery Academy', 'Bantul'),
		('club-arrow-004', 'arrowhead', 'Arrowhead Club Yogyakarta', 'Yogyakarta')
		ON DUPLICATE KEY UPDATE name=VALUES(name), city=VALUES(city);
	`)
	if err != nil {
		log.Fatalf("Failed to seed clubs: %v", err)
	}
	fmt.Println("✅ Seeded Clubs")

	// 2. Archers
	_, err = db.Exec(`
		INSERT INTO archers (uuid, id, username, email, full_name, bow_type, club_id, gender, status) VALUES
		('arc-yudhy-001', 'ARC-YUDHY', 'yudhy.kristianto', 'yudhy.kristianto@example.com', 'Yudhy Kristianto', 'recurve', 'club-sac-002', 'male', 'active'),
		('arc-adam-002', 'ARC-ADAM', 'adam.pratama', 'adam.pratama@example.com', 'Adam Pratama', 'recurve', 'club-fast-003', 'male', 'active'),
		('arc-unggul-003', 'ARC-UNGGUL', 'unggul.saputro', 'unggul.saputro@example.com', 'Unggul Saputro', 'recurve', 'club-jas-001', 'male', 'active'),
		('arc-bagus-004', 'ARC-BAGUS', 'bagus.wibowo', 'bagus.wibowo@example.com', 'Bagus Wibowo', 'compound', 'club-arrow-004', 'male', 'active'),
		('arc-cahyo-005', 'ARC-CAHYO', 'cahyo.nugroho', 'cahyo.nugroho@example.com', 'Cahyo Nugroho', 'compound', 'club-sac-002', 'male', 'active'),
		('arc-deni-006', 'ARC-DENI', 'deni.firmansyah', 'deni.firmansyah@example.com', 'Deni Firmansyah', 'barebow', 'club-jas-001', 'male', 'active'),
		('arc-eko-007', 'ARC-EKO', 'eko.prasetyo', 'eko.prasetyo@example.com', 'Eko Prasetyo', 'barebow', 'club-fast-003', 'male', 'active'),
		('arc-fajar-008', 'ARC-FAJAR', 'fajar.ramadhan', 'fajar.ramadhan@example.com', 'Fajar Ramadhan', 'recurve', 'club-sac-002', 'male', 'active')
		ON DUPLICATE KEY UPDATE full_name=VALUES(full_name), bow_type=VALUES(bow_type), club_id=VALUES(club_id);
	`)
	if err != nil {
		log.Fatalf("Failed to seed archers: %v", err)
	}
	fmt.Println("✅ Seeded Archers")

	// 3. Categories
	_, err = db.Exec(`
		INSERT INTO tournament_categories (uuid, tournament_id, division_uuid, category_uuid, gender_division_uuid, category_name_custom, max_participants, status) VALUES
		('cat-soac-recurve-men', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Recurve Men 70m Umum', 64, 'active'),
		('cat-soac-compound-men', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '349a5218-fed0-4305-ab16-e636501bb5df', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Compound Men 50m Umum', 64, 'active'),
		('cat-soac-barebow-men', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '94bf104d-8ef2-4dd0-a1b1-2d82b46a1bdc', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Barebow Men 50m Umum', 64, 'active'),
		('cat-soac-standard-u15', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'd3fc4532-fde0-11f0-87db-c3c8a1ce2650', 'f879b964-0929-45d3-9a6a-7d80f3cf708f', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'Standard Bow 30m U-15', 32, 'active')
		ON DUPLICATE KEY UPDATE division_uuid=VALUES(division_uuid), category_uuid=VALUES(category_uuid), gender_division_uuid=VALUES(gender_division_uuid), category_name_custom=VALUES(category_name_custom);
	`)
	if err != nil {
		log.Fatalf("Failed to seed categories: %v", err)
	}
	fmt.Println("✅ Seeded Tournament Categories")

	// 4. Teams
	_, err = db.Exec(`
		INSERT INTO teams (uuid, tournament_id, category_id, team_name, status) VALUES
		('team-soac-sac-rec', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'cat-soac-recurve-men', 'Sleman Archery Club Recurve Team', 'active'),
		('team-soac-jas-rec', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'cat-soac-recurve-men', 'JAS Recurve Squad', 'active')
		ON DUPLICATE KEY UPDATE team_name=VALUES(team_name);
	`)
	if err != nil {
		log.Fatalf("Failed to seed teams: %v", err)
	}
	fmt.Println("✅ Seeded Teams")

	// 5. Participants
	_, err = db.Exec(`
		INSERT INTO tournament_participants (uuid, tournament_id, archer_id, category_id, payment_amount, payment_status, target_name, back_number, registration_source) VALUES
		('part-soac-001', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-yudhy-001', 'cat-soac-recurve-men', 350000.00, 'paid', '1A', 'A01', 'self_register'),
		('part-soac-002', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-adam-002', 'cat-soac-recurve-men', 350000.00, 'paid', '1B', 'A02', 'self_register'),
		('part-soac-003', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-unggul-003', 'cat-soac-recurve-men', 350000.00, 'paid', '1C', 'A03', 'self_register'),
		('part-soac-004', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-bagus-004', 'cat-soac-compound-men', 350000.00, 'paid', '2A', 'B01', 'self_register'),
		('part-soac-005', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-cahyo-005', 'cat-soac-compound-men', 350000.00, 'paid', '2B', 'B02', 'self_register'),
		('part-soac-006', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-deni-006', 'cat-soac-barebow-men', 350000.00, 'paid', '3A', 'C01', 'self_register'),
		('part-soac-007', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-eko-007', 'cat-soac-barebow-men', 350000.00, 'paid', '3B', 'C02', 'self_register'),
		('part-soac-008', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'arc-fajar-008', 'cat-soac-standard-u15', 350000.00, 'paid', '4A', 'D01', 'self_register')
		ON DUPLICATE KEY UPDATE payment_status=VALUES(payment_status), target_name=VALUES(target_name), back_number=VALUES(back_number);
	`)
	if err != nil {
		log.Fatalf("Failed to seed participants: %v", err)
	}
	fmt.Println("✅ Seeded Participants")

	// 6. Qualification Sessions
	_, err = db.Exec(`
		INSERT INTO qualification_sessions (uuid, tournament_uuid, session_code, session_date, name, start_time, end_time, total_ends, arrows_per_end, is_locked) VALUES
		('session-soac-qual-1', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'SOAC-SESI-1', '2026-09-15', 'Kualifikasi Sesi 1', '2026-09-15 08:00:00', '2026-09-15 12:00:00', 6, 6, 0),
		('session-soac-qual-2', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'SOAC-SESI-2', '2026-09-15', 'Kualifikasi Sesi 2', '2026-09-15 13:30:00', '2026-09-15 17:30:00', 6, 6, 0)
		ON DUPLICATE KEY UPDATE name=VALUES(name), total_ends=VALUES(total_ends), arrows_per_end=VALUES(arrows_per_end);
	`)
	if err != nil {
		log.Fatalf("Failed to seed sessions: %v", err)
	}
	fmt.Println("✅ Seeded Qualification Sessions")

	// 7. Tournament Targets
	_, err = db.Exec(`
		INSERT INTO tournament_targets (uuid, tournament_uuid, target_name, board_number) VALUES
		('target-soac-1a', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '1A', 1),
		('target-soac-1b', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '1B', 1),
		('target-soac-1c', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '1C', 1),
		('target-soac-1d', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '1D', 1),
		('target-soac-2a', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '2A', 2),
		('target-soac-2b', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '2B', 2),
		('target-soac-2c', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '2C', 2),
		('target-soac-2d', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '2D', 2),
		('target-soac-3a', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '3A', 3),
		('target-soac-3b', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '3B', 3),
		('target-soac-4a', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '4A', 4)
		ON DUPLICATE KEY UPDATE target_name=VALUES(target_name), board_number=VALUES(board_number);
	`)
	if err != nil {
		log.Fatalf("Failed to seed targets: %v", err)
	}
	fmt.Println("✅ Seeded Tournament Targets")

	// 8. Target Board Qualification
	_, err = db.Exec(`
		INSERT INTO target_board_qualification (uuid, session_uuid, category_uuid, board_number, code) VALUES
		('board-soac-1', 'session-soac-qual-1', 'cat-soac-recurve-men', 1, '001JFR'),
		('board-soac-2', 'session-soac-qual-1', 'cat-soac-compound-men', 2, '002JFR'),
		('board-soac-3', 'session-soac-qual-1', 'cat-soac-barebow-men', 3, '003JFR'),
		('board-soac-4', 'session-soac-qual-1', 'cat-soac-standard-u15', 4, '004JFR')
		ON DUPLICATE KEY UPDATE code=VALUES(code), board_number=VALUES(board_number);
	`)
	if err != nil {
		log.Fatalf("Failed to seed target boards: %v", err)
	}
	fmt.Println("✅ Seeded Target Boards")

	// 9. Qualification Target Assignments
	_, err = db.Exec(`
		INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, target_board_id) VALUES
		('assign-1-1a', 'session-soac-qual-1', 'part-soac-001', 'target-soac-1a', 'board-soac-1'),
		('assign-1-1b', 'session-soac-qual-1', 'part-soac-002', 'target-soac-1b', 'board-soac-1'),
		('assign-1-1c', 'session-soac-qual-1', 'part-soac-003', 'target-soac-1c', 'board-soac-1'),
		('assign-2-2a', 'session-soac-qual-1', 'part-soac-004', 'target-soac-2a', 'board-soac-2'),
		('assign-2-2b', 'session-soac-qual-1', 'part-soac-005', 'target-soac-2b', 'board-soac-2'),
		('assign-3-3a', 'session-soac-qual-1', 'part-soac-006', 'target-soac-3a', 'board-soac-3'),
		('assign-3-3b', 'session-soac-qual-1', 'part-soac-007', 'target-soac-3b', 'board-soac-3'),
		('assign-4-4a', 'session-soac-qual-1', 'part-soac-008', 'target-soac-4a', 'board-soac-4')
		ON DUPLICATE KEY UPDATE target_board_id=VALUES(target_board_id);
	`)
	if err != nil {
		log.Fatalf("Failed to seed assignments: %v", err)
	}
	fmt.Println("✅ Seeded Target Assignments")

	// 10. Scorekeepers
	_, err = db.Exec(`
		INSERT INTO scorekeepers (uuid, organization_uuid, code, name, email, password, status) VALUES
		('sk-001', '3efcb640-8c52-4b5a-a45e-24e2e08d54df', 'SK123', 'Scorekeeper Sleman 1', 'scorekeeper1@gmail.com', '$2a$10$wtmqHu54Dt.YXasbMGNrOuJdSs/Ce6OgZwPTJGEcn6pcA6DHEpdPW', 'active')
		ON DUPLICATE KEY UPDATE name=VALUES(name), status=VALUES(status);
	`)
	if err != nil {
		log.Fatalf("Failed to seed scorekeeper: %v", err)
	}
	fmt.Println("✅ Seeded Scorekeeper (Code: SK123)")

	fmt.Println("\n🎉 Database Seeding Completed Successfully for Scoring Flow!")
}
