<?php
/**
 * AccreditationPDF.php
 *
 * PDF class for ArcheryHub Accreditation / Check-in lists.
 * Template modeled after ianseo's Accreditation PrintOut module.
 *
 * Supports three layouts selected via type:
 *   alphabetical  – participants sorted A-Z, grouped by first initial
 *   by-club       – participants grouped by club / organisation
 *   session       – participants grouped by qualification session, then target
 *
 * Column widths (all layouts use 180 mm body width, 15 mm margins each side):
 *   Alphabetical / By-club:
 *     No.(8) Nama(52) Klub(38) Sesi(8) Target(10) Umur(16) Divisi(30) Hadir(18) = 180
 *
 *   Session:
 *     Target(12) Nama(54) Klub(40) Umur(16) Divisi(30) Daftar Ulang(14) No.Dada(14) = 180
 */

require_once(__DIR__ . '/lib/tcpdf/tcpdf.php');

class AccreditationPDF extends TCPDF
{
    // -------------------------------------------------------
    // Public properties – set before calling AddPage()
    // -------------------------------------------------------
    public string $TournamentName    = '';
    public string $TournamentLocation = '';
    public string $TournamentDate    = '';   // formatted display string
    public string $DocTitle          = 'Daftar Akreditasi Peserta';

    // -------------------------------------------------------
    // Layout constants
    // -------------------------------------------------------
    const BODY_W   = 180;   // mm – usable body width
    const ROW_H    = 5.5;   // mm – data row height
    const HEAD_H   = 5.5;   // mm – column header height
    const SECT_H   = 6.0;   // mm – section/group header height
    const LEFT_M   = 15;    // mm – left margin

    // Column widths – alpha/by-club layout
    const AW_NO    = 8;
    const AW_NAME  = 52;
    const AW_CLUB  = 38;
    const AW_SES   = 8;
    const AW_TGT   = 10;
    const AW_AGE   = 16;
    const AW_DIV   = 30;
    const AW_CHK   = 18;   // single check-in box

    // Column widths – session layout
    const SW_TGT   = 12;
    const SW_NAME  = 54;
    const SW_CLUB  = 40;
    const SW_AGE   = 16;
    const SW_DIV   = 30;
    const SW_CHK1  = 14;   // "Daftar Ulang" box
    const SW_CHK2  = 14;   // "Nomor Dada" box

    // -------------------------------------------------------
    // Constructor
    // -------------------------------------------------------
    public function __construct(string $title = 'Daftar Akreditasi Peserta')
    {
        parent::__construct('P', 'mm', 'A4');
        $this->DocTitle = $title;
        $this->setPrintHeader(true);
        $this->setPrintFooter(true);
        $this->SetMargins(self::LEFT_M, 36, self::LEFT_M);
        $this->SetHeaderMargin(10);
        $this->SetFooterMargin(12);
        $this->SetAutoPageBreak(true, 22);
    }

    // -------------------------------------------------------
    // Header – printed on every page
    // -------------------------------------------------------
    public function Header(): void
    {
        // Tournament name
        $this->SetFont('helvetica', 'B', 12);
        $this->SetTextColor(0, 0, 0);
        $this->Cell(0, 6, $this->TournamentName ?: 'Kejuaraan Panahan', 0, 1, 'C');

        // Location
        if ($this->TournamentLocation) {
            $this->SetFont('helvetica', '', 8);
            $this->Cell(0, 4, $this->TournamentLocation, 0, 1, 'C');
        }

        // Date
        if ($this->TournamentDate) {
            $this->SetFont('helvetica', 'I', 7);
            $this->Cell(0, 3.5, $this->TournamentDate, 0, 1, 'C');
        }

        $this->Ln(1.5);

        // Document title bar (dark – ianseo style)
        $this->SetFont('helvetica', 'B', 9.5);
        $this->SetFillColor(30, 30, 30);
        $this->SetTextColor(255, 255, 255);
        $this->Cell(0, 6.5, strtoupper($this->DocTitle), 0, 1, 'C', true);
        $this->SetTextColor(0, 0, 0);
        $this->SetFillColor(255, 255, 255);
    }

    // -------------------------------------------------------
    // Footer – printed on every page
    // -------------------------------------------------------
    public function Footer(): void
    {
        $this->SetY(-11);
        $this->SetFont('helvetica', '', 6.5);
        $this->SetTextColor(110, 110, 110);
        $this->Cell(90, 8, '  [ ] = Belum hadir     [v] = Sudah hadir', 0, 0, 'L');
        $this->Cell(90, 8, 'Hal. ' . $this->getAliasNumPage() . '/' . $this->getAliasNbPages()
            . '   ArcheryHub.id   ' . date('d/m/Y H:i'), 0, 0, 'R');
    }

