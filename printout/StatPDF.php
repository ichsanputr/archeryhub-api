<?php
require_once(__DIR__ . '/lib/tcpdf/tcpdf.php');

class StatPDF extends TCPDF {
    public $TournamentName = '';
    public $TournamentLocation = '';
    public $DocTitle = '';

    public function __construct($Title = 'Statistics') {
        parent::__construct('P', 'mm', 'A4');
        $this->DocTitle = $Title;
        $this->setPrintHeader(true);
        $this->setPrintFooter(true);
        $this->SetMargins(15, 30, 15);
        $this->SetHeaderMargin(10);
        $this->SetFooterMargin(10);
        $this->SetAutoPageBreak(TRUE, 20);
    }

    public function Header() {
        $this->SetFont('helvetica', 'B', 12);
        $this->Cell(0, 7, $this->TournamentName, 0, 1, 'C');
        $this->SetFont('helvetica', '', 9);
        $this->Cell(0, 5, $this->TournamentLocation, 0, 1, 'C');
        $this->SetFont('helvetica', 'B', 11);
        $this->Ln(2);
        $this->Cell(0, 7, $this->DocTitle, 'B', 1, 'C');
    }

    public function Footer() {
        $this->SetY(-15);
        $this->SetFont('helvetica', 'I', 8);
        $this->Cell(0, 10, 'Page ' . $this->getAliasNumPage() . '/' . $this->getAliasNbPages() . ' - archeris.net - ' . date('Y-m-d H:i'), 0, 0, 'C');
    }

    public function StyledTable($header, $data, $widths, $aligns = []) {
        $this->SetFont('helvetica', 'B', 8);
        $this->SetFillColor(230, 230, 230);
        
        foreach ($header as $i => $h) {
            $this->Cell($widths[$i], 7, $h, 1, 0, 'C', 1);
        }
        $this->Ln();

        $this->SetFont('helvetica', '', 8);
        $fill = 0;
        foreach ($data as $row) {
            $this->SetFillColor(245, 245, 245);
            foreach ($row as $i => $cell) {
                $align = isset($aligns[$i]) ? $aligns[$i] : 'L';
                $this->Cell($widths[$i], 6, $cell, 1, 0, $align, $fill);
            }
            $this->Ln();
            $fill = !$fill;
        }
    }
}
?>

