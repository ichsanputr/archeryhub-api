package handler

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"Archeris-api/models"
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jung-kurt/gofpdf"
	"github.com/nwaples/rardecode/v2"
)

// Default HTML Certificate Template
const DefaultHTMLCertificateTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; background: #ffffff; color: #0f172a; margin: 0; padding: 40px; }
  .cert-border { border: 10px solid #0f172a; padding: 30px; border-radius: 16px; position: relative; }
  .cert-inner { border: 2px solid #b7fb23; padding: 40px; border-radius: 8px; text-align: center; }
  .title { font-size: 38px; font-weight: 900; letter-spacing: 2px; text-transform: uppercase; color: #0f172a; margin-bottom: 8px; }
  .subtitle { font-size: 14px; font-weight: 700; color: #64748b; text-transform: uppercase; letter-spacing: 3px; margin-bottom: 30px; }
  .presented-to { font-size: 14px; font-weight: 600; color: #475569; margin-bottom: 12px; }
  .archer-name { font-size: 32px; font-weight: 900; color: #0f172a; border-bottom: 2px solid #b7fb23; display: inline-block; padding-bottom: 6px; margin-bottom: 24px; }
  .desc { font-size: 16px; line-height: 1.6; color: #334155; max-w: 650px; margin: 0 auto 30px; }
  .event-name { font-weight: 800; color: #0f172a; }
  .category-name { font-weight: 800; color: #0f172a; }
  .footer-grid { display: flex; justify-content: space-between; align-items: flex-end; margin-top: 50px; }
  .cert-no { font-family: monospace; font-size: 11px; color: #64748b; font-weight: 700; }
  .signature-box { text-align: center; }
  .signature-img { max-height: 60px; margin-bottom: 8px; }
  .signatory-name { font-size: 14px; font-weight: 800; color: #0f172a; }
  .signatory-title { font-size: 11px; color: #64748b; font-weight: 600; }
</style>
</head>
<body>
  <div class="cert-border">
    <div class="cert-inner">
      <div class="subtitle">Official Certificate of Accomplishment</div>
      <div class="title">Sertifikat Penghargaan</div>
      <div class="presented-to">Diberikan Kepada:</div>
      <div class="archer-name">{{ArcherName}}</div>
      <div class="desc">
        Atas partisipasi dan prestasi luar biasa pada kejuaraan panahan <span class="event-name">{{EventName}}</span> pada kategori <span class="category-name">{{CategoryName}}</span>.
      </div>
      <div class="footer-grid">
        <div style="text-align: left;">
          <div class="cert-no">No: {{CertificateNo}}</div>
          <div class="cert-no">Tanggal: {{IssueDate}}</div>
        </div>
        <div class="signature-box">
          {{SignatureHTML}}
          <div class="signatory-name">{{OrganizerName}}</div>
          <div class="signatory-title">Penyelenggara Event</div>
        </div>
      </div>
    </div>
  </div>
</body>
</html>`

// GetCertificateTemplate returns current HTML template and background setting for an event
func GetCertificateTemplate(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var cert models.EventCertificate
		err := db.Get(&cert, "SELECT * FROM tournament_certificates WHERE tournament_id = ? LIMIT 1", eventID)
		if err != nil {
			// Return default template
			defaultTmpl := DefaultHTMLCertificateTemplate
			cert = models.EventCertificate{
				UUID:         uuid.New().String(),
				EventID:      eventID,
				HTMLTemplate: &defaultTmpl,
			}
		}

		c.JSON(http.StatusOK, cert)
	}
}

// SaveCertificateTemplate updates or inserts certificate template configuration
func SaveCertificateTemplate(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		var req models.SaveCertificateTemplateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid"})
			return
		}

		var existingID string
		err := db.Get(&existingID, "SELECT uuid FROM tournament_certificates WHERE tournament_id = ? LIMIT 1", eventID)
		if err != nil {
			// Insert new
			newUUID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO tournament_certificates (uuid, tournament_id, html_template, background_url, signature_url)
				VALUES (?, ?, ?, ?, ?)
			`, newUUID, eventID, req.HTMLTemplate, req.BackgroundURL, req.SignatureURL)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan template sertifikat: " + err.Error()})
				return
			}
		} else {
			// Update
			_, err = db.Exec(`
				UPDATE tournament_certificates 
				SET html_template = ?, background_url = ?, signature_url = ?, updated_at = NOW()
				WHERE tournament_id = ?
			`, req.HTMLTemplate, req.BackgroundURL, req.SignatureURL, eventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui template sertifikat"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Template sertifikat berhasil disimpan"})
	}
}

// GetArcherCertificates returns earned certificates for the authenticated archer
func GetArcherCertificates(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		archerID, _ := c.Get("archer_id")

		userStr := ""
		if userID != nil {
			userStr = userID.(string)
		}
		archerStr := ""
		if archerID != nil {
			archerStr = archerID.(string)
		}

		type QueryItem struct {
			CertUUID       string    `db:"cert_uuid"`
			CertificateNo  *string   `db:"certificate_no"`
			PDFURL         *string   `db:"pdf_url"`
			IssueDate      time.Time `db:"issue_date"`
			RegistrationID string    `db:"registration_id"`
			EventID        string    `db:"event_id"`
			EventSlug      string    `db:"event_slug"`
			EventName      string    `db:"event_name"`
			EventBanner    *string   `db:"event_banner"`
			CategoryName   string    `db:"category_name"`
			ArcherName     string    `db:"archer_name"`
			RegDate        time.Time `db:"registration_date"`
		}

		var items []QueryItem
		query := `
			SELECT 
				ac.uuid as cert_uuid,
				ac.certificate_no,
				ac.pdf_url,
				COALESCE(ac.issue_date, ep.registration_date) as issue_date,
				ep.uuid as registration_id,
				e.uuid as event_id,
				COALESCE(e.slug, e.uuid) as event_slug,
				e.name as event_name,
				e.banner_url as event_banner,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' - ', COALESCE(rag.name, ''))) as category_name,
				COALESCE(a.full_name, 'Peserta Archeris') as archer_name,
				ep.registration_date
			FROM archer_certificates ac
			JOIN tournament_participants ep ON ac.registration_id = ep.uuid
			JOIN tournaments e ON (ac.tournament_id = e.uuid OR ep.tournament_id = e.uuid)
			LEFT JOIN archers a ON (ep.archer_id = a.uuid OR ac.archer_id = a.uuid)
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE (ep.archer_id = ? OR ep.archer_id = ? OR a.uuid = ? OR a.uuid = ? OR a.id = ? OR ac.archer_id = ? OR ac.archer_id = ?)
			  AND ac.pdf_url IS NOT NULL AND ac.pdf_url != ''
			ORDER BY ac.created_at DESC, ep.registration_date DESC
		`
		err := db.Select(&items, query, archerStr, userStr, archerStr, userStr, userStr, archerStr, userStr)
		if err != nil {
			fmt.Printf("GetArcherCertificates query error: %v\n", err)
			c.JSON(http.StatusOK, []models.CertificateItemResponse{})
			return
		}
		if len(items) == 0 {
			c.JSON(http.StatusOK, []models.CertificateItemResponse{})
			return
		}

		var res []models.CertificateItemResponse
		for _, it := range items {
			certNo := ""
			if it.CertificateNo != nil {
				certNo = *it.CertificateNo
			}
			pdfURL := ""
			if it.PDFURL != nil {
				pdfURL = *it.PDFURL
			}

			res = append(res, models.CertificateItemResponse{
				ID:              it.CertUUID,
				EventID:         it.EventID,
				EventSlug:       it.EventSlug,
				EventName:       it.EventName,
				EventBanner:     it.EventBanner,
				CategoryName:    it.CategoryName,
				ArcherName:      it.ArcherName,
				Title:           "Sertifikat Turnamen",
				CertificateNo:   certNo,
				IssueDate:       it.IssueDate,
				PDFURL:          pdfURL,
				VerificationURL: fmt.Sprintf("/certificates/%s", it.CertUUID),
			})
		}

		c.JSON(http.StatusOK, res)
	}
}

// GenerateCertificatePDF builds A4 Landscape PDF from event template and participant data
func GenerateCertificatePDF(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		regID := c.Param("id")

		type CertData struct {
			RegistrationID string    `db:"registration_id"`
			EventID        string    `db:"event_id"`
			EventName      string    `db:"event_name"`
			OrganizerName  string    `db:"organizer_name"`
			ArcherName     string    `db:"archer_name"`
			CategoryName   string    `db:"category_name"`
			RegDate        time.Time `db:"registration_date"`
			PaymentStatus  string    `db:"payment_status"`
		}

		var d CertData
		query := `
			SELECT 
				ep.uuid as registration_id,
				e.uuid as event_id,
				e.name as event_name,
				COALESCE(o.name, 'Panitia Pelaksana') as organizer_name,
				COALESCE(a.full_name, 'Peserta Archeris') as archer_name,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' - ', COALESCE(rag.name, ''))) as category_name,
				ep.registration_date,
				ep.payment_status
			FROM tournament_participants ep
			JOIN tournaments e ON ep.tournament_id = e.uuid
			LEFT JOIN organizers o ON e.organizer_id = o.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE ep.uuid = ? OR ep.qr_raw = ?
			LIMIT 1
		`
		err := db.Get(&d, query, regID, regID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pendaftaran atau sertifikat tidak ditemukan"})
			return
		}

		// Ensure certificate entry & cert_no
		var certNo string
		var issueDate time.Time
		err = db.QueryRow("SELECT certificate_no, issue_date FROM archer_certificates WHERE registration_id = ?", d.RegistrationID).Scan(&certNo, &issueDate)
		if err != nil {
			certNo = fmt.Sprintf("CERT-%d-%s", time.Now().Year(), strings.ToUpper(strings.ReplaceAll(uuid.New().String()[:8], "-", "")))
			issueDate = d.RegDate
			newUUID := uuid.New().String()
			db.Exec(`
				INSERT INTO archer_certificates (uuid, tournament_id, archer_id, registration_id, certificate_no, issue_date)
				VALUES (?, ?, ?, ?, ?, ?)
			`, newUUID, d.EventID, d.RegistrationID, d.RegistrationID, certNo, issueDate)
		}

		// Fetch Event Certificate Settings
		var certConfig models.EventCertificate
		db.Get(&certConfig, "SELECT * FROM tournament_certificates WHERE tournament_id = ? LIMIT 1", d.EventID)

		// Create A4 Landscape PDF (297mm x 210mm)
		pdf := gofpdf.New("L", "mm", "A4", "")
		pdf.AddPage()

		// Outer Decorative Border
		pdf.SetLineWidth(2.5)
		pdf.SetDrawColor(15, 23, 42) // Navy
		pdf.Rect(10, 10, 277, 190, "D")

		pdf.SetLineWidth(0.8)
		pdf.SetDrawColor(183, 251, 35) // Neon Primary
		pdf.Rect(14, 14, 269, 182, "D")

		// Header Badge
		pdf.SetY(26)
		pdf.SetFont("Arial", "B", 10)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(277, 6, "OFFICIAL CERTIFICATE OF PARTICIPATION", "", 1, "C", false, 0, "")

		// Title
		pdf.SetFont("Arial", "B", 28)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(277, 14, "SERTIFIKAT PENGHARGAAN", "", 1, "C", false, 0, "")

		pdf.Ln(4)
		pdf.SetFont("Arial", "", 11)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(277, 6, "Diberikan dengan bangga kepada:", "", 1, "C", false, 0, "")

		// Archer Name
		pdf.Ln(2)
		pdf.SetFont("Arial", "B", 24)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(277, 12, d.ArcherName, "", 1, "C", false, 0, "")

		// Underline for name
		pdf.SetLineWidth(1.0)
		pdf.SetDrawColor(183, 251, 35)
		pdf.Line(98, 80, 198, 80)

		// Description Text
		pdf.SetY(90)
		pdf.SetFont("Arial", "", 12)
		pdf.SetTextColor(51, 65, 85)
		descLine1 := fmt.Sprintf("Atas keikutsertaan dan prestasi pada kejuaraan panahan resmi")
		descLine2 := fmt.Sprintf("%s", d.EventName)
		descLine3 := fmt.Sprintf("Kategori: %s", d.CategoryName)

		pdf.CellFormat(277, 7, descLine1, "", 1, "C", false, 0, "")
		pdf.SetFont("Arial", "B", 14)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(277, 8, descLine2, "", 1, "C", false, 0, "")
		pdf.SetFont("Arial", "B", 11)
		pdf.SetTextColor(71, 85, 105)
		pdf.CellFormat(277, 7, descLine3, "", 1, "C", false, 0, "")

		// Footer Grid
		pdf.SetY(148)
		pdf.SetX(25)
		pdf.SetFont("Courier", "B", 9)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(120, 5, fmt.Sprintf("No. Sertifikat: %s", certNo), "", 1, "L", false, 0, "")

		pdf.SetX(25)
		pdf.CellFormat(120, 5, fmt.Sprintf("Tanggal Terbit : %s", issueDate.Format("02 January 2006")), "", 0, "L", false, 0, "")

		// Signatory Column
		pdf.SetX(180)
		pdf.SetFont("Arial", "B", 11)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(90, 6, d.OrganizerName, "", 1, "C", false, 0, "")
		pdf.SetX(180)
		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(90, 5, "Penyelenggara Event", "", 1, "C", false, 0, "")

		// Verification info at bottom right
		pdf.SetY(182)
		pdf.SetX(25)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(148, 163, 184)
		pdf.CellFormat(247, 5, fmt.Sprintf("Verifikasi Keaslian: %s/certificates/verify/%s", utils.GetAPIBaseURL(), certNo), "", 1, "R", false, 0, "")

		download := c.Query("download")
		disposition := "inline"
		if download == "true" {
			disposition = "attachment"
		}
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("%s; filename=Certificate-%s.pdf", disposition, certNo))

		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencetak sertifikat PDF"})
		}
	}
}

// VerifyCertificate handles public authenticity verification of a certificate
func VerifyCertificate(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		certNo := c.Param("cert_no")

		type CertDetails struct {
			CertificateNo string    `db:"certificate_no"`
			IssueDate     time.Time `db:"issue_date"`
			ArcherName    string    `db:"archer_name"`
			EventName     string    `db:"event_name"`
			CategoryName  string    `db:"category_name"`
			OrganizerName string    `db:"organizer_name"`
		}

		var d CertDetails
		query := `
			SELECT 
				ac.certificate_no,
				ac.issue_date,
				COALESCE(a.full_name, 'Peserta Archeris') as archer_name,
				e.name as event_name,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' - ', COALESCE(rag.name, ''))) as category_name,
				COALESCE(o.name, 'Panitia Pelaksana') as organizer_name
			FROM archer_certificates ac
			JOIN tournament_participants ep ON ac.registration_id = ep.uuid
			JOIN tournaments e ON ac.tournament_id = e.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN organizers o ON e.organizer_id = o.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE ac.certificate_no = ? OR ac.uuid = ?
			LIMIT 1
		`
		err := db.Get(&d, query, certNo, certNo)
		if err != nil {
			c.JSON(http.StatusNotFound, models.VerifyCertificateResponse{
				Valid:         false,
				CertificateNo: certNo,
				Status:        "Sertifikat Tidak Ditemukan atau Tidak Valid",
			})
			return
		}

		c.JSON(http.StatusOK, models.VerifyCertificateResponse{
			Valid:         true,
			CertificateNo: d.CertificateNo,
			ArcherName:    d.ArcherName,
			EventName:     d.EventName,
			CategoryName:  d.CategoryName,
			OrganizerName: d.OrganizerName,
			IssueDate:     d.IssueDate,
			Status:        "Tersertifikasi Asli & Terverifikasi oleh Archeris.net",
		})
	}
}


type MatchedCert struct {
	Filename      string `json:"filename"`
	ArcherName    string `json:"archer_name"`
	AthleteCode   string `json:"athlete_code"`
	CategoryName  string `json:"category_name"`
	CertificateNo string `json:"certificate_no"`
	PDFURL        string `json:"pdf_url"`
}

type UnmatchedCert struct {
	Filename string `json:"filename"`
	PDFURL   string `json:"pdf_url"`
}

type ParticipantInfo struct {
	RegistrationID string `db:"registration_id"`
	ArcherID       string `db:"archer_id"`
	AthleteCode    string `db:"athlete_code"`
	FullName       string `db:"full_name"`
	ArcherUUID     string `db:"archer_uuid"`
	CategoryName   string `db:"category_name"`
	CategoryCode   string `db:"category_code"`
}

type ExtractedCertFile struct {
	RelativePath string
	Filename     string
	Data         []byte
}

func isCertFileExt(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".pdf" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp"
}

func cleanString(s string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")
	return strings.ToLower(reg.ReplaceAllString(s, ""))
}

func matchParticipant(relPath string, participants []ParticipantInfo) *ParticipantInfo {
	baseName := filepath.Base(relPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	fullPathLower := strings.ToLower(relPath)
	cleanPath := cleanString(relPath)
	cleanBase := cleanString(nameWithoutExt)

	var candidates []ParticipantInfo

	// 1. Direct Athlete Code match
	for _, p := range participants {
		if p.AthleteCode != "" {
			cleanCode := cleanString(p.AthleteCode)
			if cleanCode != "" && strings.Contains(cleanPath, cleanCode) {
				candidates = append(candidates, p)
			}
		}
	}

	// 2. Full Name match if no athlete code match
	if len(candidates) == 0 {
		for _, p := range participants {
			if p.FullName != "" {
				cleanFullName := cleanString(p.FullName)
				if len(cleanFullName) >= 3 && (strings.Contains(cleanPath, cleanFullName) || strings.Contains(cleanFullName, cleanBase)) {
					candidates = append(candidates, p)
				}
			}
		}
	}

	// 3. UUID match
	if len(candidates) == 0 {
		for _, p := range participants {
			if (p.ArcherUUID != "" && strings.Contains(fullPathLower, strings.ToLower(p.ArcherUUID))) ||
				(p.RegistrationID != "" && strings.Contains(fullPathLower, strings.ToLower(p.RegistrationID))) {
				candidates = append(candidates, p)
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return &candidates[0]
	}

	// 4. Multi-category disambiguation: match category name / code in path
	for _, p := range candidates {
		cleanCatName := cleanString(p.CategoryName)
		cleanCatCode := cleanString(p.CategoryCode)
		if (cleanCatName != "" && strings.Contains(cleanPath, cleanCatName)) ||
			(cleanCatCode != "" && strings.Contains(cleanPath, cleanCatCode)) {
			return &p
		}
	}

	// Sub-words matching for category (e.g., "recurve", "barebow", "compound", "standard")
	for _, p := range candidates {
		catWords := strings.Fields(strings.ToLower(p.CategoryName))
		for _, w := range catWords {
			if len(w) >= 4 && strings.Contains(fullPathLower, w) {
				return &p
			}
		}
	}

	return &candidates[0]
}

func extractArchiveFiles(filename string, data []byte) ([]ExtractedCertFile, error) {
	var extracted []ExtractedCertFile
	lowerName := strings.ToLower(filename)

	if strings.HasSuffix(lowerName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, zf := range zr.File {
			if zf.FileInfo().IsDir() || strings.Contains(zf.Name, "__MACOSX") || strings.HasPrefix(filepath.Base(zf.Name), ".") {
				continue
			}
			ext := strings.ToLower(filepath.Ext(zf.Name))
			if !isCertFileExt(ext) {
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				continue
			}
			buf := bytes.NewBuffer(nil)
			io.Copy(buf, rc)
			rc.Close()

			extracted = append(extracted, ExtractedCertFile{
				RelativePath: zf.Name,
				Filename:     filepath.Base(zf.Name),
				Data:         buf.Bytes(),
			})
		}
	} else if strings.HasSuffix(lowerName, ".rar") {
		rarReader, err := rardecode.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		for {
			header, err := rarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			if header.IsDir || strings.Contains(header.Name, "__MACOSX") || strings.HasPrefix(filepath.Base(header.Name), ".") {
				continue
			}
			ext := strings.ToLower(filepath.Ext(header.Name))
			if !isCertFileExt(ext) {
				continue
			}
			buf := bytes.NewBuffer(nil)
			io.Copy(buf, rarReader)

			extracted = append(extracted, ExtractedCertFile{
				RelativePath: header.Name,
				Filename:     filepath.Base(header.Name),
				Data:         buf.Bytes(),
			})
		}
	}

	return extracted, nil
}

func UploadCertificatesZIP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		var participants []ParticipantInfo
		err = db.Select(&participants, `
			SELECT 
				ep.uuid as registration_id,
				COALESCE(ep.archer_id, ep.uuid) as archer_id,
				COALESCE(ep.back_number, a.id, '') as athlete_code,
				COALESCE(a.full_name, '') as full_name,
				COALESCE(a.uuid, '') as archer_uuid,
				COALESCE(
					NULLIF(ec.category_name_custom, ''),
					CONCAT_WS(' ', rbt.name, rag.name, rgd.name),
					''
				) as category_name,
				COALESCE(ec.category_name_custom, '') as category_code
			FROM tournament_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ep.tournament_id = ? OR ep.tournament_id = ?
		`, eventUUID, eventID)

		if err != nil && err.Error() != "sql: no rows in result set" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data partisipan: " + err.Error()})
			return
		}

		uploadDir := filepath.Join(".", "uploads", "certificates", eventUUID)
		os.MkdirAll(uploadDir, os.ModePerm)

		var allFilesToProcess []ExtractedCertFile
		batchName := "Upload Batch"

		// Parse multipart form (supports multiple files, zip, rar, or single images/pdfs)
		form, err := c.MultipartForm()
		if err == nil && form != nil && len(form.File) > 0 {
			for _, fileHeaders := range form.File {
				for _, fh := range fileHeaders {
					lowerName := strings.ToLower(fh.Filename)
					f, err := fh.Open()
					if err != nil {
						continue
					}
					buf := bytes.NewBuffer(nil)
					io.Copy(buf, f)
					f.Close()

					if strings.HasSuffix(lowerName, ".zip") || strings.HasSuffix(lowerName, ".rar") {
						batchName = fh.Filename
						extracted, _ := extractArchiveFiles(fh.Filename, buf.Bytes())
						allFilesToProcess = append(allFilesToProcess, extracted...)
					} else if isCertFileExt(filepath.Ext(lowerName)) {
						if batchName == "Upload Batch" {
							batchName = fh.Filename
						}
						allFilesToProcess = append(allFilesToProcess, ExtractedCertFile{
							RelativePath: fh.Filename,
							Filename:     filepath.Base(fh.Filename),
							Data:         buf.Bytes(),
						})
					}
				}
			}
		} else {
			// Single file fallback
			file, header, err := c.Request.FormFile("file")
			if err != nil {
				file, header, err = c.Request.FormFile("zip_file")
			}
			if err == nil && file != nil {
				defer file.Close()
				buf := bytes.NewBuffer(nil)
				io.Copy(buf, file)
				lowerName := strings.ToLower(header.Filename)

				if strings.HasSuffix(lowerName, ".zip") || strings.HasSuffix(lowerName, ".rar") {
					batchName = header.Filename
					extracted, _ := extractArchiveFiles(header.Filename, buf.Bytes())
					allFilesToProcess = append(allFilesToProcess, extracted...)
				} else if isCertFileExt(filepath.Ext(lowerName)) {
					batchName = header.Filename
					allFilesToProcess = append(allFilesToProcess, ExtractedCertFile{
						RelativePath: header.Filename,
						Filename:     filepath.Base(header.Filename),
						Data:         buf.Bytes(),
					})
				}
			}
		}

		if len(allFilesToProcess) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada berkas sertifikat valid (.zip, .rar, .pdf, .png, .jpg) yang ditemukan"})
			return
		}

		var matched []MatchedCert
		var unmatched []UnmatchedCert
		batchID := uuid.New().String()

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		for _, item := range allFilesToProcess {
			match := matchParticipant(item.RelativePath, participants)

			// Generate safe unique filename on disk
			safePrefix := uuid.New().String()[:8]
			cleanBaseName := strings.ReplaceAll(filepath.Base(item.Filename), " ", "_")
			diskFilename := fmt.Sprintf("%s_%s", safePrefix, cleanBaseName)
			outPath := filepath.Join(uploadDir, diskFilename)

			err := os.WriteFile(outPath, item.Data, 0644)
			if err != nil {
				continue
			}

			fileURL := "/uploads/certificates/" + eventUUID + "/" + diskFilename

			if match != nil {
				certNo := fmt.Sprintf("CERT-%d-%s-%s", time.Now().Year(), strings.ToUpper(eventUUID[:6]), strings.ToUpper(uuid.New().String()[:6]))
				_, _ = tx.Exec(`
					INSERT INTO archer_certificates (uuid, tournament_id, archer_id, registration_id, certificate_no, pdf_url, original_filename, upload_batch_id, issue_date, created_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
					ON DUPLICATE KEY UPDATE pdf_url = VALUES(pdf_url), original_filename = VALUES(original_filename), upload_batch_id = VALUES(upload_batch_id)
				`, uuid.New().String(), eventUUID, match.ArcherID, match.RegistrationID, certNo, fileURL, item.Filename, batchID)

				matched = append(matched, MatchedCert{
					Filename:      item.RelativePath,
					ArcherName:    match.FullName,
					AthleteCode:   match.AthleteCode,
					CategoryName:  match.CategoryName,
					CertificateNo: certNo,
					PDFURL:        fileURL,
				})
			} else {
				unmatched = append(unmatched, UnmatchedCert{
					Filename: item.RelativePath,
					PDFURL:   fileURL,
				})
			}
		}

		var organizerID string
		_ = tx.Get(&organizerID, "SELECT COALESCE(organizer_id, '') FROM tournaments WHERE uuid = ?", eventUUID)

		_, err = tx.Exec(`
			INSERT INTO certificate_upload_batches (uuid, tournament_id, organizer_id, zip_filename, total_files, matched, unmatched, uploaded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
		`, batchID, eventUUID, organizerID, batchName, len(allFilesToProcess), len(matched), len(unmatched))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data batch sertifikat", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data sertifikat", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"batch_id":        batchID,
			"total_files":     len(allFilesToProcess),
			"matched_count":   len(matched),
			"unmatched_count": len(unmatched),
			"matched":         matched,
			"unmatched":       unmatched,
		})
	}
}

