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
	curr := strings.ToUpper(strings.TrimSpace(currency))
	switch curr {
	case "USD":
		return fmt.Sprintf("$ %.2f", amount)
	case "EUR":
		return fmt.Sprintf("€ %.2f", amount)
	case "GBP":
		return fmt.Sprintf("£ %.2f", amount)
	case "SGD":
		return fmt.Sprintf("S$ %.2f", amount)
	case "MYR":
		return fmt.Sprintf("RM %.2f", amount)
	case "AUD":
		return fmt.Sprintf("A$ %.2f", amount)
	default:
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

		// Create Clean, Modern Minimalist White PDF
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.SetMargins(18, 16, 18)
		pdf.SetAutoPageBreak(true, 18)
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

		// --- 1. HEADER (Minimalist Brand + Invoice Reference) ---
		if validLogoPath != "" {
			opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
			pdf.ImageOptions(validLogoPath, 18, 16, 13, 0, false, opt, 0, "")

			pdf.SetFont("Arial", "B", 13)
			pdf.SetTextColor(15, 23, 42) // Slate 900
			pdf.SetXY(34, 16)
			pdf.CellFormat(70, 4.5, "ARCHERIS", "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(100, 116, 139) // Slate 500
			pdf.SetXY(34, 21)
			pdf.CellFormat(70, 3.8, "PT Archeris Teknologi Indonesia", "", 1, "L", false, 0, "")
			pdf.SetXY(34, 25)
			pdf.CellFormat(70, 3.8, "support@archeris.net  *  archeris.net", "", 1, "L", false, 0, "")
		} else {
			pdf.SetFont("Arial", "B", 14)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetXY(18, 16)
			pdf.CellFormat(80, 5, "ARCHERIS", "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(100, 116, 139)
			pdf.SetXY(18, 21.5)
			pdf.CellFormat(80, 3.8, "PT Archeris Teknologi Indonesia", "", 1, "L", false, 0, "")
			pdf.SetXY(18, 25.5)
			pdf.CellFormat(80, 3.8, "support@archeris.net  *  archeris.net", "", 1, "L", false, 0, "")
		}

		// Right Header: INVOICE Title & Details
		pdf.SetFont("Arial", "B", 18)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetXY(105, 15)
		pdf.CellFormat(87, 6, "INVOICE", "", 1, "R", false, 0, "")

		pdf.SetFont("Arial", "B", 8.5)
		pdf.SetTextColor(51, 65, 85)
		pdf.SetXY(105, 22)
		pdf.CellFormat(87, 4, fmt.Sprintf("REF: #%s", t.Reference), "", 1, "R", false, 0, "")

		statusBadge := "PAID / LUNAS"
		if isFree {
			statusBadge = "FREE / GRATIS"
		}
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetXY(105, 26.5)
		pdf.CellFormat(87, 4, fmt.Sprintf("Tanggal: %s  |  Status: %s", issueDateFormatted, statusBadge), "", 1, "R", false, 0, "")

		// --- 2. DIVIDER LINE ---
		pdf.SetDrawColor(229, 231, 235) // Slate 200
		pdf.SetLineWidth(0.2)
		pdf.Line(18, 33, 192, 33)

		// --- 3. METADATA SECTION (Clean 2-Column Text, No Boxes) ---
		metaY := 38.0

		// Left Column: Bill To
		pdf.SetXY(18, metaY)
		pdf.SetTextColor(148, 163, 184) // Slate 400
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(82, 3.5, "DITAGIHKAN KEPADA (BILLED TO)", "", 1, "L", false, 0, "")

		userName := t.UserName
		if userName == "" {
			userName = "Pendaftar"
		}
		pdf.SetXY(18, metaY+4.5)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(82, 4.5, userName, "", 1, "L", false, 0, "")

		pdf.SetXY(18, metaY+9.5)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(82, 4, t.UserEmail, "", 1, "L", false, 0, "")

		if t.AthleteName != nil && *t.AthleteName != "" && *t.AthleteName != userName {
			pdf.SetXY(18, metaY+14)
			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "", 7.5)
			pdf.CellFormat(82, 3.5, fmt.Sprintf("Peserta: %s", *t.AthleteName), "", 1, "L", false, 0, "")
		}

		// Right Column: Transaction / Event Details
		pdf.SetXY(105, metaY)
		pdf.SetTextColor(148, 163, 184) // Slate 400
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(87, 3.5, "DETAIL PEMBAYARAN (PAYMENT DETAILS)", "", 1, "L", false, 0, "")

		eventName := "Transaksi Layanan Archeris"
		if t.EventName != nil && *t.EventName != "" {
			eventName = *t.EventName
		} else if t.PlanName != nil && *t.PlanName != "" {
			eventName = *t.PlanName
		}
		pdf.SetXY(105, metaY+4.5)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(87, 4.5, eventName, "", 1, "L", false, 0, "")

		pdf.SetXY(105, metaY+9.5)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(87, 4, fmt.Sprintf("Metode: %s", methodStr), "", 1, "L", false, 0, "")

		pdf.SetXY(105, metaY+14)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 7.5)
		pdf.CellFormat(87, 3.5, fmt.Sprintf("Waktu Bayar: %s", paidDateFormatted), "", 1, "L", false, 0, "")

		// --- 4. ITEMS TABLE (Clean Minimalist Table, No Backgrounds) ---
		tableY := 60.0
		pdf.SetY(tableY)
		pdf.SetX(18)
		pdf.SetDrawColor(203, 213, 225) // Slate 300
		pdf.SetTextColor(100, 116, 139)  // Slate 500
		pdf.SetFont("Arial", "B", 7)
		pdf.SetLineWidth(0.2)

		// Table Headers: Total = 10 + 82 + 47 + 12 + 23 = 174mm
		pdf.CellFormat(10, 6.5, "NO", "TB", 0, "C", false, 0, "")
		pdf.CellFormat(82, 6.5, "DESKRIPSI ITEM / PESERTA", "TB", 0, "L", false, 0, "")
		pdf.CellFormat(47, 6.5, "KATEGORI / DIVISI", "TB", 0, "L", false, 0, "")
		pdf.CellFormat(12, 6.5, "QTY", "TB", 0, "C", false, 0, "")
		pdf.CellFormat(23, 6.5, "JUMLAH", "TB", 1, "R", false, 0, "")

		pdf.SetDrawColor(241, 245, 249) // Slate 100 subtle row line
		pdf.SetTextColor(15, 23, 42)

		if len(participants) > 0 {
			for idx, p := range participants {
				pdf.SetX(18)
				pdf.SetFont("Arial", "", 7.5)
				pdf.CellFormat(10, 7.5, fmt.Sprintf("%d", idx+1), "B", 0, "C", false, 0, "")

				pDesc := p.AthleteName
				if p.ClubName != "" {
					pDesc = fmt.Sprintf("%s (%s)", p.AthleteName, p.ClubName)
				}
				if len(pDesc) > 42 {
					pDesc = pDesc[:39] + "..."
				}
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(82, 7.5, pDesc, "B", 0, "L", false, 0, "")

				pCat := p.CategoryName
				if pCat == "" {
					pCat = "-"
				}
				if len(pCat) > 26 {
					pCat = pCat[:23] + "..."
				}
				pdf.SetFont("Arial", "", 7.5)
				pdf.CellFormat(47, 7.5, pCat, "B", 0, "L", false, 0, "")

				pdf.CellFormat(12, 7.5, "1", "B", 0, "C", false, 0, "")

				amtStr := formatInvoiceMoney(p.Amount, currency)
				if isFree || p.Amount == 0 {
					amtStr = "Gratis"
				}
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(23, 7.5, amtStr, "B", 1, "R", false, 0, "")
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

			pdf.SetX(18)
			pdf.SetFont("Arial", "", 7.5)
			pdf.CellFormat(10, 8, "1", "B", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(82, 8, desc, "B", 0, "L", false, 0, "")
			pdf.SetFont("Arial", "", 7.5)
			pdf.CellFormat(47, 8, catStr, "B", 0, "L", false, 0, "")
			pdf.CellFormat(12, 8, qty, "B", 0, "C", false, 0, "")

			amtStr := formatInvoiceMoney(unitPrice, currency)
			if isFree || unitPrice == 0 {
				amtStr = "Gratis"
			}
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(23, 8, amtStr, "B", 1, "R", false, 0, "")
		}

		// --- 5. SUMMARY & NOTES SECTION (Clean & Spaced) ---
		summaryY := pdf.GetY() + 8
		if summaryY < 115 {
			summaryY = 115
		}

		// Left: Notes (No box, clean text)
		pdf.SetXY(18, summaryY)
		pdf.SetTextColor(148, 163, 184)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(90, 3.5, "CATATAN / KETERANGAN RESMI", "", 1, "L", false, 0, "")

		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(18, summaryY+5)
		pdf.CellFormat(90, 3.5, "* Transaksi terverifikasi resmi dalam database Archeris.", "", 1, "L", false, 0, "")
		pdf.SetXY(18, summaryY+9)
		pdf.CellFormat(90, 3.5, "* Dokumen ini merupakan bukti pembayaran dan registrasi yang sah.", "", 1, "L", false, 0, "")
		pdf.SetXY(18, summaryY+13)
		pdf.CellFormat(90, 3.5, "* Bantuan teknis & pertanyaan resmi: support@archeris.net", "", 1, "L", false, 0, "")

		// Right: Financial Calculation
		pdf.SetXY(118, summaryY)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(40, 5, "Subtotal", "", 0, "L", false, 0, "")

		subtotalStr := formatInvoiceMoney(t.Amount, currency)
		if isFree || t.Amount == 0 {
			subtotalStr = "Rp 0"
		}
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(34, 5, subtotalStr, "", 1, "R", false, 0, "")

		if t.FeeAmount > 0 {
			pdf.SetX(118)
			pdf.SetTextColor(100, 116, 139)
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(40, 5, "Biaya Layanan", "", 0, "L", false, 0, "")
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(34, 5, formatInvoiceMoney(t.FeeAmount, currency), "", 1, "R", false, 0, "")
		}

		// Total line
		currY := pdf.GetY() + 2
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetLineWidth(0.2)
		pdf.Line(118, currY, 192, currY)

		pdf.SetXY(118, currY+2.5)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(40, 6, "Total Pembayaran", "", 0, "L", false, 0, "")

		totalAmountStr := formatInvoiceMoney(t.TotalAmount, currency)
		if isFree || t.TotalAmount == 0 {
			totalAmountStr = "Rp 0 (Gratis)"
		}
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(34, 6, totalAmountStr, "", 1, "R", false, 0, "")

		// --- 6. FOOTER ---
		pdf.SetY(272)
		pdf.SetDrawColor(229, 231, 235)
		pdf.SetLineWidth(0.2)
		pdf.Line(18, 272, 192, 272)

		pdf.SetY(274)
		pdf.SetX(18)
		pdf.SetTextColor(148, 163, 184) // Slate 400
		pdf.SetFont("Arial", "", 7)
		printTimestamp := time.Now().Format("02 Jan 2006, 15:04 WIB")
		pdf.CellFormat(87, 4, fmt.Sprintf("Diterbitkan otomatis: %s", printTimestamp), "", 0, "L", false, 0, "")
		pdf.CellFormat(87, 4, "Archeris Tournament Management System  *  archeris.net", "", 1, "R", false, 0, "")

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
