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

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/Archeris?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	fmt.Println("ðŸ¹ Seeding 12 marketplace products to VPS...")

	products := []struct {
		Name     string
		Category string
		Price    float64
	}{
		{"Busur ILF Premium Recurve", "equipment", 4500000.00},
		{"Arrow Carbon Link 300 (12pcs)", "equipment", 1250000.00},
		{"Quiver Side Leather Modern", "accessories", 350000.00},
		{"Finger Tab Cordovan Professional", "accessories", 850000.00},
		{"Chest Guard Breathable Mesh", "apparel", 150000.00},
		{"Arm Guard Polymer Lightweight", "accessories", 120000.00},
		{"Bow Stand Foldable Aluminum", "accessories", 250000.00},
		{"Sight Recurve Precision X", "equipment", 1850000.00},
		{"Target Face JVD 80cm (10pcs)", "training", 150000.00},
		{"Backstop Net Panahan 3x3m", "training", 2100000.00},
		{"Jersey Archeris Official 2026", "apparel", 225000.00},
		{"Compound Bow Release Aid", "accessories", 1450000.00},
	}

	sellerID := "1038a62b-1c83-11f1-87db-c3c8a1ce2650"

	for i, p := range products {
		productUUID := uuid.New().String()
		slug := fmt.Sprintf("produk-%d-%d", time.Now().Unix()%10000, i)
		imgURL := fmt.Sprintf("https://picsum.photos/seed/%s/600/600", productUUID)
		stock := rand.Intn(50) + 5

		_, err := db.Exec(`
			INSERT INTO products (uuid, seller_id, name, slug, description, price, category, stock, status, image_url)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active', ?)
		`, productUUID, sellerID, p.Name, slug, "Kualitas terbaik untuk kebutuhan panahan Anda. "+p.Name+" dirancang untuk performa maksimal.", p.Price, p.Category, stock, imgURL)

		if err != nil {
			fmt.Printf("Error inserting product %s: %v\n", p.Name, err)
		} else {
			fmt.Printf("âœ… Seeded Product: %s\n", p.Name)
		}
	}

	fmt.Println("ðŸš€ Product seeding complete!")
}