// GetBatchProgress returns the extraction and matching progress for a certificate upload batch
func GetBatchProgress(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		batchID := c.Param("batchId")

		type BatchProgress struct {
			UUID           string    `db:"uuid" json:"batch_id"`
			EventID        string    `db:"tournament_id" json:"event_id"`
			ZipFilename    string    `db:"zip_filename" json:"zip_filename"`
			TotalFiles     int       `db:"total_files" json:"total_files"`
			ProcessedFiles int       `db:"processed_files" json:"processed_files"`
			Matched        int       `db:"matched" json:"matched"`
			Unmatched      int       `db:"unmatched" json:"unmatched"`
			Status         string    `db:"status" json:"status"`
			ErrorMessage   *string   `db:"error_message" json:"error_message"`
			UploadedAt     time.Time `db:"uploaded_at" json:"uploaded_at"`
		}

		var batch BatchProgress
		err := db.Get(&batch, `
			SELECT uuid, tournament_id, zip_filename, total_files, COALESCE(processed_files, 0) as processed_files,
				matched, unmatched, COALESCE(status, 'completed') as status, error_message, uploaded_at
			FROM certificate_upload_batches
			WHERE uuid = ?
			LIMIT 1
		`, batchID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Batch tidak ditemukan"})
			return
		}

		percent := 100
		if batch.TotalFiles > 0 {
			percent = int((float64(batch.ProcessedFiles) / float64(batch.TotalFiles)) * 100)
			if percent > 100 {
				percent = 100
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"batch_id":        batch.UUID,
			"status":          batch.Status,
			"total_files":     batch.TotalFiles,
			"processed_files": batch.ProcessedFiles,
			"matched_count":   batch.Matched,
			"unmatched_count": batch.Unmatched,
			"progress_pct":    percent,
			"error_message":   batch.ErrorMessage,
		})
	}
}