    // -------------------------------------------------------
    // Section header bar  (dark, white text – ianseo style)
    // Used for: alphabet letter blocks, session blocks, club blocks
    // -------------------------------------------------------
    public function DrawSectionHeader(string $label, bool $breakIfFull = true): void
    {
        if ($breakIfFull && $this->GetY() > 255) {
            $this->AddPage();
        } else {
            $this->Ln(2);
        }
        $this->SetFont('helvetica', 'B', 9);
        $this->SetFillColor(40, 40, 40);
        $this->SetTextColor(255, 255, 255);
        $this->SetDrawColor(40, 40, 40);
        $this->SetLineWidth(0.3);
        $this->Cell(self::BODY_W, self::SECT_H, '  ' . $label, 1, 1, 'L', true);
        $this->SetTextColor(0, 0, 0);
        $this->SetFillColor(255, 255, 255);
        $this->SetDrawColor(170, 170, 170);
        $this->SetLineWidth(0.2);
    }

    // -------------------------------------------------------
    // Sub-group header bar (light grey)
    // Used for: target sub-groups within a session, etc.
    // -------------------------------------------------------
    public function DrawSubHeader(string $label): void
    {
        $this->SetFont('helvetica', 'B', 7.5);
        $this->SetFillColor(215, 215, 215);
        $this->SetTextColor(0, 0, 0);
        $this->SetDrawColor(170, 170, 170);
        $this->Cell(self::BODY_W, 5.0, '  ' . $label, 1, 1, 'L', true);
        $this->SetFillColor(255, 255, 255);
    }

    // -------------------------------------------------------
    // Column header row – Alphabetical / By-club layout
    // -------------------------------------------------------
    public function DrawAlphaTableHeader(): void
    {
        $this->SetFont('helvetica', 'B', 7);
        $this->SetFillColor(200, 200, 200);
        $this->SetTextColor(0, 0, 0);
        $this->SetDrawColor(150, 150, 150);
        $this->SetLineWidth(0.3);

        $this->Cell(self::AW_NO,   self::HEAD_H, 'No.',               1, 0, 'C', true);
        $this->Cell(self::AW_NAME, self::HEAD_H, 'Nama Peserta',      1, 0, 'L', true);
        $this->Cell(self::AW_CLUB, self::HEAD_H, 'Klub / Organisasi', 1, 0, 'L', true);
        $this->Cell(self::AW_SES,  self::HEAD_H, 'Ses',               1, 0, 'C', true);
        $this->Cell(self::AW_TGT,  self::HEAD_H, 'Target',            1, 0, 'C', true);
        $this->Cell(self::AW_AGE,  self::HEAD_H, 'Kel. Umur',         1, 0, 'C', true);
        $this->Cell(self::AW_DIV,  self::HEAD_H, 'Divisi / Busur',    1, 0, 'C', true);
        $this->Cell(self::AW_CHK,  self::HEAD_H, 'Hadir  [ ]',        1, 1, 'C', true);

        $this->SetDrawColor(170, 170, 170);
        $this->SetLineWidth(0.2);
    }

    // -------------------------------------------------------
    // Column header row – Session layout
    // -------------------------------------------------------
    public function DrawSessionTableHeader(): void
    {
        $this->SetFont('helvetica', 'B', 7);
        $this->SetFillColor(200, 200, 200);
        $this->SetTextColor(0, 0, 0);
        $this->SetDrawColor(150, 150, 150);
        $this->SetLineWidth(0.3);

        $this->Cell(self::SW_TGT,  self::HEAD_H, 'Target',            1, 0, 'C', true);
        $this->Cell(self::SW_NAME, self::HEAD_H, 'Nama Peserta',      1, 0, 'L', true);
        $this->Cell(self::SW_CLUB, self::HEAD_H, 'Klub / Organisasi', 1, 0, 'L', true);
        $this->Cell(self::SW_AGE,  self::HEAD_H, 'Kel. Umur',         1, 0, 'C', true);
        $this->Cell(self::SW_DIV,  self::HEAD_H, 'Divisi / Busur',    1, 0, 'C', true);
        $this->SetFont('helvetica', 'B', 6);
        $this->Cell(self::SW_CHK1, self::HEAD_H, 'Daftar Ulang [ ]',  1, 0, 'C', true);
        $this->Cell(self::SW_CHK2, self::HEAD_H, 'Nomor Dada  [ ]',   1, 1, 'C', true);

        $this->SetDrawColor(170, 170, 170);
        $this->SetLineWidth(0.2);
    }

