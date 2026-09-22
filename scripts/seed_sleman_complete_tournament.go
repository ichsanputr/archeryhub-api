package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

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

	fmt.Println("🏹 Starting Comprehensive Seeding for Sleman Open Archery Championship 2026...")

	// 1. Organizer
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
		log.Fatalf("Organizer error: %v", err)
	}
	fmt.Println("✅ 1. Organizer Seeded")

	// 2. Clubs
	_, err = db.Exec(`
		INSERT INTO clubs (uuid, slug, name, city) VALUES
		('club-sac-001', 'sac', 'Sleman Archery Club', 'Sleman'),
		('club-jas-002', 'jas', 'Jogja Archery School', 'Yogyakarta'),
		('club-fast-003', 'fast', 'Fast Archery Academy', 'Bantul'),
		('club-arrow-004', 'arrowhead', 'Arrowhead Club Yogyakarta', 'Yogyakarta'),
		('club-mac-005', 'mataram', 'Mataram Archery Club', 'Sleman')
		ON DUPLICATE KEY UPDATE name=VALUES(name), city=VALUES(city);
	`)
	if err != nil {
		log.Fatalf("Clubs error: %v", err)
	}
	fmt.Println("✅ 2. Clubs Seeded")

	// 3. Tournament
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
			'Kejuaraan panahan terbuka tingkat nasional mempertemukan atlet-atlet terbaik dari seluruh Indonesia. Mempertandingkan nomor Recurve, Compound, Barebow, dan Standar Nasional beregu & perorangan sesuai regulasi resmi World Archery & PERPANI.',
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
			venue=VALUES(venue);
	`)
	if err != nil {
		log.Fatalf("Tournament error: %v", err)
	}
	fmt.Println("✅ 3. Tournament Seeded")

	// 4. Categories (Individual + Teams + Mixed Teams)
	_, err = db.Exec(`
		INSERT INTO tournament_categories (uuid, tournament_id, division_uuid, category_uuid, gender_division_uuid, tournament_type_uuid, category_name_custom, max_participants, status) VALUES
		-- Individual Categories
		('cat-soac-rec-men', 't-soac-2026-full', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'Recurve Men 70m Umum', 64, 'active'),
		('cat-soac-rec-women', 't-soac-2026-full', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'Recurve Women 70m Umum', 64, 'active'),
		('cat-soac-comp-men', 't-soac-2026-full', '349a5218-fed0-4305-ab16-e636501bb5df', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'Compound Men 50m Umum', 64, 'active'),
		('cat-soac-comp-women', 't-soac-2026-full', '349a5218-fed0-4305-ab16-e636501bb5df', 'f235b870-724b-44ac-8683-b665df0c0548', 'afbded2f-705c-480f-84d2-6962bcb4b2ef', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'Compound Women 50m Umum', 64, 'active'),
		('cat-soac-bare-men', 't-soac-2026-full', '94bf104d-8ef2-4dd0-a1b1-2d82b46a1bdc', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', 'da2740c8-f7ac-460f-a8c2-46f0c6ec844f', 'Barebow Men 50m Umum', 64, 'active'),

		-- Team Categories
		('cat-soac-rec-team-men', 't-soac-2026-full', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', '3bfbc4ad-afb2-44b2-a686-8d5fd46e5e2f', 'Recurve Men Team 70m', 16, 'active'),
		('cat-soac-rec-mixed', 't-soac-2026-full', '5c95a503-4fd4-465f-9dd3-b08568181792', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', '160f7979-706c-41bb-ba42-111186ad31ab', 'Recurve Mixed Team 70m', 16, 'active'),
		('cat-soac-comp-team-men', 't-soac-2026-full', '349a5218-fed0-4305-ab16-e636501bb5df', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', '3bfbc4ad-afb2-44b2-a686-8d5fd46e5e2f', 'Compound Men Team 50m', 16, 'active'),
		('cat-soac-comp-mixed', 't-soac-2026-full', '349a5218-fed0-4305-ab16-e636501bb5df', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', '160f7979-706c-41bb-ba42-111186ad31ab', 'Compound Mixed Team 50m', 16, 'active'),
		('cat-soac-bare-team-men', 't-soac-2026-full', '94bf104d-8ef2-4dd0-a1b1-2d82b46a1bdc', 'f235b870-724b-44ac-8683-b665df0c0548', 'd60f4939-4d88-4ded-bf4d-6d8cff4de5ae', '3bfbc4ad-afb2-44b2-a686-8d5fd46e5e2f', 'Barebow Men Team 50m', 16, 'active')
		ON DUPLICATE KEY UPDATE 
			category_name_custom=VALUES(category_name_custom),
			tournament_type_uuid=VALUES(tournament_type_uuid),
			status=VALUES(status);
	`)
	if err != nil {
		log.Fatalf("Categories error: %v", err)
	}
	fmt.Println("✅ 4. Categories & Team Categories Seeded")

	// 5. Clean Previous Tournament Participants & Score Records for fresh clean state
	_, _ = db.Exec("DELETE tm FROM team_members tm JOIN teams t ON tm.team_id = t.uuid WHERE t.tournament_id = 't-soac-2026-full'")
	_, _ = db.Exec("DELETE FROM teams WHERE tournament_id = 't-soac-2026-full'")
	_, _ = db.Exec("DELETE m FROM elimination_matches m JOIN elimination_brackets b ON m.bracket_uuid = b.uuid WHERE b.tournament_uuid = 't-soac-2026-full'")
	_, _ = db.Exec("DELETE e FROM elimination_entries e JOIN elimination_brackets b ON e.bracket_uuid = b.uuid WHERE b.tournament_uuid = 't-soac-2026-full'")
	_, _ = db.Exec("DELETE FROM elimination_brackets WHERE tournament_uuid = 't-soac-2026-full'")
	_, _ = db.Exec("DELETE FROM qualification_target_assignments WHERE session_uuid IN ('session-soac-qual-1', 'session-soac-qual-2')")
	_, _ = db.Exec("DELETE FROM target_board_qualification WHERE session_uuid IN ('session-soac-qual-1', 'session-soac-qual-2')")
	_, _ = db.Exec("DELETE FROM qualification_arrow_scores WHERE end_score_uuid IN (SELECT uuid FROM qualification_end_scores WHERE session_uuid IN ('session-soac-qual-1', 'session-soac-qual-2'))")
	_, _ = db.Exec("DELETE FROM qualification_end_scores WHERE session_uuid IN ('session-soac-qual-1', 'session-soac-qual-2')")
	_, _ = db.Exec("DELETE FROM tournament_targets WHERE tournament_uuid = 't-soac-2026-full'")
	_, _ = db.Exec("DELETE FROM tournament_participants WHERE tournament_id = 't-soac-2026-full'")

	// 6. Archers (Insert with unique usernames & emails)
	archersData := []struct {
		UUID      string
		ID        string
		Username  string
		Email     string
		FullName  string
		BowType   string
		ClubID    string
		Gender    string
		AvatarURL string
	}{
		// SAC Recurve Men
		{"arc-soac-rm-1", "ARC-SOAC-01", "rizky.soac", "rizky.soac@example.com", "Rizky Pratama", "recurve", "club-sac-001", "male", "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200&q=80"},
		{"arc-soac-rm-2", "ARC-SOAC-02", "yudhy.soac", "yudhy.soac@example.com", "Yudhy Kristianto", "recurve", "club-sac-001", "male", "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200&q=80"},
		{"arc-soac-rm-3", "ARC-SOAC-03", "dimas.soac", "dimas.soac@example.com", "Dimas Anggoro", "recurve", "club-sac-001", "male", "https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=200&q=80"},
		// SAC Recurve Women
		{"arc-soac-rw-1", "ARC-SOAC-04", "annisa.soac", "annisa.soac@example.com", "Annisa Fitri", "recurve", "club-sac-001", "female", "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200&q=80"},
		{"arc-soac-rw-2", "ARC-SOAC-05", "maya.soac", "maya.soac@example.com", "Maya Indah", "recurve", "club-sac-001", "female", "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=200&q=80"},

		// JAS Recurve Men
		{"arc-soac-rm-4", "ARC-SOAC-06", "unggul.soac", "unggul.soac@example.com", "Unggul Saputro", "recurve", "club-jas-002", "male", "https://images.unsplash.com/photo-1492562080023-ab3db95bfbce?w=200&q=80"},
		{"arc-soac-rm-5", "ARC-SOAC-07", "bagas.soac", "bagas.soac@example.com", "Bagas Wicaksono", "recurve", "club-jas-002", "male", "https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?w=200&q=80"},
		{"arc-soac-rm-6", "ARC-SOAC-08", "gilang.soac", "gilang.soac@example.com", "Gilang Ramadhan", "recurve", "club-jas-002", "male", "https://images.unsplash.com/photo-1519085360753-af0119f7cbe7?w=200&q=80"},
		// JAS Recurve Women
		{"arc-soac-rw-3", "ARC-SOAC-09", "dian.soac", "dian.soac@example.com", "Dian Permata", "recurve", "club-jas-002", "female", "https://images.unsplash.com/photo-1544005313-94ddf0286df2?w=200&q=80"},
		{"arc-soac-rw-4", "ARC-SOAC-10", "larasati.soac", "larasati.soac@example.com", "Larasati Putri", "recurve", "club-jas-002", "female", "https://images.unsplash.com/photo-1524504388940-b1c1722653e1?w=200&q=80"},

		// FAST Recurve Men
		{"arc-soac-rm-7", "ARC-SOAC-11", "adam.soac", "adam.soac@example.com", "Adam Pratama", "recurve", "club-fast-003", "male", "https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=200&q=80"},
		{"arc-soac-rm-8", "ARC-SOAC-12", "rian.soac", "rian.soac@example.com", "Rian Kurniawan", "recurve", "club-fast-003", "male", "https://images.unsplash.com/photo-1539571696357-5a69c17a67c6?w=200&q=80"},
		{"arc-soac-rm-9", "ARC-SOAC-13", "farhan.soac", "farhan.soac@example.com", "Farhan Hidayat", "recurve", "club-fast-003", "male", "https://images.unsplash.com/photo-1501196354995-cbb51c65aaea?w=200&q=80"},
		// FAST Recurve Women
		{"arc-soac-rw-5", "ARC-SOAC-14", "ratna.soac", "ratna.soac@example.com", "Ratna Sari", "recurve", "club-fast-003", "female", "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=200&q=80"},

		// ARROWHEAD Recurve Men
		{"arc-soac-rm-10", "ARC-SOAC-15", "hendra.soac", "hendra.soac@example.com", "Hendra Wijaya", "recurve", "club-arrow-004", "male", "https://images.unsplash.com/photo-1522075469751-3a6694fb2f61?w=200&q=80"},
		{"arc-soac-rm-11", "ARC-SOAC-16", "tegar.soac", "tegar.soac@example.com", "Tegar Prasetyo", "recurve", "club-arrow-004", "male", "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200&q=80"},
		{"arc-soac-rm-12", "ARC-SOAC-17", "ilham.soac", "ilham.soac@example.com", "Ilham Fauzi", "recurve", "club-arrow-004", "male", "https://images.unsplash.com/photo-1492562080023-ab3db95bfbce?w=200&q=80"},

		// Compound Archers
		{"arc-soac-cmp-1", "ARC-SOAC-18", "bagus.soac", "bagus.soac@example.com", "Bagus Wibowo", "compound", "club-arrow-004", "male", "https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=200&q=80"},
		{"arc-soac-cmp-2", "ARC-SOAC-19", "arya.soac", "arya.soac@example.com", "Arya Satria", "compound", "club-arrow-004", "female", "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200&q=80"},
		{"arc-soac-cmp-3", "ARC-SOAC-20", "cahyo.soac", "cahyo.soac@example.com", "Cahyo Nugroho", "compound", "club-sac-001", "male", "https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?w=200&q=80"},
		{"arc-soac-cmp-4", "ARC-SOAC-21", "bayu.soac", "bayu.soac@example.com", "Bayu Tri Laksono", "compound", "club-sac-001", "female", "https://images.unsplash.com/photo-1544005313-94ddf0286df2?w=200&q=80"},
		{"arc-soac-cmp-5", "ARC-SOAC-22", "kevin.soac", "kevin.soac@example.com", "Kevin Sanjaya", "compound", "club-jas-002", "male", "https://images.unsplash.com/photo-1519085360753-af0119f7cbe7?w=200&q=80"},
		{"arc-soac-cmp-6", "ARC-SOAC-23", "nisa.soac", "nisa.soac@example.com", "Nisa Sabrina", "compound", "club-jas-002", "female", "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=200&q=80"},

		// Barebow Archers
		{"arc-soac-bar-1", "ARC-SOAC-24", "deni.soac", "deni.soac@example.com", "Deni Firmansyah", "barebow", "club-jas-002", "male", "https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=200&q=80"},
		{"arc-soac-bar-2", "ARC-SOAC-25", "wahyu.soac", "wahyu.soac@example.com", "Wahyu Pratama", "barebow", "club-jas-002", "male", "https://images.unsplash.com/photo-1501196354995-cbb51c65aaea?w=200&q=80"},
		{"arc-soac-bar-3", "ARC-SOAC-26", "eko.soac", "eko.soac@example.com", "Eko Prasetyo", "barebow", "club-fast-003", "male", "https://images.unsplash.com/photo-1539571696357-5a69c17a67c6?w=200&q=80"},
		{"arc-soac-bar-4", "ARC-SOAC-27", "sigit.soac", "sigit.soac@example.com", "Sigit Nugraha", "barebow", "club-fast-003", "male", "https://images.unsplash.com/photo-1522075469751-3a6694fb2f61?w=200&q=80"},
		{"arc-soac-bar-5", "ARC-SOAC-28", "fajar.soac", "fajar.soac@example.com", "Fajar Ramadhan", "barebow", "club-sac-001", "male", "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200&q=80"},
		{"arc-soac-bar-6", "ARC-SOAC-29", "budi.soac", "budi.soac@example.com", "Budi Santoso", "barebow", "club-sac-001", "male", "https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=200&q=80"},
	}

	for _, a := range archersData {
		_, err := db.Exec(`
			INSERT INTO archers (uuid, id, username, email, full_name, bow_type, club_id, gender, status, avatar_url)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active', ?)
			ON DUPLICATE KEY UPDATE 
				full_name=VALUES(full_name),
				bow_type=VALUES(bow_type),
				club_id=VALUES(club_id),
				gender=VALUES(gender),
				avatar_url=VALUES(avatar_url);
		`, a.UUID, a.ID, a.Username, a.Email, a.FullName, a.BowType, a.ClubID, a.Gender, a.AvatarURL)
		if err != nil {
			log.Fatalf("Insert archer error %s: %v", a.FullName, err)
		}
	}
	fmt.Println("✅ 5. Archers Seeded (29 athletes)")

	// 7. Tournament Participants
	participantsData := []struct {
		UUID       string
		ArcherUUID string
		CatUUID    string
		TargetName string
		BackNo     string
		Score      int
		Rank       int
	}{
		// Recurve Men
		{"part-soac-rm-01", "arc-soac-rm-1", "cat-soac-rec-men", "1A", "RM-01", 672, 1},
		{"part-soac-rm-02", "arc-soac-rm-4", "cat-soac-rec-men", "1B", "RM-02", 665, 2},
		{"part-soac-rm-03", "arc-soac-rm-7", "cat-soac-rec-men", "1C", "RM-03", 660, 3},
		{"part-soac-rm-04", "arc-soac-rm-2", "cat-soac-rec-men", "1D", "RM-04", 658, 4},
		{"part-soac-rm-05", "arc-soac-rm-10", "cat-soac-rec-men", "2A", "RM-05", 655, 5},
		{"part-soac-rm-06", "arc-soac-rm-5", "cat-soac-rec-men", "2B", "RM-06", 650, 6},
		{"part-soac-rm-07", "arc-soac-rm-3", "cat-soac-rec-men", "2C", "RM-07", 645, 7},
		{"part-soac-rm-08", "arc-soac-rm-8", "cat-soac-rec-men", "2D", "RM-08", 642, 8},
		{"part-soac-rm-09", "arc-soac-rm-6", "cat-soac-rec-men", "3A", "RM-09", 638, 9},
		{"part-soac-rm-10", "arc-soac-rm-11", "cat-soac-rec-men", "3B", "RM-10", 635, 10},
		{"part-soac-rm-11", "arc-soac-rm-9", "cat-soac-rec-men", "3C", "RM-11", 630, 11},
		{"part-soac-rm-12", "arc-soac-rm-12", "cat-soac-rec-men", "3D", "RM-12", 625, 12},

		// Recurve Women
		{"part-soac-rw-01", "arc-soac-rw-3", "cat-soac-rec-women", "4A", "RW-01", 660, 1},
		{"part-soac-rw-02", "arc-soac-rw-1", "cat-soac-rec-women", "4B", "RW-02", 654, 2},
		{"part-soac-rw-03", "arc-soac-rw-5", "cat-soac-rec-women", "4C", "RW-03", 648, 3},
		{"part-soac-rw-04", "arc-soac-rw-4", "cat-soac-rec-women", "4D", "RW-04", 642, 4},
		{"part-soac-rw-05", "arc-soac-rw-2", "cat-soac-rec-women", "5A", "RW-05", 638, 5},

		// Compound Men & Women
		{"part-soac-cmp-01", "arc-soac-cmp-1", "cat-soac-comp-men", "6A", "CM-01", 692, 1},
		{"part-soac-cmp-02", "arc-soac-cmp-3", "cat-soac-comp-men", "6B", "CM-02", 688, 2},
		{"part-soac-cmp-03", "arc-soac-cmp-5", "cat-soac-comp-men", "6C", "CM-03", 686, 3},
		{"part-soac-cmp-04", "arc-soac-cmp-2", "cat-soac-comp-women", "7A", "CW-01", 685, 1},
		{"part-soac-cmp-05", "arc-soac-cmp-4", "cat-soac-comp-women", "7B", "CW-02", 680, 2},
		{"part-soac-cmp-06", "arc-soac-cmp-6", "cat-soac-comp-women", "7C", "CW-03", 678, 3},

		// Barebow Men
		{"part-soac-bar-01", "arc-soac-bar-1", "cat-soac-bare-men", "8A", "BM-01", 625, 1},
		{"part-soac-bar-02", "arc-soac-bar-3", "cat-soac-bare-men", "8B", "BM-02", 618, 2},
		{"part-soac-bar-03", "arc-soac-bar-5", "cat-soac-bare-men", "8C", "BM-03", 615, 3},
		{"part-soac-bar-04", "arc-soac-bar-2", "cat-soac-bare-men", "8D", "BM-04", 610, 4},
		{"part-soac-bar-05", "arc-soac-bar-4", "cat-soac-bare-men", "9A", "BM-05", 602, 5},
		{"part-soac-bar-06", "arc-soac-bar-6", "cat-soac-bare-men", "9B", "BM-06", 595, 6},
	}

	for _, p := range participantsData {
		_, err := db.Exec(`
			INSERT INTO tournament_participants (
				uuid, tournament_id, archer_id, category_id, payment_amount, payment_status,
				target_name, back_number, registration_source, qual_score, qual_rank
			) VALUES (?, 't-soac-2026-full', ?, ?, 350000.00, 'paid', ?, ?, 'self_register', ?, ?)
		`, p.UUID, p.ArcherUUID, p.CatUUID, p.TargetName, p.BackNo, p.Score, p.Rank)
		if err != nil {
			log.Fatalf("Insert participant error %s: %v", p.UUID, err)
		}
	}
	fmt.Println("✅ 6. Participants Seeded")

	// 8. Sessions & Target Butts
	_, _ = db.Exec(`
		INSERT INTO qualification_sessions (uuid, tournament_uuid, session_code, session_date, name, start_time, end_time, total_ends, arrows_per_end, is_locked) VALUES
		('session-soac-qual-1', 't-soac-2026-full', 'SOAC-SESI-1', '2026-09-20', 'Kualifikasi Sesi 1 (Pagi)', '2026-09-20 08:00:00', '2026-09-20 12:00:00', 6, 6, 1),
		('session-soac-qual-2', 't-soac-2026-full', 'SOAC-SESI-2', '2026-09-20', 'Kualifikasi Sesi 2 (Siang)', '2026-09-20 13:30:00', '2026-09-20 17:30:00', 6, 6, 1)
		ON DUPLICATE KEY UPDATE is_locked=VALUES(is_locked);
	`)

	// 8b. Tournament Targets (Bantalan 1-10 with slots A, B, C, D)
	targetsSQL := `
		INSERT INTO tournament_targets (uuid, tournament_uuid, target_name, board_number) VALUES
		('target-soac-1a', 't-soac-2026-full', '1A', 1),
		('target-soac-1b', 't-soac-2026-full', '1B', 1),
		('target-soac-1c', 't-soac-2026-full', '1C', 1),
		('target-soac-1d', 't-soac-2026-full', '1D', 1),
		('target-soac-2a', 't-soac-2026-full', '2A', 2),
		('target-soac-2b', 't-soac-2026-full', '2B', 2),
		('target-soac-2c', 't-soac-2026-full', '2C', 2),
		('target-soac-2d', 't-soac-2026-full', '2D', 2),
		('target-soac-3a', 't-soac-2026-full', '3A', 3),
		('target-soac-3b', 't-soac-2026-full', '3B', 3),
		('target-soac-3c', 't-soac-2026-full', '3C', 3),
		('target-soac-3d', 't-soac-2026-full', '3D', 3),
		('target-soac-4a', 't-soac-2026-full', '4A', 4),
		('target-soac-4b', 't-soac-2026-full', '4B', 4),
		('target-soac-4c', 't-soac-2026-full', '4C', 4),
		('target-soac-4d', 't-soac-2026-full', '4D', 4),
		('target-soac-5a', 't-soac-2026-full', '5A', 5),
		('target-soac-5b', 't-soac-2026-full', '5B', 5),
		('target-soac-5c', 't-soac-2026-full', '5C', 5),
		('target-soac-5d', 't-soac-2026-full', '5D', 5),
		('target-soac-6a', 't-soac-2026-full', '6A', 6),
		('target-soac-6b', 't-soac-2026-full', '6B', 6),
		('target-soac-6c', 't-soac-2026-full', '6C', 6),
		('target-soac-6d', 't-soac-2026-full', '6D', 6),
		('target-soac-7a', 't-soac-2026-full', '7A', 7),
		('target-soac-7b', 't-soac-2026-full', '7B', 7),
		('target-soac-7c', 't-soac-2026-full', '7C', 7),
		('target-soac-7d', 't-soac-2026-full', '7D', 7),
		('target-soac-8a', 't-soac-2026-full', '8A', 8),
		('target-soac-8b', 't-soac-2026-full', '8B', 8),
		('target-soac-8c', 't-soac-2026-full', '8C', 8),
		('target-soac-8d', 't-soac-2026-full', '8D', 8),
		('target-soac-9a', 't-soac-2026-full', '9A', 9),
		('target-soac-9b', 't-soac-2026-full', '9B', 9),
		('target-soac-9c', 't-soac-2026-full', '9C', 9),
		('target-soac-9d', 't-soac-2026-full', '9D', 9),
		('target-soac-10a', 't-soac-2026-full', '10A', 10),
		('target-soac-10b', 't-soac-2026-full', '10B', 10),
		('target-soac-10c', 't-soac-2026-full', '10C', 10),
		('target-soac-10d', 't-soac-2026-full', '10D', 10)
		ON DUPLICATE KEY UPDATE target_name=VALUES(target_name), board_number=VALUES(board_number);
	`
	if _, err := db.Exec(targetsSQL); err != nil {
		log.Fatalf("Tournament targets error: %v", err)
	}
	fmt.Println("✅ 8a. Tournament Targets Seeded (10 Boards x 4 Slots = 40 Targets)")

	// 8c. Target Boards per Category (target_board_qualification)
	targetBoardsSQL := `
		INSERT INTO target_board_qualification (uuid, session_uuid, category_uuid, board_number, code) VALUES
		('tbq-soac-1', 'session-soac-qual-1', 'cat-soac-rec-men', 1, '001REC'),
		('tbq-soac-2', 'session-soac-qual-1', 'cat-soac-rec-men', 2, '002REC'),
		('tbq-soac-3', 'session-soac-qual-1', 'cat-soac-rec-men', 3, '003REC'),
		('tbq-soac-4', 'session-soac-qual-1', 'cat-soac-rec-women', 4, '004RCW'),
		('tbq-soac-5', 'session-soac-qual-1', 'cat-soac-rec-women', 5, '005RCW'),
		('tbq-soac-6', 'session-soac-qual-1', 'cat-soac-comp-men', 6, '006COM'),
		('tbq-soac-7', 'session-soac-qual-1', 'cat-soac-comp-women', 7, '007COW'),
		('tbq-soac-8', 'session-soac-qual-1', 'cat-soac-bare-men', 8, '008BAR'),
		('tbq-soac-9', 'session-soac-qual-1', 'cat-soac-bare-men', 9, '009BAR'),
		('tbq-soac-10', 'session-soac-qual-1', 'cat-soac-bare-men', 10, '010EXT')
		ON DUPLICATE KEY UPDATE code=VALUES(code), board_number=VALUES(board_number);
	`
	if _, err := db.Exec(targetBoardsSQL); err != nil {
		log.Fatalf("Target board qualification error: %v", err)
	}
	fmt.Println("✅ 8b. Target Boards Qualification Seeded")

	// 8d. Qualification Target Assignments
	for _, p := range participantsData {
		targetUUID := fmt.Sprintf("target-soac-%s", strings.ToLower(p.TargetName))
		// Extract board number
		boardNumStr := ""
		for _, r := range p.TargetName {
			if r >= '0' && r <= '9' {
				boardNumStr += string(r)
			}
		}
		boardNum, _ := strconv.Atoi(boardNumStr)
		targetBoardUUID := fmt.Sprintf("tbq-soac-%d", boardNum)
		assignmentUUID := fmt.Sprintf("qta-soac-%s", p.UUID)

		_, err := db.Exec(`
			INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, target_board_id)
			VALUES (?, 'session-soac-qual-1', ?, ?, ?)
			ON DUPLICATE KEY UPDATE target_uuid=VALUES(target_uuid), target_board_id=VALUES(target_board_id);
		`, assignmentUUID, p.UUID, targetUUID, targetBoardUUID)
		if err != nil {
			log.Fatalf("Target assignment error %s: %v", p.UUID, err)
		}
	}
	fmt.Println("✅ 8c. Qualification Target Assignments Seeded (29 archers mapped)")

	// 8e. Elimination Target Boards (target_board_elimination)
	_, _ = db.Exec(`
		INSERT INTO target_board_elimination (uuid, bracket_uuid, category_uuid, board_number, code) VALUES
		('tbe-soac-rm-1', 'brk-soac-rec-men', 'cat-soac-rec-men', 1, 'E01RM'),
		('tbe-soac-rm-2', 'brk-soac-rec-men', 'cat-soac-rec-men', 2, 'E02RM'),
		('tbe-soac-rm-3', 'brk-soac-rec-men', 'cat-soac-rec-men', 3, 'E03RM'),
		('tbe-soac-rm-4', 'brk-soac-rec-men', 'cat-soac-rec-men', 4, 'E04RM'),
		('tbe-soac-rmt-1', 'brk-soac-rec-team', 'cat-soac-rec-team-men', 5, 'E05RMT'),
		('tbe-soac-rmt-2', 'brk-soac-rec-team', 'cat-soac-rec-team-men', 6, 'E06RMT')
		ON DUPLICATE KEY UPDATE code=VALUES(code);
	`)
	fmt.Println("✅ 8d. Target Boards Elimination Seeded")

	// 9. Qualification Ends
	for _, p := range participantsData {
		baseEnd := p.Score / 6
		rem := p.Score % 6
		for endNum := 1; endNum <= 6; endNum++ {
			endScore := baseEnd
			if endNum <= rem {
				endScore++
			}
			xCount := 2
			tenCount := 3
			endUUID := fmt.Sprintf("end-%s-%d", p.UUID, endNum)
			_, _ = db.Exec(`
				INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end)
				VALUES (?, 'session-soac-qual-1', ?, ?, ?, ?, ?)
			`, endUUID, p.UUID, endNum, endScore, xCount, tenCount)
		}
	}
	fmt.Println("✅ 7. Qualification End Scores Seeded")

	// 10. Official Teams (`teams` & `team_members`)
	type TeamSeed struct {
		UUID           string
		CategoryID     string
		TeamName       string
		Rank           int
		TotalScore     int
		TotalX         int
		ParticipantIDs []string
	}

	teamsToSeed := []TeamSeed{
		// Recurve Men Team (3 members each)
		{
			UUID:           "team-rec-m-sac",
			CategoryID:     "cat-soac-rec-team-men",
			TeamName:       "Sleman Archery Club Recurve Men",
			Rank:           1,
			TotalScore:     1975,
			TotalX:         65,
			ParticipantIDs: []string{"part-soac-rm-01", "part-soac-rm-04", "part-soac-rm-07"},
		},
		{
			UUID:           "team-rec-m-jas",
			CategoryID:     "cat-soac-rec-team-men",
			TeamName:       "Jogja Archery School Recurve Men",
			Rank:           2,
			TotalScore:     1953,
			TotalX:         58,
			ParticipantIDs: []string{"part-soac-rm-02", "part-soac-rm-06", "part-soac-rm-09"},
		},
		{
			UUID:           "team-rec-m-fast",
			CategoryID:     "cat-soac-rec-team-men",
			TeamName:       "Fast Archery Academy Recurve Men",
			Rank:           3,
			TotalScore:     1932,
			TotalX:         52,
			ParticipantIDs: []string{"part-soac-rm-03", "part-soac-rm-08", "part-soac-rm-11"},
		},
		{
			UUID:           "team-rec-m-arrow",
			CategoryID:     "cat-soac-rec-team-men",
			TeamName:       "Arrowhead Club Yogyakarta Recurve Men",
			Rank:           4,
			TotalScore:     1915,
			TotalX:         48,
			ParticipantIDs: []string{"part-soac-rm-05", "part-soac-rm-10", "part-soac-rm-12"},
		},

		// Recurve Mixed Team (1 M + 1 F)
		{
			UUID:           "team-rec-mix-sac",
			CategoryID:     "cat-soac-rec-mixed",
			TeamName:       "Sleman Archery Club Recurve Mixed",
			Rank:           1,
			TotalScore:     1326,
			TotalX:         45,
			ParticipantIDs: []string{"part-soac-rm-01", "part-soac-rw-02"},
		},
		{
			UUID:           "team-rec-mix-jas",
			CategoryID:     "cat-soac-rec-mixed",
			TeamName:       "Jogja Archery School Recurve Mixed",
			Rank:           2,
			TotalScore:     1325,
			TotalX:         44,
			ParticipantIDs: []string{"part-soac-rm-02", "part-soac-rw-01"},
		},
		{
			UUID:           "team-rec-mix-fast",
			CategoryID:     "cat-soac-rec-mixed",
			TeamName:       "Fast Archery Academy Recurve Mixed",
			Rank:           3,
			TotalScore:     1308,
			TotalX:         38,
			ParticipantIDs: []string{"part-soac-rm-03", "part-soac-rw-03"},
		},

		// Compound Mixed Team (1 M + 1 F)
		{
			UUID:           "team-cmp-mix-arrow",
			CategoryID:     "cat-soac-comp-mixed",
			TeamName:       "Arrowhead Compound Mixed",
			Rank:           1,
			TotalScore:     1377,
			TotalX:         72,
			ParticipantIDs: []string{"part-soac-cmp-01", "part-soac-cmp-04"},
		},
		{
			UUID:           "team-cmp-mix-sac",
			CategoryID:     "cat-soac-comp-mixed",
			TeamName:       "SAC Compound Mixed",
			Rank:           2,
			TotalScore:     1368,
			TotalX:         68,
			ParticipantIDs: []string{"part-soac-cmp-02", "part-soac-cmp-05"},
		},
		{
			UUID:           "team-cmp-mix-jas",
			CategoryID:     "cat-soac-comp-mixed",
			TeamName:       "JAS Compound Mixed",
			Rank:           3,
			TotalScore:     1364,
			TotalX:         64,
			ParticipantIDs: []string{"part-soac-cmp-03", "part-soac-cmp-06"},
		},

		// Barebow Men Team (2 members)
		{
			UUID:           "team-bar-m-jas",
			CategoryID:     "cat-soac-bare-team-men",
			TeamName:       "JAS Barebow Men Team",
			Rank:           1,
			TotalScore:     1235,
			TotalX:         32,
			ParticipantIDs: []string{"part-soac-bar-01", "part-soac-bar-04"},
		},
		{
			UUID:           "team-bar-m-fast",
			CategoryID:     "cat-soac-bare-team-men",
			TeamName:       "FAST Barebow Men Team",
			Rank:           2,
			TotalScore:     1220,
			TotalX:         28,
			ParticipantIDs: []string{"part-soac-bar-02", "part-soac-bar-05"},
		},
		{
			UUID:           "team-bar-m-sac",
			CategoryID:     "cat-soac-bare-team-men",
			TeamName:       "SAC Barebow Men Team",
			Rank:           3,
			TotalScore:     1210,
			TotalX:         26,
			ParticipantIDs: []string{"part-soac-bar-03", "part-soac-bar-06"},
		},
	}

	for _, tm := range teamsToSeed {
		_, err := db.Exec(`
			INSERT INTO teams (uuid, tournament_id, category_id, event_id, team_name, team_rank, total_score, total_x_count, status)
			VALUES (?, 't-soac-2026-full', ?, ?, ?, ?, ?, ?, 'active')
		`, tm.UUID, tm.CategoryID, tm.CategoryID, tm.TeamName, tm.Rank, tm.TotalScore, tm.TotalX)
		if err != nil {
			log.Fatalf("Insert team error %s: %v", tm.TeamName, err)
		}

		for idx, pID := range tm.ParticipantIDs {
			memberUUID := fmt.Sprintf("tm-%s-%d", tm.UUID, idx+1)
			_, err := db.Exec(`
				INSERT INTO team_members (uuid, team_id, participant_id, member_order)
				VALUES (?, ?, ?, ?)
			`, memberUUID, tm.UUID, pID, idx+1)
			if err != nil {
				log.Fatalf("Insert team member error %s: %v", tm.TeamName, err)
			}
		}
	}
	fmt.Println("✅ 8. Official Teams & Members Seeded (Recurve Men, Recurve Mixed, Compound Mixed, Barebow Teams)")

	// 11. Elimination Brackets & Match Trees
	// Bracket 1: Recurve Men Individual (8-Archer Bracket)
	bracketRM := "brk-soac-rec-men"
	_, err = db.Exec(`
		INSERT INTO elimination_brackets (uuid, bracket_id, tournament_uuid, category_uuid, bracket_type, format, bracket_size, status)
		VALUES (?, 'BRK-SOAC-RM', 't-soac-2026-full', 'cat-soac-rec-men', 'individual', 'recurve_set', 8, 'completed')
	`, bracketRM)
	if err != nil {
		log.Fatalf("Bracket error: %v", err)
	}

	entrySeeds := []struct {
		UUID   string
		PartID string
		Seed   int
		Score  int
	}{
		{"entry-rm-1", "part-soac-rm-01", 1, 672},
		{"entry-rm-2", "part-soac-rm-02", 2, 665},
		{"entry-rm-3", "part-soac-rm-03", 3, 660},
		{"entry-rm-4", "part-soac-rm-04", 4, 658},
		{"entry-rm-5", "part-soac-rm-05", 5, 655},
		{"entry-rm-6", "part-soac-rm-06", 6, 650},
		{"entry-rm-7", "part-soac-rm-07", 7, 645},
		{"entry-rm-8", "part-soac-rm-08", 8, 642},
	}

	for _, es := range entrySeeds {
		_, _ = db.Exec(`
			INSERT INTO elimination_entries (uuid, bracket_uuid, participant_type, participant_uuid, seed, qual_total_score)
			VALUES (?, ?, 'archer', ?, ?, ?)
		`, es.UUID, bracketRM, es.PartID, es.Seed, es.Score)
	}

	matchesSeed := []struct {
		UUID    string
		MatchID string
		Round   int
		MatchNo int
		EntryA  string
		EntryB  string
		Winner  string
		PointsA int
		PointsB int
	}{
		// Quarterfinals (Round 1)
		{"m-rm-qf-1", "M-RM-QF-1", 1, 1, "entry-rm-1", "entry-rm-8", "entry-rm-1", 6, 2},
		{"m-rm-qf-2", "M-RM-QF-2", 1, 2, "entry-rm-4", "entry-rm-5", "entry-rm-4", 6, 4},
		{"m-rm-qf-3", "M-RM-QF-3", 1, 3, "entry-rm-3", "entry-rm-6", "entry-rm-3", 6, 0},
		{"m-rm-qf-4", "M-RM-QF-4", 1, 4, "entry-rm-2", "entry-rm-7", "entry-rm-2", 6, 2},

		// Semifinals (Round 2)
		{"m-rm-sf-1", "M-RM-SF-1", 2, 1, "entry-rm-1", "entry-rm-4", "entry-rm-1", 6, 4},
		{"m-rm-sf-2", "M-RM-SF-2", 2, 2, "entry-rm-3", "entry-rm-2", "entry-rm-2", 5, 6},

		// Bronze Match (Round 3)
		{"m-rm-bronze", "M-RM-BRONZE", 3, 2, "entry-rm-4", "entry-rm-3", "entry-rm-3", 2, 6},

		// Gold Final (Round 3)
		{"m-rm-gold", "M-RM-GOLD", 3, 1, "entry-rm-1", "entry-rm-2", "entry-rm-1", 6, 4},
	}

	for _, m := range matchesSeed {
		_, _ = db.Exec(`
			INSERT INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, winner_entry_uuid, status, total_points_a, total_points_b)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'finished', ?, ?)
		`, m.UUID, m.MatchID, bracketRM, m.Round, m.MatchNo, m.EntryA, m.EntryB, m.Winner, m.PointsA, m.PointsB)
	}

	// Bracket 2: Recurve Men Team (4-Team Bracket)
	bracketRMT := "brk-soac-rec-team"
	_, _ = db.Exec(`
		INSERT INTO elimination_brackets (uuid, bracket_id, tournament_uuid, category_uuid, bracket_type, format, bracket_size, status)
		VALUES (?, 'BRK-SOAC-RMT', 't-soac-2026-full', 'cat-soac-rec-team-men', 'team3', 'recurve_set', 4, 'completed')
	`, bracketRMT)

	teamEntries := []struct {
		UUID   string
		TeamID string
		Seed   int
		Score  int
	}{
		{"entry-team-1", "team-rec-m-sac", 1, 1975},
		{"entry-team-2", "team-rec-m-jas", 2, 1953},
		{"entry-team-3", "team-rec-m-fast", 3, 1932},
		{"entry-team-4", "team-rec-m-arrow", 4, 1915},
	}

	for _, te := range teamEntries {
		_, _ = db.Exec(`
			INSERT INTO elimination_entries (uuid, bracket_uuid, participant_type, participant_uuid, seed, qual_total_score)
			VALUES (?, ?, 'team', ?, ?, ?)
		`, te.UUID, bracketRMT, te.TeamID, te.Seed, te.Score)
	}

	teamMatches := []struct {
		UUID    string
		MatchID string
		Round   int
		MatchNo int
		EntryA  string
		EntryB  string
		Winner  string
		PointsA int
		PointsB int
	}{
		// Semifinals
		{"m-rmt-sf-1", "M-RMT-SF-1", 1, 1, "entry-team-1", "entry-team-4", "entry-team-1", 6, 0},
		{"m-rmt-sf-2", "M-RMT-SF-2", 1, 2, "entry-team-2", "entry-team-3", "entry-team-2", 6, 2},
		// Bronze
		{"m-rmt-bronze", "M-RMT-BRONZE", 2, 2, "entry-team-4", "entry-team-3", "entry-team-3", 2, 6},
		// Gold Final
		{"m-rmt-gold", "M-RMT-GOLD", 2, 1, "entry-team-1", "entry-team-2", "entry-team-1", 6, 4},
	}

	for _, tm := range teamMatches {
		_, _ = db.Exec(`
			INSERT INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, winner_entry_uuid, status, total_points_a, total_points_b)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'finished', ?, ?)
		`, tm.UUID, tm.MatchID, bracketRMT, tm.Round, tm.MatchNo, tm.EntryA, tm.EntryB, tm.Winner, tm.PointsA, tm.PointsB)
	}

	fmt.Println("✅ 9. Elimination Brackets & Match Trees Seeded with Finals & Winners")
	fmt.Println("\n🎉🎉 COMPREHENSIVE SEEDING COMPLETED SUCCESSFULLY!")
}