func GetCertificateUploadBatches(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			eventUUID = eventID
		}
		var batches []map[string]interface{}
		
		rows, err := db.Queryx("SELECT uuid, tournament_id as event_id, organizer_id, zip_filename, total_files, matched, unmatched, uploaded_at FROM certificate_upload_batches WHERE tournament_id = ? OR tournament_id = ? ORDER BY uploaded_at DESC", eventUUID, eventID)
		if err != nil {
			c.JSON(http.StatusOK, []interface{}{})
			return
		}
		defer rows.Close()

		for rows.Next() {
			results := make(map[string]interface{})
			err = rows.MapScan(results)
			if err == nil {
				// Convert []byte to string for string types
				for k, v := range results {
					if b, ok := v.([]byte); ok {
						results[k] = string(b)
					}
				}
				batches = append(batches, results)
			}
		}
		
		if batches == nil {
			batches = []map[string]interface{}{}
		}
		c.JSON(http.StatusOK, batches)
	}
}

func UploadParticipantCertificate(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		type PartData struct {
			UUID        string `db:"uuid"`
			ArcherID    string `db:"archer_id"`
			FullName    string `db:"full_name"`
			AthleteCode string `db:"athlete_code"`
		}
		var part PartData
		err = db.Get(&part, `
			SELECT ep.uuid, COALESCE(ep.archer_id, ep.uuid) AS archer_id, COALESCE(a.full_name, '') AS full_name, COALESCE(ep.back_number, a.id, '') AS athlete_code
			FROM tournament_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE (ep.uuid = ? OR ep.archer_id = ?) AND (ep.tournament_id = ? OR ep.tournament_id = ?)
			LIMIT 1
		`, participantID, participantID, eventUUID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data peserta tidak ditemukan"})
			return
		}

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Berkas sertifikat tidak ditemukan"})
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if !isCertFileExt(ext) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format berkas tidak didukung. Gunakan .pdf, .png, .jpg, .jpeg, atau .webp"})
			return
		}

		uploadDir := filepath.Join(".", "uploads", "certificates", eventUUID)
		_ = os.MkdirAll(uploadDir, os.ModePerm)
		safePrefix := uuid.New().String()[:8]
		safeFilename := fmt.Sprintf("%s_%s", safePrefix, filepath.Base(header.Filename))
		outPath := filepath.Join(uploadDir, safeFilename)
		outFile, err := os.Create(outPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan berkas"})
			return
		}
		_, err = io.Copy(outFile, file)
		outFile.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menulis berkas"})
			return
		}

		pdfURL := "/uploads/certificates/" + eventUUID + "/" + safeFilename
		certNo := fmt.Sprintf("CERT-%d-%s-%s", time.Now().Year(), strings.ToUpper(eventUUID[:6]), strings.ToUpper(uuid.New().String()[:6]))

		certUUID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO archer_certificates (uuid, tournament_id, archer_id, registration_id, certificate_no, pdf_url, original_filename, issue_date, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		`, certUUID, eventUUID, part.ArcherID, part.UUID, certNo, pdfURL, header.Filename)
		if err != nil {
			// Update if registration_id duplicate
			db.Exec(`
				UPDATE archer_certificates
				SET pdf_url = ?, original_filename = ?
				WHERE tournament_id = ? AND registration_id = ?
			`, pdfURL, header.Filename, eventUUID, part.UUID)
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         "Sertifikat berhasil diunggah",
			"certificate_no":  certNo,
			"pdf_url":         pdfURL,
			"participant_id":  part.UUID,
			"registration_id": part.UUID,
			"archer_id":       part.ArcherID,
		})
	}
}

func ManualAssignCertificate(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		var req struct {
			BatchID          string `json:"batch_id"`
			OriginalFilename string `json:"original_filename"`
			ParticipantID    string `json:"participant_id"`
			PDFURL           string `json:"pdf_url"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid"})
			return
		}

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		var archerID string
		err = db.Get(&archerID, "SELECT COALESCE(archer_id, uuid) FROM tournament_participants WHERE uuid = ?", req.ParticipantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Peserta tidak ditemukan"})
			return
		}

		certNo := fmt.Sprintf("CERT-%d-%s-%s", time.Now().Year(), strings.ToUpper(eventUUID[:6]), strings.ToUpper(uuid.New().String()[:6]))

		_, err = db.Exec(`
			INSERT INTO archer_certificates (uuid, tournament_id, archer_id, registration_id, certificate_no, pdf_url, original_filename, upload_batch_id, issue_date, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE pdf_url = VALUES(pdf_url), original_filename = VALUES(original_filename), upload_batch_id = VALUES(upload_batch_id)
		`, uuid.New().String(), eventUUID, archerID, req.ParticipantID, certNo, req.PDFURL, req.OriginalFilename, req.BatchID)

		if err != nil {
			// Update if registration_id duplicate
			db.Exec(`
				UPDATE archer_certificates
				SET pdf_url = ?, original_filename = ?, upload_batch_id = ?
				WHERE tournament_id = ? AND registration_id = ?
			`, req.PDFURL, req.OriginalFilename, req.BatchID, eventUUID, req.ParticipantID)
		}

		if req.BatchID != "" {
			db.Exec(`UPDATE certificate_upload_batches SET matched = matched + 1, unmatched = GREATEST(unmatched - 1, 0) WHERE uuid = ?`, req.BatchID)
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         "Berhasil assign sertifikat",
			"certificate_no":  certNo,
			"participant_id":  req.ParticipantID,
			"registration_id": req.ParticipantID,
			"pdf_url":         req.PDFURL,
		})
	}
}

