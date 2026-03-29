package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type SeedData struct {
	BowTypes       []string
	AgeGroups      []string
	EventTypes     []string
	GenderDivs     []string
	Archers        []string
	Organizations  []string
}

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/archeryhub?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	fmt.Println("🌱 Seeding 5 new events to VPS...")

	data := SeedData{
		BowTypes: []string{
			"349a5218-fed0-4305-ab16-e636501bb5df", // Compound
			"5c95a503-4fd4-465f-9dd3-b08568181792", // Recurve
			"94bf104d-8ef2-4dd0-a1b1-2d82b46a1bdc", // Barebow
		},
		AgeGroups: []string{
			"55c3beff-8d11-4c93-8a5a-25eff347af4a", // U-13
			"f0ab19a5-efe2-4ceb-b177-57966249af04", // U-18
			"f235b870-724b-44ac-8683-b665df0c0548", // Umum
		},
		EventTypes: []string{
			"da2740c8-f7ac-460f-a8c2-46f0c6ec844f", // Individual
			"3bfbc4ad-afb2-44b2-a686-8d5fd46e5e2f", // Team
		},
		GenderDivs: []string{
			"d60f4939-4d88-4ded-bf4d-6d8cff4de5ae", // Men
			"afbded2f-705c-480f-84d2-6962bcb4b2ef", // Women
		},
		Archers: []string{
			"0f935f1a-f51e-4228-9db8-41c41e6688f1", "11e0974c-a7f6-4b76-811f-5291137f164e",
			"13d10c7f-0c12-4336-8920-657465ebac8e", "1535520e-b139-4e0d-b0ec-eaed0e28b38e",
			"1bfd6f26-e0e7-40e3-b7ce-0a1e9cd91351", "205e215d-7a35-4432-b5a8-13e225283fe5",
			"32133e95-ffa4-4360-9c24-abbb80de8652", "36d8060c-517b-4375-84a2-dfdc28a6252d",
			"47eb76e8-9cbe-4df2-9d80-a1435b6ca2a0", "50ecd06f-dba7-4442-98d1-f0a239d716f1",
		},
		Organizations: []string{
			"34d712a4-f992-11f0-87db-c3c8a1ce2650", // PERPANI DKI
			"34d7162e-f992-11f0-87db-c3c8a1ce2650", // Yogyakarta Archery
		},
	}

	ts := time.Now().Unix()
	events := []struct {
		Name  string
		Slug  string
		Venue string
		City  string
	}{
		{"Sleman Junior Cup 2026", fmt.Sprintf("sleman-junior-cup-2026-%d", ts), "Lapangan Kedulan", "Sleman"},
		{"Solo Barebow Championship", fmt.Sprintf("solo-barebow-championship-%d", ts), "Sritex Arena Solo", "Solo"},
		{"Sultan Hamengkubuwono Cup X", fmt.Sprintf("sultan-hb-cup-x-2026-%d", ts), "Stadion Mandala Krida", "Yogyakarta"},
		{"ArcheryHub Pro Series Jakarta", fmt.Sprintf("archeryhub-pro-jakarta-2026-%d", ts), "Gelora Bung Karno", "Jakarta"},
		{"Brawijaya Traditional Open", fmt.Sprintf("brawijaya-traditional-open-2026-%d", ts), "Universitas Brawijaya", "Malang"},
	}

	for i, e := range events {
		eventUUID := uuid.New().String()
		startDate := time.Now().AddDate(0, i+1, 0)
		endDate := startDate.AddDate(0, 0, 2)
		regDeadline := startDate.AddDate(0, 0, -7)

		// 1. Insert Event
		eventCode := fmt.Sprintf("SEED-%d%d", time.Now().Unix()%10000, i)
		bannerURL := fmt.Sprintf("https://picsum.photos/seed/%s/1200/400", uuid.New().String())
		logoURL := fmt.Sprintf("https://picsum.photos/seed/%s/400/400", uuid.New().String())
		_, err := db.Exec(`
			INSERT INTO events (uuid, slug, code, name, venue, city, start_date, end_date, registration_deadline, status, organizer_id, entry_fee, description, banner_url, logo_url)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', 'a1fdd1c4-632a-44d9-9be4-c96461e4530e', 150000.00, ?, ?, ?)
		`, eventUUID, e.Slug, eventCode, e.Name, e.Venue, e.City, startDate, endDate, regDeadline, "Dynamic seeded event for mobile app testing.", bannerURL, logoURL)
		if err != nil {
			fmt.Printf("Error inserting event %s: %v\n", e.Name, err)
			continue
		}

		// 2. Insert Categories (3 per event)
		catUUIDs := []string{}
		for j := 0; j < 3; j++ {
			catUUID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO event_categories (uuid, event_id, division_uuid, category_uuid, event_type_uuid, gender_division_uuid, status)
				VALUES (?, ?, ?, ?, ?, ?, 'active')
			`, catUUID, eventUUID, data.BowTypes[rand.Intn(len(data.BowTypes))], data.AgeGroups[rand.Intn(len(data.AgeGroups))], data.EventTypes[rand.Intn(len(data.EventTypes))], data.GenderDivs[rand.Intn(len(data.GenderDivs))])
			if err == nil {
				catUUIDs = append(catUUIDs, catUUID)
			}
		}

		// 3. Insert Participants (10 per event)
		for j := 0; j < 10; j++ {
			partUUID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, registration_date, payment_amount)
				VALUES (?, ?, ?, ?, 'lunas', ?, 150000.00)
			`, partUUID, eventUUID, data.Archers[rand.Intn(len(data.Archers))], catUUIDs[rand.Intn(len(catUUIDs))], time.Now())
		}

		// 4. Insert Schedule (3 per event)
		titles := []string{"Registration", "Qualification", "Final Round"}
		for j, title := range titles {
			schedUUID := uuid.New().String()
			st := startDate.Add(time.Duration(8+j*3) * time.Hour)
			et := st.Add(2 * time.Hour)
			_, err = db.Exec(`
				INSERT INTO event_schedule (uuid, event_id, title, start_time, end_time, day_order, sort_order)
				VALUES (?, ?, ?, ?, ?, 1, ?)
			`, schedUUID, eventUUID, title, st, et, j+1)
		}

		// 5. Insert Gallery (4 images)
		for j := 1; j <= 4; j++ {
			imgUUID := uuid.New().String()
			imgURL := fmt.Sprintf("https://picsum.photos/seed/%s/800/600", uuid.New().String())
			_, err = db.Exec(`
				INSERT INTO event_images (uuid, event_id, url, caption, display_order, is_primary)
				VALUES (?, ?, ?, ?, ?, ?)
			`, imgUUID, eventUUID, imgURL, fmt.Sprintf("Action Shot %d", j), j, j == 1)
		}

		fmt.Printf("✅ Seeded Event: %s (%s)\n", e.Name, eventUUID)
	}

	fmt.Println("🚀 Seeding complete!")
}
