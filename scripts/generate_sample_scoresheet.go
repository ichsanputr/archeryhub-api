package main

import (
	"Archeris-api/utils"
	"bytes"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

type SampleEntry struct {
	TargetNo     int
	TargetName   string
	ArcherName   string
	ClubName     string
	CategoryName string
	ArrowsPerEnd int
}

func main() {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)

	eventName := "Tournament Testing 1"
	locDateStr := "Stadion Panahan Kalasan Yogyakarta, Sleman, 20-23 Sep 2026"
	sessionName := "Kualifikasi Sesi 1 (Pagi)"
	dateStr := "20-23 Sep 2026"

	// ── Exact IanSeo 4-Up Layout Coordinates (A4 = 210mm x 297mm) ─────────
	// Margins: 10mm. defScoreW = 90mm, defScoreH = 125.5mm
	// Top Cards (A, B): Y = 10.0mm (ends at Y = 135.5mm)
	// Center QR: Y = 135.5mm (ends at Y = 160.5mm)
	// Bottom Cards (C, D): Y = 161.5mm (ends at Y = 287.0mm)
	quadrantCoords := map[string]struct{ x, y float64 }{
		"A": {x: 10.0, y: 10.0},
		"B": {x: 110.0, y: 10.0},
		"C": {x: 10.0, y: 161.5},
		"D": {x: 110.0, y: 161.5},
	}

	samples := map[string]SampleEntry{
		"A": {TargetNo: 1, TargetName: "1A", ArcherName: "AHMAD SYARIF", ClubName: "D'A - D'Archers Club", CategoryName: "Compound U-18 Men", ArrowsPerEnd: 6},
		"B": {TargetNo: 1, TargetName: "1B", ArcherName: "BAGAS PRASETYO", ClubName: "SAC - Solo Archery", CategoryName: "Compound U-18 Men", ArrowsPerEnd: 6},
		"C": {TargetNo: 1, TargetName: "1C", ArcherName: "JOKO FIRMANSYAH", ClubName: "D'A - D'Archers Club", CategoryName: "Compound U-18 Men", ArrowsPerEnd: 6},
		"D": {TargetNo: 1, TargetName: "1D", ArcherName: "RIZKY KURNIAWAN", ClubName: "D'A - D'Archers Club", CategoryName: "Compound U-18 Men", ArrowsPerEnd: 6},
	}

	renderQuadrantCard := func(x, y float64, slotLetter string, boardNo int, e SampleEntry) {
		targetLabel := fmt.Sprintf("%d%s", boardNo, slotLetter)
		archerName := strings.ToUpper(e.ArcherName)
		clubDisplay := e.ClubName
		catDisplay := e.CategoryName
		arrowsPerEnd := e.ArrowsPerEnd

		// ── 1. HEADER (IanSeo Standard) ──────────────────────────────────
		tgtW := 22.0
		tgtX := x + 90.0 - tgtW - 2.0

		// Tournament Title (Top Left)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(x+2, y+2)
		pdf.CellFormat(86.0-tgtW, 3.5, eventName, "", 1, "L", false, 0, "")

		// Location & Date
		pdf.SetFont("Arial", "", 6.2)
		pdf.SetTextColor(60, 60, 60)
		pdf.SetXY(x+2, y+5.5)
		pdf.CellFormat(86.0-tgtW, 3.2, locDateStr, "", 1, "L", false, 0, "")

		// Target Number (Big Bold IanSeo Target e.g. "1A")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "B", 20)
		pdf.SetXY(tgtX, y+7.0)
		pdf.CellFormat(tgtW, 8.5, targetLabel, "", 0, "R", false, 0, "")

		// Category / Division under Target Number
		pdf.SetFont("Arial", "B", 7.5)
		pdf.SetDrawColor(60, 60, 60)
		pdf.SetLineWidth(0.2)
		pdf.SetXY(tgtX, y+16.0)
		pdf.CellFormat(tgtW, 4.0, catDisplay, "T", 0, "C", false, 0, "")

		// Archer Underline Row
		pdf.SetDrawColor(180, 180, 180)
		pdf.SetLineWidth(0.15)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "", 7)
		labelW := pdf.GetStringWidth("Archer: ")
		pdf.SetXY(x+2, y+9.5)
		pdf.CellFormat(labelW, 5.0, "Archer: ", "B", 0, "L", false, 0, "")

		nameW := (tgtX - 3.0) - (x + 2.0 + labelW)
		if nameW < 10 {
			nameW = 10
		}
		pdf.SetFont("Arial", "B", 10.5)
		pdf.CellFormat(nameW, 5.0, archerName, "B", 0, "L", false, 0, "")

		// Country / Club Underline Row
		pdf.SetFont("Arial", "", 7)
		cLabelW := pdf.GetStringWidth("Country: ")
		pdf.SetXY(x+2, y+15.0)
		pdf.CellFormat(cLabelW, 5.0, "Country: ", "B", 0, "L", false, 0, "")

		cValW := (tgtX - 3.0) - (x + 2.0 + cLabelW)
		if cValW < 10 {
			cValW = 10
		}
		pdf.SetFont("Arial", "B", 7.5)
		pdf.CellFormat(cValW, 5.0, clubDisplay, "B", 0, "L", false, 0, "")

		// ── 2. SCORING TABLE GRID (Exact IanSeo 6-Arrow Matrix) ────────
		yTable := y + 22.0
		pdf.SetDrawColor(51, 51, 51)
		pdf.SetLineWidth(0.18)
		pdf.SetFillColor(232, 232, 232) // IanSeo Gray #E8E8E8
		pdf.SetTextColor(0, 0, 0)

		hHeader := 4.5
		hRow := 5.8
		totalEnds := 6

		if arrowsPerEnd >= 6 {
			// Standard 6-Arrow Format (End 1..6 x 6 arrows = 36 arrows)
			wEnd := 7.5
			wAr := 6.9
			wProg := 12.0
			wTot := 13.5
			w10X := 6.8
			wX := 6.8

			// Header Row
			pdf.SetXY(x+1, yTable)
			pdf.SetFont("Arial", "I", 6.2)
			pdf.CellFormat(wEnd, hHeader, "Session", "1", 0, "C", true, 0, "")

			pdf.SetFont("Arial", "B", 7)
			for a := 1; a <= 6; a++ {
				pdf.CellFormat(wAr, hHeader, fmt.Sprintf("%d", a), "1", 0, "C", true, 0, "")
			}

			pdf.SetFont("Arial", "B", 5.8)
			pdf.CellFormat(wProg, hHeader, "TotalProg", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wTot, hHeader, "Total", "1", 0, "C", true, 0, "")
			pdf.CellFormat(w10X, hHeader, "10+X", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wX, hHeader, "X", "1", 1, "C", true, 0, "")

			// 6 Ends Rows
			for end := 1; end <= totalEnds; end++ {
				pdf.SetXY(x+1, yTable+hHeader+float64(end-1)*hRow)
				pdf.SetFillColor(232, 232, 232)
				pdf.SetFont("Arial", "B", 7)
				pdf.CellFormat(wEnd, hRow, fmt.Sprintf("%d", end), "1", 0, "C", true, 0, "")

				for a := 1; a <= 6; a++ {
					pdf.CellFormat(wAr, hRow, "", "1", 0, "C", false, 0, "")
				}
				pdf.CellFormat(wProg, hRow, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(wTot, hRow, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(w10X, hRow, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(wX, hRow, "", "1", 1, "C", false, 0, "")
			}

			// Total Row
			yTotRow := yTable + hHeader + float64(totalEnds)*hRow
			hTot := 5.8
			pdf.SetXY(x+1, yTotRow)
			pdf.SetFont("Arial", "B", 8.5)
			pdf.CellFormat(wEnd+6*wAr, hTot, "Total ", "0", 0, "R", false, 0, "")
			pdf.CellFormat(wProg, hTot, "", "0", 0, "C", false, 0, "")
			pdf.CellFormat(wTot, hTot, "", "1", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "B", 7.5)
			pdf.CellFormat(w10X, hTot, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(wX, hTot, "", "1", 1, "C", false, 0, "")
		}

		// ── 3. SIGNATURES (IanSeo Style) ──────────────────────────────
		ySig := y + 125.5 - 10.0
		pdf.SetDrawColor(51, 51, 51)
		pdf.SetLineWidth(0.18)

		wSig := 38.0
		pdf.SetFont("Arial", "B", 6.2)
		pdf.SetXY(x+4, ySig)
		pdf.CellFormat(wSig, 3.2, "Archer", "T", 0, "C", false, 0, "")

		pdf.SetXY(x+48, ySig)
		pdf.CellFormat(wSig, 3.2, "Scorer", "T", 1, "C", false, 0, "")

		// Subtle Footer Metadata
		pdf.SetFont("Arial", "I", 5.2)
		pdf.SetTextColor(120, 120, 120)
		pdf.SetXY(x+2, y+122.0)
		pdf.CellFormat(86, 2.5, fmt.Sprintf("%s | %s | %s | WA Rules", eventName, sessionName, dateStr), "", 1, "C", false, 0, "")
	}

	pdf.AddPage()
	boardNo := 1
	boardCode := "001TGT"

	// 1. Render Quadrants A, B, C, D
	for _, letter := range []string{"A", "B", "C", "D"} {
		coord := quadrantCoords[letter]
		entry := samples[letter]
		renderQuadrantCard(coord.x, coord.y, letter, boardNo, entry)
	}

	// 2. Central Target Board QR Code (Exact IanSeo DrawQRCode_ISK_NG Center White Square)
	// Center is (105mm, 148.5mm) -> Square: (92.5, 135.5, 25mm x 25mm)
	qrX := (210.0 - 25.0) / 2.0
	qrY := ((297.0 - 25.0) / 2.0) - 0.5

	// Clean white background square with subtle gray border (exact DrawQRCode_ISK_NG)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(120, 120, 120)
	pdf.SetLineWidth(0.15)
	pdf.Rect(qrX, qrY, 25.0, 25.0, "FD")

	// High-density QR Code inside the square with space for board code label below
	qrPng, err := utils.GenerateQRCode(boardCode, 256)
	if err == nil && len(qrPng) > 0 {
		imgName := fmt.Sprintf("qr_board_%d_%s", boardNo, boardCode)
		opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.RegisterImageOptionsReader(imgName, opt, bytes.NewReader(qrPng))
		pdf.ImageOptions(imgName, qrX+3.0, qrY+1.5, 19.0, 19.0, false, opt, 0, "")

		// Clean small board code label below the QR code
		pdf.SetFont("Arial", "B", 6.5)
		pdf.SetTextColor(40, 40, 40)
		pdf.SetXY(qrX, qrY+20.8)
		pdf.CellFormat(25.0, 3.2, boardCode, "0", 0, "C", false, 0, "")
	}

	outPath := "sample_ianseo_scoresheet.pdf"
	err = pdf.OutputFileAndClose(outPath)
	if err != nil {
		fmt.Printf("Error generating PDF: %v\n", err)
		return
	}
	fmt.Printf("SUCCESS: Generated %s\n", outPath)
}
