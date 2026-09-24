package main

import (
	"Archeris-api/handler"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	db, err := sqlx.Connect("mysql", "root:@tcp(127.0.0.1:3306)/archeris?parseTime=true")
	if err != nil {
		fmt.Printf("Database connection failed: %v\n", err)
		return
	}
	defer db.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/tournaments/:id/qualification/scoresheet", handler.GetQualificationScoresheet(db))

	req, _ := http.NewRequest("GET", "/tournaments/tournament-testing-1/qualification/scoresheet?session=TT1-SESI-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		fmt.Printf("Failed: status %d, body: %s\n", w.Code, w.Body.String())
		return
	}

	pdfBytes := w.Body.Bytes()
	outPath := "sample_4in1_scoresheet.pdf"
	_ = os.WriteFile(outPath, pdfBytes, 0644)
	fmt.Printf("SUCCESS: Saved %d bytes to %s\n", len(pdfBytes), outPath)
}
