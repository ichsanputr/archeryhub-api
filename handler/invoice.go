package handler

import (
	"Archeris-api/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/jung-kurt/gofpdf"
)

func GenerateInvoicePDF(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")

		type EnrichedTransaction struct {
			models.PaymentTransaction
			Description string  `json:"description" db:"description"`
			PlanName    *string `json:"plan_name" db:"plan_name"`
			EventName   *string `json:"event_name" db:"event_name"`
			AthleteName *string `json:"athlete_name" db:"athlete_name"`
			Division    *string `json:"division" db:"division"`
			Category    *string `json:"category" db:"category"`
			UserEmail   string  `db:"user_email"`
			UserName    string  `db:"user_name"`
		}

		var t EnrichedTransaction
		query := `
			SELECT 
				t.*,
				CASE 
					WHEN t.subscription_plan_id IS NOT NULL THEN p.name
					WHEN t.registration_id IS NOT NULL THEN CONCAT('Registrasi: ', a.full_name)
					WHEN t.event_id IS NOT NULL THEN CONCAT('Platform Fee: ', e.name)
					ELSE 'Transaksi Archeris'
				END as description,
				p.name as plan_name,
				e.name as event_name,
				a.full_name as athlete_name,
				rbt.name as division,
				COALESCE(ec.category_name_custom, rag.name) as category,
				u.email as user_email,
				COALESCE(u.full_name, u.username) as user_name
			FROM payment_transactions t
			LEFT JOIN subscription_plans p ON t.subscription_plan_id = p.id
			LEFT JOIN event_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN events e ON t.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN (
				SELECT uuid, email, full_name, username FROM archers
				UNION ALL
				SELECT uuid, email, name as full_name, slug as username FROM organizations
				UNION ALL
				SELECT uuid, email, store_name as full_name, slug as username FROM sellers
			) u ON t.user_id = u.uuid
			WHERE t.reference = ?
		`
		err := db.Get(&t, query, reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}

		if t.Status != "paid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hanya transaksi lunas yang dapat mengeluarkan invoice"})
			return
		}

		// Create PDF
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()

		// --- Header Section ---
		pdf.SetFillColor(15, 23, 42) // Navy
		pdf.Rect(0, 0, 210, 60, "F")

		// Logo / Title
		pdf.SetTextColor(217, 255, 0) // Neon Primary
		pdf.SetFont("Arial", "B", 24)
		pdf.Text(20, 25, "archeris.net")
		
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Arial", "", 10)
		pdf.Text(20, 32, "PT. Archeris Teknologi Indonesia")
		pdf.Text(20, 37, "Official Payment Receipt")

		// Invoice Title
		pdf.SetFont("Arial", "B", 34)
		pdf.Text(120, 35, "INVOICE")

		// --- Info Section ---
		pdf.SetY(70)
		pdf.SetX(20)
		pdf.SetTextColor(100, 116, 139) // Gray
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(90, 5, "DITAGIHKAN KEPADA:", "", 0, "L", false, 0, "")
		pdf.CellFormat(90, 5, "DETAIL TRANSAKSI:", "", 1, "R", false, 0, "")

		pdf.SetX(20)
		pdf.SetTextColor(15, 23, 42) // Navy
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(90, 7, t.UserName, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(90, 7, "ID: "+t.Reference, "", 1, "R", false, 0, "")

		pdf.SetX(20)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(90, 5, t.UserEmail, "", 0, "L", false, 0, "")
		pdf.CellFormat(90, 5, "Tanggal: "+t.PaidAt.Format("02 Jan 2006"), "", 1, "R", false, 0, "")

		pdf.SetX(20)
		pdf.CellFormat(90, 5, "", "", 0, "L", false, 0, "")
		pdf.CellFormat(90, 5, "Metode: "+*t.PaymentMethod, "", 1, "R", false, 0, "")

		// --- Table Section ---
		pdf.SetY(110)
		pdf.SetX(20)
		pdf.SetFillColor(241, 245, 249) // Slate 50
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(100, 10, " DESKRIPSI LAYANAN", "B", 0, "L", true, 0, "")
		pdf.CellFormat(20, 10, "QTY", "B", 0, "C", true, 0, "")
		pdf.CellFormat(25, 10, "HARGA", "B", 0, "R", true, 0, "")
		pdf.CellFormat(25, 10, "TOTAL ", "B", 1, "R", true, 0, "")

		pdf.SetX(20)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Arial", "B", 10)
		
		desc := t.Description
		qty := "1 Item"
		unitPrice := t.Amount
		if t.SubscriptionPlanID != nil {
			qty = fmt.Sprintf("%d Bln", t.Months)
			unitPrice = t.Amount / float64(t.Months)
		}

		pdf.CellFormat(100, 15, " "+desc, "", 0, "L", false, 0, "")
		pdf.CellFormat(20, 15, qty, "", 0, "C", false, 0, "")
		pdf.CellFormat(25, 15, fmt.Sprintf("%.0f", unitPrice), "", 0, "R", false, 0, "")
		pdf.CellFormat(25, 15, fmt.Sprintf("%.0f", t.Amount), "", 1, "R", false, 0, "")

		// --- Summary Section ---
		pdf.SetY(180)
		pdf.Line(20, 175, 190, 175)
		
		pdf.SetX(120)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(40, 10, "Subtotal", "", 0, "L", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("Rp %.0f", t.Amount), "", 1, "R", false, 0, "")

		pdf.SetX(120)
		pdf.CellFormat(40, 10, "Pajak / Admin", "", 0, "L", false, 0, "")
		pdf.CellFormat(30, 10, "Rp 0", "", 1, "R", false, 0, "")

		pdf.SetX(120)
		pdf.SetFillColor(15, 23, 42)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(40, 12, " TOTAL", "", 0, "L", true, 0, "")
		pdf.SetTextColor(217, 255, 0)
		pdf.CellFormat(30, 12, fmt.Sprintf("Rp %.0f ", t.TotalAmount), "", 1, "R", true, 0, "")

		// --- Footer ---
		pdf.SetY(260)
		pdf.SetTextColor(148, 163, 184)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(170, 10, "TERIMA KASIH TELAH MENGGUNAKAN LAYANAN archeris.net", "", 0, "C", false, 0, "")

		// Output to browser
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=Invoice-%s.pdf", t.Reference))
		
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		}
	}
}