func GetEventCertificates(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			eventUUID = eventID
		}
		
		var certs []map[string]interface{}
		
		query := `
		SELECT 
			ac.uuid, ac.tournament_id as event_id, ac.archer_id, ac.registration_id, ac.registration_id AS participant_id,
			ac.certificate_no, ac.issue_date, ac.pdf_url, ac.original_filename, ac.created_at,
			COALESCE(a.full_name, '') as archer_name,
			COALESCE(ep.back_number, a.id, '') as athlete_code,
			COALESCE(cl.name, '') as club_name,
			COALESCE(
				NULLIF(ec.category_name_custom, ''),
				CONCAT_WS(' ', rbt.name, rag.name, rgd.name),
				''
			) as category_name
		FROM archer_certificates ac
		LEFT JOIN archers a ON ac.archer_id = a.uuid
		LEFT JOIN clubs cl ON a.club_id = cl.uuid
		LEFT JOIN tournament_participants ep ON ac.registration_id = ep.uuid
		LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
		LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
		LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
		LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
		WHERE ac.tournament_id = ? OR ac.tournament_id = ?
		ORDER BY ac.created_at DESC
		`
		
		rows, err := db.Queryx(query, eventUUID, eventID)
		if err != nil {
			c.JSON(http.StatusOK, []interface{}{})
			return
		}
		defer rows.Close()

		for rows.Next() {
			results := make(map[string]interface{})
			err = rows.MapScan(results)
			if err == nil {
				for k, v := range results {
					if b, ok := v.([]byte); ok {
						results[k] = string(b)
					}
				}
				certs = append(certs, results)
			}
		}
		
		if certs == nil {
			certs = []map[string]interface{}{}
		}
		
		c.JSON(http.StatusOK, certs)
	}
}

