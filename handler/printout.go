package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/jung-kurt/gofpdf"
)

// ─────────────────────────────────────────────────────────────────────────────
// DATA STRUCTURES
// ─────────────────────────────────────────────────────────────────────────────

type PrintEventInfo struct {
	UUID      string         `db:"uuid"`
	Name      string         `db:"name"`
	Slug      string         `db:"slug"`
	Venue     sql.NullString `db:"venue"`
	Location  sql.NullString `db:"location"`
	City      sql.NullString `db:"city"`
	StartDate sql.NullTime   `db:"start_date"`
	EndDate   sql.NullTime   `db:"end_date"`
	LogoURL   sql.NullString `db:"logo_url"`
	OrgName   sql.NullString `db:"org_name"`
}

type PrintScoresheetEntry struct {
	ParticipantUUID string         `db:"participant_uuid"`
	TargetNo        int            `db:"target_no"`
	TargetName      string         `db:"target_name"`
	ArcherName      string         `db:"archer_name"`
	ArcherCode      sql.NullString `db:"archer_code"`
	ClubName        string         `db:"club_name"`
	CategoryName    string         `db:"category_name"`
	DivisionName    string         `db:"division_name"`
	AgeGroup        string         `db:"age_group"`
	Gender          string         `db:"gender"`
	SessionCode     string         `db:"session_code"`
	SessionName     string         `db:"session_name"`
	BirthDate       sql.NullTime   `db:"birth_date"`
	Email           sql.NullString `db:"email"`
}

type PrintParticipantRow struct {
	AthleteName     string         `db:"athlete_name"`
	AthleteCode     sql.NullString `db:"athlete_code"`
	ClubName        string         `db:"club_name"`
	AgeGroup        string         `db:"age_group"`
	DivisionName    string         `db:"division_name"`
	CategoryName    string         `db:"category_name"`
	SessionCode     string         `db:"session_code"`
	TargetName      string         `db:"target_name"`
	Gender          string         `db:"gender"`
	ParticipantUUID string         `db:"participant_uuid"`
}

type PrintClassStatRow struct {
	AgeGroup     string `db:"age_group"`
	DivisionName string `db:"division_name"`
	Gender       string `db:"gender"`
	TotalCount   int    `db:"total_count"`
}

