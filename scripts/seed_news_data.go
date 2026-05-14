package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/Archeris?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	fmt.Println("ðŸ“° Seeding 10 news articles to VPS...")

	titles := []string{
		"Tips Memilih Busur Pertama untuk Pemula",
		"Persiapan Mental Sebelum Turnamen Panahan Besar",
		"Mengenal Perbedaan Recurve dan Compound untuk Atlet Muda",
		"Jadwal Turnamen Archeris Seri Nasional 2026",
		"Review: Stabilizer Karbon Terbaru di Marketplace Archeris",
		"Bagaimana Cara Menghitung Skor Panahan yang Benar?",
		"Kisah Inspiratif: Perjalanan Menuju Medali Emas POPDA",
		"Pentingnya Latihan Fisik di Luar Area Panahan",
		"Inovasi Teknologi Scoring Real-time di Archeris",
		"Daftar Klub Panahan Terverifikasi di Indonesia",
	}

	authorID := "a1fdd1c4-632a-44d9-9be4-c96461e4530e"
	authorName := "Admin Archeris"
	orgID := "a1fdd1c4-632a-44d9-9be4-c96461e4530e"

	for i, title := range titles {
		articleUUID := uuid.New().String()
		slug := fmt.Sprintf("artikel-%d-%d", time.Now().Unix()%10000, i)
		imgURL := fmt.Sprintf("https://picsum.photos/seed/%s/800/450", articleUUID)
		publishAt := time.Now().AddDate(0, 0, -i*2)

		_, err := db.Exec(`
			INSERT INTO news (uuid, organization_id, title, slug, excerpt, content, image_url, category, status, author_name, author_id, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'pengumuman', 'published', ?, ?, ?)
		`, articleUUID, orgID, title, slug, "Ringkasan singkat tentang "+title, "Isi konten artikel lengkap untuk "+title+". Detail dan informasi mendalam tentang topik ini akan sangat berguna bagi para pemanah.", imgURL, authorName, authorID, publishAt)

		if err != nil {
			fmt.Printf("Error inserting news %s: %v\n", title, err)
		} else {
			fmt.Printf("âœ… Seeded News: %s\n", title)
		}
	}

	fmt.Println("ðŸš€ News seeding complete!")
}

