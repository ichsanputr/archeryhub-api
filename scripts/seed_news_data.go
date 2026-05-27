package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type NewsItem struct {
	Title   string
	Excerpt string
	Content string
}

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/archeris?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	fmt.Println("📰 Seeding news articles to VPS in English...")

	articles := []NewsItem{
		{
			Title:   "List of Verified Archery Clubs in Indonesia",
			Excerpt: "A list of verified archery clubs in Indonesia.",
			Content: "Here is the complete list of verified archery clubs in Indonesia. These clubs offer professional training facilities and certified coaches to help archers of all skill levels improve.",
		},
		{
			Title:   "Understanding the Difference Between Recurve and Compound Bows for Young Athletes",
			Excerpt: "Key differences between Recurve and Compound bows to help young athletes make the right choice.",
			Content: "Choosing the right bow type is crucial for young archers. Recurve bows are traditional and focus on basic form, while Compound bows use a pulley system to reduce draw weight. This guide breaks down the benefits of each for youth training.",
		},
		{
			Title:   "Inspirational Story: The Journey to a POPDA Gold Medal",
			Excerpt: "An inspiring journey of a young archer winning a gold medal at POPDA.",
			Content: "Winning a gold medal at the regional sports week (POPDA) requires dedication, focus, and hours of practice. Read this inspiring story of a young archer who overcame obstacles to achieve their dream on the shooting line.",
		},
		{
			Title:   "Tips on Choosing the First Bow for Beginners",
			Excerpt: "A guide to selecting the right bow for beginner archers.",
			Content: "For beginner archers, choosing the first bow can be overwhelming. Learn about draw weight, bow length, and limb materials to find the perfect fit for your starting training sessions.",
		},
		{
			Title:   "Real-time Scoring Technology Innovation on ArcheryHub",
			Excerpt: "How ArcheryHub is revolutionizing tournament scoring with real-time digital updates.",
			Content: "ArcheryHub introduces state-of-the-art real-time scoring. Scorekeepers can enter results via mobile web, updating leaderboards instantly for spectators and coaches around the world.",
		},
		{
			Title:   "Review: The Latest Carbon Stabilizer in the ArcheryHub Marketplace",
			Excerpt: "An in-depth look at the latest carbon stabilizers available on the marketplace.",
			Content: "Stabilizers are essential for dampening bow vibration and improving stability. We review the latest carbon stabilizers in the ArcheryHub marketplace to help you pick the best gear for your setup.",
		},
		{
			Title:   "Tips for Choosing Your First Archery Equipment",
			Excerpt: "Essential gear guide for beginner archers starting their journey.",
			Content: "Starting your archery journey? Here are the essential tips on selecting your first bow, arrows, finger tab, and arm guard to ensure safe and effective practice sessions.",
		},
		{
			Title:   "The Importance of Physical Training Outside the Archery Range",
			Excerpt: "Why off-range physical fitness is crucial for improving your shooting form.",
			Content: "Archery is not just about aiming; it requires core strength, stability, and shoulder endurance. Learn why physical workouts like core training and cardio outside the range can significantly improve your shooting accuracy.",
		},
		{
			Title:   "How to Calculate Archery Scores Correctly",
			Excerpt: "A simple guide to understanding target face scoring and rules.",
			Content: "Understanding how target face rings translate to points is fundamental. This guide explains scoring rules for indoor and outdoor rounds, including X rings and target ring colors.",
		},
		{
			Title:   "Congratulations to the Winners of the ArcheryHub Championship 2024",
			Excerpt: "Celebrating the outstanding archers who triumphed at our annual championship.",
			Content: "The ArcheryHub Championship 2024 has successfully concluded. We congratulate all the winners across all divisions for their exceptional performances and sportsmanship.",
		},
	}

	authorID := "a1fdd1c4-632a-44d9-9be4-c96461e4530e"
	authorName := "Admin Archeris"
	orgID := "a1fdd1c4-632a-44d9-9be4-c96461e4530e"

	for i, art := range articles {
		articleUUID := uuid.New().String()
		
		// generate slug based on title
		slug := strings.ToLower(art.Title)
		slug = strings.ReplaceAll(slug, " ", "-")
		slug = strings.ReplaceAll(slug, ":", "")
		slug = strings.ReplaceAll(slug, "?", "")
		slug = strings.ReplaceAll(slug, ",", "")
		slug = strings.ReplaceAll(slug, "&", "and")

		imgURL := fmt.Sprintf("https://picsum.photos/seed/%s/800/450", articleUUID)
		publishAt := time.Now().AddDate(0, 0, -i*2)

		_, err := db.Exec(`
			INSERT INTO news (uuid, organization_id, title, slug, excerpt, content, image_url, category, status, author_name, author_id, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'pengumuman', 'published', ?, ?, ?)
			ON DUPLICATE KEY UPDATE title = VALUES(title), excerpt = VALUES(excerpt), content = VALUES(content)
		`, articleUUID, orgID, art.Title, slug, art.Excerpt, art.Content, imgURL, authorName, authorID, publishAt)

		if err != nil {
			fmt.Printf("Error inserting news %s: %v\n", art.Title, err)
		} else {
			fmt.Printf("✅ Seeded News: %s\n", art.Title)
		}
	}

	fmt.Println("🚀 News seeding complete!")
}