type PrintClubStatRow struct {
	ClubName   string `db:"club_name"`
	TotalCount int    `db:"total_count"`
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPER FUNCTIONS
// ─────────────────────────────────────────────────────────────────────────────

func fetchPrintEvent(db *sqlx.DB, eventID string) (*PrintEventInfo, error) {
	var ev PrintEventInfo
	query := `
		SELECT e.uuid, e.name, e.slug, e.venue, e.location, e.city, e.start_date, e.end_date, e.logo_url,
		       COALESCE(o.name, '') AS org_name
		FROM tournaments e
		LEFT JOIN organizers o ON e.organizer_id = o.uuid
		WHERE e.uuid = ? OR e.slug = ?
		LIMIT 1
	`
	err := db.Get(&ev, query, eventID, eventID)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func formatPrintDateRange(start, end sql.NullTime) string {
	months := map[int]string{
		1: "Januari", 2: "Februari", 3: "Maret", 4: "April", 5: "Mei", 6: "Juni",
		7: "Juli", 8: "Agustus", 9: "September", 10: "Oktober", 11: "November", 12: "Desember",
	}
	if !start.Valid {
		return "-"
	}
	st := start.Time
	if !end.Valid || end.Time.Equal(st) {
		return fmt.Sprintf("%d %s %d", st.Day(), months[int(st.Month())], st.Year())
	}
	et := end.Time
	if st.Month() == et.Month() && st.Year() == et.Year() {
		return fmt.Sprintf("%d - %d %s %d", st.Day(), et.Day(), months[int(st.Month())], st.Year())
	}
	return fmt.Sprintf("%d %s %d - %d %s %d", st.Day(), months[int(st.Month())], st.Year(), et.Day(), months[int(et.Month())], et.Year())
}

func printPdfHeader(pdf *gofpdf.Fpdf, ev *PrintEventInfo, docTitle string) {
	pdf.SetFillColor(15, 23, 42) // Navy #0f172a
	pdf.Rect(0, 0, 210, 24, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(10, 6)
	pdf.CellFormat(120, 6, ev.Name, "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetXY(130, 6)
	pdf.CellFormat(70, 6, docTitle, "", 0, "R", false, 0, "")

	pdf.SetTextColor(203, 213, 225) // Slate-300
	pdf.SetFont("Arial", "", 8)
	loc := ev.Venue.String
	if ev.City.Valid && ev.City.String != "" {
		if loc != "" {
			loc += ", "
		}
		loc += ev.City.String
	}
	dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)
	pdf.SetXY(10, 14)
	pdf.CellFormat(190, 5, fmt.Sprintf("%s | %s", loc, dateStr), "", 0, "L", false, 0, "")

	pdf.SetY(30)
	pdf.SetTextColor(15, 23, 42)
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. SCORESHEET PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetQualificationScoresheet(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		sessionCode := c.Param("sessionCode")

		categoryID := c.Query("category_id")
		targetFromStr := c.Query("target_from")
		targetToStr := c.Query("target_to")
		blankMode := c.Query("blank") == "1"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT 
				ep.uuid AS participant_uuid,
				COALESCE(et.board_number, 0) AS target_no,
				COALESCE(et.target_name, ep.target_name, '1A') AS target_name,
				a.full_name AS archer_name,
				ep.back_number AS archer_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(rbt.name, '-') AS division_name,
				COALESCE(rag.name, '-') AS age_group,
				COALESCE(rgd.name, '-') AS gender,
				COALESCE(qs.session_code, '1') AS session_code,
				COALESCE(qs.name, 'Sesi 1') AS session_name,
				a.birth_date,
				a.email
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN tournament_targets et ON qta.target_uuid = et.uuid
			WHERE ep.tournament_id = ?
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if sessionCode != "" && sessionCode != "all" {
			query += " AND (qs.session_code = ? OR qs.uuid = ?)"
			args = append(args, sessionCode, sessionCode)
		}

		if categoryID != "" {
			query += " AND (ep.category_id = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}

		if targetFromStr != "" {
			if tf, err := strconv.Atoi(targetFromStr); err == nil {
				query += " AND et.board_number >= ?"
				args = append(args, tf)
			}
		}

		if targetToStr != "" {
			if tt, err := strconv.Atoi(targetToStr); err == nil {
				query += " AND et.board_number <= ?"
				args = append(args, tt)
			}
		}

		query += " ORDER BY qs.session_code ASC, et.board_number ASC, ep.target_name ASC, a.full_name ASC"

		var entries []PrintScoresheetEntry
		err = db.Select(&entries, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data scoresheet: " + err.Error()})
			return
		}

		if len(entries) == 0 && blankMode {
			entries = append(entries, PrintScoresheetEntry{
				TargetName:   "1A",
				ArcherName:   "........................................",
				ClubName:     "-",
				CategoryName: "-",
				SessionName:  "Sesi 1",
				SessionCode:  "1",
			})
		}

		pdf := gofpdf.New("P", "mm", "A4", "")

		for _, e := range entries {
			pdf.AddPage()

			// Header Banner
			printPdfHeader(pdf, ev, "LEMBAR SKOR KUALIFIKASI")

			// Athlete info Card
			pdf.SetFillColor(248, 250, 252)
			pdf.SetDrawColor(226, 232, 240)
			pdf.RoundedRect(10, 28, 190, 22, 2, "1234", "FD")

			// Target Badge (Right)
			pdf.SetFillColor(15, 23, 42)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Arial", "B", 16)
			pdf.SetXY(168, 30)
			pdf.CellFormat(30, 18, e.TargetName, "", 0, "C", true, 0, "")

			// Info Text (Left)
			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(14, 31)
			pdf.CellFormat(25, 5, "NAMA ATLET:", "", 0, "L", false, 0, "")
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 10)
			archerName := e.ArcherName
			if blankMode {
				archerName = "...................................................."
			}
			pdf.CellFormat(80, 5, archerName, "", 0, "L", false, 0, "")

			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(20, 5, "SESI:", "", 0, "L", false, 0, "")
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(30, 5, fmt.Sprintf("%s (%s)", e.SessionName, e.SessionCode), "", 1, "L", false, 0, "")

			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "B", 8)
			pdf.SetX(14)
			pdf.CellFormat(25, 5, "KATEGORI:", "", 0, "L", false, 0, "")
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(80, 5, e.CategoryName, "", 0, "L", false, 0, "")

			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(20, 5, "KLUB:", "", 0, "L", false, 0, "")
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(30, 5, e.ClubName, "", 1, "L", false, 0, "")

			// Scoresheet Table (6 Ends, 6 Arrows)
			pdf.SetY(54)
			pdf.SetX(10)
			pdf.SetFillColor(241, 245, 249)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetDrawColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 8)

			pdf.CellFormat(14, 8, "End", "1", 0, "C", true, 0, "")
			pdf.CellFormat(16, 8, "1", "1", 0, "C", true, 0, "")
			pdf.CellFormat(16, 8, "2", "1", 0, "C", true, 0, "")
			pdf.CellFormat(16, 8, "3", "1", 0, "C", true, 0, "")
			pdf.CellFormat(16, 8, "4", "1", 0, "C", true, 0, "")
			pdf.CellFormat(16, 8, "5", "1", 0, "C", true, 0, "")
			pdf.CellFormat(16, 8, "6", "1", 0, "C", true, 0, "")
			pdf.CellFormat(26, 8, "End Total", "1", 0, "C", true, 0, "")
			pdf.CellFormat(28, 8, "Running Total", "1", 0, "C", true, 0, "")
			pdf.CellFormat(13, 8, "10+X", "1", 0, "C", true, 0, "")
			pdf.CellFormat(13, 8, "X", "1", 1, "C", true, 0, "")

			pdf.SetFont("Arial", "B", 9)
			for endNum := 1; endNum <= 6; endNum++ {
				pdf.SetX(10)
				pdf.SetFillColor(248, 250, 252)
				pdf.CellFormat(14, 18, fmt.Sprintf("%d", endNum), "1", 0, "C", true, 0, "")
				pdf.CellFormat(16, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(16, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(16, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(16, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(16, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(16, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(26, 18, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(28, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(13, 18, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(13, 18, "", "1", 1, "C", false, 0, "")
			}

			// Total Row
			pdf.SetX(10)
			pdf.SetFillColor(241, 245, 249)
			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(110, 10, " TOTAL SKOR BABAK INI", "1", 0, "R", true, 0, "")
			pdf.CellFormat(26, 10, "", "1", 0, "C", true, 0, "")
			pdf.CellFormat(28, 10, "", "1", 0, "C", true, 0, "")
			pdf.CellFormat(13, 10, "", "1", 0, "C", true, 0, "")
			pdf.CellFormat(13, 10, "", "1", 1, "C", true, 0, "")

			// Signatures Block
			pdf.SetY(190)
			pdf.SetDrawColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(15, 23, 42)

			pdf.Line(15, 215, 65, 215)
			pdf.SetXY(15, 217)
			pdf.CellFormat(50, 5, "Tanda Tangan Atlet", "", 0, "C", false, 0, "")

			pdf.Line(80, 215, 130, 215)
			pdf.SetXY(80, 217)
			pdf.CellFormat(50, 5, "Pencatat Skor (Scorer)", "", 0, "C", false, 0, "")

			pdf.Line(145, 215, 195, 215)
			pdf.SetXY(145, 217)
			pdf.CellFormat(50, 5, "Wasit / Bantalan (Judge)", "", 1, "C", false, 0, "")
		}

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=Scoresheet-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output scoresheet PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. PARTICIPANT LIST PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetEventParticipantList(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		listType := c.DefaultQuery("type", "alphabetical")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				a.full_name AS athlete_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(rag.name, '-') AS age_group,
				COALESCE(rbt.name, '-') AS division_name,
				COALESCE(rgd.name, '-') AS gender,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(qs.session_code, '-') AS session_code,
				COALESCE(et.target_name, ep.target_name, '-') AS target_name,
				ep.uuid AS participant_uuid
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN tournament_targets et ON qta.target_uuid = et.uuid
			WHERE ep.tournament_id = ?
		`

		if listType == "by-club" {
			query += " ORDER BY cl.name ASC, a.full_name ASC"
		} else if listType == "by-category" {
			query += " ORDER BY category_name ASC, et.board_number ASC, ep.target_name ASC, a.full_name ASC"
		} else {
			query += " ORDER BY a.full_name ASC"
		}

		var participants []PrintParticipantRow
		err = db.Select(&participants, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar peserta: " + err.Error()})
			return
		}

		docTitle := "Daftar Peserta (Abjad A-Z)"
		if listType == "by-club" {
			docTitle = "Daftar Peserta (Per Klub)"
		} else if listType == "by-category" {
			docTitle = "Daftar Peserta (Per Kategori)"
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		printPdfHeader(pdf, ev, docTitle)

		pdf.SetFillColor(241, 245, 249)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetFont("Arial", "B", 8)

		pdf.CellFormat(10, 7, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(16, 7, "Target", "1", 0, "C", true, 0, "")
		pdf.CellFormat(18, 7, "No Dada", "1", 0, "C", true, 0, "")
		pdf.CellFormat(55, 7, "Nama Atlet", "1", 0, "L", true, 0, "")
		pdf.CellFormat(45, 7, "Klub / Kontingen", "1", 0, "L", true, 0, "")
		pdf.CellFormat(46, 7, "Kategori Lomba", "1", 1, "L", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		for i, p := range participants {
			if pdf.GetY() > 270 {
				pdf.AddPage()
				printPdfHeader(pdf, ev, docTitle)
				pdf.SetFillColor(241, 245, 249)
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(10, 7, "No", "1", 0, "C", true, 0, "")
				pdf.CellFormat(16, 7, "Target", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 7, "No Dada", "1", 0, "C", true, 0, "")
				pdf.CellFormat(55, 7, "Nama Atlet", "1", 0, "L", true, 0, "")
				pdf.CellFormat(45, 7, "Klub / Kontingen", "1", 0, "L", true, 0, "")
				pdf.CellFormat(46, 7, "Kategori Lomba", "1", 1, "L", true, 0, "")
				pdf.SetFont("Arial", "", 8)
			}

			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			}
			pdf.CellFormat(10, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(16, 6, p.TargetName, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(18, 6, p.AthleteCode.String, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(55, 6, p.AthleteName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(45, 6, p.ClubName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(46, 6, p.CategoryName, "1", 1, "L", fill, 0, "")
		}

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=Participants-%s-%s.pdf", listType, ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output participants PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. STATISTICS CLASSES PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetEventStatisticsClasses(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				COALESCE(rag.name, 'Umum') AS age_group,
				COALESCE(rbt.name, 'Standar') AS division_name,
				COALESCE(rgd.name, 'Campuran') AS gender,
				COUNT(ep.uuid) AS total_count
			FROM tournament_participants ep
			JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ep.tournament_id = ?
			GROUP BY rag.name, rbt.name, rgd.name
			ORDER BY division_name ASC, age_group ASC, gender ASC
		`

		var stats []PrintClassStatRow
		err = db.Select(&stats, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik kelas: " + err.Error()})
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		printPdfHeader(pdf, ev, "STATISTIK KELAS & DIVISI")

		pdf.SetFillColor(241, 245, 249)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetFont("Arial", "B", 9)

		pdf.CellFormat(15, 8, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(60, 8, "Divisi Busur", "1", 0, "L", true, 0, "")
		pdf.CellFormat(55, 8, "Kelas Usia", "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 8, "Gender", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 8, "Peserta", "1", 1, "R", true, 0, "")

		pdf.SetFont("Arial", "", 9)
		grandTotal := 0
		for i, s := range stats {
			grandTotal += s.TotalCount
			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			}
			pdf.CellFormat(15, 7, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(60, 7, s.DivisionName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(55, 7, s.AgeGroup, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(30, 7, s.Gender, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(30, 7, fmt.Sprintf("%d", s.TotalCount), "1", 1, "R", fill, 0, "")
		}

		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(15, 23, 42)
		pdf.SetTextColor(255, 255, 255)
		pdf.CellFormat(160, 9, " TOTAL KESELURUHAN PESERTA ", "1", 0, "R", true, 0, "")
		pdf.CellFormat(30, 9, fmt.Sprintf("%d", grandTotal), "1", 1, "R", true, 0, "")

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=Statistics-Classes-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output statistics PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. STATISTICS CLUBS PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetEventStatisticsClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COUNT(ep.uuid) AS total_count
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			WHERE ep.tournament_id = ?
			GROUP BY cl.name
			ORDER BY total_count DESC, cl.name ASC
		`

		var stats []PrintClubStatRow
		err = db.Select(&stats, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik klub: " + err.Error()})
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		printPdfHeader(pdf, ev, "STATISTIK KLUB & KONTINGEN")

		pdf.SetFillColor(241, 245, 249)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetFont("Arial", "B", 9)

		pdf.CellFormat(15, 8, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(135, 8, "Nama Klub / Kontingen", "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 8, "Jumlah Atlet", "1", 1, "R", true, 0, "")

		pdf.SetFont("Arial", "", 9)
		grandTotal := 0
		for i, s := range stats {
			grandTotal += s.TotalCount
			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			}
			pdf.CellFormat(15, 7, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(135, 7, s.ClubName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(40, 7, fmt.Sprintf("%d", s.TotalCount), "1", 1, "R", fill, 0, "")
		}

		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(15, 23, 42)
		pdf.SetTextColor(255, 255, 255)
		pdf.CellFormat(150, 9, " TOTAL ATLET TERDAFTAR ", "1", 0, "R", true, 0, "")
		pdf.CellFormat(40, 9, fmt.Sprintf("%d", grandTotal), "1", 1, "R", true, 0, "")

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=Statistics-Clubs-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output statistics clubs PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. QUALIFICATION START LIST PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetQualificationStartListPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		sessionCode := c.Query("session")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				COALESCE(qs.session_code, '1') AS session_code,
				COALESCE(qs.name, 'Sesi 1') AS session_name,
				COALESCE(et.board_number, 0) AS board_number,
				COALESCE(et.target_name, ep.target_name, '-') AS target_name,
				a.full_name AS athlete_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(rgd.name, '-') AS gender
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN tournament_targets et ON qta.target_uuid = et.uuid
			WHERE ep.tournament_id = ?
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if sessionCode != "" && sessionCode != "all" {
			query += " AND (qs.session_code = ? OR qs.session_code IS NULL)"
			args = append(args, sessionCode)
		}

		query += " ORDER BY qs.session_code ASC, et.board_number ASC, ep.target_name ASC, a.full_name ASC"

		type StartListRow struct {
			SessionCode  string         `db:"session_code"`
			SessionName  string         `db:"session_name"`
			BoardNumber  int            `db:"board_number"`
			TargetName   string         `db:"target_name"`
			AthleteName  string         `db:"athlete_name"`
			AthleteCode  sql.NullString `db:"athlete_code"`
			ClubName     string         `db:"club_name"`
			CategoryName string         `db:"category_name"`
			Gender       string         `db:"gender"`
		}

		var rows []StartListRow
		err = db.Select(&rows, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil start list: " + err.Error()})
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		printPdfHeader(pdf, ev, "START LIST BANTALAN")

		currentSession := ""
		for _, r := range rows {
			if r.SessionCode != currentSession {
				currentSession = r.SessionCode
				pdf.SetY(pdf.GetY() + 4)
				pdf.SetFillColor(15, 23, 42)
				pdf.SetTextColor(255, 255, 255)
				pdf.SetFont("Arial", "B", 9)
				pdf.CellFormat(190, 7, fmt.Sprintf(" %s (Sesi %s)", r.SessionName, r.SessionCode), "", 1, "L", true, 0, "")

				pdf.SetFillColor(241, 245, 249)
				pdf.SetTextColor(15, 23, 42)
				pdf.SetDrawColor(203, 213, 225)
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(16, 7, "Target", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 7, "No Dada", "1", 0, "C", true, 0, "")
				pdf.CellFormat(56, 7, "Nama Atlet", "1", 0, "L", true, 0, "")
				pdf.CellFormat(50, 7, "Klub / Kontingen", "1", 0, "L", true, 0, "")
				pdf.CellFormat(50, 7, "Kategori Lomba", "1", 1, "L", true, 0, "")
			}

			if pdf.GetY() > 270 {
				pdf.AddPage()
				printPdfHeader(pdf, ev, "START LIST BANTALAN")
				pdf.SetFillColor(241, 245, 249)
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(16, 7, "Target", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 7, "No Dada", "1", 0, "C", true, 0, "")
				pdf.CellFormat(56, 7, "Nama Atlet", "1", 0, "L", true, 0, "")
				pdf.CellFormat(50, 7, "Klub / Kontingen", "1", 0, "L", true, 0, "")
				pdf.CellFormat(50, 7, "Kategori Lomba", "1", 1, "L", true, 0, "")
			}

			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(16, 6, r.TargetName, "1", 0, "C", false, 0, "")
			pdf.CellFormat(18, 6, r.AthleteCode.String, "1", 0, "C", false, 0, "")
			pdf.CellFormat(56, 6, r.AthleteName, "1", 0, "L", false, 0, "")
			pdf.CellFormat(50, 6, r.ClubName, "1", 0, "L", false, 0, "")
			pdf.CellFormat(50, 6, r.CategoryName, "1", 1, "L", false, 0, "")
		}

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=StartList-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output start list PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. QUALIFICATION RESULTS PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetQualificationResultsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				ep.category_id,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				a.full_name AS athlete_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(et.target_name, ep.target_name, '-') AS target_name,
				COALESCE(SUM(qes.total_score_end), 0) AS total_score,
				COALESCE(SUM(qes.ten_count_end), 0) AS total_10,
				COALESCE(SUM(qes.x_count_end), 0) AS total_x
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN tournament_targets et ON qta.target_uuid = et.uuid
			LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
			WHERE ep.tournament_id = ?
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if categoryID != "" {
			query += " AND (ep.category_id = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}

		query += " GROUP BY ep.uuid ORDER BY category_name ASC, total_score DESC, total_x DESC, total_10 DESC, a.full_name ASC"

		type ResultRow struct {
			CategoryID   string         `db:"category_id"`
			CategoryName string         `db:"category_name"`
			AthleteName  string         `db:"athlete_name"`
			AthleteCode  sql.NullString `db:"athlete_code"`
			ClubName     string         `db:"club_name"`
			TargetName   string         `db:"target_name"`
			TotalScore   int            `db:"total_score"`
			Total10      int            `db:"total_10"`
			TotalX       int            `db:"total_x"`
		}

		var results []ResultRow
		err = db.Select(&results, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil hasil kualifikasi: " + err.Error()})
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		printPdfHeader(pdf, ev, "HASIL RESMI KUALIFIKASI")

		currentCat := ""
		rank := 0
		for _, r := range results {
			if r.CategoryName != currentCat {
				currentCat = r.CategoryName
				rank = 0
				pdf.SetY(pdf.GetY() + 4)
				pdf.SetFillColor(15, 23, 42)
				pdf.SetTextColor(255, 255, 255)
				pdf.SetFont("Arial", "B", 9)
				pdf.CellFormat(190, 7, fmt.Sprintf(" %s", r.CategoryName), "", 1, "L", true, 0, "")

				pdf.SetFillColor(241, 245, 249)
				pdf.SetTextColor(15, 23, 42)
				pdf.SetDrawColor(203, 213, 225)
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(12, 7, "Pos", "1", 0, "C", true, 0, "")
				pdf.CellFormat(16, 7, "Target", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 7, "No Dada", "1", 0, "C", true, 0, "")
				pdf.CellFormat(54, 7, "Nama Atlet", "1", 0, "L", true, 0, "")
				pdf.CellFormat(44, 7, "Klub / Kontingen", "1", 0, "L", true, 0, "")
				pdf.CellFormat(22, 7, "Total", "1", 0, "C", true, 0, "")
				pdf.CellFormat(12, 7, "10s", "1", 0, "C", true, 0, "")
				pdf.CellFormat(12, 7, "Xs", "1", 1, "C", true, 0, "")
			}

			if pdf.GetY() > 270 {
				pdf.AddPage()
				printPdfHeader(pdf, ev, "HASIL RESMI KUALIFIKASI")
				pdf.SetFillColor(241, 245, 249)
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(12, 7, "Pos", "1", 0, "C", true, 0, "")
				pdf.CellFormat(16, 7, "Target", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 7, "No Dada", "1", 0, "C", true, 0, "")
				pdf.CellFormat(54, 7, "Nama Atlet", "1", 0, "L", true, 0, "")
				pdf.CellFormat(44, 7, "Klub / Kontingen", "1", 0, "L", true, 0, "")
				pdf.CellFormat(22, 7, "Total", "1", 0, "C", true, 0, "")
				pdf.CellFormat(12, 7, "10s", "1", 0, "C", true, 0, "")
				pdf.CellFormat(12, 7, "Xs", "1", 1, "C", true, 0, "")
			}

			rank++
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(12, 6, fmt.Sprintf("%d", rank), "1", 0, "C", false, 0, "")
			pdf.CellFormat(16, 6, r.TargetName, "1", 0, "C", false, 0, "")
			pdf.CellFormat(18, 6, r.AthleteCode.String, "1", 0, "C", false, 0, "")
			pdf.CellFormat(54, 6, r.AthleteName, "1", 0, "L", false, 0, "")
			pdf.CellFormat(44, 6, r.ClubName, "1", 0, "L", false, 0, "")
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(22, 6, fmt.Sprintf("%d", r.TotalScore), "1", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(12, 6, fmt.Sprintf("%d", r.Total10), "1", 0, "C", false, 0, "")
			pdf.CellFormat(12, 6, fmt.Sprintf("%d", r.TotalX), "1", 1, "C", false, 0, "")
		}

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=Results-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output results PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. MEDAL STANDINGS PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetMedalStandingsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type ClubMedalRow struct {
			ClubName string `db:"club_name"`
			Total    int    `db:"total_count"`
		}
		var clubRows []ClubMedalRow
		db.Select(&clubRows, `
			SELECT COALESCE(cl.name, 'Individu') AS club_name, COUNT(ep.uuid) AS total_count
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			WHERE ep.tournament_id = ?
			GROUP BY cl.name ORDER BY total_count DESC LIMIT 25
		`, ev.UUID)

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		printPdfHeader(pdf, ev, "KLASEMEN PEROLEHAN MEDALI")

		pdf.SetFillColor(241, 245, 249)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetFont("Arial", "B", 9)

		pdf.CellFormat(15, 8, "Pos", "1", 0, "C", true, 0, "")
		pdf.CellFormat(85, 8, "Nama Klub / Kontingen", "1", 0, "L", true, 0, "")
		pdf.CellFormat(22, 8, "Emas", "1", 0, "C", true, 0, "")
		pdf.CellFormat(22, 8, "Perak", "1", 0, "C", true, 0, "")
		pdf.CellFormat(22, 8, "Perunggu", "1", 0, "C", true, 0, "")
		pdf.CellFormat(24, 8, "Total", "1", 1, "C", true, 0, "")

		pdf.SetFont("Arial", "", 9)
		for i, cRow := range clubRows {
			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			}
			pdf.CellFormat(15, 7, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(85, 7, cRow.ClubName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(22, 7, "-", "1", 0, "C", fill, 0, "")
			pdf.CellFormat(22, 7, "-", "1", 0, "C", fill, 0, "")
			pdf.CellFormat(22, 7, "-", "1", 0, "C", fill, 0, "")
			pdf.CellFormat(24, 7, "-", "1", 1, "C", fill, 0, "")
		}

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=MedalStandings-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output medal standings PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 8. TARGET LABELS PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetTargetLabelsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		sessionCode := c.Query("session")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				COALESCE(et.target_name, ep.target_name, '1A') AS target_name,
				a.full_name AS athlete_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(qs.session_code, '1') AS session_code
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN tournament_targets et ON qta.target_uuid = et.uuid
			WHERE ep.tournament_id = ?
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if sessionCode != "" && sessionCode != "all" {
			query += " AND (qs.session_code = ? OR qs.session_code IS NULL)"
			args = append(args, sessionCode)
		}

		query += " ORDER BY et.board_number ASC, ep.target_name ASC, a.full_name ASC"

		type LabelRow struct {
			TargetName   string         `db:"target_name"`
			AthleteName  string         `db:"athlete_name"`
			AthleteCode  sql.NullString `db:"athlete_code"`
			ClubName     string         `db:"club_name"`
			CategoryName string         `db:"category_name"`
			SessionCode  string         `db:"session_code"`
		}

		var labels []LabelRow
		err = db.Select(&labels, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data label target: " + err.Error()})
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()

		// Draw sticker grid (2 cols x 7 rows per page)
		cardW := 92.0
		cardH := 36.0
		marginX := 10.0
		marginY := 10.0
		gapX := 6.0
		gapY := 4.0

		for i, l := range labels {
			itemsPerPage := 14
			pageIndex := i % itemsPerPage
			if i > 0 && pageIndex == 0 {
				pdf.AddPage()
			}

			col := pageIndex % 2
			row := pageIndex / 2

			x := marginX + float64(col)*(cardW+gapX)
			y := marginY + float64(row)*(cardH+gapY)

			// Border
			pdf.SetDrawColor(15, 23, 42)
			pdf.SetFillColor(255, 255, 255)
			pdf.RoundedRect(x, y, cardW, cardH, 2, "1234", "D")

			// Target Box
			pdf.SetFillColor(15, 23, 42)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Arial", "B", 18)
			pdf.SetXY(x+3, y+3)
			pdf.CellFormat(22, 30, l.TargetName, "", 0, "C", true, 0, "")

			// Details Text
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 10)
			pdf.SetXY(x+28, y+4)
			pdf.CellFormat(60, 5, l.AthleteName, "", 1, "L", false, 0, "")

			pdf.SetTextColor(71, 85, 105)
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(x+28, y+11)
			pdf.CellFormat(60, 4, l.ClubName, "", 1, "L", false, 0, "")

			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "", 7)
			pdf.SetXY(x+28, y+18)
			pdf.CellFormat(60, 4, l.CategoryName, "", 1, "L", false, 0, "")

			pdf.SetXY(x+28, y+24)
			pdf.CellFormat(60, 4, fmt.Sprintf("Sesi %s - %s", l.SessionCode, ev.Name), "", 1, "L", false, 0, "")
		}

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=TargetLabels-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output target labels PDF"})
		}
	}
}