func DeleteArcherCertificate(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		certID := c.Param("certId")

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			eventUUID = eventID
		}

		if certID == "clear-all" {
			// Clean up physical files in upload directory
			uploadDir := filepath.Join(".", "uploads", "certificates", eventUUID)
			_ = os.RemoveAll(uploadDir)

			_, err = db.Exec("DELETE FROM archer_certificates WHERE tournament_id = ? OR tournament_id = ?", eventUUID, eventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membersihkan sertifikat: " + err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Semua sertifikat turnamen berhasil dibersihkan"})
			return
		}

		// Find certificate to remove physical file if exists
		var certItems []struct {
			UUID   string  `db:"uuid"`
			PDFURL *string `db:"pdf_url"`
		}
		_ = db.Select(&certItems, `
			SELECT uuid, pdf_url FROM archer_certificates 
			WHERE (uuid = ? OR registration_id = ? OR archer_id = ? OR certificate_no = ?)
			  AND (tournament_id = ? OR tournament_id = ?)
		`, certID, certID, certID, certID, eventUUID, eventID)

		for _, it := range certItems {
			if it.PDFURL != nil && *it.PDFURL != "" && strings.HasPrefix(*it.PDFURL, "/uploads/") {
				localPath := filepath.Join(".", *it.PDFURL)
				_ = os.Remove(localPath)
			}
		}

		result, err := db.Exec(`
			DELETE FROM archer_certificates 
			WHERE (uuid = ? OR registration_id = ? OR archer_id = ? OR certificate_no = ?) 
			  AND (tournament_id = ? OR tournament_id = ?)
		`, certID, certID, certID, certID, eventUUID, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus sertifikat: " + err.Error()})
			return
		}

		rowsAff, _ := result.RowsAffected()
		if rowsAff == 0 {
			// Try fallback matching without tournament_id
			_, _ = db.Exec("DELETE FROM archer_certificates WHERE uuid = ? OR registration_id = ?", certID, certID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Sertifikat berhasil dihapus"})
	}
}

