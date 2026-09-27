package handler

import (
	"Archeris-api/models"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/jung-kurt/gofpdf"
)

func formatInvoiceMoney(amount float64, currency string) string {
	if strings.ToUpper(currency) == "USD" {
		return fmt.Sprintf("$ %.2f", amount)
	}
	intVal := int64(amount)
	sign := ""
	if intVal < 0 {
		sign = "-"
		intVal = -intVal
	}
	s := fmt.Sprintf("%d", intVal)
	var res []byte
	n := len(s)
	for i, c := range []byte(s) {
		if i > 0 && (n-i)%3 == 0 {
			res = append(res, '.')
		}
		res = append(res, c)
	}
	return fmt.Sprintf("%sRp %s", sign, string(res))
}

func GenerateInvoicePDF(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")

		type EnrichedTransaction struct {
			models.PaymentTransaction
			Description  string  `json:"description" db:"description"`
			PlanName     *string `json:"plan_name" db:"plan_name"`
			EventName    *string `json:"event_name" db:"event_name"`
			PageSettings *string `json:"page_settings" db:"page_settings"`
			AthleteName  *string `json:"athlete_name" db:"athlete_name"`
			Division     *string `json:"division" db:"division"`
			Category     *string `json:"category" db:"category"`
			UserEmail    string  `db:"user_email"`
			UserName     string  `db:"user_name"`
		}

		var t EnrichedTransaction
		query := `
			SELECT 
				t.*,
				CASE 
					WHEN t.subscription_plan_id IS NOT NULL THEN p.name
					WHEN t.registration_id IS NOT NULL THEN CONCAT('Registrasi Turnamen: ', a.full_name)
					WHEN t.tournament_id IS NOT NULL THEN CONCAT('Turnamen: ', e.name)
					ELSE 'Transaksi Layanan Archeris'
				END as description,
				p.name as plan_name,
				e.name as event_name,
				e.page_settings as page_settings,
				a.full_name as athlete_name,
				rbt.name as division,
				COALESCE(ec.category_name_custom, rag.name) as category,
				u.email as user_email,
				COALESCE(u.full_name, u.username) as user_name
			FROM payment_transactions t
			LEFT JOIN subscription_plans p ON t.subscription_plan_id = p.id
			LEFT JOIN tournament_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN tournaments e ON COALESCE(t.tournament_id, ep.tournament_id) = e.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN (
				SELECT uuid, email, full_name, username FROM archers
				UNION ALL
				SELECT uuid, email, name as full_name, slug as username FROM organizers
			) u ON t.user_id = u.uuid
			WHERE t.reference = ?
		`
		err := db.Get(&t, query, reference)
		if err != nil {
			// Fallback: Check quota_purchases
			var q struct {
				UUID             string    `db:"uuid"`
				OrganizerID      string    `db:"organizer_id"`
				PlanID           int       `db:"plan_id"`
				QuotaType        string    `db:"quota_type"`
				Quantity         int       `db:"quantity"`
				UnitPrice        float64   `db:"unit_price"`
				TotalAmount      float64   `db:"total_amount"`
				Currency         string    `db:"currency"`
				PaymentStatus    string    `db:"payment_status"`
				PaymentMethod    string    `db:"payment_method"`
				PaymentReference string    `db:"payment_reference"`
				PurchasedAt      time.Time `db:"purchased_at"`
				PlanName         string    `db:"plan_name"`
				OrgName          string    `db:"org_name"`
				OrgEmail         string    `db:"org_email"`
			}
			qQuery := `
				SELECT q.uuid, q.organizer_id, q.plan_id, q.quota_type, q.quantity, q.unit_price, 
					   q.total_amount, q.currency, q.payment_status, COALESCE(q.payment_method, 'Mayar') as payment_method, 
					   q.payment_reference, q.purchased_at, COALESCE(p.name, 'Paket Kuota Event') as plan_name,
					   COALESCE(o.name, 'Organizer') as org_name, COALESCE(o.email, '') as org_email
				FROM quota_purchases q
				LEFT JOIN subscription_plans p ON q.plan_id = p.id
				LEFT JOIN organizers o ON q.organizer_id = o.uuid
				WHERE q.payment_reference = ? OR q.uuid = ?
				LIMIT 1
			`
			if errQ := db.Get(&q, qQuery, reference, reference); errQ == nil {
				t.UUID = q.UUID
				t.Reference = q.PaymentReference
				t.UserID = q.OrganizerID
				t.Amount = q.TotalAmount
				t.TotalAmount = q.TotalAmount
				t.PaymentMethod = &q.PaymentMethod
				t.Status = q.PaymentStatus
				t.CreatedAt = q.PurchasedAt
				t.PaidAt = &q.PurchasedAt
				t.Description = fmt.Sprintf("%s (%d Event)", q.PlanName, q.Quantity)
				t.PlanName = &q.PlanName
				t.UserName = q.OrgName
				t.UserEmail = q.OrgEmail
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
				return
			}
		}

		// Authorization check: verify owner or admin if user is authenticated
		userID, exists := c.Get("user_id")
		userRole, _ := c.Get("role")
		if exists && userID != nil {
			userIDStr := fmt.Sprintf("%v", userID)
			if userRole != "admin" && userRole != "root" && t.UserID != userIDStr {
				c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke invoice transaksi ini"})
				return
			}
		}

		if t.Status != "paid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hanya transaksi lunas yang dapat mengeluarkan invoice"})
			return
		}

		// Determine currency and free state
		isFree := (t.TotalAmount == 0 && t.Amount == 0) || (t.PaymentMethod != nil && (strings.ToLower(*t.PaymentMethod) == "free" || strings.ToLower(*t.PaymentMethod) == "free_registration"))
		currency := "IDR"
		if t.PaymentMethod != nil && strings.ToLower(*t.PaymentMethod) == "paypal" {
			currency = "USD"
		}
		if t.PageSettings != nil && *t.PageSettings != "" {
			var ps struct {
				Currency string `json:"currency"`
			}
			if err := json.Unmarshal([]byte(*t.PageSettings), &ps); err == nil && ps.Currency != "" {
				currency = strings.ToUpper(ps.Currency)
			}
		}

		// Format payment method display
		methodStr := "Pendaftaran Gratis (Free)"
		if !isFree {
			if t.PaymentMethod != nil && *t.PaymentMethod != "" {
				m := strings.ToLower(*t.PaymentMethod)
				switch m {
				case "manual", "manual_transfer", "bank_transfer":
					methodStr = "Transfer Bank (Manual)"
				case "mayar":
					methodStr = "Mayar (QRIS / VA / E-Wallet)"
				case "paypal":
					methodStr = "PayPal"
				default:
					methodStr = *t.PaymentMethod
				}
			} else {
				methodStr = "Transfer Bank"
			}
		}

		// Dates
		var paidTime time.Time
		if t.PaidAt != nil {
			paidTime = *t.PaidAt
		} else {
			paidTime = t.CreatedAt
		}
		paidDateFormatted := paidTime.Format("02 Jan 2006, 15:04 WIB")
		issueDateFormatted := t.CreatedAt.Format("02 Jan 2006")

		// Query detailed participants if any
		type InvoiceParticipant struct {
			AthleteName  string  `db:"athlete_name"`
			ClubName     string  `db:"club_name"`
			CategoryName string  `db:"category_name"`
			Amount       float64 `db:"payment_amount"`
		}
		var participants []InvoiceParticipant
		regID := ""
		if t.RegistrationID != nil {
			regID = *t.RegistrationID
		}
		_ = db.Select(&participants, `
			SELECT 
				COALESCE(a.full_name, 'Peserta') as athlete_name,
				COALESCE(c.name, '') as club_name,
				COALESCE(tc.category_name_custom, CONCAT_WS(' ', rbt.name, rag.name, rgd.name), '') as category_name,
				tp.payment_amount
			FROM tournament_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			LEFT JOIN tournament_categories tc ON tp.category_id = tc.uuid
			LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON tc.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
			WHERE (tp.payment_id = ? AND tp.payment_id != '') OR (tp.uuid = ? AND tp.uuid != '')
			ORDER BY tp.created_at ASC
		`, t.UUID, regID)

		// Create Clean Black and White PDF
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.SetMargins(15, 12, 15)
		pdf.SetAutoPageBreak(true, 15)
		pdf.AddPage()

		// Locate Logo
		logoPaths := []string{
			"public/logo.png",
			"api/public/logo.png",
			"../public/logo.png",
			"../api/public/logo.png",
		}
		var validLogoPath string
		for _, lp := range logoPaths {
			if _, err := os.Stat(lp); err == nil {
				validLogoPath = lp
				break
			}
		}

		// --- 1. HEADER SECTION (Left: Logo + Company Info | Right: INVOICE + Details) ---
		if validLogoPath != "" {
			opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
			// Logo positioned cleanly on the left (Width: 16mm)
			pdf.ImageOptions(validLogoPath, 15, 12, 16, 0, false, opt, 0, "")

			// Company Info positioned beside the logo at X=34 (Zero Overlap)
			pdf.SetFont("Arial", "B", 13)
			pdf.SetTextColor(15, 23, 42) // Black / Dark Slate
			pdf.SetXY(34, 12)
			pdf.CellFormat(75, 5, "ARCHERIS", "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(100, 116, 139) // Slate Gray
			pdf.SetXY(34, 17.5)
			pdf.CellFormat(75, 4, "PT Archeris Teknologi Indonesia", "", 1, "L", false, 0, "")
			pdf.SetXY(34, 21.5)
			pdf.CellFormat(75, 4, "Tournament & Scoring Management System", "", 1, "L", false, 0, "")
			pdf.SetXY(34, 25.5)
			pdf.CellFormat(75, 4, "www.archeris.net  |  support@archeris.net", "", 1, "L", false, 0, "")
		} else {
			pdf.SetFont("Arial", "B", 16)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetXY(15, 12)
			pdf.CellFormat(90, 6, "ARCHERIS", "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(100, 116, 139)
			pdf.SetXY(15, 18)
			pdf.CellFormat(90, 4, "PT Archeris Teknologi Indonesia", "", 1, "L", false, 0, "")
			pdf.SetXY(15, 22)
			pdf.CellFormat(90, 4, "Tournament & Scoring Management System", "", 1, "L", false, 0, "")
			pdf.SetXY(15, 26)
			pdf.CellFormat(90, 4, "www.archeris.net  |  support@archeris.net", "", 1, "L", false, 0, "")
		}

		// Right Header: INVOICE Title & Info
		pdf.SetFont("Arial", "B", 20)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetXY(110, 11)
		pdf.CellFormat(85, 7, "INVOICE", "", 1, "R", false, 0, "")

		pdf.SetFont("Arial", "B", 9)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetXY(110, 18.5)
		pdf.CellFormat(85, 4.5, fmt.Sprintf("NO: %s", t.Reference), "", 1, "R", false, 0, "")

		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetXY(110, 23.5)
		pdf.CellFormat(85, 4, fmt.Sprintf("Waktu Transaksi: %s", paidDateFormatted), "", 1, "R", false, 0, "")

		// Status Badge (Clean Box)
		statusBadge := "LUNAS / PAID"
		if isFree {
			statusBadge = "GRATIS / FREE"
		}
		pdf.SetXY(142, 29)
		pdf.SetFillColor(241, 245, 249) // Slate 100
		pdf.SetDrawColor(203, 213, 225) // Slate 300
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 7.5)
		pdf.CellFormat(53, 6, "STATUS: "+statusBadge, "1", 1, "C", true, 0, "")

		// --- 2. DIVIDER LINE ---
		pdf.SetDrawColor(226, 232, 240) // Slate 200
		pdf.SetLineWidth(0.3)
		pdf.Line(15, 38, 195, 38)

		// --- 3. BILLING & EVENT DETAILS (2 Bordered Cards) ---
		cardY := 42.0
		cardH := 25.0

		// Left Card: Bill To
		pdf.SetFillColor(248, 250, 252) // Slate 50
		pdf.SetDrawColor(226, 232, 240) // Slate 200
		pdf.SetLineWidth(0.2)
		pdf.Rect(15, cardY, 87, cardH, "FD")

		pdf.SetXY(18, cardY+2.5)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(81, 3.5, "DITAGIHKAN KEPADA (BILL TO):", "", 1, "L", false, 0, "")

		userName := t.UserName
		if userName == "" {
			userName = "Pendaftar"
		}
		pdf.SetXY(18, cardY+6.5)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 9.5)
		pdf.CellFormat(81, 4.5, userName, "", 1, "L", false, 0, "")

		pdf.SetXY(18, cardY+11.5)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(81, 4, t.UserEmail, "", 1, "L", false, 0, "")

		extraUser := "Peserta Terverifikasi"
		if t.AthleteName != nil && *t.AthleteName != "" && *t.AthleteName != userName {
			extraUser = fmt.Sprintf("Nama Atlet: %s", *t.AthleteName)
		}
		pdf.SetXY(18, cardY+16)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 7.5)
		pdf.CellFormat(81, 4, extraUser, "", 1, "L", false, 0, "")

		// Right Card: Event & Payment Info
		pdf.Rect(108, cardY, 87, cardH, "FD")

		pdf.SetXY(111, cardY+2.5)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(81, 3.5, "INFORMASI KEGIATAN & TRANSAKSI:", "", 1, "L", false, 0, "")

		eventName := "Transaksi Layanan Archeris"
		if t.EventName != nil && *t.EventName != "" {
			eventName = *t.EventName
		} else if t.PlanName != nil && *t.PlanName != "" {
			eventName = *t.PlanName
		}
		pdf.SetXY(111, cardY+6.5)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(81, 4.5, eventName, "", 1, "L", false, 0, "")

		pdf.SetXY(111, cardY+11.5)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(81, 4, fmt.Sprintf("Metode: %s", methodStr), "", 1, "L", false, 0, "")

		pdf.SetXY(111, cardY+16)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 7.5)
		pdf.CellFormat(81, 4, fmt.Sprintf("Tanggal Terbit: %s", issueDateFormatted), "", 1, "L", false, 0, "")

		// --- 4. ITEMS TABLE ---
		tableY := 71.0
		pdf.SetY(tableY)
		pdf.SetX(15)
		pdf.SetFillColor(241, 245, 249) // Slate 100
		pdf.SetDrawColor(203, 213, 225) // Slate 300
		pdf.SetTextColor(51, 65, 85)   // Slate 700
		pdf.SetFont("Arial", "B", 7.5)
		pdf.SetLineWidth(0.2)

		// Table Headers: Total = 10 + 85 + 45 + 15 + 25 = 180mm
		pdf.CellFormat(10, 7.5, "NO", "TB", 0, "C", true, 0, "")
		pdf.CellFormat(85, 7.5, " DESKRIPSI LAYANAN / ITEM", "TB", 0, "L", true, 0, "")
		pdf.CellFormat(45, 7.5, "KATEGORI / DIVISI", "TB", 0, "L", true, 0, "")
		pdf.CellFormat(15, 7.5, "QTY", "TB", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7.5, "TOTAL ", "TB", 1, "R", true, 0, "")

		pdf.SetDrawColor(226, 232, 240) // Slate 200
		pdf.SetTextColor(15, 23, 42)

		if len(participants) > 0 {
			for idx, p := range participants {
				pdf.SetX(15)
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(10, 8, fmt.Sprintf("%d", idx+1), "B", 0, "C", false, 0, "")

				pDesc := p.AthleteName
				if p.ClubName != "" {
					pDesc = fmt.Sprintf("%s (%s)", p.AthleteName, p.ClubName)
				}
				if len(pDesc) > 45 {
					pDesc = pDesc[:42] + "..."
				}
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(85, 8, " "+pDesc, "B", 0, "L", false, 0, "")

				pCat := p.CategoryName
				if pCat == "" {
					pCat = "-"
				}
				if len(pCat) > 24 {
					pCat = pCat[:21] + "..."
				}
				pdf.SetFont("Arial", "", 7.5)
				pdf.CellFormat(45, 8, pCat, "B", 0, "L", false, 0, "")

				pdf.CellFormat(15, 8, "1", "B", 0, "C", false, 0, "")

				amtStr := formatInvoiceMoney(p.Amount, currency)
				if isFree || p.Amount == 0 {
					amtStr = "Gratis"
				}
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(25, 8, amtStr+" ", "B", 1, "R", false, 0, "")
			}
		} else {
			desc := t.Description
			qty := "1"
			unitPrice := t.Amount
			if t.SubscriptionPlanID != nil && t.Months > 0 {
				qty = fmt.Sprintf("%d Bln", t.Months)
			}

			catStr := "-"
			if t.Division != nil && t.Category != nil {
				catStr = fmt.Sprintf("%s %s", *t.Division, *t.Category)
			} else if t.PlanName != nil {
				catStr = *t.PlanName
			}

			pdf.SetX(15)
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(10, 8.5, "1", "B", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(85, 8.5, " "+desc, "B", 0, "L", false, 0, "")
			pdf.SetFont("Arial", "", 7.5)
			pdf.CellFormat(45, 8.5, catStr, "B", 0, "L", false, 0, "")
			pdf.CellFormat(15, 8.5, qty, "B", 0, "C", false, 0, "")

			amtStr := formatInvoiceMoney(unitPrice, currency)
			if isFree || unitPrice == 0 {
				amtStr = "Gratis"
			}
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(25, 8.5, amtStr+" ", "B", 1, "R", false, 0, "")
		}

		// --- 5. SUMMARY & NOTES SECTION ---
		summaryY := pdf.GetY() + 6
		if summaryY < 120 {
			summaryY = 120
		}

		// Left: Official Verification Note Box (X = 15, Width = 95mm)
		pdf.SetY(summaryY)
		pdf.SetX(15)
		pdf.SetFillColor(248, 250, 252) // Slate 50
		pdf.SetDrawColor(226, 232, 240) // Slate 200
		pdf.SetLineWidth(0.2)
		pdf.Rect(15, summaryY, 95, 32, "FD")

		pdf.SetXY(18, summaryY+3)
		pdf.SetTextColor(51, 65, 85)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(89, 3.5, "CATATAN TRANSAKSI & PENDAFTARAN:", "", 1, "L", false, 0, "")

		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(18, summaryY+7.5)
		pdf.CellFormat(89, 3.5, "- Transaksi telah terverifikasi resmi dalam sistem Archeris.", "", 1, "L", false, 0, "")
		pdf.SetXY(18, summaryY+11.5)
		pdf.CellFormat(89, 3.5, "- Data peserta telah tercatat aktif pada daftar turnamen.", "", 1, "L", false, 0, "")
		pdf.SetXY(18, summaryY+15.5)
		pdf.CellFormat(89, 3.5, "- Dokumen ini merupakan bukti pendaftaran yang sah.", "", 1, "L", false, 0, "")
		pdf.SetXY(18, summaryY+19.5)
		pdf.CellFormat(89, 3.5, "- Bantuan teknis & pertanyaan: support@archeris.net", "", 1, "L", false, 0, "")

		// Right: Financial Calculation (X = 120, Width = 75mm)
		pdf.SetY(summaryY)
		pdf.SetDrawColor(226, 232, 240)

		pdf.SetX(120)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 8.5)
		pdf.CellFormat(40, 5.5, "Subtotal", "", 0, "L", false, 0, "")
		subtotalStr := formatInvoiceMoney(t.Amount, currency)
		if isFree || t.Amount == 0 {
			subtotalStr = "Rp 0"
		}
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 8.5)
		pdf.CellFormat(35, 5.5, subtotalStr, "", 1, "R", false, 0, "")

		if t.FeeAmount > 0 {
			pdf.SetX(120)
			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "", 8.5)
			pdf.CellFormat(40, 5, "Biaya Layanan", "", 0, "L", false, 0, "")
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 8.5)
			pdf.CellFormat(35, 5, formatInvoiceMoney(t.FeeAmount, currency), "", 1, "R", false, 0, "")
		}

		// Total Box (Bordered, Sleek Black/Slate)
		totalY := pdf.GetY() + 2
		pdf.SetY(totalY)
		pdf.SetX(120)
		pdf.SetFillColor(241, 245, 249) // Slate 100
		pdf.SetDrawColor(15, 23, 42)    // Dark Navy / Black Border
		pdf.SetLineWidth(0.3)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 8.5)
		pdf.CellFormat(38, 8.5, " TOTAL LUNAS", "LTB", 0, "L", true, 0, "")

		totalAmountStr := formatInvoiceMoney(t.TotalAmount, currency)
		if isFree || t.TotalAmount == 0 {
			totalAmountStr = "Rp 0 (GRATIS)"
		}
		pdf.SetFont("Arial", "B", 9.5)
		pdf.CellFormat(37, 8.5, totalAmountStr+" ", "RTB", 1, "R", true, 0, "")

		// --- 6. FOOTER ---
		pdf.SetY(275)
		pdf.SetDrawColor(226, 232, 240)
		pdf.SetLineWidth(0.2)
		pdf.Line(15, 275, 195, 275)

		pdf.SetY(277)
		pdf.SetX(15)
		pdf.SetTextColor(148, 163, 184) // Slate 400
		pdf.SetFont("Arial", "", 7)
		printTimestamp := time.Now().Format("02 Jan 2006, 15:04:05 WIB")
		pdf.CellFormat(90, 4, fmt.Sprintf("Diterbitkan otomatis pada: %s", printTimestamp), "", 0, "L", false, 0, "")
		pdf.CellFormat(90, 4, "Archeris Tournament Management System  |  archeris.net", "", 1, "R", false, 0, "")

		// Output to browser
		download := c.Query("download")
		disposition := "inline"
		if download == "true" {
			disposition = "attachment"
		}
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("%s; filename=Invoice-%s.pdf", disposition, t.Reference))

		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		}
	}
}
