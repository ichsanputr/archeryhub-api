<?php
require_once(__DIR__ . '/lib/tcpdf/tcpdf.php');

class ListPDF extends TCPDF {
    public $TournamentName = '';
    public $TournamentLocation = '';
    public $DocTitle = '';

    public function __construct($Title = 'List') {
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
        $this->Cell(0, 10, 'Page ' . $this->getAliasNumPage() . '/' . $this->getAliasNbPages() . ' - ArcheryHub.id - ' . date('Y-m-d H:i'), 0, 0, 'C');
    }
}
?>