// GenerateAllCertificates creates certificates for all paid participants for an event
func GenerateAllCertificates(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			eventUUID = eventID
		}

		type PaidParticipant struct {
			RegistrationID string `db:"registration_id"`
			ArcherID       string `db:"archer_id"`
			AthleteCode    string `db:"athlete_code"`
			FullName       string `db:"full_name"`
		}

		var participants []PaidParticipant
		query := `
			SELECT 
				ep.uuid as registration_id,
				ep.archer_id,
				COALESCE(ep.back_number, CAST(a.id AS CHAR), '') as athlete_code,
				COALESCE(a.full_name, '') as full_name
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			WHERE ep.tournament_id = ? AND ep.payment_status IN ('paid', 'lunas')
		`
		err = db.Select(&participants, query, eventUUID)
		if err != nil && err.Error() != "sql: no rows in result set" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data peserta: " + err.Error()})
			return
		}

		if len(participants) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada peserta dengan status pembayaran Lunas yang terdaftar"})
			return
		}

		generatedCount := 0
		year := time.Now().Year()

		for _, p := range participants {
			var exists string
			err := db.Get(&exists, "SELECT uuid FROM archer_certificates WHERE tournament_id = ? AND registration_id = ? LIMIT 1", eventUUID, p.RegistrationID)
			if err != nil {
				certNo := fmt.Sprintf("CERT-%d-%s", year, strings.ToUpper(strings.ReplaceAll(uuid.New().String()[:8], "-", "")))
				newUUID := uuid.New().String()
				issueDate := time.Now()

				_, err = db.Exec(`
					INSERT INTO archer_certificates (uuid, tournament_id, archer_id, registration_id, certificate_no, issue_date, created_at)
					VALUES (?, ?, ?, ?, ?, ?, NOW())
				`, newUUID, eventUUID, p.ArcherID, p.RegistrationID, certNo, issueDate)
				if err == nil {
					generatedCount++
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         fmt.Sprintf("Berhasil menerbitkan %d sertifikat baru dari %d peserta lunas", generatedCount, len(participants)),
			"generated_count": generatedCount,
			"total_eligible":  len(participants),
		})
	}
}

// ClearAllCertificates deletes all issued certificates for an event
func ClearAllCertificates(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			eventUUID = eventID
		}

		// Clean up physical files in upload directory
		uploadDir := filepath.Join(".", "uploads", "certificates", eventUUID)
		_ = os.RemoveAll(uploadDir)

		_, err = db.Exec("DELETE FROM archer_certificates WHERE tournament_id = ? OR tournament_id = ?", eventUUID, eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membersihkan sertifikat: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Semua sertifikat turnamen berhasil dibersihkan"})
	}
}

