package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	BirthDate       sql.NullString `db:"birth_date"`
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

func newOrisPdf(orientation string) *gofpdf.Fpdf {
	if orientation == "" {
		orientation = "P"
	}
	pdf := gofpdf.New(orientation, "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 18)

	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		pdf.SetFont("Arial", "I", 7)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetDrawColor(226, 232, 240)

		if orientation == "L" {
			pdf.Line(10, 196, 287, 196)
			pdf.SetX(10)
			pdf.CellFormat(95, 8, fmt.Sprintf("Report Created: %s UTC+7", time.Now().Format("02 Jan 2006 15:04:05")), "", 0, "L", false, 0, "")
			pdf.CellFormat(87, 8, "World Archery / PERPANI ORIS Timing & Results", "", 0, "C", false, 0, "")
			pdf.CellFormat(95, 8, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "R", false, 0, "")
		} else {
			pdf.Line(10, 283, 200, 283)
			pdf.SetX(10)
			pdf.CellFormat(65, 8, fmt.Sprintf("Report Created: %s UTC+7", time.Now().Format("02 Jan 2006 15:04:05")), "", 0, "L", false, 0, "")
			pdf.CellFormat(60, 8, "World Archery / PERPANI ORIS Timing & Results", "", 0, "C", false, 0, "")
			pdf.CellFormat(65, 8, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "R", false, 0, "")
		}
	})

	return pdf
}

func setPdfHeaders(c *gin.Context, filename string) {
	c.Header("Content-Type", "application/pdf")
	disposition := "attachment"
	if c.Query("inline") == "1" || c.Query("preview") == "1" || c.Query("view") == "1" || gin.Mode() != gin.ReleaseMode {
		disposition = "inline"
	}
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, filename))
}

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
		1: "Jan", 2: "Feb", 3: "Mar", 4: "Apr", 5: "May", 6: "Jun",
		7: "Jul", 8: "Aug", 9: "Sep", 10: "Oct", 11: "Nov", 12: "Dec",
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
		return fmt.Sprintf("%d-%d %s %d", st.Day(), et.Day(), months[int(st.Month())], st.Year())
	}
	if st.Year() == et.Year() {
		return fmt.Sprintf("%d %s - %d %s %d", st.Day(), months[int(st.Month())], et.Day(), months[int(et.Month())], st.Year())
	}
	return fmt.Sprintf("%d %s %d - %d %s %d", st.Day(), months[int(st.Month())], st.Year(), et.Day(), months[int(et.Month())], et.Year())
}

func generateNocCode(clubName string) string {
	clean := strings.ToUpper(strings.TrimSpace(clubName))
	if clean == "" || clean == "INDIVIDU / TANPA KLUB" || clean == "INDIVIDU" {
		return "IND"
	}
	if strings.Contains(clean, "KLATEN") {
		return "KLA"
	}
	if strings.Contains(clean, "SLEMAN") {
		return "SLM"
	}
	if strings.Contains(clean, "BOYOLALI") {
		return "BYL"
	}
	if strings.Contains(clean, "YOGYAKARTA") {
		return "YOG"
	}
	if strings.Contains(clean, "BANTUL") {
		return "BTL"
	}
	if strings.Contains(clean, "KULON PROGO") {
		return "KLP"
	}
	if strings.Contains(clean, "MAGELANG") {
		return "MGL"
	}

	words := strings.Fields(clean)
	var filtered []string
	for _, w := range words {
		if w != "CLUB" && w != "ARCHERY" && w != "TEAM" && w != "PENGKAB" && w != "PENGKOT" && w != "PERPANI" && w != "CENTER" && w != "SQUAD" && len(w) > 0 {
			filtered = append(filtered, w)
		}
	}
	if len(filtered) > 0 {
		first := filtered[0]
		if len(first) >= 3 {
			return first[:3]
		}
		return first
	}
	if len(words) > 0 && len(words[0]) >= 3 {
		return words[0][:3]
	}
	if len(clean) >= 3 {
		return clean[:3]
	}
	return clean
}

func formatDOB(raw string) string {
	if raw == "" || raw == "0000-00-00" {
		return ""
	}
	datePart := strings.Split(raw, "T")[0]
	t, err := time.Parse("2006-01-02", datePart)
	if err == nil {
		return t.Format("02 Jan 2006")
	}
	return datePart
}

func setupOrisPdf(docCode string) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 15, 10)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AliasNbPages("{nb}")

	pdf.SetFooterFunc(func() {
		curY := 282.0
		pdf.SetLineWidth(0.3)
		pdf.SetDrawColor(0, 0, 0)
		pdf.Line(5, curY, 205, curY)
		pdf.SetLineWidth(0.1)

		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(0, 0, 0)

		prefix := "AR_"
		if strings.HasPrefix(docCode, "AR_") {
			prefix = docCode
		} else if docCode != "" {
			prefix = "AR_" + docCode
		}
		pdf.SetXY(10, curY+1)
		pdf.CellFormat(60, 4, prefix, "", 0, "L", false, 0, "")

		now := time.Now().Format("02 Jan 2006 15:04")
		pdf.SetXY(75, curY+1)
		pdf.CellFormat(60, 4, fmt.Sprintf("Report Created: %s @ UTC+07:00", now), "", 0, "C", false, 0, "")

		pdf.SetXY(140, curY+1)
		pdf.CellFormat(60, 4, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	return pdf
}

func printPdfHeader(pdf *gofpdf.Fpdf, ev *PrintEventInfo, docTitle string) {
	printOrisPdfHeaderDetailed(pdf, ev, "", docTitle, "", "")
}

func printOrisPdfHeader(pdf *gofpdf.Fpdf, ev *PrintEventInfo, docCode, docTitle string) {
	printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, "", "")
}

func printOrisPdfHeaderDetailed(pdf *gofpdf.Fpdf, ev *PrintEventInfo, docCode, docTitle, eventName, phaseName string) {
	pdf.SetMargins(10, 15, 10)
	pdf.SetAutoPageBreak(true, 16)

	loc := ""
	if ev.City.Valid && ev.City.String != "" {
		loc = ev.City.String
		if ev.Location.Valid && ev.Location.String != "" && !strings.Contains(strings.ToLower(ev.Location.String), strings.ToLower(ev.City.String)) {
			loc += ", " + ev.Location.String
		}
	} else if ev.Venue.Valid && ev.Venue.String != "" {
		loc = ev.Venue.String
	} else if ev.Location.Valid && ev.Location.String != "" {
		loc = ev.Location.String
	}
	if loc == "" {
		loc = "Indonesia"
	}

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 10)
	pdf.SetXY(10, 5)
	pdf.MultiCell(40, 5, loc, "", "L", false)

	pdf.SetXY(10, 20)
	dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)
	pdf.MultiCell(40, 5, dateStr, "", "L", false)

	pdf.SetFont("Arial", "B", 11)
	pdf.SetXY(50, 5)
	pdf.CellFormat(150, 5, ev.Name, "", 0, "L", false, 0, "")

	if eventName != "" {
		pdf.SetXY(50, 12.5)
		pdf.CellFormat(150, 5, eventName, "", 0, "L", false, 0, "")
	}

	if phaseName != "" {
		pdf.SetXY(50, 19.5)
		pdf.CellFormat(150, 5, phaseName, "", 0, "L", false, 0, "")
	}

	pdf.SetLineWidth(0.3)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(5, 30, 205, 30)
	pdf.SetLineWidth(0.1)

	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(10, 30)
	pdf.CellFormat(190, 7, strings.ToUpper(docTitle), "", 1, "C", false, 0, "")

	pdf.SetY(41)
}