    // -------------------------------------------------------
    // Data row – Alphabetical / By-club layout
    // -------------------------------------------------------
    public function DrawAlphaRow(
        int    $no,
        string $name,
        string $club,
        string $session,
        string $target,
        string $ageGroup,
        string $division,
        bool   $fill = false
    ): void {
        $this->SetFont('helvetica', '', 7);
        $this->SetTextColor(0, 0, 0);
        $this->SetFillColor($fill ? 245 : 255, $fill ? 245 : 255, $fill ? 245 : 255);
        $this->SetDrawColor(170, 170, 170);

        $this->Cell(self::AW_NO,   self::ROW_H, $no,              1, 0, 'C', $fill);
        $this->Cell(self::AW_NAME, self::ROW_H, $name,            1, 0, 'L', $fill);
        $this->Cell(self::AW_CLUB, self::ROW_H, $club ?: '-',     1, 0, 'L', $fill);
        $this->Cell(self::AW_SES,  self::ROW_H, $session ?: '-',  1, 0, 'C', $fill);
        $this->Cell(self::AW_TGT,  self::ROW_H, $target ?: '-',   1, 0, 'C', $fill);
        $this->Cell(self::AW_AGE,  self::ROW_H, $ageGroup ?: '-', 1, 0, 'C', $fill);
        $this->Cell(self::AW_DIV,  self::ROW_H, $division ?: '-', 1, 0, 'L', $fill);
        $this->Cell(self::AW_CHK,  self::ROW_H, '',               1, 1, 'C', false); // empty check-in
    }

    // -------------------------------------------------------
    // Data row – Session layout
    // -------------------------------------------------------
    public function DrawSessionRow(
        string $target,
        string $name,
        string $club,
        string $ageGroup,
        string $division,
        bool   $fill = false
    ): void {
        $this->SetFont('helvetica', '', 7);
        $this->SetTextColor(0, 0, 0);
        $this->SetFillColor($fill ? 245 : 255, $fill ? 245 : 255, $fill ? 245 : 255);
        $this->SetDrawColor(170, 170, 170);

        $this->Cell(self::SW_TGT,  self::ROW_H, $target ?: '-',   1, 0, 'C', $fill);
        $this->Cell(self::SW_NAME, self::ROW_H, $name,            1, 0, 'L', $fill);
        $this->Cell(self::SW_CLUB, self::ROW_H, $club ?: '-',     1, 0, 'L', $fill);
        $this->Cell(self::SW_AGE,  self::ROW_H, $ageGroup ?: '-', 1, 0, 'C', $fill);
        $this->Cell(self::SW_DIV,  self::ROW_H, $division ?: '-', 1, 0, 'L', $fill);
        $this->Cell(self::SW_CHK1, self::ROW_H, '',               1, 0, 'C', false); // Daftar Ulang
        $this->Cell(self::SW_CHK2, self::ROW_H, '',               1, 1, 'C', false); // Nomor Dada
    }

    // -------------------------------------------------------
    // Group total line  (right-aligned italic, ianseo style)
    // -------------------------------------------------------
    public function DrawGroupTotal(int $count): void
    {
        $this->SetFont('helvetica', 'I', 6.5);
        $this->SetTextColor(80, 80, 80);
        $this->Cell(self::BODY_W, 4, 'Jumlah peserta: ' . $count . ' orang  ', 0, 1, 'R');
        $this->SetTextColor(0, 0, 0);
    }

    // -------------------------------------------------------
    // Helper: does the current position fit N more rows?
    // -------------------------------------------------------
    public function CanFit(int $rows = 1): bool
    {
        // 297mm page – 22mm autobreak zone – rows
        return $this->GetY() <= (297 - 22 - ($rows * self::ROW_H));
    }

    // -------------------------------------------------------
    // Helper: reprint table header when a page break has
    // occurred mid-group (match ianseo's "Continue" behaviour)
    // -------------------------------------------------------
    public function RepeatHeaderOnNewPage(string $type): void
    {
        $this->AddPage();
        if ($type === 'session') {
            $this->DrawSessionTableHeader();
        } else {
            $this->DrawAlphaTableHeader();
        }
    }
}