func printOrisSignatures(pdf *gofpdf.Fpdf, yPos float64) {
	if yPos > 240 {
		pdf.AddPage()
		yPos = 35
	}
	pdf.SetY(yPos + 8)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(0, 0, 0)

	// Left: Technical Delegate
	pdf.Line(25, yPos+28, 85, yPos+28)
	pdf.SetXY(25, yPos+29)
	pdf.CellFormat(60, 4, "Technical Delegate", "", 0, "C", false, 0, "")

	// Right: Chief Judge
	pdf.Line(125, yPos+28, 185, yPos+28)
	pdf.SetXY(125, yPos+29)
	pdf.CellFormat(60, 4, "Chief Judge / Ketua Wasit", "", 1, "C", false, 0, "")
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. SCORESHEET PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func GetQualificationScoresheet(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		sessionCode := c.Param("sessionCode")
		if sessionCode == "" {
			sessionCode = c.Param("session")
		}
		if sessionCode == "" {
			sessionCode = c.Query("session")
		}
		if sessionCode == "" {
			sessionCode = c.Query("session_code")
		}

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
		pdf.SetAutoPageBreak(false, 0)

		loc := ev.Venue.String
		if ev.City.Valid && ev.City.String != "" {
			if loc != "" {
				loc += ", "
			}
			loc += ev.City.String
		}
		dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)

		renderHalfSheet := func(y0 float64, e PrintScoresheetEntry) {
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)

			// 1. Header (Tournament & Category)
			pdf.SetFont("Arial", "B", 9.5)
			pdf.SetXY(10, y0)
			pdf.CellFormat(115, 4.5, ev.Name, "", 0, "L", false, 0, "")

			pdf.SetFont("Arial", "B", 9.5)
			pdf.SetXY(125, y0)
			catText := e.CategoryName
			if catText == "" || catText == "-" {
				catText = "Qualification Round"
			}
			pdf.CellFormat(75, 4.5, catText, "", 1, "R", false, 0, "")

			pdf.SetFont("Arial", "", 7.5)
			pdf.SetTextColor(80, 80, 80)
			pdf.SetXY(10, y0+4.5)
			pdf.CellFormat(115, 3.5, fmt.Sprintf("%s | %s", loc, dateStr), "", 0, "L", false, 0, "")

			pdf.SetXY(125, y0+4.5)
			sessLabel := fmt.Sprintf("Session %s", e.SessionCode)
			if e.SessionName != "" && e.SessionName != "Sesi "+e.SessionCode {
				sessLabel = fmt.Sprintf("%s (%s)", e.SessionName, e.SessionCode)
			}
			pdf.CellFormat(75, 3.5, fmt.Sprintf("%s | Qualification Scoresheet", sessLabel), "", 1, "R", false, 0, "")

			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.2)
			pdf.Line(10, y0+9, 200, y0+9)

			// 2. Athlete Card Info
			pdf.SetTextColor(0, 0, 0)
			pdf.SetLineWidth(0.15)
			pdf.Rect(10, y0+10.5, 18, 11, "D")
			pdf.SetFont("Arial", "B", 14)
			pdf.SetXY(10, y0+10.5)
			tgt := e.TargetName
			if tgt == "" || tgt == "-" {
				tgt = ""
			}
			pdf.CellFormat(18, 11, tgt, "", 0, "C", false, 0, "")

			archerName := strings.ToUpper(e.ArcherName)
			if blankMode || archerName == "" {
				archerName = "...................................................."
			}
			noc := generateNocCode(e.ClubName)
			clubDisplay := e.ClubName
			if clubDisplay == "" || clubDisplay == "-" {
				clubDisplay = "Individu"
			}

			pdf.SetFont("Arial", "B", 9.5)
			pdf.SetXY(30, y0+10.5)
			pdf.CellFormat(122, 4, archerName, "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetXY(30, y0+14.5)
			pdf.CellFormat(122, 3.5, fmt.Sprintf("[%s] %s", noc, clubDisplay), "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "I", 7.5)
			pdf.SetTextColor(70, 70, 70)
			pdf.SetXY(30, y0+18)
			pdf.CellFormat(122, 3.5, e.CategoryName, "", 1, "L", false, 0, "")

			// Right Info Box
			pdf.SetTextColor(0, 0, 0)
			pdf.Rect(155, y0+10.5, 45, 11, "D")
			pdf.SetFont("Arial", "B", 7.5)
			pdf.SetXY(156, y0+11.5)
			pdf.CellFormat(43, 3.5, fmt.Sprintf("Target: %s", tgt), "", 1, "L", false, 0, "")
			pdf.SetFont("Arial", "", 7.5)
			pdf.SetXY(156, y0+16)
			pdf.CellFormat(43, 3.5, fmt.Sprintf("Session: %s", e.SessionCode), "", 1, "L", false, 0, "")

			// 3. Side-by-Side 2-Halves Scoring Grid (World Archery / IanSeo Standard)
			yGrid := y0 + 23.5
			renderDistanceGrid := func(xOff float64, halfTitle string) {
				pdf.SetLineWidth(0.1)
				pdf.SetFillColor(245, 245, 245)
				pdf.SetFont("Arial", "B", 7)
				pdf.SetXY(xOff, yGrid)
				pdf.CellFormat(7, 5, "End", "1", 0, "C", true, 0, "")
				pdf.CellFormat(9, 5, "1", "1", 0, "C", true, 0, "")
				pdf.CellFormat(9, 5, "2", "1", 0, "C", true, 0, "")
				pdf.CellFormat(9, 5, "3", "1", 0, "C", true, 0, "")
				pdf.CellFormat(15, 5, "Total", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 5, "R. Total", "1", 0, "C", true, 0, "")
				pdf.CellFormat(13, 5, "10+X", "1", 0, "C", true, 0, "")
				pdf.CellFormat(13, 5, "X", "1", 1, "C", true, 0, "")

				rowH := 6.5
				pdf.SetFont("Arial", "B", 8)
				for end := 1; end <= 6; end++ {
					pdf.SetXY(xOff, yGrid+5+float64(end-1)*rowH)
					pdf.SetFillColor(250, 250, 250)
					pdf.CellFormat(7, rowH, fmt.Sprintf("%d", end), "1", 0, "C", true, 0, "")
					pdf.CellFormat(9, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(9, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(9, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(15, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(18, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(13, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(13, rowH, "", "1", 1, "C", false, 0, "")
				}

				// Subtotal Row
				pdf.SetXY(xOff, yGrid+5+6*rowH)
				pdf.SetFillColor(240, 240, 240)
				pdf.SetFont("Arial", "B", 7.5)
				pdf.CellFormat(34, 5.5, halfTitle, "1", 0, "R", true, 0, "")
				pdf.CellFormat(15, 5.5, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 5.5, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(13, 5.5, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(13, 5.5, "", "1", 1, "C", true, 0, "")
			}

			// Left: 1st Half / Distance 1 (X: 10 to 103)
			renderDistanceGrid(10, "1st Half Total ")
			// Right: 2nd Half / Distance 2 (X: 107 to 200)
			renderDistanceGrid(107, "2nd Half Total ")

			// 4. Grand Total Summary Bar
			yTot := yGrid + 5 + 6*6.5 + 6.5
			pdf.SetXY(10, yTot)
			pdf.SetFillColor(245, 245, 245)
			pdf.SetFont("Arial", "B", 7.5)
			pdf.CellFormat(42, 6.5, " 1st Half: _____________", "1", 0, "L", true, 0, "")
			pdf.CellFormat(42, 6.5, " 2nd Half: _____________", "1", 0, "L", true, 0, "")
			pdf.CellFormat(48, 6.5, " Total 10+X: ______   Total X: ______", "1", 0, "C", true, 0, "")
			pdf.CellFormat(58, 6.5, " FINAL TOTAL: _____________", "1", 1, "C", true, 0, "")

			// 5. Signatures Block
			ySig := yTot + 9.5
			pdf.SetFont("Arial", "", 7.5)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.15)

			pdf.Line(15, ySig+5, 65, ySig+5)
			pdf.SetXY(15, ySig+6)
			pdf.CellFormat(50, 3.5, "Archer's Signature", "", 0, "C", false, 0, "")

			pdf.Line(80, ySig+5, 130, ySig+5)
			pdf.SetXY(80, ySig+6)
			pdf.CellFormat(50, 3.5, "Scorer's Signature", "", 0, "C", false, 0, "")

			pdf.Line(145, ySig+5, 195, ySig+5)
			pdf.SetXY(145, ySig+6)
			pdf.CellFormat(50, 3.5, "Target Captain / Judge", "", 1, "C", false, 0, "")

			pdf.SetFont("Arial", "I", 6)
			pdf.SetTextColor(110, 110, 110)
			pdf.SetXY(10, ySig+10.5)
			pdf.CellFormat(190, 3, "The signatures certify the correctness of the arrow scores, 10s and Xs in accordance with World Archery Rules.", "", 1, "C", false, 0, "")
		}

		// Loop entries by 2 (2 scoresheets per A4 page)
		totalCount := len(entries)
		for i := 0; i < totalCount; i += 2 {
			pdf.AddPage()

			// Top half scoresheet
			renderHalfSheet(8.0, entries[i])

			// Center dashed cut-line
			pdf.SetDrawColor(180, 180, 180)
			pdf.SetLineWidth(0.15)
			pdf.SetFont("Arial", "I", 6.5)
			pdf.SetTextColor(130, 130, 130)
			pdf.SetXY(10, 146.5)
			pdf.CellFormat(190, 3, "- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - Cut Here - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -", "", 0, "C", false, 0, "")

			// Bottom half scoresheet
			if i+1 < totalCount {
				renderHalfSheet(152.0, entries[i+1])
			} else if blankMode {
				blankEntry := PrintScoresheetEntry{
					TargetName:   entries[i].TargetName,
					ArcherName:   "........................................",
					ClubName:     "-",
					CategoryName: entries[i].CategoryName,
					SessionName:  entries[i].SessionName,
					SessionCode:  entries[i].SessionCode,
				}
				renderHalfSheet(152.0, blankEntry)
			}
		}

		setPdfHeaders(c, fmt.Sprintf("Scoresheet-%s.pdf", ev.Slug))
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
				COALESCE(CAST(a.date_of_birth AS CHAR), CAST(a.birth_date AS CHAR), '') AS birth_date,
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
			query += " ORDER BY category_name ASC, cl.name ASC, a.full_name ASC"
		} else {
			query += " ORDER BY a.full_name ASC"
		}

		var participants []PrintParticipantRow
		err = db.Select(&participants, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar peserta: " + err.Error()})
			return
		}

		if listType == "by-club" {
			GetEntriesByClubPrintout(db)(c)
			return
		}

		if listType == "by-category" {
			// [C32A] ENTRIES BY EVENT
			docCode := "C32A"
			docTitle := "ENTRIES BY EVENT"
			pdf := setupOrisPdf(docCode)

			renderHeader := func(y float64) {
				pdf.SetLineWidth(0.1)
				pdf.SetDrawColor(0, 0, 0)
				pdf.Rect(10, y-1, 190, 5.5, "D")
				pdf.SetFont("Arial", "B", 8)
				pdf.SetTextColor(0, 0, 0)
				pdf.SetXY(10, y)
				pdf.CellFormat(15, 3.5, "NOC", "", 0, "L", false, 0, "")
				pdf.CellFormat(40, 3.5, "Country", "", 0, "L", false, 0, "")
				pdf.CellFormat(15, 3.5, "Back No.  ", "", 0, "R", false, 0, "")
				pdf.CellFormat(15, 3.5, "W. Rank   ", "", 0, "R", false, 0, "")
				pdf.CellFormat(25, 3.5, "Date of Birth   ", "", 0, "R", false, 0, "")
				pdf.CellFormat(80, 3.5, "Name", "", 1, "L", false, 0, "")
				pdf.SetY(y + 5.5)
			}

			currentCat := ""
			lastClub := ""
			for _, p := range participants {
				if currentCat != p.CategoryName {
					currentCat = p.CategoryName
					lastClub = ""
					pdf.AddPage()
					printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, currentCat, "Entries")
					renderHeader(41.0)
				}

				if pdf.GetY() > 265 {
					pdf.AddPage()
					printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, currentCat, "Entries")
					renderHeader(41.0)
				}

				noc := generateNocCode(p.ClubName)
				dobStr := formatDOB(p.BirthDate.String)
				targetNo := strings.TrimLeft(strings.TrimSpace(p.TargetName), "0")
				if targetNo == "-" {
					targetNo = ""
				}

				if lastClub != p.ClubName {
					pdf.SetY(pdf.GetY() + 3.5)
					lastClub = p.ClubName

					pdf.SetFont("Arial", "", 8)
					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(10, pdf.GetY())
					pdf.CellFormat(15, 3.5, noc, "", 0, "L", false, 0, "")
					pdf.CellFormat(40, 3.5, p.ClubName, "", 0, "L", false, 0, "")
				} else {
					pdf.SetFont("Arial", "", 8)
					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(10, pdf.GetY())
					pdf.CellFormat(15, 3.5, "", "", 0, "L", false, 0, "")
					pdf.CellFormat(40, 3.5, "", "", 0, "L", false, 0, "")
				}

				pdf.CellFormat(15, 3.5, targetNo+"  ", "", 0, "R", false, 0, "")
				pdf.CellFormat(15, 3.5, "", "", 0, "R", false, 0, "")
				pdf.CellFormat(25, 3.5, dobStr+"   ", "", 0, "R", false, 0, "")
				pdf.CellFormat(80, 3.5, strings.ToUpper(p.AthleteName), "", 1, "L", false, 0, "")
			}

			setPdfHeaders(c, fmt.Sprintf("%s-entries-by-event-C32A.pdf", ev.Slug))
			_ = pdf.Output(c.Writer)
			return
		}

		// Default: Alphabetical [C32B] ENTRIES BY NAME
		docCode := "C32B"
		docTitle := "ENTRIES BY NAME"
		pdf := setupOrisPdf(docCode)

		renderHeader := func(y float64) {
			pdf.SetLineWidth(0.1)
			pdf.SetDrawColor(0, 0, 0)
			pdf.Rect(10, y-1, 190, 5.5, "D")
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(10, y)
			pdf.CellFormat(45, 3.5, "Name", "", 0, "L", false, 0, "")
			pdf.CellFormat(10, 3.5, "NOC", "", 0, "L", false, 0, "")
			pdf.CellFormat(40, 3.5, "Country", "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "W. Rank   ", "", 0, "R", false, 0, "")
			pdf.CellFormat(25, 3.5, "Date of Birth   ", "", 0, "R", false, 0, "")
			pdf.CellFormat(15, 3.5, "Back No.  ", "", 0, "R", false, 0, "")
			pdf.CellFormat(40, 3.5, "Event", "", 1, "L", false, 0, "")
			pdf.SetY(y + 5.5)
		}

		pdf.AddPage()
		printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, "", "Entries")
		renderHeader(41.0)

		lastLetter := ""
		for _, p := range participants {
			if pdf.GetY() > 265 {
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, "", "Entries")
				renderHeader(41.0)
			}

			firstChar := ""
			trimmed := strings.TrimSpace(p.AthleteName)
			if len(trimmed) > 0 {
				firstChar = strings.ToUpper(string(trimmed[0]))
			}

			if lastLetter != "" && lastLetter != firstChar {
				pdf.SetY(pdf.GetY() + 3.5)
			}
			lastLetter = firstChar

			noc := generateNocCode(p.ClubName)
			dobStr := formatDOB(p.BirthDate.String)
			targetNo := strings.TrimLeft(strings.TrimSpace(p.TargetName), "0")
			if targetNo == "-" {
				targetNo = ""
			}

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(10, pdf.GetY())
			pdf.CellFormat(45, 3.5, strings.ToUpper(p.AthleteName), "", 0, "L", false, 0, "")
			pdf.CellFormat(10, 3.5, noc, "", 0, "L", false, 0, "")
			pdf.CellFormat(40, 3.5, p.ClubName, "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "", "", 0, "R", false, 0, "")
			pdf.CellFormat(25, 3.5, dobStr+"   ", "", 0, "R", false, 0, "")
			pdf.CellFormat(15, 3.5, targetNo+"  ", "", 0, "R", false, 0, "")
			pdf.CellFormat(40, 3.5, p.CategoryName, "", 1, "L", false, 0, "")
		}

		setPdfHeaders(c, fmt.Sprintf("%s-entries-by-name-C32B.pdf", ev.Slug))
		_ = pdf.Output(c.Writer)
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

		pdf := setupOrisPdf("C03")
		pdf.AddPage()
		printOrisPdfHeaderDetailed(pdf, ev, "C03", "NUMBER OF ENTRIES BY CLASS/DIVISION", "", "Entries")

		pdf.SetFillColor(245, 245, 245)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetLineWidth(0.1)
		pdf.SetFont("Arial", "B", 8)

		pdf.CellFormat(12, 6.5, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(68, 6.5, "Division / Bow Type", "1", 0, "L", true, 0, "")
		pdf.CellFormat(55, 6.5, "Age Class", "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 6.5, "Gender", "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 6.5, "Athletes", "1", 1, "R", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		grandTotal := 0
		for i, s := range stats {
			grandTotal += s.TotalCount
			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(250, 250, 250)
			}
			pdf.CellFormat(12, 5.5, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(68, 5.5, s.DivisionName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(55, 5.5, s.AgeGroup, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(30, 5.5, s.Gender, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(25, 5.5, fmt.Sprintf("%d", s.TotalCount), "1", 1, "R", fill, 0, "")
		}

		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(240, 240, 240)
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(165, 6.5, " TOTAL COMPETITORS ", "1", 0, "R", true, 0, "")
		pdf.CellFormat(25, 6.5, fmt.Sprintf("%d", grandTotal), "1", 1, "R", true, 0, "")

		setPdfHeaders(c, fmt.Sprintf("%s-statistics-classes-divisions-C03.pdf", ev.Slug))
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

		pdf := setupOrisPdf("C04")
		pdf.AddPage()
		printOrisPdfHeaderDetailed(pdf, ev, "C04", "NUMBER OF ENTRIES BY CLUB/CONTINGENT", "", "Entries")

		pdf.SetFillColor(245, 245, 245)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetLineWidth(0.1)
		pdf.SetFont("Arial", "B", 8)

		pdf.CellFormat(12, 6.5, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(16, 6.5, "Code", "1", 0, "C", true, 0, "")
		pdf.CellFormat(132, 6.5, "Club / Contingent Name", "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 6.5, "Total Athletes", "1", 1, "R", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		grandTotal := 0
		for i, s := range stats {
			grandTotal += s.TotalCount
			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(250, 250, 250)
			}
			noc := generateNocCode(s.ClubName)
			pdf.CellFormat(12, 5.5, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(16, 5.5, noc, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(132, 5.5, s.ClubName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(30, 5.5, fmt.Sprintf("%d", s.TotalCount), "1", 1, "R", fill, 0, "")
		}

		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(240, 240, 240)
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(160, 6.5, fmt.Sprintf(" TOTAL (%d CLUBS) ", len(stats)), "1", 0, "R", true, 0, "")
		pdf.CellFormat(30, 6.5, fmt.Sprintf("%d", grandTotal), "1", 1, "R", true, 0, "")

		setPdfHeaders(c, fmt.Sprintf("%s-statistics-clubs-contingents-C04.pdf", ev.Slug))
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
				COALESCE(CAST(a.date_of_birth AS CHAR), CAST(a.birth_date AS CHAR), '') AS birth_date,
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
			BirthDate    sql.NullString `db:"birth_date"`
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

		pdf := setupOrisPdf("C32C")

		renderTableHeader := func(y float64) {
			pdf.SetLineWidth(0.1)
			pdf.SetDrawColor(0, 0, 0)
			pdf.Rect(10, y-1, 190, 5.5, "D")
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(10, y)
			pdf.CellFormat(15, 3.5, "Target", "", 0, "L", false, 0, "")
			pdf.CellFormat(50, 3.5, "Name", "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "NOC", "", 0, "L", false, 0, "")
			pdf.CellFormat(45, 3.5, "Country", "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "W. Rank   ", "", 0, "R", false, 0, "")
			pdf.CellFormat(50, 3.5, "Date of Birth", "", 1, "L", false, 0, "")
			pdf.SetY(y + 5.5)
		}

		currentEvent := ""
		oldTargetBoard := ""

		for _, r := range rows {
			if r.CategoryName != currentEvent {
				currentEvent = r.CategoryName
				oldTargetBoard = ""
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, "C32C", "START LIST BY TARGET", currentEvent, "Qualification Round")
				renderTableHeader(41.0)
			}

			rawTarget := strings.TrimLeft(strings.TrimSpace(r.TargetName), "0")
			targetBoard := ""
			targetLetter := ""
			for i, ch := range rawTarget {
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					targetBoard = strings.TrimSpace(rawTarget[:i])
					targetLetter = strings.ToUpper(strings.TrimSpace(rawTarget[i:]))
					break
				}
			}
			if targetBoard == "" {
				targetBoard = rawTarget
			}

			if oldTargetBoard != "" && oldTargetBoard != targetBoard {
				if pdf.GetY() > 255 {
					pdf.AddPage()
					printOrisPdfHeaderDetailed(pdf, ev, "C32C", "START LIST BY TARGET", currentEvent, "Qualification Round")
					renderTableHeader(41.0)
				} else {
					pdf.SetY(pdf.GetY() + 3.5)
				}
			}

			if pdf.GetY() > 265 {
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, "C32C", "START LIST BY TARGET", currentEvent, "Qualification Round")
				renderTableHeader(41.0)
			}

			targetCol := ""
			if oldTargetBoard != targetBoard {
				targetCol = fmt.Sprintf("%s %s", targetBoard, targetLetter)
				oldTargetBoard = targetBoard
			} else {
				targetCol = fmt.Sprintf("   %s", targetLetter)
			}

			dobStr := ""
			if r.BirthDate.Valid && r.BirthDate.String != "" && r.BirthDate.String != "0000-00-00" {
				datePart := strings.Split(r.BirthDate.String, "T")[0]
				t, err := time.Parse("2006-01-02", datePart)
				if err == nil {
					dobStr = t.Format("02 Jan 2006")
				} else {
					dobStr = datePart
				}
			}

			noc := generateNocCode(r.ClubName)

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(10, pdf.GetY())
			pdf.CellFormat(15, 3.5, targetCol, "", 0, "L", false, 0, "")
			pdf.CellFormat(50, 3.5, strings.ToUpper(r.AthleteName), "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, noc, "", 0, "L", false, 0, "")
			pdf.CellFormat(45, 3.5, r.ClubName, "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "", "", 0, "R", false, 0, "")
			pdf.CellFormat(50, 3.5, dobStr, "", 1, "L", false, 0, "")
		}

		if len(rows) == 0 {
			pdf.AddPage()
			printOrisPdfHeaderDetailed(pdf, ev, "C32C", "START LIST BY TARGET", "All Categories", "Qualification Round")
			renderTableHeader(41.0)
			pdf.SetFont("Arial", "I", 9)
			pdf.SetXY(10, 55)
			pdf.CellFormat(190, 8, "Belum ada data alokasi bantalan kualifikasi.", "", 1, "C", false, 0, "")
		}

		setPdfHeaders(c, fmt.Sprintf("%s-qualification-start-list-C32C.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output start list PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. QUALIFICATION RESULTS PDF HANDLER (ORIS C73A / C73B)
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
				COALESCE(SUM(CASE WHEN qs.session_code = '1' OR qs.session_code IS NULL THEN qes.total_score_end ELSE 0 END), 0) AS score_s1,
				COALESCE(SUM(CASE WHEN qs.session_code = '2' THEN qes.total_score_end ELSE 0 END), 0) AS score_s2,
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
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
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
			ScoreS1      int            `db:"score_s1"`
			ScoreS2      int            `db:"score_s2"`
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

		pdf := setupOrisPdf("C73A")

		renderResultsHeader := func(y float64) {
			pdf.SetLineWidth(0.1)
			pdf.SetDrawColor(0, 0, 0)
			pdf.Rect(10, y-1, 190, 5.5, "D")
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(10, y)
			pdf.CellFormat(13, 3.5, "Rank", "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "Back No.", "", 0, "R", false, 0, "")
			pdf.CellFormat(70, 3.5, "Name", "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "NOC", "", 0, "L", false, 0, "")
			pdf.CellFormat(15, 3.5, "50m-1", "", 0, "R", false, 0, "")
			pdf.CellFormat(15, 3.5, "50m-2", "", 0, "R", false, 0, "")
			pdf.CellFormat(10, 3.5, "10+X", "", 0, "R", false, 0, "")
			pdf.CellFormat(10, 3.5, "X", "", 0, "R", false, 0, "")
		}

		currentCategory := ""
		rank := 0
		for _, r := range results {
			if r.CategoryName != currentCategory {
				currentCategory = r.CategoryName
				rank = 0
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, "C73A", "RESULTS", currentCategory, "Qualification Round")
				renderResultsHeader(41.0)
			}

			rank++
			if pdf.GetY() > 265 {
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, "C73A", "RESULTS", currentCategory, "Qualification Round")
				renderResultsHeader(41.0)
			}

			athleteDisplay := strings.ToUpper(r.AthleteName)
			if r.AthleteCode.Valid && r.AthleteCode.String != "" {
				athleteDisplay = fmt.Sprintf("%s  (%s)", strings.ToUpper(r.AthleteName), r.AthleteCode.String)
			}

			targetNo := strings.TrimLeft(strings.TrimSpace(r.TargetName), "0")
			if targetNo == "-" {
				targetNo = ""
			}
			noc := generateNocCode(r.ClubName)

			rankStr := fmt.Sprintf("%d", rank)
			scoreStr := fmt.Sprintf("%d", r.TotalScore)
			xCountStr := fmt.Sprintf("%d", r.TotalX)
			tenCountStr := fmt.Sprintf("%d", r.Total10)
			d1Str := fmt.Sprintf("%d", r.ScoreS1)
			d2Str := fmt.Sprintf("%d", r.ScoreS2)
			if d1Str == "0" && d2Str == "0" {
				d1Str = "-"
				d2Str = "-"
			}

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(10, pdf.GetY())

			pdf.CellFormat(10, 3.5, rankStr+" ", "", 0, "R", false, 0, "")
			pdf.CellFormat(12, 3.5, targetNo+"  ", "", 0, "R", false, 0, "")
			pdf.CellFormat(15, 3.5, "", "", 0, "R", false, 0, "")
			pdf.CellFormat(60, 3.5, athleteDisplay, "", 0, "L", false, 0, "")
			pdf.CellFormat(10, 3.5, noc, "", 0, "L", false, 0, "")
			pdf.CellFormat(35, 3.5, r.ClubName, "", 0, "L", false, 0, "")
			pdf.CellFormat(12, 3.5, d1Str, "", 0, "R", false, 0, "")
			pdf.CellFormat(12, 3.5, d2Str, "", 0, "R", false, 0, "")
			pdf.CellFormat(12, 3.5, scoreStr, "", 0, "R", false, 0, "")
			pdf.CellFormat(6, 3.5, tenCountStr, "", 0, "R", false, 0, "")
			pdf.CellFormat(6, 3.5, xCountStr, "", 1, "R", false, 0, "")
		}

		if len(results) == 0 {
			pdf.AddPage()
			printOrisPdfHeaderDetailed(pdf, ev, "C73A", "RESULTS", "All Categories", "Qualification Round")
			renderResultsHeader(41.0)
			pdf.SetFont("Arial", "I", 9)
			pdf.SetXY(10, 55)
			pdf.CellFormat(190, 8, "Belum ada skor kualifikasi.", "", 1, "C", false, 0, "")
		}

		setPdfHeaders(c, fmt.Sprintf("%s-qualification-individual-results-C73A.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output results PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 6.5. TEAM QUALIFICATION RESULTS PDF HANDLER (ORIS C73C)
// ─────────────────────────────────────────────────────────────────────────────

type TeamQualArcher struct {
	Name       string
	Bib        string
	TotalScore int
	Total10    int
	TotalX     int
}

type TeamQualGroup struct {
	Rank       int
	TeamName   string
	ClubName   string
	NocCode    string
	TotalScore int
	Total10    int
	TotalX     int
	Archers    []TeamQualArcher
}

func GetTeamQualificationResultsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")
		isMixed := c.Query("type") == "mixed" || c.Query("type") == "mix_team"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				ep.category_id,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(rgd.name, 'Umum') AS gender_name,
				a.full_name AS athlete_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
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
			LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
			WHERE ep.tournament_id = ?
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if categoryID != "" {
			query += " AND (ep.category_id = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}

		query += " GROUP BY ep.uuid ORDER BY category_name ASC, club_name ASC, total_score DESC, total_x DESC, total_10 DESC, a.full_name ASC"

		type RawRow struct {
			CategoryID   string         `db:"category_id"`
			CategoryName string         `db:"category_name"`
			GenderName   string         `db:"gender_name"`
			AthleteName  string         `db:"athlete_name"`
			AthleteCode  sql.NullString `db:"athlete_code"`
			ClubName     string         `db:"club_name"`
			TotalScore   int            `db:"total_score"`
			Total10      int            `db:"total_10"`
			TotalX       int            `db:"total_x"`
		}

		var rows []RawRow
		_ = db.Select(&rows, query, args...)

		catMap := make(map[string]map[string][]TeamQualArcher)
		var catOrder []string
		for _, r := range rows {
			if _, exists := catMap[r.CategoryName]; !exists {
				catMap[r.CategoryName] = make(map[string][]TeamQualArcher)
				catOrder = append(catOrder, r.CategoryName)
			}
			catMap[r.CategoryName][r.ClubName] = append(catMap[r.CategoryName][r.ClubName], TeamQualArcher{
				Name:       r.AthleteName,
				Bib:        r.AthleteCode.String,
				TotalScore: r.TotalScore,
				Total10:    r.Total10,
				TotalX:     r.TotalX,
			})
		}

		pdf := setupOrisPdf("C73C")

		renderTeamHeader := func(y float64) {
			pdf.SetLineWidth(0.1)
			pdf.SetDrawColor(0, 0, 0)
			pdf.Rect(10, y-1, 190, 9.0, "D")
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(0, 0, 0)

			// Line 1
			pdf.SetXY(10, y)
			pdf.CellFormat(15, 7, "Rank", "", 0, "L", false, 0, "")
			pdf.CellFormat(50, 7, "NOC", "", 0, "L", false, 0, "")
			pdf.CellFormat(50, 7, "Name", "", 0, "L", false, 0, "")

			pdf.SetXY(125, y)
			pdf.CellFormat(20, 3.5, "Individual", "", 0, "R", false, 0, "")
			pdf.CellFormat(15, 3.5, "Team", "", 0, "R", false, 0, "")

			// Line 2
			pdf.SetXY(125, y+3.5)
			pdf.CellFormat(20, 3.5, "Total", "", 0, "R", false, 0, "")
			pdf.CellFormat(15, 3.5, "Total", "", 0, "R", false, 0, "")

			pdf.SetY(y + 9.0)
		}

		maxMembers := 3
		if isMixed {
			maxMembers = 2
		}

		hasAnyTeams := false
		for _, catName := range catOrder {
			clubMap := catMap[catName]
			var teams []TeamQualGroup

			for clubName, archers := range clubMap {
				if len(archers) >= maxMembers {
					tScore := 0
					t10 := 0
					tX := 0
					var teamArchers []TeamQualArcher
					for i := 0; i < maxMembers && i < len(archers); i++ {
						tScore += archers[i].TotalScore
						t10 += archers[i].Total10
						tX += archers[i].TotalX
						teamArchers = append(teamArchers, archers[i])
					}
					noc := generateNocCode(clubName)
					teams = append(teams, TeamQualGroup{
						TeamName:   fmt.Sprintf("%s (%s)", clubName, catName),
						ClubName:   clubName,
						NocCode:    noc,
						TotalScore: tScore,
						Total10:    t10,
						TotalX:     tX,
						Archers:    teamArchers,
					})
				}
			}

			// Sort teams by total_score desc, total_x desc, total_10 desc
			for i := 0; i < len(teams); i++ {
				for j := i + 1; j < len(teams); j++ {
					if teams[j].TotalScore > teams[i].TotalScore ||
						(teams[j].TotalScore == teams[i].TotalScore && teams[j].TotalX > teams[i].TotalX) ||
						(teams[j].TotalScore == teams[i].TotalScore && teams[j].TotalX == teams[i].TotalX && teams[j].Total10 > teams[i].Total10) {
						teams[i], teams[j] = teams[j], teams[i]
					}
				}
			}

			if len(teams) == 0 {
				continue
			}
			hasAnyTeams = true

			teamCatLabel := fmt.Sprintf("%s Team", catName)
			if isMixed {
				teamCatLabel = fmt.Sprintf("%s Mixed Team", catName)
			}

			pdf.AddPage()
			printOrisPdfHeaderDetailed(pdf, ev, "C73C", "RESULTS", teamCatLabel, "Qualification Round")
			renderTeamHeader(41.0)

			for tRank, t := range teams {
				if pdf.GetY() > 255 {
					pdf.AddPage()
					printOrisPdfHeaderDetailed(pdf, ev, "C73C", "RESULTS", teamCatLabel, "Qualification Round")
					renderTeamHeader(41.0)
				}

				if tRank > 0 {
					pdf.SetY(pdf.GetY() + 2.5)
				}

				for i, ath := range t.Archers {
					pdf.SetFont("Arial", "", 8)
					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(10, pdf.GetY())

					if i == 0 {
						nocTeam := fmt.Sprintf("%s - %s", t.NocCode, t.ClubName)
						pdf.CellFormat(15, 3.5, fmt.Sprintf("%d", tRank+1), "", 0, "R", false, 0, "")
						pdf.CellFormat(50, 3.5, nocTeam, "", 0, "L", false, 0, "")
						pdf.CellFormat(50, 3.5, strings.ToUpper(ath.Name), "", 0, "L", false, 0, "")
						pdf.CellFormat(20, 3.5, fmt.Sprintf("%d", ath.TotalScore), "", 0, "R", false, 0, "")
						pdf.CellFormat(15, 3.5, fmt.Sprintf("%d", t.TotalScore), "", 0, "R", false, 0, "")
						pdf.CellFormat(30, 3.5, "", "", 1, "L", false, 0, "")
					} else {
						pdf.CellFormat(15, 3.5, "", "", 0, "R", false, 0, "")
						pdf.CellFormat(50, 3.5, "", "", 0, "L", false, 0, "")
						pdf.CellFormat(50, 3.5, strings.ToUpper(ath.Name), "", 0, "L", false, 0, "")
						pdf.CellFormat(20, 3.5, fmt.Sprintf("%d", ath.TotalScore), "", 0, "R", false, 0, "")
						pdf.CellFormat(15, 3.5, "", "", 0, "R", false, 0, "")
						pdf.CellFormat(30, 3.5, "", "", 1, "L", false, 0, "")
					}
				}
			}
		}

		if !hasAnyTeams {
			pdf.AddPage()
			printOrisPdfHeaderDetailed(pdf, ev, "C73C", "RESULTS", "All Categories Team", "Qualification Round")
			renderTeamHeader(41.0)
			pdf.SetFont("Arial", "I", 9)
			pdf.SetXY(10, 55)
			pdf.CellFormat(190, 8, "Belum ada tim yang memenuhi syarat kualifikasi beregu.", "", 1, "C", false, 0, "")
		}

		setPdfHeaders(c, fmt.Sprintf("%s-qualification-team-results-C73C.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output team qualification results PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. MEDALLISTS BY EVENT PDF HANDLER (ORIS C93)
// ─────────────────────────────────────────────────────────────────────────────

type MedallistItem struct {
	CategoryName string
	DateStr      string
	MedalType    string // "GOLD", "SILVER", "BRONZE"
	AthleteName  string
	NocCode      string
	ClubName     string
}

func GetMedallistsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type BracketMatchData struct {
			CategoryName string         `db:"category_name"`
			BracketSize  int            `db:"bracket_size"`
			MatchNo      int            `db:"match_no"`
			WinnerEntry  sql.NullString `db:"winner_entry_uuid"`
			ScheduledAt  sql.NullTime   `db:"scheduled_at"`
			EntryAUUID   sql.NullString `db:"entry_a_uuid"`
			NameA        sql.NullString `db:"name_a"`
			ClubA        sql.NullString `db:"club_a"`
			EntryBUUID   sql.NullString `db:"entry_b_uuid"`
			NameB        sql.NullString `db:"name_b"`
			ClubB        sql.NullString `db:"club_b"`
		}

		var matches []BracketMatchData
		_ = db.Select(&matches, `
			SELECT
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(eb.bracket_size, 8) AS bracket_size,
				em.match_no,
				em.winner_entry_uuid,
				em.scheduled_at,
				em.entry_a_uuid,
				CASE
					WHEN eeA.participant_type = 'team' THEN tA.team_name
					WHEN eeA.participant_type = 'archer' THEN aA.full_name
					ELSE ''
				END AS name_a,
				CASE
					WHEN eeA.participant_type = 'team' THEN COALESCE(cA_t.name, 'Individu')
					ELSE COALESCE(cA.name, 'Individu')
				END AS club_a,
				em.entry_b_uuid,
				CASE
					WHEN eeB.participant_type = 'team' THEN tB.team_name
					WHEN eeB.participant_type = 'archer' THEN aB.full_name
					ELSE ''
				END AS name_b,
				CASE
					WHEN eeB.participant_type = 'team' THEN COALESCE(cB_t.name, 'Individu')
					ELSE COALESCE(cB.name, 'Individu')
				END AS club_b
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN clubs cA ON aA.club_id = cA.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN clubs cA_t ON tA.event_id = cA_t.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN clubs cB ON aB.club_id = cB.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			LEFT JOIN clubs cB_t ON tB.event_id = cB_t.uuid
			WHERE eb.tournament_uuid = ?
			ORDER BY category_name ASC, em.match_no ASC
		`, ev.UUID)

		// Group medallists by Category
		catMedallists := make(map[string][]MedallistItem)
		var catList []string

		for _, m := range matches {
			if _, exists := catMedallists[m.CategoryName]; !exists {
				catMedallists[m.CategoryName] = []MedallistItem{}
				catList = append(catList, m.CategoryName)
			}

			dateStr := ""
			if m.ScheduledAt.Valid {
				dateStr = m.ScheduledAt.Time.Format("Mon 02 Jan")
			}

			// Finals match (Gold & Silver)
			if m.MatchNo == m.BracketSize-1 || (m.BracketSize == 2 && m.MatchNo == 1) {
				goldName, goldClub := m.NameA.String, m.ClubA.String
				silverName, silverClub := m.NameB.String, m.ClubB.String
				if m.WinnerEntry.Valid && m.EntryBUUID.Valid && m.WinnerEntry.String == m.EntryBUUID.String {
					goldName, goldClub = m.NameB.String, m.ClubB.String
					silverName, silverClub = m.NameA.String, m.ClubA.String
				}
				if goldName != "" {
					catMedallists[m.CategoryName] = append(catMedallists[m.CategoryName], MedallistItem{
						CategoryName: m.CategoryName,
						DateStr:      dateStr,
						MedalType:    "GOLD",
						AthleteName:  goldName,
						NocCode:      generateNocCode(goldClub),
						ClubName:     goldClub,
					})
				}
				if silverName != "" {
					catMedallists[m.CategoryName] = append(catMedallists[m.CategoryName], MedallistItem{
						CategoryName: m.CategoryName,
						DateStr:      dateStr,
						MedalType:    "SILVER",
						AthleteName:  silverName,
						NocCode:      generateNocCode(silverClub),
						ClubName:     silverClub,
					})
				}
			}

			// Bronze Match
			if m.MatchNo == m.BracketSize && m.BracketSize > 2 {
				bronzeName, bronzeClub := m.NameA.String, m.ClubA.String
				if m.WinnerEntry.Valid && m.EntryBUUID.Valid && m.WinnerEntry.String == m.EntryBUUID.String {
					bronzeName, bronzeClub = m.NameB.String, m.ClubB.String
				}
				if bronzeName != "" {
					catMedallists[m.CategoryName] = append(catMedallists[m.CategoryName], MedallistItem{
						CategoryName: m.CategoryName,
						DateStr:      dateStr,
						MedalType:    "BRONZE",
						AthleteName:  bronzeName,
						NocCode:      generateNocCode(bronzeClub),
						ClubName:     bronzeClub,
					})
				}
			}
		}

		pdf := setupOrisPdf("C93")
		pdf.AddPage()
		printOrisPdfHeaderDetailed(pdf, ev, "C93", "MEDALLISTS BY EVENT", "", "Medallists by Event")

		renderHeader := func(y float64) {
			pdf.SetLineWidth(0.1)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(10, y)
			pdf.CellFormat(45, 7, "Event Name", "1", 0, "L", false, 0, "")
			pdf.CellFormat(20, 7, "Date", "1", 0, "R", false, 0, "")
			pdf.CellFormat(15, 7, "Medal", "1", 0, "L", false, 0, "")
			pdf.CellFormat(50, 7, "Name", "1", 0, "L", false, 0, "")
			pdf.CellFormat(60, 7, "NOC", "1", 1, "L", false, 0, "")
		}

		renderHeader(41.0)

		hasMedals := false
		for _, catName := range catList {
			items := catMedallists[catName]
			if len(items) == 0 {
				continue
			}
			hasMedals = true

			blockH := float64(len(items)) * 7.0
			if pdf.GetY()+blockH > 265 {
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, "C93", "MEDALLISTS BY EVENT", "", "Medallists by Event")
				renderHeader(41.0)
			}

			startY := pdf.GetY()
			dateStr := items[0].DateStr

			// Event Name cell
			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(10, startY)
			pdf.CellFormat(45, 7, catName, "", 0, "L", false, 0, "")
			pdf.SetXY(10, startY)
			pdf.CellFormat(45, blockH, "", "1", 0, "L", false, 0, "")

			// Date cell
			pdf.SetXY(55, startY)
			pdf.CellFormat(20, 7, dateStr, "", 0, "R", false, 0, "")
			pdf.SetXY(55, startY)
			pdf.CellFormat(20, blockH, "", "1", 0, "L", false, 0, "")

			// Medals rows
			for i, it := range items {
				rowY := startY + float64(i)*7.0
				pdf.SetXY(75, rowY)
				pdf.CellFormat(15, 7, it.MedalType, "1", 0, "L", false, 0, "")
				pdf.CellFormat(50, 7, strings.ToUpper(it.AthleteName), "1", 0, "L", false, 0, "")
				nocTeam := fmt.Sprintf("%-6s %s", it.NocCode, it.ClubName)
				pdf.CellFormat(60, 7, nocTeam, "1", 1, "L", false, 0, "")
			}
		}

		if !hasMedals {
			pdf.SetFont("Arial", "I", 9)
			pdf.SetXY(10, 55)
			pdf.CellFormat(190, 8, "Belum ada data peraih medali.", "", 1, "C", false, 0, "")
		}

		setPdfHeaders(c, fmt.Sprintf("%s-medallists-by-event-C93.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output medallists PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 7.5. MEDAL STANDINGS PDF HANDLER (ORIS C95)
// ─────────────────────────────────────────────────────────────────────────────

type ClubMedalTally struct {
	Rank    int
	NOC     string
	Club    string
	IndivG  int
	IndivS  int
	IndivB  int
	TeamG   int
	TeamS   int
	TeamB   int
	TotalG  int
	TotalS  int
	TotalB  int
	RankTot int
}

func GetMedalStandingsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type BracketWinnerData struct {
			BracketSize     int            `db:"bracket_size"`
			MatchNo         int            `db:"match_no"`
			WinnerEntry     sql.NullString `db:"winner_entry_uuid"`
			ParticipantType string         `db:"participant_type"`
			EntryAUUID      sql.NullString `db:"entry_a_uuid"`
			ClubA           sql.NullString `db:"club_a"`
			EntryBUUID      sql.NullString `db:"entry_b_uuid"`
			ClubB           sql.NullString `db:"club_b"`
		}

		var matches []BracketWinnerData
		_ = db.Select(&matches, `
			SELECT
				COALESCE(eb.bracket_size, 8) AS bracket_size,
				em.match_no,
				em.winner_entry_uuid,
				COALESCE(eeA.participant_type, 'archer') AS participant_type,
				em.entry_a_uuid,
				CASE
					WHEN eeA.participant_type = 'team' THEN COALESCE(cA_t.name, 'Individu')
					ELSE COALESCE(cA.name, 'Individu')
				END AS club_a,
				em.entry_b_uuid,
				CASE
					WHEN eeB.participant_type = 'team' THEN COALESCE(cB_t.name, 'Individu')
					ELSE COALESCE(cB.name, 'Individu')
				END AS club_b
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN clubs cA ON aA.club_id = cA.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN clubs cA_t ON tA.event_id = cA_t.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN clubs cB ON aB.club_id = cB.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			LEFT JOIN clubs cB_t ON tB.event_id = cB_t.uuid
			WHERE eb.tournament_uuid = ?
		`, ev.UUID)

		clubMap := make(map[string]*ClubMedalTally)

		getOrCreate := func(name string) *ClubMedalTally {
			if name == "" {
				name = "Individu / Tanpa Klub"
			}
			if _, ok := clubMap[name]; !ok {
				clubMap[name] = &ClubMedalTally{
					NOC:  generateNocCode(name),
					Club: name,
				}
			}
			return clubMap[name]
		}

		for _, m := range matches {
			isTeam := m.ParticipantType == "team"

			if m.MatchNo == m.BracketSize-1 || (m.BracketSize == 2 && m.MatchNo == 1) {
				goldClub := m.ClubA.String
				silverClub := m.ClubB.String
				if m.WinnerEntry.Valid && m.EntryBUUID.Valid && m.WinnerEntry.String == m.EntryBUUID.String {
					goldClub = m.ClubB.String
					silverClub = m.ClubA.String
				}
				if goldClub != "" {
					g := getOrCreate(goldClub)
					if isTeam {
						g.TeamG++
					} else {
						g.IndivG++
					}
					g.TotalG++
				}
				if silverClub != "" {
					s := getOrCreate(silverClub)
					if isTeam {
						s.TeamS++
					} else {
						s.IndivS++
					}
					s.TotalS++
				}
			}
			if m.MatchNo == m.BracketSize && m.BracketSize > 2 {
				bronzeClub := m.ClubA.String
				if m.WinnerEntry.Valid && m.EntryBUUID.Valid && m.WinnerEntry.String == m.EntryBUUID.String {
					bronzeClub = m.ClubB.String
				}
				if bronzeClub != "" {
					b := getOrCreate(bronzeClub)
					if isTeam {
						b.TeamB++
					} else {
						b.IndivB++
					}
					b.TotalB++
				}
			}
		}

		var standings []ClubMedalTally
		for _, v := range clubMap {
			standings = append(standings, *v)
		}

		// Sort WA Olympic Rules: TotalG DESC -> TotalS DESC -> TotalB DESC -> (TotalG+TotalS+TotalB) DESC -> Club ASC
		for i := 0; i < len(standings); i++ {
			for j := i + 1; j < len(standings); j++ {
				totI := standings[i].TotalG + standings[i].TotalS + standings[i].TotalB
				totJ := standings[j].TotalG + standings[j].TotalS + standings[j].TotalB
				if standings[j].TotalG > standings[i].TotalG ||
					(standings[j].TotalG == standings[i].TotalG && standings[j].TotalS > standings[i].TotalS) ||
					(standings[j].TotalG == standings[i].TotalG && standings[j].TotalS == standings[i].TotalS && standings[j].TotalB > standings[i].TotalB) ||
					(standings[j].TotalG == standings[i].TotalG && standings[j].TotalS == standings[i].TotalS && standings[j].TotalB == standings[i].TotalB && totJ > totI) {
					standings[i], standings[j] = standings[j], standings[i]
				}
			}
		}

		pdf := setupOrisPdf("C95")
		pdf.AddPage()
		printOrisPdfHeaderDetailed(pdf, ev, "C95", "MEDAL STANDINGS", "", "Medal Standings")

		// 2-level header
		pdf.SetLineWidth(0.1)
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(10, 41)

		// Row 1
		pdf.CellFormat(10, 14, "Rank", "1", 0, "C", false, 0, "")
		pdf.CellFormat(45, 14, "NOC", "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 7, "Individual", "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 7, "Team", "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 7, "Total", "1", 0, "C", false, 0, "")
		pdf.CellFormat(15, 7, "Rank by", "TLR", 1, "C", false, 0, "")

		// Row 2
		pdf.SetXY(65, 48)
		pdf.CellFormat(10, 7, "G", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "S", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "B", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "Tot", "1", 0, "C", false, 0, "")

		pdf.CellFormat(10, 7, "G", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "S", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "B", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "Tot", "1", 0, "C", false, 0, "")

		pdf.CellFormat(10, 7, "G", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "S", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "B", "1", 0, "C", false, 0, "")
		pdf.CellFormat(10, 7, "Tot", "1", 0, "C", false, 0, "")
		pdf.CellFormat(15, 7, "Total", "BLR", 1, "C", false, 0, "")

		// Data rows
		totIndG, totIndS, totIndB, totIndT := 0, 0, 0, 0
		totTmG, totTmS, totTmB, totTmT := 0, 0, 0, 0
		totAllG, totAllS, totAllB, totAllT := 0, 0, 0, 0

		valOrBlank := func(v int) string {
			if v > 0 {
				return fmt.Sprintf("%d", v)
			}
			return ""
		}

		rank := 0
		for _, r := range standings {
			if pdf.GetY() > 265 {
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, "C95", "MEDAL STANDINGS", "", "Medal Standings")
			}

			rank++
			indTot := r.IndivG + r.IndivS + r.IndivB
			tmTot := r.TeamG + r.TeamS + r.TeamB
			allTot := r.TotalG + r.TotalS + r.TotalB

			totIndG += r.IndivG
			totIndS += r.IndivS
			totIndB += r.IndivB
			totIndT += indTot

			totTmG += r.TeamG
			totTmS += r.TeamS
			totTmB += r.TeamB
			totTmT += tmTot

			totAllG += r.TotalG
			totAllS += r.TotalS
			totAllB += r.TotalB
			totAllT += allTot

			pdf.SetFont("Arial", "", 8)
			pdf.SetXY(10, pdf.GetY())
			pdf.CellFormat(10, 7, fmt.Sprintf("%d", rank), "1", 0, "R", false, 0, "")
			pdf.CellFormat(8, 7, r.NOC, "TLB", 0, "L", false, 0, "")
			pdf.CellFormat(37, 7, r.Club, "TRB", 0, "L", false, 0, "")

			pdf.CellFormat(10, 7, valOrBlank(r.IndivG), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(r.IndivS), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(r.IndivB), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(indTot), "1", 0, "R", false, 0, "")

			pdf.CellFormat(10, 7, valOrBlank(r.TeamG), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(r.TeamS), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(r.TeamB), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(tmTot), "1", 0, "R", false, 0, "")

			pdf.CellFormat(10, 7, valOrBlank(r.TotalG), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(r.TotalS), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(r.TotalB), "1", 0, "R", false, 0, "")
			pdf.CellFormat(10, 7, valOrBlank(allTot), "1", 0, "R", false, 0, "")

			pdf.CellFormat(15, 7, fmt.Sprintf("%d", rank), "1", 1, "R", false, 0, "")
		}

		// Summary Total Row
		pdf.SetY(pdf.GetY() + 0.25)
		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(10, pdf.GetY())
		pdf.CellFormat(55, 7, "Total: ", "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totIndG), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totIndS), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totIndB), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totIndT), "1", 0, "R", false, 0, "")

		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totTmG), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totTmS), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totTmB), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totTmT), "1", 0, "R", false, 0, "")

		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totAllG), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totAllS), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totAllB), "1", 0, "R", false, 0, "")
		pdf.CellFormat(10, 7, fmt.Sprintf("%d", totAllT), "1", 1, "R", false, 0, "")

		setPdfHeaders(c, fmt.Sprintf("%s-medal-standings-C95.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output medal standings PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 7.6. FINAL RANKINGS PDF HANDLER (ORIS C76A / C76B)
// ─────────────────────────────────────────────────────────────────────────────

type FinalRankRow struct {
	Rank        int
	Bib         string
	AthleteName string
	ClubName    string
	QualScore   int
	LastRound   string
}

func GetFinalRankingsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")
		isTeam := c.Query("type") == "team" || c.Query("type") == "mix_team"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		docCode := "C76A"
		docTitle := "RESULTS SUMMARY"
		phaseName := "Individual Final Ranking"
		if isTeam {
			docCode = "C76B"
			phaseName = "Team Final Ranking"
		}

		type FinalRankRawData struct {
			CategoryID   string         `db:"category_id"`
			CategoryName string         `db:"category_name"`
			Bib          sql.NullString `db:"athlete_code"`
			AthleteName  string         `db:"athlete_name"`
			ClubName     string         `db:"club_name"`
			QualScore    int            `db:"qual_score"`
		}

		var rawEntries []FinalRankRawData
		query := `
			SELECT
				COALESCE(ep.category_id, ec.uuid) AS category_id,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				a.full_name AS athlete_name,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(SUM(qes.total_score_end), 0) AS qual_score
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
			WHERE ep.tournament_id = ?
		`
		var args []interface{}
		args = append(args, ev.UUID)
		if categoryID != "" {
			query += " AND (ep.category_id = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}
		query += " GROUP BY ep.uuid ORDER BY category_name ASC, qual_score DESC, a.full_name ASC"

		_ = db.Select(&rawEntries, query, args...)

		catGroup := make(map[string][]FinalRankRawData)
		var catNames []string
		for _, r := range rawEntries {
			if _, ok := catGroup[r.CategoryName]; !ok {
				catGroup[r.CategoryName] = []FinalRankRawData{}
				catNames = append(catNames, r.CategoryName)
			}
			catGroup[r.CategoryName] = append(catGroup[r.CategoryName], r)
		}

		pdf := setupOrisPdf(docCode)

		if len(catNames) == 0 {
			pdf.AddPage()
			printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, "", phaseName)
		}

		for _, cat := range catNames {
			athletes := catGroup[cat]
			if len(athletes) == 0 {
				continue
			}

			pdf.AddPage()
			printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, cat, phaseName)

			renderHeader := func() {
				pdf.SetDrawColor(0, 0, 0)
				pdf.SetLineWidth(0.1)
				pdf.Rect(10, 41, 190, 6, "D")

				pdf.SetFont("Arial", "B", 7.5)
				pdf.SetTextColor(0, 0, 0)
				pdf.SetXY(10, 41)
				if !isTeam {
					pdf.CellFormat(12, 6, "Rk", "", 0, "R", false, 0, "")
					pdf.CellFormat(2, 6, "", "", 0, "L", false, 0, "")
					pdf.CellFormat(58, 6, "Name", "", 0, "L", false, 0, "")
					pdf.CellFormat(15, 6, "NOC", "", 0, "L", false, 0, "")
					pdf.CellFormat(55, 6, "Country / Club", "", 0, "L", false, 0, "")
					pdf.CellFormat(18, 6, "RR. Score", "", 0, "R", false, 0, "")
					pdf.CellFormat(10, 6, "1/8", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, 6, "1/4", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, 6, "1/2", "", 1, "C", false, 0, "")
				} else {
					pdf.CellFormat(12, 6, "Rk", "", 0, "R", false, 0, "")
					pdf.CellFormat(2, 6, "", "", 0, "L", false, 0, "")
					pdf.CellFormat(15, 6, "NOC", "", 0, "L", false, 0, "")
					pdf.CellFormat(55, 6, "Country / Team", "", 0, "L", false, 0, "")
					pdf.CellFormat(58, 6, "Name", "", 0, "L", false, 0, "")
					pdf.CellFormat(18, 6, "RR. Score", "", 0, "R", false, 0, "")
					pdf.CellFormat(10, 6, "1/4", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, 6, "1/2", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, 6, "Finals", "", 1, "C", false, 0, "")
				}
			}

			renderHeader()
			curY := 47.5
			rowHeight := 3.8
			jumpLines := map[int]bool{5: true, 9: true, 17: true, 33: true}
			oldRank := -1

			if !isTeam {
				for idx, ath := range athletes {
					rank := idx + 1
					if curY+rowHeight > 275 {
						pdf.AddPage()
						printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, cat, phaseName)
						renderHeader()
						curY = 47.5
						oldRank = -1
					}

					if oldRank != rank && jumpLines[rank] {
						curY += 2.5
						oldRank = -1
					}

					rankStr := ""
					if oldRank != rank {
						rankStr = fmt.Sprintf("%d", rank)
					}

					nocCode := generateNocCode(ath.ClubName)
					pdf.SetFont("Arial", "", 8)
					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(10, curY)
					pdf.CellFormat(12, rowHeight, rankStr, "", 0, "R", false, 0, "")
					pdf.CellFormat(2, rowHeight, "", "", 0, "L", false, 0, "")
					pdf.CellFormat(58, rowHeight, strings.ToUpper(ath.AthleteName), "", 0, "L", false, 0, "")
					pdf.CellFormat(15, rowHeight, nocCode, "", 0, "L", false, 0, "")
					pdf.CellFormat(55, rowHeight, ath.ClubName, "", 0, "L", false, 0, "")
					scoreTxt := "-"
					if ath.QualScore > 0 {
						scoreTxt = fmt.Sprintf("%d", ath.QualScore)
					}
					pdf.CellFormat(18, rowHeight, scoreTxt, "", 0, "R", false, 0, "")
					pdf.CellFormat(10, rowHeight, "-", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, rowHeight, "-", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, rowHeight, "-", "", 1, "C", false, 0, "")

					curY += rowHeight
					oldRank = rank
				}
			} else {
				// Group by club as teams
				clubTeams := make(map[string][]FinalRankRawData)
				var clubList []string
				for _, ath := range athletes {
					if _, ok := clubTeams[ath.ClubName]; !ok {
						clubTeams[ath.ClubName] = []FinalRankRawData{}
						clubList = append(clubList, ath.ClubName)
					}
					clubTeams[ath.ClubName] = append(clubTeams[ath.ClubName], ath)
				}

				for tIdx, clubName := range clubList {
					members := clubTeams[clubName]
					rank := tIdx + 1
					neededH := float64(len(members))*rowHeight + 3.0
					if curY+neededH > 275 {
						pdf.AddPage()
						printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, cat, phaseName)
						renderHeader()
						curY = 47.5
					}

					rankStr := fmt.Sprintf("%d", rank)
					nocCode := generateNocCode(clubName)

					pdf.SetFont("Arial", "", 8)
					pdf.SetTextColor(0, 0, 0)

					firstAth := ""
					if len(members) > 0 {
						firstAth = strings.ToUpper(members[0].AthleteName)
					}

					teamScore := 0
					for _, m := range members {
						teamScore += m.QualScore
					}
					teamScoreTxt := "-"
					if teamScore > 0 {
						teamScoreTxt = fmt.Sprintf("%d", teamScore)
					}

					pdf.SetXY(10, curY)
					pdf.CellFormat(12, rowHeight, rankStr, "", 0, "R", false, 0, "")
					pdf.CellFormat(2, rowHeight, "", "", 0, "L", false, 0, "")
					pdf.CellFormat(15, rowHeight, nocCode, "", 0, "L", false, 0, "")
					pdf.CellFormat(55, rowHeight, clubName, "", 0, "L", false, 0, "")
					pdf.CellFormat(58, rowHeight, firstAth, "", 0, "L", false, 0, "")
					pdf.CellFormat(18, rowHeight, teamScoreTxt, "", 0, "R", false, 0, "")
					pdf.CellFormat(10, rowHeight, "-", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, rowHeight, "-", "", 0, "C", false, 0, "")
					pdf.CellFormat(10, rowHeight, "-", "", 1, "C", false, 0, "")
					curY += rowHeight

					for k := 1; k < len(members); k++ {
						pdf.SetXY(10, curY)
						pdf.CellFormat(12, rowHeight, "", "", 0, "R", false, 0, "")
						pdf.CellFormat(2, rowHeight, "", "", 0, "L", false, 0, "")
						pdf.CellFormat(15, rowHeight, "", "", 0, "L", false, 0, "")
						pdf.CellFormat(55, rowHeight, "", "", 0, "L", false, 0, "")
						pdf.CellFormat(58, rowHeight, strings.ToUpper(members[k].AthleteName), "", 0, "L", false, 0, "")
						pdf.CellFormat(18, rowHeight, "", "", 0, "R", false, 0, "")
						pdf.CellFormat(10, rowHeight, "", "", 0, "C", false, 0, "")
						pdf.CellFormat(10, rowHeight, "", "", 0, "C", false, 0, "")
						pdf.CellFormat(10, rowHeight, "", "", 1, "C", false, 0, "")
						curY += rowHeight
					}
					curY += 2.0 // Team spacer
				}
			}
		}

		filename := fmt.Sprintf("%s-individual-final-ranking-C76A.pdf", ev.Slug)
		if isTeam {
			filename = fmt.Sprintf("%s-team-final-ranking-C76B.pdf", ev.Slug)
		}
		setPdfHeaders(c, filename)
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output final rankings PDF"})
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
		pdf.SetAutoPageBreak(false, 0)
		pdf.AddPage()

		// 3 columns x 8 rows (24 labels per A4 page, IanSeo standard)
		cardW := 63.0
		cardH := 32.5
		marginX := 7.0
		marginY := 11.0
		gapX := 3.5
		gapY := 2.5
		itemsPerPage := 24

		for i, l := range labels {
			pageIndex := i % itemsPerPage
			if i > 0 && pageIndex == 0 {
				pdf.AddPage()
			}

			col := pageIndex % 3
			row := pageIndex / 3

			x := marginX + float64(col)*(cardW+gapX)
			y := marginY + float64(row)*(cardH+gapY)

			// Outer Box Border
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.15)
			pdf.Rect(x, y, cardW, cardH, "D")

			// Top Header (Tournament Name)
			pdf.SetFont("Arial", "B", 6)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(x+1, y+1)
			pdf.CellFormat(cardW-2, 3, ev.Name, "", 1, "C", false, 0, "")

			// Header Divider Line
			pdf.SetLineWidth(0.1)
			pdf.Line(x, y+4.2, x+cardW, y+4.2)

			// Target Box (Left)
			pdf.SetLineWidth(0.15)
			pdf.Rect(x+1.5, y+5.5, 14, 13, "D")
			pdf.SetFont("Arial", "B", 13)
			pdf.SetXY(x+1.5, y+5.5)
			pdf.CellFormat(14, 13, l.TargetName, "", 0, "C", false, 0, "")

			// Athlete & Category Details (Right)
			noc := generateNocCode(l.ClubName)
			athleteDisplayName := strings.ToUpper(l.AthleteName)

			// Athlete Name
			pdf.SetFont("Arial", "B", 7.5)
			pdf.SetXY(x+17, y+5.2)
			pdf.CellFormat(cardW-18, 3.8, athleteDisplayName, "", 1, "L", false, 0, "")

			// NOC & Club Name
			pdf.SetFont("Arial", "", 6.5)
			pdf.SetXY(x+17, y+9.2)
			clubText := fmt.Sprintf("[%s] %s", noc, l.ClubName)
			if len(clubText) > 28 {
				clubText = clubText[:28] + "..."
			}
			pdf.CellFormat(cardW-18, 3.5, clubText, "", 1, "L", false, 0, "")

			// Category
			pdf.SetFont("Arial", "I", 6.5)
			pdf.SetXY(x+17, y+13.0)
			catText := l.CategoryName
			if len(catText) > 28 {
				catText = catText[:28] + "..."
			}
			pdf.CellFormat(cardW-18, 3.5, catText, "", 1, "L", false, 0, "")

			// Session / Archer Code
			pdf.SetFont("Arial", "", 6)
			pdf.SetXY(x+17, y+16.8)
			pdf.CellFormat(cardW-18, 3.2, fmt.Sprintf("Session %s", l.SessionCode), "", 1, "L", false, 0, "")

			// Bottom Section (Verification / Equipment Line)
			pdf.SetLineWidth(0.1)
			pdf.Line(x, y+22.0, x+cardW, y+22.0)
			pdf.SetFont("Arial", "", 5.5)
			pdf.SetTextColor(70, 70, 70)
			pdf.SetXY(x+2, y+23.0)
			codeDisplay := l.AthleteCode.String
			if codeDisplay == "" {
				codeDisplay = "-"
			}
			pdf.CellFormat(cardW-4, 3, fmt.Sprintf("Archer ID: %s   |   WORLD ARCHERY ORIS", codeDisplay), "", 1, "L", false, 0, "")

			pdf.SetXY(x+2, y+26.5)
			pdf.CellFormat(cardW-4, 3, "Equip. Insp: [  ]   |   Signature: _____________", "", 1, "L", false, 0, "")
		}

		setPdfHeaders(c, fmt.Sprintf("%s-target-labels.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output target labels PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 9. TEAM / MIXED TEAM ELIMINATION SCORESHEET PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

type TeamMemberInfo struct {
	TeamUUID   string `db:"team_id"`
	MemberName string `db:"member_name"`
	Order      int    `db:"member_order"`
}

type EliminationMatchTeamRow struct {
	MatchUUID    string         `db:"match_uuid"`
	BracketUUID  string         `db:"bracket_uuid"`
	BracketSize  int            `db:"bracket_size"`
	BracketType  string         `db:"bracket_type"`
	Format       string         `db:"format"`
	CategoryName string         `db:"category_name"`
	RoundNo      int            `db:"round_no"`
	MatchNo      int            `db:"match_no"`
	ScheduledAt  sql.NullTime   `db:"scheduled_at"`
	TargetName   sql.NullString `db:"target_name"`
	BoardNumber  sql.NullInt64  `db:"board_number"`
	TypeA        sql.NullString `db:"type_a"`
	SeedA        sql.NullInt64  `db:"seed_a"`
	TeamUUIDA    sql.NullString `db:"team_uuid_a"`
	TeamNameA    sql.NullString `db:"team_name_a"`
	ClubNameA    sql.NullString `db:"club_name_a"`
	TypeB        sql.NullString `db:"type_b"`
	SeedB        sql.NullInt64  `db:"seed_b"`
	TeamUUIDB    sql.NullString `db:"team_uuid_b"`
	TeamNameB    sql.NullString `db:"team_name_b"`
	ClubNameB    sql.NullString `db:"club_name_b"`
}

func getPrintElimRoundLabel(bracketSize, roundNo, matchNo int) string {
	if matchNo == bracketSize && bracketSize > 2 {
		return "Perebutan Juara 3 (Bronze Medal)"
	}
	if matchNo == bracketSize-1 && bracketSize > 2 {
		return "Final Emas (Gold Medal)"
	}
	totalRounds := 0
	n := bracketSize
	for n > 1 {
		n /= 2
		totalRounds++
	}
	roundsFromFinal := totalRounds - roundNo
	switch roundsFromFinal {
	case 0:
		return "Final"
	case 1:
		return "Semifinal"
	case 2:
		return "Perempat Final (Quarterfinal)"
	case 3:
		return "Babak 1/8 (1/8 Final)"
	case 4:
		return "Babak 1/16 (1/16 Final)"
	default:
		if roundsFromFinal > 0 {
			return fmt.Sprintf("Babak 1/%d", 1<<roundsFromFinal)
		}
		return fmt.Sprintf("Babak %d", roundNo)
	}
}

func GetTeamEliminationScoresheetPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		bracketID := c.Query("bracket_id")
		if bracketID == "" {
			bracketID = c.Param("bracketId")
		}
		if bracketID == "" {
			bracketID = c.Param("bracket_id")
		}
		matchID := c.Query("match_id")
		categoryID := c.Query("category_id")
		isMixedParam := c.Query("is_mixed") == "1" || c.Query("type") == "mixed" || c.Query("type") == "mix_team"
		blankMode := c.Query("blank") == "1"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				em.uuid AS match_uuid,
				eb.uuid AS bracket_uuid,
				COALESCE(eb.bracket_size, 8) AS bracket_size,
				COALESCE(eb.bracket_type, 'team') AS bracket_type,
				COALESCE(eb.format, 'recurve_set') AS format,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				em.round_no,
				em.match_no,
				em.scheduled_at,
				COALESCE(et.target_name, '') AS target_name,
				COALESCE(et.board_number, 0) AS board_number,
				eeA.participant_type AS type_a,
				COALESCE(eeA.seed, 0) AS seed_a,
				CASE
					WHEN eeA.participant_type = 'team' THEN tA.uuid
					ELSE aA.uuid
				END AS team_uuid_a,
				CASE
					WHEN eeA.participant_type = 'team' THEN tA.team_name
					ELSE aA.full_name
				END AS team_name_a,
				COALESCE(cA.name, 'Individu') AS club_name_a,
				eeB.participant_type AS type_b,
				COALESCE(eeB.seed, 0) AS seed_b,
				CASE
					WHEN eeB.participant_type = 'team' THEN tB.uuid
					ELSE aB.uuid
				END AS team_uuid_b,
				CASE
					WHEN eeB.participant_type = 'team' THEN tB.team_name
					ELSE aB.full_name
				END AS team_name_b,
				COALESCE(cB.name, 'Individu') AS club_name_b
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN tournament_targets et ON em.target_uuid = et.uuid
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN clubs cA ON aA.club_id = cA.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN clubs cB ON aB.club_id = cB.uuid
			WHERE eb.tournament_uuid = ? AND (em.is_bye = 0 OR em.is_bye IS NULL)
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if bracketID != "" {
			query += " AND (eb.uuid = ? OR eb.bracket_id = ?)"
			args = append(args, bracketID, bracketID)
		}
		if matchID != "" {
			query += " AND (em.uuid = ? OR em.match_id = ?)"
			args = append(args, matchID, matchID)
		}
		if categoryID != "" {
			query += " AND (eb.category_uuid = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}

		query += " ORDER BY COALESCE(et.board_number, 9999) ASC, em.round_no ASC, em.match_no ASC"

		var matches []EliminationMatchTeamRow
		_ = db.Select(&matches, query, args...)

		// Fetch team members map
		teamMembersMap := make(map[string][]string)
		var teamIDs []string
		for _, m := range matches {
			if m.TeamUUIDA.Valid && m.TeamUUIDA.String != "" {
				teamIDs = append(teamIDs, m.TeamUUIDA.String)
			}
			if m.TeamUUIDB.Valid && m.TeamUUIDB.String != "" {
				teamIDs = append(teamIDs, m.TeamUUIDB.String)
			}
		}

		if len(teamIDs) > 0 {
			mQuery, mArgs, inErr := sqlx.In(`
				SELECT tm.team_id, a.full_name AS member_name, tm.member_order
				FROM team_members tm
				JOIN tournament_participants tp ON tm.participant_id = tp.uuid
				JOIN archers a ON tp.archer_id = a.uuid
				WHERE tm.team_id IN (?)
				ORDER BY tm.team_id ASC, tm.member_order ASC
			`, teamIDs)
			if inErr == nil {
				var members []TeamMemberInfo
				if selErr := db.Select(&members, mQuery, mArgs...); selErr == nil {
					for _, mem := range members {
						teamMembersMap[mem.TeamUUID] = append(teamMembersMap[mem.TeamUUID], mem.MemberName)
					}
				}
			}
		}

		// Fallback if no matches found
		if len(matches) == 0 {
			bType := "team"
			if isMixedParam {
				bType = "mixed_team"
			}
			matches = append(matches, EliminationMatchTeamRow{
				BracketSize:  8,
				BracketType:  bType,
				Format:       "recurve_set",
				CategoryName: "Beregu / Mix Team",
				RoundNo:      1,
				MatchNo:      1,
			})
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.SetAutoPageBreak(false, 0)

		loc := ev.Venue.String
		if ev.City.Valid && ev.City.String != "" {
			if loc != "" {
				loc += ", "
			}
			loc += ev.City.String
		}
		dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)

		for _, m := range matches {
			pdf.AddPage()

			isMixed := isMixedParam || m.BracketType == "mixed_team" || m.BracketType == "mix_team" || strings.Contains(strings.ToLower(m.CategoryName), "mix")
			arrowsPerEnd := 6
			athletesCount := 3
			docTitle := "Team Elimination Match Scoresheet"
			if isMixed {
				arrowsPerEnd = 4
				athletesCount = 2
				docTitle = "Mixed Team Elimination Match Scoresheet"
			}

			// 1. Header (Tournament & Document Info)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.2)

			pdf.SetFont("Arial", "B", 10)
			pdf.SetXY(10, 10)
			pdf.CellFormat(120, 5, ev.Name, "", 0, "L", false, 0, "")

			pdf.SetFont("Arial", "B", 10)
			pdf.SetXY(130, 10)
			pdf.CellFormat(70, 5, docTitle, "", 1, "R", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(70, 70, 70)
			pdf.SetXY(10, 15)
			pdf.CellFormat(120, 4, fmt.Sprintf("%s | %s", loc, dateStr), "", 0, "L", false, 0, "")

			pdf.SetXY(130, 15)
			pdf.CellFormat(70, 4, "World Archery / PERPANI Official Match Record", "", 1, "R", false, 0, "")

			pdf.Line(10, 20, 200, 20)

			// 2. Round & Match Info Ribbon
			roundName := getPrintElimRoundLabel(m.BracketSize, m.RoundNo, m.MatchNo)
			pdf.SetFillColor(245, 245, 245)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.15)
			pdf.SetFont("Arial", "B", 9)
			pdf.SetXY(10, 22)

			targetInfo := "Target: -"
			if m.TargetName.Valid && m.TargetName.String != "" {
				targetInfo = fmt.Sprintf("Target: %s", m.TargetName.String)
			}
			schedInfo := ""
			if m.ScheduledAt.Valid {
				schedInfo = fmt.Sprintf(" | Time: %s", m.ScheduledAt.Time.Format("02 Jan 15:04"))
			}

			infoText := fmt.Sprintf(" %s - %s | %s%s", m.CategoryName, roundName, targetInfo, schedInfo)
			pdf.CellFormat(190, 7, infoText, "1", 1, "L", true, 0, "")

			// 3. Team A Card (Left) & Team B Card (Right)
			cardW := 92.0
			cardH := 28.0
			yCards := 31.0

			// Team A Box
			pdf.SetXY(10, yCards)
			pdf.Rect(10, yCards, cardW, cardH, "D")

			seedATxt := "-"
			if m.SeedA.Valid && m.SeedA.Int64 > 0 {
				seedATxt = fmt.Sprintf("#%d", m.SeedA.Int64)
			}
			pdf.Rect(12, yCards+2, 12, 6, "D")
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(12, yCards+2)
			pdf.CellFormat(12, 6, seedATxt, "", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "B", 9.5)
			pdf.SetXY(26, yCards+2)
			teamAName := strings.ToUpper(m.TeamNameA.String)
			if blankMode || teamAName == "" {
				teamAName = "TEAM A: ...................................."
			}
			pdf.CellFormat(74, 6, teamAName, "", 1, "L", false, 0, "")

			pdf.SetTextColor(60, 60, 60)
			pdf.SetFont("Arial", "", 7.5)
			pdf.SetXY(12, yCards+9)
			clubAName := m.ClubNameA.String
			if blankMode || clubAName == "" {
				clubAName = "-"
			}
			pdf.CellFormat(88, 3.5, fmt.Sprintf("Club / Contingent: %s", clubAName), "", 1, "L", false, 0, "")

			// Member names A
			memsA := teamMembersMap[m.TeamUUIDA.String]
			pdf.SetFont("Arial", "I", 7)
			pdf.SetTextColor(80, 80, 80)
			for i := 0; i < athletesCount; i++ {
				pdf.SetXY(12, yCards+13+float64(i)*4.2)
				athName := "..................................................."
				if i < len(memsA) && !blankMode {
					athName = strings.ToUpper(memsA[i])
				}
				pdf.CellFormat(88, 3.8, fmt.Sprintf("%d. %s", i+1, athName), "", 1, "L", false, 0, "")
			}

			// Team B Box
			xCardB := 108.0
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(xCardB, yCards)
			pdf.Rect(xCardB, yCards, cardW, cardH, "D")

			seedBTxt := "-"
			if m.SeedB.Valid && m.SeedB.Int64 > 0 {
				seedBTxt = fmt.Sprintf("#%d", m.SeedB.Int64)
			}
			pdf.Rect(xCardB+2, yCards+2, 12, 6, "D")
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(xCardB+2, yCards+2)
			pdf.CellFormat(12, 6, seedBTxt, "", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "B", 9.5)
			pdf.SetXY(xCardB+16, yCards+2)
			teamBName := strings.ToUpper(m.TeamNameB.String)
			if blankMode || teamBName == "" {
				teamBName = "TEAM B: ...................................."
			}
			pdf.CellFormat(74, 6, teamBName, "", 1, "L", false, 0, "")

			pdf.SetTextColor(60, 60, 60)
			pdf.SetFont("Arial", "", 7.5)
			pdf.SetXY(xCardB+2, yCards+9)
			clubBName := m.ClubNameB.String
			if blankMode || clubBName == "" {
				clubBName = "-"
			}
			pdf.CellFormat(88, 3.5, fmt.Sprintf("Club / Contingent: %s", clubBName), "", 1, "L", false, 0, "")

			// Member names B
			memsB := teamMembersMap[m.TeamUUIDB.String]
			pdf.SetFont("Arial", "I", 7)
			pdf.SetTextColor(80, 80, 80)
			for i := 0; i < athletesCount; i++ {
				pdf.SetXY(xCardB+2, yCards+13+float64(i)*4.2)
				athName := "..................................................."
				if i < len(memsB) && !blankMode {
					athName = strings.ToUpper(memsB[i])
				}
				pdf.CellFormat(88, 3.8, fmt.Sprintf("%d. %s", i+1, athName), "", 1, "L", false, 0, "")
			}

			// 4. Scoresheet Table (IanSeo Standard)
			yTable := yCards + cardH + 3.0
			pdf.SetY(yTable)
			pdf.SetX(10)
			pdf.SetFillColor(245, 245, 245)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetFont("Arial", "B", 7.5)

			isSetSystem := m.Format == "recurve_set" || strings.Contains(strings.ToLower(m.Format), "set")
			setColTitle := "Set Pts"
			if !isSetSystem {
				setColTitle = "Total"
			}

			if arrowsPerEnd == 6 {
				// 3-Archer Team Header (6 Arrows per end)
				arrowColW := 7.0
				pdf.CellFormat(10, 7, "Set", "1", 0, "C", true, 0, "")
				for a := 1; a <= 6; a++ {
					pdf.CellFormat(arrowColW, 7, fmt.Sprintf("%d", a), "1", 0, "C", true, 0, "")
				}
				pdf.CellFormat(15, 7, "Total", "1", 0, "C", true, 0, "")
				pdf.CellFormat(15, 7, setColTitle, "1", 0, "C", true, 0, "")

				// Center VS
				pdf.CellFormat(16, 7, "VS", "1", 0, "C", true, 0, "")

				// Team B side
				pdf.CellFormat(15, 7, setColTitle, "1", 0, "C", true, 0, "")
				pdf.CellFormat(15, 7, "Total", "1", 0, "C", true, 0, "")
				for a := 1; a <= 6; a++ {
					pdf.CellFormat(arrowColW, 7, fmt.Sprintf("%d", a), "1", 0, "C", true, 0, "")
				}
				pdf.CellFormat(10, 7, "Set", "1", 1, "C", true, 0, "")

				// 4 Sets / Ends Rows
				rowH := 18.0
				for setNum := 1; setNum <= 4; setNum++ {
					pdf.SetX(10)
					pdf.SetFont("Arial", "B", 8.5)
					pdf.SetFillColor(250, 250, 250)
					pdf.CellFormat(10, rowH, fmt.Sprintf("%d", setNum), "1", 0, "C", true, 0, "")

					for a := 1; a <= 6; a++ {
						pdf.CellFormat(arrowColW, rowH, "", "1", 0, "C", false, 0, "")
					}
					pdf.CellFormat(15, rowH, "", "1", 0, "C", true, 0, "")
					pdf.CellFormat(15, rowH, "", "1", 0, "C", false, 0, "")

					pdf.CellFormat(16, rowH, fmt.Sprintf("S%d", setNum), "1", 0, "C", true, 0, "")

					pdf.CellFormat(15, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(15, rowH, "", "1", 0, "C", true, 0, "")
					for a := 1; a <= 6; a++ {
						pdf.CellFormat(arrowColW, rowH, "", "1", 0, "C", false, 0, "")
					}
					pdf.CellFormat(10, rowH, fmt.Sprintf("%d", setNum), "1", 1, "C", true, 0, "")
				}

				// Shoot-Off Row (3 arrows for 3-person team)
				pdf.SetX(10)
				pdf.SetFont("Arial", "B", 7.5)
				pdf.SetFillColor(245, 245, 245)
				pdf.CellFormat(10, 10, "S.O", "1", 0, "C", true, 0, "")
				for a := 1; a <= 3; a++ {
					pdf.CellFormat(arrowColW, 10, "", "1", 0, "C", false, 0, "")
				}
				pdf.CellFormat(arrowColW*3, 10, "Shoot-Off", "1", 0, "C", true, 0, "")
				pdf.CellFormat(15, 10, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(15, 10, "", "1", 0, "C", false, 0, "")

				pdf.CellFormat(16, 10, "TIE", "1", 0, "C", true, 0, "")

				pdf.CellFormat(15, 10, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(15, 10, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(arrowColW*3, 10, "Shoot-Off", "1", 0, "C", true, 0, "")
				for a := 1; a <= 3; a++ {
					pdf.CellFormat(arrowColW, 10, "", "1", 0, "C", false, 0, "")
				}
				pdf.CellFormat(10, 10, "S.O", "1", 1, "C", true, 0, "")

				// Match Final Result Summary Row
				pdf.SetX(10)
				pdf.SetFont("Arial", "B", 8.5)
				pdf.SetFillColor(240, 240, 240)
				pdf.CellFormat(52, 9, " TOTAL SET POINTS / SCORE", "1", 0, "R", true, 0, "")
				pdf.CellFormat(15, 9, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(15, 9, "", "1", 0, "C", true, 0, "")

				pdf.CellFormat(16, 9, "WINNER", "1", 0, "C", true, 0, "")

				pdf.CellFormat(15, 9, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(15, 9, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(62, 9, "TOTAL SET POINTS / SCORE ", "1", 1, "L", true, 0, "")

			} else {
				// 2-Archer Mixed Team Header (4 Arrows per end)
				arrowColW := 9.0
				pdf.CellFormat(10, 7, "Set", "1", 0, "C", true, 0, "")
				for a := 1; a <= 4; a++ {
					pdf.CellFormat(arrowColW, 7, fmt.Sprintf("%d", a), "1", 0, "C", true, 0, "")
				}
				pdf.CellFormat(18, 7, "Total", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 7, setColTitle, "1", 0, "C", true, 0, "")

				// Center VS
				pdf.CellFormat(16, 7, "VS", "1", 0, "C", true, 0, "")

				// Team B side
				pdf.CellFormat(18, 7, setColTitle, "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 7, "Total", "1", 0, "C", true, 0, "")
				for a := 1; a <= 4; a++ {
					pdf.CellFormat(arrowColW, 7, fmt.Sprintf("%d", a), "1", 0, "C", true, 0, "")
				}
				pdf.CellFormat(10, 7, "Set", "1", 1, "C", true, 0, "")

				// 4 Sets / Ends Rows
				rowH := 18.0
				for setNum := 1; setNum <= 4; setNum++ {
					pdf.SetX(10)
					pdf.SetFont("Arial", "B", 8.5)
					pdf.SetFillColor(250, 250, 250)
					pdf.CellFormat(10, rowH, fmt.Sprintf("%d", setNum), "1", 0, "C", true, 0, "")

					for a := 1; a <= 4; a++ {
						pdf.CellFormat(arrowColW, rowH, "", "1", 0, "C", false, 0, "")
					}
					pdf.CellFormat(18, rowH, "", "1", 0, "C", true, 0, "")
					pdf.CellFormat(18, rowH, "", "1", 0, "C", false, 0, "")

					pdf.CellFormat(16, rowH, fmt.Sprintf("S%d", setNum), "1", 0, "C", true, 0, "")

					pdf.CellFormat(18, rowH, "", "1", 0, "C", false, 0, "")
					pdf.CellFormat(18, rowH, "", "1", 0, "C", true, 0, "")
					for a := 1; a <= 4; a++ {
						pdf.CellFormat(arrowColW, rowH, "", "1", 0, "C", false, 0, "")
					}
					pdf.CellFormat(10, rowH, fmt.Sprintf("%d", setNum), "1", 1, "C", true, 0, "")
				}

				// Shoot-Off Row (2 arrows for Mixed Team)
				pdf.SetX(10)
				pdf.SetFont("Arial", "B", 7.5)
				pdf.SetFillColor(245, 245, 245)
				pdf.CellFormat(10, 10, "S.O", "1", 0, "C", true, 0, "")
				for a := 1; a <= 2; a++ {
					pdf.CellFormat(arrowColW, 10, "", "1", 0, "C", false, 0, "")
				}
				pdf.CellFormat(arrowColW*2, 10, "Shoot-Off", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 10, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 10, "", "1", 0, "C", false, 0, "")

				pdf.CellFormat(16, 10, "TIE", "1", 0, "C", true, 0, "")

				pdf.CellFormat(18, 10, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(18, 10, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(arrowColW*2, 10, "Shoot-Off", "1", 0, "C", true, 0, "")
				for a := 1; a <= 2; a++ {
					pdf.CellFormat(arrowColW, 10, "", "1", 0, "C", false, 0, "")
				}
				pdf.CellFormat(10, 10, "S.O", "1", 1, "C", true, 0, "")

				// Match Final Result Summary Row
				pdf.SetX(10)
				pdf.SetFont("Arial", "B", 8.5)
				pdf.SetFillColor(240, 240, 240)
				pdf.CellFormat(46, 9, " TOTAL SET POINTS / SCORE", "1", 0, "R", true, 0, "")
				pdf.CellFormat(18, 9, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 9, "", "1", 0, "C", true, 0, "")

				pdf.CellFormat(16, 9, "WINNER", "1", 0, "C", true, 0, "")

				pdf.CellFormat(18, 9, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(18, 9, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(56, 9, "TOTAL SET POINTS / SCORE ", "1", 1, "L", true, 0, "")
			}

			// 5. Signatures Block
			ySig := 228.0
			pdf.SetY(ySig)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.15)
			pdf.SetFont("Arial", "", 7.5)

			pdf.Line(15, ySig+10, 65, ySig+10)
			pdf.SetXY(15, ySig+11)
			pdf.CellFormat(50, 4, "Team A Captain Signature", "", 0, "C", false, 0, "")

			pdf.Line(80, ySig+10, 130, ySig+10)
			pdf.SetXY(80, ySig+11)
			pdf.CellFormat(50, 4, "Target Judge Signature", "", 0, "C", false, 0, "")

			pdf.Line(145, ySig+10, 195, ySig+10)
			pdf.SetXY(145, ySig+11)
			pdf.CellFormat(50, 4, "Team B Captain Signature", "", 1, "C", false, 0, "")

			pdf.SetFont("Arial", "I", 6.5)
			pdf.SetTextColor(110, 110, 110)
			pdf.SetXY(10, ySig+16)
			pdf.CellFormat(190, 3, "The signatures certify the correctness of the match result in accordance with World Archery Rules.", "", 1, "C", false, 0, "")
		}

		setPdfHeaders(c, fmt.Sprintf("TeamEliminationScoresheet-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output team elimination scoresheet PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 10. ELIMINATION MATCH SCHEDULE (ORIS C58) PDF HANDLER
// ─────────────────────────────────────────────────────────────────────────────

type EliminationScheduleRow struct {
	MatchUUID    string         `db:"match_uuid"`
	BracketUUID  string         `db:"bracket_uuid"`
	BracketSize  int            `db:"bracket_size"`
	BracketType  string         `db:"bracket_type"`
	CategoryName string         `db:"category_name"`
	RoundNo      int            `db:"round_no"`
	MatchNo      int            `db:"match_no"`
	ScheduledAt  sql.NullTime   `db:"scheduled_at"`
	TargetName   sql.NullString `db:"target_name"`
	BoardNumber  sql.NullInt64  `db:"board_number"`
	Status       string         `db:"status"`
	SeedA        sql.NullInt64  `db:"seed_a"`
	NameA        sql.NullString `db:"name_a"`
	ClubA        sql.NullString `db:"club_a"`
	SeedB        sql.NullInt64  `db:"seed_b"`
	NameB        sql.NullString `db:"name_b"`
	ClubB        sql.NullString `db:"club_b"`
}

func getIanseoPhaseName(bracketSize, roundNo, matchNo int) string {
	if matchNo == bracketSize && bracketSize > 2 {
		return "Bronze"
	}
	if matchNo == bracketSize-1 && bracketSize > 2 {
		return "Gold"
	}
	totalRounds := 0
	n := bracketSize
	for n > 1 {
		n /= 2
		totalRounds++
	}
	roundsFromFinal := totalRounds - roundNo
	switch roundsFromFinal {
	case 0:
		return "Gold"
	case 1:
		return "Semi"
	case 2:
		return "1/4"
	case 3:
		return "1/8"
	case 4:
		return "1/16"
	case 5:
		return "1/32"
	case 6:
		return "1/64"
	default:
		if roundsFromFinal > 0 {
			return fmt.Sprintf("1/%d", 1<<roundsFromFinal)
		}
		return "Final"
	}
}

func GetEliminationSchedulePrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		bracketID := c.Query("bracket_id")
		categoryID := c.Query("category_id")
		dateFilter := c.Query("date")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				em.uuid AS match_uuid,
				eb.uuid AS bracket_uuid,
				COALESCE(eb.bracket_size, 8) AS bracket_size,
				COALESCE(eb.bracket_type, 'individual') AS bracket_type,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				em.round_no,
				em.match_no,
				em.scheduled_at,
				COALESCE(et.target_name, '') AS target_name,
				COALESCE(et.board_number, 0) AS board_number,
				em.status,
				COALESCE(eeA.seed, 0) AS seed_a,
				CASE
					WHEN eeA.participant_type = 'team' THEN tA.team_name
					WHEN eeA.participant_type = 'archer' THEN aA.full_name
					ELSE ''
				END AS name_a,
				CASE
					WHEN eeA.participant_type = 'team' THEN COALESCE(cA_team.name, 'Individu')
					ELSE COALESCE(cA.name, 'Individu')
				END AS club_a,
				COALESCE(eeB.seed, 0) AS seed_b,
				CASE
					WHEN eeB.participant_type = 'team' THEN tB.team_name
					WHEN eeB.participant_type = 'archer' THEN aB.full_name
					ELSE ''
				END AS name_b,
				CASE
					WHEN eeB.participant_type = 'team' THEN COALESCE(cB_team.name, 'Individu')
					ELSE COALESCE(cB.name, 'Individu')
				END AS club_b
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN tournament_targets et ON em.target_uuid = et.uuid
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN clubs cA ON aA.club_id = cA.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN clubs cA_team ON tA.event_id = cA_team.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN clubs cB ON aB.club_id = cB.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			LEFT JOIN clubs cB_team ON tB.event_id = cB_team.uuid
			WHERE eb.tournament_uuid = ? AND (em.is_bye = 0 OR em.is_bye IS NULL)
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if bracketID != "" {
			query += " AND (eb.uuid = ? OR eb.bracket_id = ?)"
			args = append(args, bracketID, bracketID)
		}
		if categoryID != "" {
			query += " AND (eb.category_uuid = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}
		if dateFilter != "" {
			query += " AND DATE(em.scheduled_at) = ?"
			args = append(args, dateFilter)
		}

		query += " ORDER BY DATE(COALESCE(em.scheduled_at, '9999-12-31')) ASC, COALESCE(et.board_number, 9999) ASC, em.round_no ASC, em.match_no ASC"

		var matches []EliminationScheduleRow
		err = db.Select(&matches, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jadwal eliminasi: " + err.Error()})
			return
		}

		pdf := setupOrisPdf("C58")
		pdf.AddPage()
		printOrisPdfHeader(pdf, ev, "C58", "DETAILED COMPETITION SCHEDULE")

		const cellH = 8.0

		// Table Header (Exact IanSeo ORIS C58)
		renderTableHeader := func() {
			pdf.SetY(45)
			pdf.SetFont("Arial", "B", 7.5)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)

			// 1. Date/Session (25)
			pdf.SetXY(10, 45)
			pdf.CellFormat(25, cellH, "Date/Session", "1", 0, "L", false, 0, "")

			// 2. Match (7)
			pdf.CellFormat(7, cellH, "Match", "1", 0, "C", false, 0, "")

			// 3. Start Time (9, 2 lines)
			pdf.CellFormat(9, cellH/2, "Start", "TLR", 0, "C", false, 0, "")
			pdf.SetXY(pdf.GetX()-9, 45+cellH/2)
			pdf.CellFormat(9, cellH/2, "Time", "BLR", 0, "C", false, 0, "")
			pdf.SetXY(pdf.GetX(), 45)

			// 4. Event (30)
			pdf.CellFormat(30, cellH, "Event", "1", 0, "L", false, 0, "")

			// 5. Round (9)
			pdf.CellFormat(9, cellH, "Round", "1", 0, "L", false, 0, "")

			// 6. R.R. Rank 1 (10, 2 lines)
			pdf.CellFormat(10, cellH/2, "R.R.", "TLR", 0, "C", false, 0, "")
			pdf.SetXY(pdf.GetX()-10, 45+cellH/2)
			pdf.CellFormat(10, cellH/2, "Rank", "BLR", 0, "C", false, 0, "")
			pdf.SetXY(pdf.GetX(), 45)

			// 7. Participant 1 (45)
			pdf.CellFormat(45, cellH, "Participant 1", "1", 0, "L", false, 0, "")

			// 8. R.R. Rank 2 (10, 2 lines)
			pdf.CellFormat(10, cellH/2, "R.R.", "TLR", 0, "C", false, 0, "")
			pdf.SetXY(pdf.GetX()-10, 45+cellH/2)
			pdf.CellFormat(10, cellH/2, "Rank", "BLR", 0, "C", false, 0, "")
			pdf.SetXY(pdf.GetX(), 45)

			// 9. Participant 2 (45)
			pdf.CellFormat(45, cellH, "Participant 2", "1", 1, "L", false, 0, "")

			pdf.SetY(53)
		}

		renderTableHeader()

		// Group matches by Date/Session (like IanSeo)
		type SessionGroup struct {
			DateKey     string
			DateDisplay string
			Matches     []EliminationScheduleRow
		}

		var groups []SessionGroup
		var currentGrp *SessionGroup

		for _, m := range matches {
			dateKey := "TBD"
			dateDisp := "Schedule"
			if m.ScheduledAt.Valid {
				dateKey = m.ScheduledAt.Time.Format("2006-01-02")
				dateDisp = m.ScheduledAt.Time.Format("Mon 2 Jan")
			}
			if currentGrp == nil || currentGrp.DateKey != dateKey {
				groups = append(groups, SessionGroup{
					DateKey:     dateKey,
					DateDisplay: dateDisp,
					Matches:     []EliminationScheduleRow{},
				})
				currentGrp = &groups[len(groups)-1]
			}
			currentGrp.Matches = append(currentGrp.Matches, m)
		}

		sessionNumber := 1
		for _, grp := range groups {
			matchCount := len(grp.Matches)
			for i, m := range grp.Matches {
				// Page break check (need room for at least 1 match)
				if pdf.GetY()+cellH > 270 {
					// close Date/Session cell border
					pdf.Line(10, pdf.GetY(), 35, pdf.GetY())
					pdf.AddPage()
					printOrisPdfHeader(pdf, ev, "C58", "DETAILED COMPETITION SCHEDULE")
					renderTableHeader()
				}

				rowY := pdf.GetY()
				pdf.SetFont("Arial", "", 7)
				pdf.SetTextColor(0, 0, 0)
				pdf.SetDrawColor(0, 0, 0)

				// 1. Date/Session Column (25mm)
				if i == 0 || rowY == 53 {
					contText := ""
					if i > 0 {
						contText = " (Cont.)"
					}
					pdf.SetFont("Arial", "B", 7)
					pdf.SetXY(10, rowY+0.5)
					pdf.CellFormat(25, 3.2, grp.DateDisplay+contText, "", 1, "L", false, 0, "")
					pdf.SetXY(10, rowY+3.7)
					pdf.CellFormat(25, 3.2, fmt.Sprintf("Session %d", sessionNumber), "", 1, "L", false, 0, "")
					pdf.Rect(10, rowY, 25, cellH, "D")
				} else {
					pdf.Line(10, rowY, 10, rowY+cellH)
					pdf.Line(35, rowY, 35, rowY+cellH)
				}

				// If last match of the session, close bottom border
				if i == matchCount-1 {
					pdf.Line(10, rowY+cellH, 35, rowY+cellH)
				}

				// 2. Match No (7mm)
				pdf.SetFont("Arial", "", 7.5)
				pdf.SetXY(35, rowY)
				matchNoStr := fmt.Sprintf("%d", m.MatchNo)
				if m.BoardNumber.Valid && m.BoardNumber.Int64 > 0 {
					matchNoStr = fmt.Sprintf("%d", m.BoardNumber.Int64)
				}
				pdf.CellFormat(7, cellH, matchNoStr, "1", 0, "C", false, 0, "")

				// 3. Start Time (9mm)
				timeStr := "-"
				if m.ScheduledAt.Valid {
					timeStr = m.ScheduledAt.Time.Format("15:04")
				}
				pdf.CellFormat(9, cellH, timeStr, "1", 0, "C", false, 0, "")

				// 4. Event (30mm)
				eventName := m.CategoryName
				if len(eventName) > 22 {
					eventName = eventName[:20] + ".."
				}
				pdf.CellFormat(30, cellH, eventName, "1", 0, "L", false, 0, "")

				// 5. Round (9mm)
				phaseCode := getIanseoPhaseName(m.BracketSize, m.RoundNo, m.MatchNo)
				pdf.CellFormat(9, cellH, phaseCode, "1", 0, "L", false, 0, "")

				// 6. R.R. Rank 1 (10mm)
				seedA := ""
				if m.SeedA.Valid && m.SeedA.Int64 > 0 {
					seedA = fmt.Sprintf("%d", m.SeedA.Int64)
				}
				pdf.CellFormat(10, cellH, seedA, "1", 0, "R", false, 0, "")

				// 7. Participant 1: Name (37mm) + NOC (8mm)
				nameA := "Bye / TBD"
				if m.NameA.Valid && m.NameA.String != "" {
					nameA = m.NameA.String
				}
				if len(nameA) > 23 {
					nameA = nameA[:21] + ".."
				}
				clubCodeA := ""
				if m.ClubA.Valid && m.ClubA.String != "" {
					clubCodeA = generateNocCode(m.ClubA.String)
				}
				pdf.CellFormat(37, cellH, nameA, "1", 0, "L", false, 0, "")
				pdf.CellFormat(8, cellH, clubCodeA, "1", 0, "L", false, 0, "")

				// 8. R.R. Rank 2 (10mm)
				seedB := ""
				if m.SeedB.Valid && m.SeedB.Int64 > 0 {
					seedB = fmt.Sprintf("%d", m.SeedB.Int64)
				}
				pdf.CellFormat(10, cellH, seedB, "1", 0, "R", false, 0, "")

				// 9. Participant 2: Name (37mm) + NOC (8mm)
				nameB := "Bye / TBD"
				if m.NameB.Valid && m.NameB.String != "" {
					nameB = m.NameB.String
				}
				if len(nameB) > 23 {
					nameB = nameB[:21] + ".."
				}
				clubCodeB := ""
				if m.ClubB.Valid && m.ClubB.String != "" {
					clubCodeB = generateNocCode(m.ClubB.String)
				}
				pdf.CellFormat(37, cellH, nameB, "1", 0, "L", false, 0, "")
				pdf.CellFormat(8, cellH, clubCodeB, "1", 1, "L", false, 0, "")
			}
			sessionNumber++
		}

		setPdfHeaders(c, fmt.Sprintf("EliminationSchedule-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output elimination schedule PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 11. ENTRIES BY CLUB / COUNTRY (ORIS C30)
// ─────────────────────────────────────────────────────────────────────────────

type ClubEntrySummary struct {
	ClubName   string `db:"club_name"`
	MenCount   int    `db:"men_count"`
	WomenCount int    `db:"women_count"`
	TotalCount int    `db:"total_count"`
}

type ClubAthleteDetail struct {
	ClubName     string         `db:"club_name"`
	AthleteName  string         `db:"athlete_name"`
	BirthDate    sql.NullString `db:"birth_date"`
	AthleteCode  sql.NullString `db:"athlete_code"`
	CategoryName string         `db:"category_name"`
	Gender       string         `db:"gender"`
	TargetName   string         `db:"target_name"`
}

func GetEntriesByClubPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// 1. Summary by Club
		var summaries []ClubEntrySummary
		_ = db.Select(&summaries, `
			SELECT
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				SUM(CASE WHEN NOT (LOWER(rgd.name) LIKE '%putri%' OR LOWER(rgd.name) LIKE '%women%' OR LOWER(rgd.name) LIKE '%woman%' OR LOWER(rgd.name) LIKE '%girl%' OR LOWER(a.gender) = 'female') THEN 1 ELSE 0 END) AS men_count,
				SUM(CASE WHEN LOWER(rgd.name) LIKE '%putri%' OR LOWER(rgd.name) LIKE '%women%' OR LOWER(rgd.name) LIKE '%woman%' OR LOWER(rgd.name) LIKE '%girl%' OR LOWER(a.gender) = 'female' THEN 1 ELSE 0 END) AS women_count,
				COUNT(ep.uuid) AS total_count
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ep.tournament_id = ?
			GROUP BY cl.name
			ORDER BY cl.name ASC
		`, ev.UUID)

		// Ensure mathematical consistency
		for i := range summaries {
			if summaries[i].MenCount+summaries[i].WomenCount < summaries[i].TotalCount {
				summaries[i].MenCount = summaries[i].TotalCount - summaries[i].WomenCount
			}
		}

		// 2. Athlete details per club
		var athletes []ClubAthleteDetail
		_ = db.Select(&athletes, `
			SELECT
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				a.full_name AS athlete_name,
				COALESCE(CAST(a.date_of_birth AS CHAR), CAST(a.birth_date AS CHAR), '') AS birth_date,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(rgd.name, '-') AS gender,
				COALESCE(et.target_name, ep.target_name, '-') AS target_name
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN tournament_targets et ON qta.target_uuid = et.uuid
			WHERE ep.tournament_id = ?
			ORDER BY cl.name ASC, a.full_name ASC
		`, ev.UUID)

		docCode := "C30"
		pdf := setupOrisPdf(docCode)
		pdf.AddPage()
		printOrisPdfHeaderDetailed(pdf, ev, docCode, "NUMBER OF ENTRIES BY COUNTRY/CLUB", "", "Entries")

		// 1. Summary Table (Ianseo ORIS Standard)
		pdf.SetFillColor(245, 245, 245)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetLineWidth(0.1)
		pdf.SetFont("Arial", "B", 8)

		pdf.CellFormat(16, 6.5, "Code", "1", 0, "C", true, 0, "")
		pdf.CellFormat(74, 6.5, "Country / Club Name", "1", 0, "L", true, 0, "")
		pdf.CellFormat(22, 6.5, "Men", "1", 0, "C", true, 0, "")
		pdf.CellFormat(22, 6.5, "Women", "1", 0, "C", true, 0, "")
		pdf.CellFormat(28, 6.5, "Total Competitors", "1", 0, "C", true, 0, "")
		pdf.CellFormat(14, 6.5, "Officials", "1", 0, "C", true, 0, "")
		pdf.CellFormat(14, 6.5, "Total", "1", 1, "C", true, 0, "")

		totalM, totalW, totalAll := 0, 0, 0
		pdf.SetFont("Arial", "", 8)
		for i, s := range summaries {
			if pdf.GetY() > 265 {
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, docCode, "NUMBER OF ENTRIES BY COUNTRY/CLUB", "", "Entries")
				pdf.SetFillColor(245, 245, 245)
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(16, 6.5, "Code", "1", 0, "C", true, 0, "")
				pdf.CellFormat(74, 6.5, "Country / Club Name", "1", 0, "L", true, 0, "")
				pdf.CellFormat(22, 6.5, "Men", "1", 0, "C", true, 0, "")
				pdf.CellFormat(22, 6.5, "Women", "1", 0, "C", true, 0, "")
				pdf.CellFormat(28, 6.5, "Total Competitors", "1", 0, "C", true, 0, "")
				pdf.CellFormat(14, 6.5, "Officials", "1", 0, "C", true, 0, "")
				pdf.CellFormat(14, 6.5, "Total", "1", 1, "C", true, 0, "")
				pdf.SetFont("Arial", "", 8)
			}

			totalM += s.MenCount
			totalW += s.WomenCount
			totalAll += s.TotalCount

			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(250, 250, 250)
			}
			noc := generateNocCode(s.ClubName)
			pdf.CellFormat(16, 5.5, noc, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(74, 5.5, s.ClubName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(22, 5.5, fmt.Sprintf("%d", s.MenCount), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(22, 5.5, fmt.Sprintf("%d", s.WomenCount), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(28, 5.5, fmt.Sprintf("%d", s.TotalCount), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(14, 5.5, "-", "1", 0, "C", fill, 0, "")
			pdf.CellFormat(14, 5.5, fmt.Sprintf("%d", s.TotalCount), "1", 1, "C", fill, 0, "")
		}

		// Total Row (Ianseo ORIS Standard)
		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(240, 240, 240)
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(90, 6.5, fmt.Sprintf(" Total: %d Clubs", len(summaries)), "1", 0, "L", true, 0, "")
		pdf.CellFormat(22, 6.5, fmt.Sprintf("%d", totalM), "1", 0, "C", true, 0, "")
		pdf.CellFormat(22, 6.5, fmt.Sprintf("%d", totalW), "1", 0, "C", true, 0, "")
		pdf.CellFormat(28, 6.5, fmt.Sprintf("%d", totalAll), "1", 0, "C", true, 0, "")
		pdf.CellFormat(14, 6.5, "-", "1", 0, "C", true, 0, "")
		pdf.CellFormat(14, 6.5, fmt.Sprintf("%d", totalAll), "1", 1, "C", true, 0, "")

		// 2. Athlete Dossier / Entries List per Club (Ianseo ORIS C30 Second Section)
		if len(athletes) > 0 {
			pdf.AddPage()
			printOrisPdfHeaderDetailed(pdf, ev, docCode, "ENTRIES BY COUNTRY/CLUB", "", "Entries")

			renderEntriesHeader := func(y float64) {
				pdf.SetLineWidth(0.1)
				pdf.SetDrawColor(0, 0, 0)
				pdf.Rect(10, y-1, 190, 5.5, "D")
				pdf.SetFont("Arial", "B", 8)
				pdf.SetTextColor(0, 0, 0)
				pdf.SetXY(10, y)
				pdf.CellFormat(12, 3.5, "NOC", "", 0, "L", false, 0, "")
				pdf.CellFormat(45, 3.5, "Country", "", 0, "L", false, 0, "")
				pdf.CellFormat(48, 3.5, "Name", "", 0, "L", false, 0, "")
				pdf.CellFormat(13, 3.5, "W. Rank", "", 0, "R", false, 0, "")
				pdf.CellFormat(24, 3.5, "Date of Birth", "", 0, "R", false, 0, "")
				pdf.CellFormat(14, 3.5, "Back No.", "", 0, "R", false, 0, "")
				pdf.CellFormat(34, 3.5, "Event", "", 1, "L", false, 0, "")
				pdf.SetY(y + 5.5)
			}

			renderEntriesHeader(41.0)

			lastClub := ""
			for _, ath := range athletes {
				if pdf.GetY() > 265 {
					pdf.AddPage()
					printOrisPdfHeaderDetailed(pdf, ev, docCode, "ENTRIES BY COUNTRY/CLUB", "", "Entries")
					renderEntriesHeader(41.0)
					lastClub = ""
				}

				noc := generateNocCode(ath.ClubName)
				dobStr := formatDOB(ath.BirthDate.String)
				targetNo := strings.TrimLeft(strings.TrimSpace(ath.TargetName), "0")
				if targetNo == "-" {
					targetNo = ""
				}

				if lastClub != ath.ClubName {
					if lastClub != "" {
						pdf.SetY(pdf.GetY() + 3.5)
					}
					lastClub = ath.ClubName

					pdf.SetFont("Arial", "", 8)
					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(10, pdf.GetY())
					pdf.CellFormat(12, 3.5, noc, "", 0, "L", false, 0, "")
					pdf.CellFormat(45, 3.5, ath.ClubName, "", 0, "L", false, 0, "")
				} else {
					pdf.SetFont("Arial", "", 8)
					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(10, pdf.GetY())
					pdf.CellFormat(12, 3.5, "", "", 0, "L", false, 0, "")
					pdf.CellFormat(45, 3.5, "", "", 0, "L", false, 0, "")
				}

				pdf.CellFormat(48, 3.5, strings.ToUpper(ath.AthleteName), "", 0, "L", false, 0, "")
				pdf.CellFormat(13, 3.5, "", "", 0, "R", false, 0, "")
				pdf.CellFormat(24, 3.5, dobStr+"  ", "", 0, "R", false, 0, "")
				pdf.CellFormat(14, 3.5, targetNo+" ", "", 0, "R", false, 0, "")
				pdf.CellFormat(34, 3.5, ath.CategoryName, "", 1, "L", false, 0, "")
			}
		}

		setPdfHeaders(c, fmt.Sprintf("%s-entries-by-club-C30.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output entries by club PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 12. ELIMINATION START LIST BY TARGET (ORIS C51A)
// ─────────────────────────────────────────────────────────────────────────────

func GetEliminationStartListPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		bracketID := c.Query("bracket_id")
		categoryID := c.Query("category_id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				em.uuid AS match_uuid,
				eb.uuid AS bracket_uuid,
				COALESCE(eb.bracket_size, 8) AS bracket_size,
				COALESCE(eb.bracket_type, 'individual') AS bracket_type,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				em.round_no,
				em.match_no,
				em.scheduled_at,
				COALESCE(et.target_name, '-') AS target_name,
				COALESCE(et.board_number, 0) AS board_number,
				em.status,
				COALESCE(eeA.seed, 0) AS seed_a,
				CASE
					WHEN eeA.participant_type = 'team' THEN tA.team_name
					WHEN eeA.participant_type = 'archer' THEN aA.full_name
					ELSE ''
				END AS name_a,
				CASE
					WHEN eeA.participant_type = 'team' THEN COALESCE(cA_team.name, 'Individu')
					ELSE COALESCE(cA.name, 'Individu')
				END AS club_a,
				COALESCE(eeB.seed, 0) AS seed_b,
				CASE
					WHEN eeB.participant_type = 'team' THEN tB.team_name
					WHEN eeB.participant_type = 'archer' THEN aB.full_name
					ELSE ''
				END AS name_b,
				CASE
					WHEN eeB.participant_type = 'team' THEN COALESCE(cB_team.name, 'Individu')
					ELSE COALESCE(cB.name, 'Individu')
				END AS club_b
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN tournament_targets et ON em.target_uuid = et.uuid
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN clubs cA ON aA.club_id = cA.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN clubs cA_team ON tA.event_id = cA_team.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN clubs cB ON aB.club_id = cB.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			LEFT JOIN clubs cB_team ON tB.event_id = cB_team.uuid
			WHERE eb.tournament_uuid = ? AND (em.is_bye = 0 OR em.is_bye IS NULL)
		`

		var args []interface{}
		args = append(args, ev.UUID)

		if bracketID != "" {
			query += " AND (eb.uuid = ? OR eb.bracket_id = ?)"
			args = append(args, bracketID, bracketID)
		}
		if categoryID != "" {
			query += " AND (eb.category_uuid = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}

		query += " ORDER BY COALESCE(et.board_number, 9999) ASC, em.round_no ASC, em.match_no ASC"

		var matches []EliminationScheduleRow
		_ = db.Select(&matches, query, args...)

		pdf := setupOrisPdf("C51A")
		pdf.AddPage()
		printOrisPdfHeader(pdf, ev, "C51A", "START LIST BY TARGET (MATCH TARGET ALLOCATIONS)")

		renderHeader := func() {
			pdf.SetFillColor(241, 245, 249)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetDrawColor(203, 213, 225)
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(16, 7, "Target", "1", 0, "C", true, 0, "")
			pdf.CellFormat(20, 7, "Time", "1", 0, "C", true, 0, "")
			pdf.CellFormat(40, 7, "Category / Event", "1", 0, "L", true, 0, "")
			pdf.CellFormat(28, 7, "Phase", "1", 0, "L", true, 0, "")
			pdf.CellFormat(43, 7, "Athlete A", "1", 0, "L", true, 0, "")
			pdf.CellFormat(43, 7, "Athlete B", "1", 1, "L", true, 0, "")
		}

		renderHeader()

		for i, m := range matches {
			if pdf.GetY() > 265 {
				pdf.AddPage()
				printOrisPdfHeader(pdf, ev, "C51A", "START LIST BY TARGET (MATCH TARGET ALLOCATIONS)")
				renderHeader()
			}

			timeStr := "-"
			if m.ScheduledAt.Valid {
				timeStr = m.ScheduledAt.Time.Format("15:04")
			}

			p1 := "BYE / TBD"
			if m.NameA.Valid && m.NameA.String != "" {
				seedStr := ""
				if m.SeedA.Valid && m.SeedA.Int64 > 0 {
					seedStr = fmt.Sprintf("[%d] ", m.SeedA.Int64)
				}
				p1 = fmt.Sprintf("%s%s", seedStr, m.NameA.String)
			}

			p2 := "BYE / TBD"
			if m.NameB.Valid && m.NameB.String != "" {
				seedStr := ""
				if m.SeedB.Valid && m.SeedB.Int64 > 0 {
					seedStr = fmt.Sprintf("[%d] ", m.SeedB.Int64)
				}
				p2 = fmt.Sprintf("%s%s", seedStr, m.NameB.String)
			}

			phaseName := getIanseoPhaseName(m.BracketSize, m.RoundNo, m.MatchNo)

			fill := i%2 == 1
			if fill {
				pdf.SetFillColor(250, 250, 250)
			}
			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(0, 0, 0)

			targetStr := "-"
			if m.TargetName.Valid && m.TargetName.String != "" {
				targetStr = m.TargetName.String
			}

			pdf.CellFormat(16, 6.5, targetStr, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(20, 6.5, timeStr, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(40, 6.5, m.CategoryName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(28, 6.5, phaseName, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(43, 6.5, p1, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(43, 6.5, p2, "1", 1, "L", fill, 0, "")
		}

		if len(matches) > 0 {
			printOrisSignatures(pdf, pdf.GetY())
		}

		setPdfHeaders(c, fmt.Sprintf("EliminationStartList-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output elimination start list PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 13. ELIMINATION BRACKET VECTOR PDF (ORIS C75A / C75C)
// ─────────────────────────────────────────────────────────────────────────────

type BracketVectorMatch struct {
	MatchUUID   string         `db:"match_uuid"`
	MatchNo     int            `db:"match_no"`
	RoundNo     int            `db:"round_no"`
	RoundName   string         `db:"round_name"`
	TargetNo    sql.NullInt64  `db:"target_number"`
	TargetName  sql.NullString `db:"target_name"`
	Status      string         `db:"status"`
	PointsA     int            `db:"total_points_a"`
	PointsB     int            `db:"total_points_b"`
	ScoreA      int            `db:"total_score_a"`
	ScoreB      int            `db:"total_score_b"`
	WinnerUUID  sql.NullString `db:"winner_entry_uuid"`
	EntryAUUID  sql.NullString `db:"entry_a_uuid"`
	EntryBUUID  sql.NullString `db:"entry_b_uuid"`
	SeedA       sql.NullInt64  `db:"seed_a"`
	SeedB       sql.NullInt64  `db:"seed_b"`
	NameA       sql.NullString `db:"name_a"`
	ClubA       sql.NullString `db:"club_a"`
	NameB       sql.NullString `db:"name_b"`
	ClubB       sql.NullString `db:"club_b"`
}

type BracketCategoryInfo struct {
	BracketUUID  string `db:"bracket_uuid"`
	CategoryUUID string `db:"category_uuid"`
	CategoryName string `db:"category_name"`
	BracketType  string `db:"bracket_type"` // individual, team3, team2
	BracketSize  int    `db:"bracket_size"`
}

func GetEliminationBracketVectorPDF(db *sqlx.DB) gin.HandlerFunc {
	return GetEliminationBracketsPrintout(db)
}

func GetEliminationBracketsPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// 1. Fetch brackets to print
		bQuery := `
			SELECT 
				eb.uuid AS bracket_uuid,
				eb.category_uuid,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, 'Category') AS category_name,
				eb.bracket_type,
				eb.bracket_size
			FROM elimination_brackets eb
			JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE eb.tournament_uuid = ?
		`
		var bArgs []interface{}
		bArgs = append(bArgs, ev.UUID)
		if categoryID != "" {
			bQuery += " AND (eb.category_uuid = ? OR ec.uuid = ?)"
			bArgs = append(bArgs, categoryID, categoryID)
		}
		bQuery += " ORDER BY category_name ASC"

		var brackets []BracketCategoryInfo
		_ = db.Select(&brackets, bQuery, bArgs...)
		if len(brackets) == 0 {
			brackets = append(brackets, BracketCategoryInfo{
				CategoryName: "Bagan Eliminasi",
			})
		}

		pdf := newOrisPdf("L") // Landscape 297 x 210 mm

		loc := ev.Venue.String
		if ev.City.Valid && ev.City.String != "" {
			if loc != "" {
				loc += ", "
			}
			loc += ev.City.String
		}
		dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)

		for _, b := range brackets {
			pdf.AddPage()

			docCode := "C75A"
			docTitle := "Elimination & Final Round - Individual Brackets"
			if b.BracketType == "team" || b.BracketType == "mix_team" {
				docCode = "C75C"
				docTitle = "Elimination & Final Round - Team Brackets"
			}

			// 1. Clean IanSeo Landscape Header
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.2)

			pdf.SetFont("Arial", "B", 11)
			pdf.SetXY(10, 10)
			pdf.CellFormat(160, 5, ev.Name, "", 0, "L", false, 0, "")

			pdf.SetFont("Arial", "B", 11)
			pdf.SetXY(175, 10)
			pdf.CellFormat(112, 5, b.CategoryName, "", 1, "R", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(70, 70, 70)
			pdf.SetXY(10, 15.5)
			pdf.CellFormat(160, 4, fmt.Sprintf("%s | %s", loc, dateStr), "", 0, "L", false, 0, "")

			pdf.SetXY(175, 15.5)
			pdf.CellFormat(112, 4, fmt.Sprintf("[%s] %s", docCode, docTitle), "", 1, "R", false, 0, "")

			pdf.Line(10, 21, 287, 21)

			// Fetch Matches for this bracket
			var matches []BracketVectorMatch
			if b.BracketUUID != "" {
				_ = db.Select(&matches, `
					SELECT
						em.uuid AS match_uuid, em.bracket_uuid, em.round_no, em.match_no,
						COALESCE(eeA.seed, 0) AS seed_a,
						CASE
							WHEN eeA.participant_type = 'team' THEN tA.team_name
							WHEN eeA.participant_type = 'archer' THEN aA.full_name
							ELSE ''
						END AS name_a,
						CASE
							WHEN eeA.participant_type = 'team' THEN COALESCE(cA_team.name, 'Individu')
							ELSE COALESCE(cA.name, 'Individu')
						END AS club_a,
						COALESCE(em.total_score_a, 0) AS total_score_a,
						COALESCE(em.total_points_a, 0) AS total_points_a,
						COALESCE(eeB.seed, 0) AS seed_b,
						CASE
							WHEN eeB.participant_type = 'team' THEN tB.team_name
							WHEN eeB.participant_type = 'archer' THEN aB.full_name
							ELSE ''
						END AS name_b,
						CASE
							WHEN eeB.participant_type = 'team' THEN COALESCE(cB_team.name, 'Individu')
							ELSE COALESCE(cB.name, 'Individu')
						END AS club_b,
						COALESCE(em.total_score_b, 0) AS total_score_b,
						COALESCE(em.total_points_b, 0) AS total_points_b,
						em.winner_entry_uuid, em.entry_a_uuid, em.entry_b_uuid, em.status
					FROM elimination_matches em
					LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
					LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
					LEFT JOIN clubs cA ON aA.club_id = cA.uuid
					LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
					LEFT JOIN clubs cA_team ON tA.event_id = cA_team.uuid
					LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
					LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
					LEFT JOIN clubs cB ON aB.club_id = cB.uuid
					LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
					LEFT JOIN clubs cB_team ON tB.event_id = cB_team.uuid
					WHERE em.bracket_uuid = ?
					ORDER BY em.round_no ASC, em.match_no ASC
				`, b.BracketUUID)
			}

			// Map matches by round & match_no
			matchMap := make(map[string]BracketVectorMatch)
			for _, m := range matches {
				key := fmt.Sprintf("%d-%d", m.RoundNo, m.MatchNo)
				matchMap[key] = m
			}

			bracketSize := b.BracketSize
			if bracketSize < 2 {
				bracketSize = 8
			}

			totalRounds := 1
			for sz := bracketSize; sz > 2; sz /= 2 {
				totalRounds++
			}

			roundLabels := []string{}
			if bracketSize >= 16 {
				roundLabels = append(roundLabels, "1/8 Elimination")
			}
			if bracketSize >= 8 {
				roundLabels = append(roundLabels, "Quarterfinals")
			}
			if bracketSize >= 4 {
				roundLabels = append(roundLabels, "Semifinals")
			}
			roundLabels = append(roundLabels, "Finals")
			roundLabels = append(roundLabels, "Medallists")

			colWidth := 277.0 / float64(len(roundLabels))
			xStart := 10.0
			yTop := 24.5

			// 2. Render Round Header Ribbon
			pdf.SetFont("Arial", "B", 7.5)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.1)

			for colIdx, label := range roundLabels {
				x := xStart + float64(colIdx)*colWidth
				pdf.SetFillColor(245, 245, 245)
				pdf.Rect(x, yTop, colWidth-2, 5.5, "FD")
				pdf.SetXY(x, yTop+1)
				pdf.CellFormat(colWidth-2, 3.5, label, "", 0, "C", false, 0, "")
			}

			// 3. Draw Vector Bracket Tree Lines
			isSetSystem := strings.Contains(strings.ToLower(b.BracketType), "recurve") || strings.Contains(strings.ToLower(b.CategoryName), "recurve")
			yTreeTop := yTop + 8.0
			treeH := 140.0

			var goldWinner, silverWinner, bronzeWinner, fourthPlace string
			var goldClub, silverClub, bronzeClub, fourthClub string

			for r := 1; r <= totalRounds; r++ {
				colIdx := r - 1
				matchesInRound := bracketSize / (1 << r)
				if matchesInRound < 1 {
					matchesInRound = 1
				}

				spacing := treeH / float64(matchesInRound)
				x := xStart + float64(colIdx)*colWidth
				lineW := colWidth - 8.0

				for mIdx := 1; mIdx <= matchesInRound; mIdx++ {
					yCenter := yTreeTop + (float64(mIdx)-0.5)*spacing
					ySlotA := yCenter - spacing*0.25
					ySlotB := yCenter + spacing*0.25

					mKey := fmt.Sprintf("%d-%d", r, mIdx)
					m, hasMatch := matchMap[mKey]

					nameA := "BYE / TBD"
					clubA := ""
					scoreA := ""
					seedA := ""
					if hasMatch && m.NameA.Valid && m.NameA.String != "" {
						nameA = strings.ToUpper(m.NameA.String)
						clubA = generateNocCode(m.ClubA.String)
						if m.SeedA.Valid && m.SeedA.Int64 > 0 {
							seedA = fmt.Sprintf("%d", m.SeedA.Int64)
						}
						if m.Status == "finished" {
							if isSetSystem {
								scoreA = fmt.Sprintf("%d", m.PointsA)
							} else {
								scoreA = fmt.Sprintf("%d", m.ScoreA)
							}
						}
					}

					nameB := "BYE / TBD"
					clubB := ""
					scoreB := ""
					seedB := ""
					if hasMatch && m.NameB.Valid && m.NameB.String != "" {
						nameB = strings.ToUpper(m.NameB.String)
						clubB = generateNocCode(m.ClubB.String)
						if m.SeedB.Valid && m.SeedB.Int64 > 0 {
							seedB = fmt.Sprintf("%d", m.SeedB.Int64)
						}
						if m.Status == "finished" {
							if isSetSystem {
								scoreB = fmt.Sprintf("%d", m.PointsB)
							} else {
								scoreB = fmt.Sprintf("%d", m.ScoreB)
							}
						}
					}

					if len(nameA) > 18 {
						nameA = nameA[:16] + ".."
					}
					if len(nameB) > 18 {
						nameB = nameB[:16] + ".."
					}

					winA := hasMatch && m.WinnerUUID.Valid && m.EntryAUUID.Valid && m.WinnerUUID.String == m.EntryAUUID.String
					winB := hasMatch && m.WinnerUUID.Valid && m.EntryBUUID.Valid && m.WinnerUUID.String == m.EntryBUUID.String

					// Track Finals Winner for Medals Box
					if r == totalRounds && hasMatch && m.Status == "finished" {
						if winA {
							goldWinner = m.NameA.String
							goldClub = m.ClubA.String
							silverWinner = m.NameB.String
							silverClub = m.ClubB.String
						} else if winB {
							goldWinner = m.NameB.String
							goldClub = m.ClubB.String
							silverWinner = m.NameA.String
							silverClub = m.ClubA.String
						}
					}

					pdf.SetDrawColor(0, 0, 0)
					pdf.SetLineWidth(0.15)

					// Line A (Upper slot)
					pdf.Line(x, ySlotA, x+lineW, ySlotA)
					if winA {
						pdf.SetFont("Arial", "B", 7)
					} else {
						pdf.SetFont("Arial", "", 7)
					}
					txtA := nameA
					if seedA != "" {
						txtA = fmt.Sprintf("%s. %s", seedA, nameA)
					}
					if clubA != "" {
						txtA = fmt.Sprintf("%s [%s]", txtA, clubA)
					}
					pdf.SetXY(x, ySlotA-3.5)
					pdf.CellFormat(lineW-8, 3.5, txtA, "", 0, "L", false, 0, "")
					pdf.SetFont("Arial", "B", 7)
					pdf.CellFormat(8, 3.5, scoreA, "", 0, "R", false, 0, "")

					// Line B (Lower slot)
					pdf.Line(x, ySlotB, x+lineW, ySlotB)
					if winB {
						pdf.SetFont("Arial", "B", 7)
					} else {
						pdf.SetFont("Arial", "", 7)
					}
					txtB := nameB
					if seedB != "" {
						txtB = fmt.Sprintf("%s. %s", seedB, nameB)
					}
					if clubB != "" {
						txtB = fmt.Sprintf("%s [%s]", txtB, clubB)
					}
					pdf.SetXY(x, ySlotB-3.5)
					pdf.CellFormat(lineW-8, 3.5, txtB, "", 0, "L", false, 0, "")
					pdf.SetFont("Arial", "B", 7)
					pdf.CellFormat(8, 3.5, scoreB, "", 0, "R", false, 0, "")

					// Right connector bracket
					xRight := x + lineW
					pdf.Line(xRight, ySlotA, xRight, ySlotB)

					// Branch leading to next round
					xNext := xStart + float64(r)*colWidth
					pdf.Line(xRight, yCenter, xNext, yCenter)
				}
			}

			// 4. Bronze Match & Medallists Summary Box (Right Columns)
			xFinalCol := xStart + float64(totalRounds-1)*colWidth
			xMedalsCol := xStart + float64(totalRounds)*colWidth
			yBronzeBox := yTreeTop + 105.0

			// Bronze Match Box
			if bracketSize >= 4 {
				pdf.SetDrawColor(0, 0, 0)
				pdf.SetLineWidth(0.15)
				pdf.SetFillColor(245, 245, 245)
				pdf.Rect(xFinalCol, yBronzeBox, colWidth-4, 4.5, "FD")
				pdf.SetFont("Arial", "B", 7)
				pdf.SetXY(xFinalCol, yBronzeBox+0.5)
				pdf.CellFormat(colWidth-4, 3.5, "BRONZE MEDAL MATCH", "", 0, "C", false, 0, "")

				bKey := fmt.Sprintf("%d-%d", totalRounds, bracketSize)
				bMatch, hasBronze := matchMap[bKey]

				bNameA := "BYE / TBD"
				bClubA := ""
				bScoreA := ""
				bNameB := "BYE / TBD"
				bClubB := ""
				bScoreB := ""
				if hasBronze {
					if bMatch.NameA.Valid && bMatch.NameA.String != "" {
						bNameA = strings.ToUpper(bMatch.NameA.String)
						bClubA = generateNocCode(bMatch.ClubA.String)
					}
					if bMatch.NameB.Valid && bMatch.NameB.String != "" {
						bNameB = strings.ToUpper(bMatch.NameB.String)
						bClubB = generateNocCode(bMatch.ClubB.String)
					}
					if bMatch.Status == "finished" {
						if isSetSystem {
							bScoreA = fmt.Sprintf("%d", bMatch.PointsA)
							bScoreB = fmt.Sprintf("%d", bMatch.PointsB)
						} else {
							bScoreA = fmt.Sprintf("%d", bMatch.ScoreA)
							bScoreB = fmt.Sprintf("%d", bMatch.ScoreB)
						}
						winBA := bMatch.WinnerUUID.Valid && bMatch.EntryAUUID.Valid && bMatch.WinnerUUID.String == bMatch.EntryAUUID.String
						winBB := bMatch.WinnerUUID.Valid && bMatch.EntryBUUID.Valid && bMatch.WinnerUUID.String == bMatch.EntryBUUID.String
						if winBA {
							bronzeWinner = bMatch.NameA.String
							bronzeClub = bMatch.ClubA.String
							fourthPlace = bMatch.NameB.String
							fourthClub = bMatch.ClubB.String
						} else if winBB {
							bronzeWinner = bMatch.NameB.String
							bronzeClub = bMatch.ClubB.String
							fourthPlace = bMatch.NameA.String
							fourthClub = bMatch.ClubA.String
						}
					}
				}

				if len(bNameA) > 16 {
					bNameA = bNameA[:14] + ".."
				}
				if len(bNameB) > 16 {
					bNameB = bNameB[:14] + ".."
				}

				yB1 := yBronzeBox + 10.0
				yB2 := yBronzeBox + 18.0
				bLineW := colWidth - 8.0

				pdf.Line(xFinalCol, yB1, xFinalCol+bLineW, yB1)
				txtBA := bNameA
				if bClubA != "" {
					txtBA = fmt.Sprintf("%s [%s]", bNameA, bClubA)
				}
				pdf.SetFont("Arial", "", 7)
				pdf.SetXY(xFinalCol, yB1-3.5)
				pdf.CellFormat(bLineW-8, 3.5, txtBA, "", 0, "L", false, 0, "")
				pdf.SetFont("Arial", "B", 7)
				pdf.CellFormat(8, 3.5, bScoreA, "", 0, "R", false, 0, "")

				pdf.Line(xFinalCol, yB2, xFinalCol+bLineW, yB2)
				txtBB := bNameB
				if bClubB != "" {
					txtBB = fmt.Sprintf("%s [%s]", bNameB, bClubB)
				}
				pdf.SetFont("Arial", "", 7)
				pdf.SetXY(xFinalCol, yB2-3.5)
				pdf.CellFormat(bLineW-8, 3.5, txtBB, "", 0, "L", false, 0, "")
				pdf.SetFont("Arial", "B", 7)
				pdf.CellFormat(8, 3.5, bScoreB, "", 0, "R", false, 0, "")

				// Connector
				pdf.Line(xFinalCol+bLineW, yB1, xFinalCol+bLineW, yB2)
				pdf.Line(xFinalCol+bLineW, (yB1+yB2)/2, xFinalCol+colWidth-2, (yB1+yB2)/2)
			}

			// 5. Official Medallists Summary Box (Far Right Column)
			yMedalsBox := yTreeTop
			medalsBoxW := colWidth - 2.0
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.15)
			pdf.SetFillColor(245, 245, 245)
			pdf.Rect(xMedalsCol, yMedalsBox, medalsBoxW, 5.5, "FD")
			pdf.SetFont("Arial", "B", 7.5)
			pdf.SetXY(xMedalsCol, yMedalsBox+1)
			pdf.CellFormat(medalsBoxW, 3.5, "MEDALLISTS", "", 0, "C", false, 0, "")

			renderMedalRow := func(yRow float64, medalCode, medalLabel, winnerName, clubName string) {
				pdf.SetFillColor(250, 250, 250)
				pdf.Rect(xMedalsCol, yRow, medalsBoxW, 14, "D")

				pdf.SetFont("Arial", "B", 7.5)
				pdf.SetXY(xMedalsCol+2, yRow+1.5)
				pdf.CellFormat(medalsBoxW-4, 3.5, fmt.Sprintf("[%s] %s", medalCode, medalLabel), "", 1, "L", false, 0, "")

				wName := strings.ToUpper(winnerName)
				if wName == "" {
					wName = "TBD"
				}
				pdf.SetFont("Arial", "B", 7)
				pdf.SetXY(xMedalsCol+2, yRow+5.5)
				pdf.CellFormat(medalsBoxW-4, 3.5, wName, "", 1, "L", false, 0, "")

				cName := clubName
				if cName == "" {
					cName = "-"
				}
				pdf.SetFont("Arial", "", 6.5)
				pdf.SetTextColor(80, 80, 80)
				pdf.SetXY(xMedalsCol+2, yRow+9.5)
				pdf.CellFormat(medalsBoxW-4, 3.5, cName, "", 1, "L", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
			}

			renderMedalRow(yMedalsBox+7, "GOLD", "Gold Medal", goldWinner, goldClub)
			renderMedalRow(yMedalsBox+23, "SILVER", "Silver Medal", silverWinner, silverClub)
			renderMedalRow(yMedalsBox+39, "BRONZE", "Bronze Medal", bronzeWinner, bronzeClub)
			renderMedalRow(yMedalsBox+55, "4TH", "4th Place", fourthPlace, fourthClub)
		}

		setPdfHeaders(c, fmt.Sprintf("%s-elimination-brackets-C75.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output elimination brackets PDF"})
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 14. OFFICERS & COMPETITION OFFICIALS (ORIS C09)


// ─────────────────────────────────────────────────────────────────────────────
// 15. COMPETITION PROGRAM & SCHEDULE (ORIS C08)
// ─────────────────────────────────────────────────────────────────────────────

type ScheduleItemRow struct {
	Title       string         `db:"title"`
	Description sql.NullString `db:"description"`
	StartTime   sql.NullTime   `db:"start_time"`
	EndTime     sql.NullTime   `db:"end_time"`
	DayOrder    int            `db:"day_order"`
	Location    sql.NullString `db:"location"`
}

func GetEventSchedulePrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type PrintSchedItem struct {
			Title       string         `db:"title"`
			Subtitle    sql.NullString `db:"subtitle"`
			StartTime   string         `db:"start_time"`
			EndTime     string         `db:"end_time"`
			DurationMin int            `db:"duration_minutes"`
			DayOrder    int            `db:"day_number"`
			Location    sql.NullString `db:"location"`
			TargetStart sql.NullInt64  `db:"target_start"`
			TargetEnd   sql.NullInt64  `db:"target_end"`
			DateStr     sql.NullString `db:"schedule_date"`
		}

		var items []PrintSchedItem
		_ = db.Select(&items, `
			SELECT 
				title, subtitle, 
				CAST(start_time AS CHAR) AS start_time, 
				CAST(end_time AS CHAR) AS end_time,
				duration_minutes, day_number, location,
				target_start, target_end,
				COALESCE(CAST(schedule_date AS CHAR), '') AS schedule_date
			FROM tournament_schedule_items
			WHERE tournament_id = ?
			ORDER BY day_number ASC, start_time ASC, sort_order ASC
		`, ev.UUID)

		if len(items) == 0 {
			// Check legacy table
			type LegacySchedRow struct {
				Title     string         `db:"title"`
				StartTime sql.NullTime   `db:"start_time"`
				EndTime   sql.NullTime   `db:"end_time"`
				DayOrder  int            `db:"day_order"`
				Location  sql.NullString `db:"location"`
			}
			var legacyRows []LegacySchedRow
			_ = db.Select(&legacyRows, `
				SELECT title, start_time, end_time, COALESCE(day_order, 1) AS day_order, location
				FROM tournament_schedules
				WHERE tournament_id = ?
				ORDER BY COALESCE(day_order, 1) ASC, start_time ASC
			`, ev.UUID)

			for _, lr := range legacyRows {
				st := "08:00"
				et := "09:00"
				if lr.StartTime.Valid {
					st = lr.StartTime.Time.Format("15:04")
				}
				if lr.EndTime.Valid {
					et = lr.EndTime.Time.Format("15:04")
				}
				items = append(items, PrintSchedItem{
					Title:       lr.Title,
					StartTime:   st,
					EndTime:     et,
					DurationMin: 60,
					DayOrder:    lr.DayOrder,
					Location:    lr.Location,
				})
			}
		}

		if len(items) == 0 {
			// Fallback standard program
			items = []PrintSchedItem{
				{Title: "Official Practice & Equipment Inspection", StartTime: "07:30", EndTime: "08:30", DurationMin: 60, DayOrder: 1, Location: sql.NullString{String: "Main Field", Valid: true}},
				{Title: "Qualification Rounds - Session 1", StartTime: "08:30", EndTime: "11:30", DurationMin: 180, DayOrder: 1, Location: sql.NullString{String: "Main Field", Valid: true}, TargetStart: sql.NullInt64{Int64: 1, Valid: true}, TargetEnd: sql.NullInt64{Int64: 30, Valid: true}},
				{Title: "Individual Elimination Matches (1/16 to Semi-Finals)", StartTime: "08:30", EndTime: "12:15", DurationMin: 225, DayOrder: 2, Location: sql.NullString{String: "Main Field", Valid: true}},
				{Title: "Team Elimination Matches & Medal Matches / Award Ceremony", StartTime: "08:30", EndTime: "15:00", DurationMin: 390, DayOrder: 3, Location: sql.NullString{String: "Main Field & Podium", Valid: true}},
			}
		}

		docCode := "C08"
		docTitle := "Schedule"
		phaseName := "Schedule"

		pdf := setupOrisPdf(docCode)
		pdf.AddPage()
		printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, "", phaseName)

		curY := 36.0
		rowH := 5.0

		// Group items by DayOrder
		dayMap := make(map[int][]PrintSchedItem)
		var days []int
		for _, it := range items {
			if _, ok := dayMap[it.DayOrder]; !ok {
				dayMap[it.DayOrder] = []PrintSchedItem{}
				days = append(days, it.DayOrder)
			}
			dayMap[it.DayOrder] = append(dayMap[it.DayOrder], it)
		}

		for _, d := range days {
			dayItems := dayMap[d]
			neededH := float64(len(dayItems))*rowH + 12
			if curY+neededH > 275 {
				pdf.AddPage()
				printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, "", phaseName)
				curY = 36.0
			}

			// Day Banner
			dateLabel := fmt.Sprintf("Day %d", d)
			if len(dayItems) > 0 && dayItems[0].DateStr.Valid && dayItems[0].DateStr.String != "" {
				t, parseErr := time.Parse("2006-01-02", dayItems[0].DateStr.String)
				if parseErr == nil {
					dateLabel = t.Format("2 Jan 2006, Monday")
				}
			} else if ev.StartDate.Valid {
				t := ev.StartDate.Time.AddDate(0, 0, d-1)
				dateLabel = t.Format("2 Jan 2006, Monday")
			}

			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFillColor(240, 240, 240)
			pdf.SetXY(10, curY)
			pdf.CellFormat(190, 5, fmt.Sprintf("  %s", dateLabel), "", 1, "L", true, 0, "")
			curY += 6.0

			for _, it := range dayItems {
				if curY+rowH > 275 {
					pdf.AddPage()
					printOrisPdfHeaderDetailed(pdf, ev, docCode, docTitle, "", phaseName)
					curY = 36.0
				}

				st := it.StartTime
				if len(st) > 5 {
					st = st[:5]
				}
				et := it.EndTime
				if len(et) > 5 {
					et = et[:5]
				}
				timeStr := fmt.Sprintf("%s-%s", st, et)

				durHours := it.DurationMin / 60
				durMins := it.DurationMin % 60
				durStr := fmt.Sprintf("%02d:%02d", durHours, durMins)

				pdf.SetFont("Arial", "", 8)
				pdf.SetTextColor(0, 0, 0)
				pdf.SetXY(10, curY)
				pdf.CellFormat(22, rowH, timeStr, "", 0, "L", false, 0, "")
				pdf.CellFormat(15, rowH, durStr, "", 0, "C", false, 0, "")

				// Description with target & subtitle
				descText := it.Title
				if it.Subtitle.Valid && it.Subtitle.String != "" {
					descText += fmt.Sprintf(" (%s)", it.Subtitle.String)
				}
				if it.TargetStart.Valid && it.TargetEnd.Valid && it.TargetStart.Int64 > 0 {
					descText += fmt.Sprintf(" [Target %d-%d]", it.TargetStart.Int64, it.TargetEnd.Int64)
				}
				if it.Location.Valid && it.Location.String != "" && it.Location.String != "Main Field" {
					descText += fmt.Sprintf(" @ %s", it.Location.String)
				}

				pdf.CellFormat(153, rowH, descText, "", 0, "L", false, 0, "")

				curY += rowH
			}
			curY += 2.5
		}

		filename := fmt.Sprintf("%s-schedule-%s.pdf", ev.Slug, docCode)
		setPdfHeaders(c, filename)
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output schedule PDF"})
		}
	}
}




