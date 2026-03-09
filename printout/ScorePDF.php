<?php
require_once(__DIR__ . '/lib/tcpdf/tcpdf.php');

class ScorePDF extends TCPDF {
    public $PrintLogo = true;
    public $PrintHeader = true;
    public $PrintDrawing = true;
    public $PrintFlags = true;
    public $PrintTotalCols = false;
    public $PrintBarcode = true;
    public $FillWithArrows = false;
    public $PrintLineNo = true;
    public $NoTensOnlyX = false;
    public $ScoreQrPersonal = false;
    public $QRCode = [];
    public $FontStd = 'helvetica';
    public $FontStdMedium = 'helvetica';
    public $FontStdSmall = 'helvetica';
    public $FontFix = 'courier';
    public $ToPaths = ['ToLeft' => '', 'ToRight' => '', 'ToBottom' => ''];

    const sideMargin = 10;

    public function __construct($Portrait = true) {
        parent::__construct(($Portrait ? 'P' : 'L'), 'mm', 'A4');
        $this->setPrintHeader(false);
        $this->setPrintFooter(false);
        $this->SetMargins(10, 10, 10);
        $this->SetAutoPageBreak(false, 10);
        $this->SetSubject('Scoresheet');
        $this->SetColors();
    }

    public function SetColors($Datum = false, $Light = false) {
        if ($this->PrintDrawing) {
            $this->SetTextColor(0, 0, 0);
            $this->SetDrawColor(51, 51, 51);
            if ($Light)
                $this->SetFillColor(248, 248, 248);
            else
                $this->SetFillColor(232, 232, 232);
        } else {
            $this->SetDrawColor(255, 255, 255);
            $this->SetFillColor(255, 255, 255);
            if ($Datum)
                $this->SetTextColor(0, 0, 0);
            else
                $this->SetTextColor(255, 255, 255);
        }
    }

    public function DrawScoreNew($TopX, $TopY, $Width, $Height, $Distance = 0, $Data = array()) {
        $NumEnd = isset($Data['NumEnds']) ? (int)$Data['NumEnds'] : 12;
        $NumArrow = isset($Data['NumArrows']) ? (int)$Data['NumArrows'] : 3;

        // Proportions from Ianseo ScorecardsLib.php
        $CellW = ($Width / ($NumArrow + 5));
        $EndNumCellW = 0.8 * $CellW;
        $ArCellW = $CellW;
        $TotalCellW = 1.4 * $CellW;
        $XNineW = 0.7 * $CellW;

        $prnGolds = isset($Data['Golds']) ? $Data['Golds'] : '10';
        $prnXNine = isset($Data['XNine']) ? $Data['XNine'] : 'X';

        $TopOffset = 30;
        
        // Calculate CellH like Ianseo
        // CellH = min(10, ($Height - 41 - BottomImage - 4*ArcInfo) / (NumEnd + ExtraRows))
        $ExtraRows = 3; 
        $CellH = min(10, ($Height - 35) / ($NumEnd + $ExtraRows));

        $this->SetColors();

        // 1. HEADER - Competition Info
        if ($this->PrintHeader) {
            $this->SetXY($TopX, $TopY);
            $this->SetFont($this->FontStd, 'B', 8);
            $this->MultiCell($Width - 20, 3.5, isset($Data['TournamentName']) ? $Data['TournamentName'] : 'Competition Name', 0, 'L', 0);
            $this->SetFont($this->FontStd, '', 6.5);
            $this->SetX($TopX);
            $this->MultiCell($Width - 20, 3, (isset($Data['TournamentLocation']) ? $Data['TournamentLocation'] : 'Location & City'), 0, 'L', 0);
        }

        // 2. ATHLETE INFO (Exact Ianseo Layout)
        $FlagOffset = 0; 
        $athleteBoxY = $TopY + ($TopOffset * 7/12);
        
        // Archer Label and Name
        $this->SetXY($TopX + $FlagOffset, $athleteBoxY);
        $this->SetFont($this->FontStd, '', 7);
        $this->SetColors(false);
        $labelW = $this->GetStringWidth("Nama Atlet: ");
        $this->Cell($labelW, $TopOffset/6, "Nama Atlet: ", 'B', 0, 'L', 0);
        
        $this->SetFont($this->FontStd, 'B', 13);
        $this->SetColors(true);
        $boxWidth = $Width - (1.6 * $CellW) - $labelW - $FlagOffset;
        $this->Cell($boxWidth, $TopOffset/6 + 1.5, isset($Data['AthleteName']) ? $Data['AthleteName'] : '', 'B', 0, 'L', 0);

        // Target Number (Top Right)
        $this->SetXY($TopX + $Width - (1.4 * $CellW), $TopY + ($TopOffset * 13/24));
        $this->SetFont($this->FontStd, 'B', 20);
        $this->SetColors(true);
        $this->Cell(1.4 * $CellW, $TopOffset * 7/24, isset($Data['TargetNo']) ? $Data['TargetNo'] : '', 0, 0, 'R', 1);

        // Category (Below Target No)
        $this->SetXY($TopX + $Width - (1.4 * $CellW), $TopY + ($TopOffset * 10/12));
        $this->SetFont($this->FontStd, 'B', 8);
        $this->Cell(1.4 * $CellW, $TopOffset * 2/12, isset($Data['Category']) ? $Data['Category'] : ' ', 'T', 0, 'C', 1);

        // Club (formerly Country)
        $this->SetXY($TopX + $FlagOffset, $TopY + ($TopOffset * 19/24));
        $this->SetFont($this->FontStd, '', 7);
        $this->SetColors(false);
        $cLabelW = $this->GetStringWidth("Klub / Organisasi: ");
        $this->Cell($cLabelW, $TopOffset/6, "Klub / Organisasi: ", 'B', 0, 'L', 0);
        $this->SetFont($this->FontStd, 'B', 7);
        $this->SetColors(true);
        $this->Cell($boxWidth, $TopOffset/6, isset($Data['Country']) ? $Data['Country'] : ' ', 'B', 0, 'L', 0);

        // Barcode / EnCode
        if ($this->PrintBarcode) {
            $this->write1DBarcode(isset($Data['TargetNo']) ? $Data['TargetNo'] : 'SCORE', 'C128', $TopX + $Width - 30, $TopY, 25, 5, 0.2, array('text'=>false), 'N');
        }

        // 3. SCORING GRID
        $GridY = $TopY + $TopOffset;
        
        $this->SetXY($TopX, $GridY);
        $this->SetFont($this->FontStd, 'I', 7);
        $this->SetFillColor(248, 248, 248);
        $this->SetColors(true, true);
        $this->Cell($EndNumCellW, $CellH, "", 0, 0, 'C', 1);
        
        $this->SetFillColor(232, 232, 232);
        $this->SetFont($this->FontStd, 'B', 9);
        $this->SetColors(false);
        for ($j = 1; $j <= $NumArrow; $j++) {
            $this->Cell($ArCellW, $CellH, $j, 1, 0, 'C', 1);
        }

        // Session Label (Right part of header)
        $this->SetFont($this->FontStd, 'B', 7);
        $this->SetColors(true);
        $headerRightW = ($TotalCellW * 2) + ($XNineW * 2);
        $this->Cell($headerRightW, $CellH / 2, "Sesi " . (isset($Data['Session']) ? $Data['Session'] : '1'), 1, 1, 'R', 1);
        
        $this->SetXY($TopX + $EndNumCellW + ($ArCellW * $NumArrow), $GridY + $CellH / 2);
        $this->Cell($TotalCellW * 0.75, $CellH / 2, "Total Prog.", 1, 0, 'C', 1);
        $this->Cell($TotalCellW * 1.25, $CellH / 2, "Total", 1, 0, 'C', 1);
        $this->Cell($XNineW, $CellH / 2, $prnGolds, 1, 0, 'C', 1);
        $this->Cell($XNineW, $CellH / 2, $prnXNine, 1, 1, 'C', 1);

        // Rows
        for ($i = 1; $i <= $NumEnd; $i++) {
            $this->SetX($TopX);
            $this->SetFont($this->FontStd, 'B', 8);
            $this->Cell($EndNumCellW, $CellH, $i, 1, 0, 'C', 1);
            
            $this->SetFont($this->FontStd, '', 10);
            for ($j = 0; $j < $NumArrow; $j++) {
                $this->Cell($ArCellW, $CellH, '', 1, 0, 'C', 0);
            }
            $this->Cell($TotalCellW * 0.75, $CellH, '', 1, 0, 'C', 0);
            $this->Cell($TotalCellW * 1.25, $CellH, '', 1, 0, 'C', 0);
            $this->Cell($XNineW, $CellH, '', 1, 0, 'C', 0);
            $this->Cell($XNineW, $CellH, '', 1, 1, 'C', 0);

            // Distance sub-total
            if ($NumEnd == 12 && $i == 6) {
                $this->SetX($TopX + $EndNumCellW);
                $this->SetFont($this->FontStd, 'B', 7);
                $this->Cell($ArCellW * $NumArrow, $CellH, "Sub-total jarak 1", 1, 0, 'C', 1);
                $this->Cell($headerRightW, $CellH, "", 1, 1, 'C', 0);
            }
        }

        // Final Total Row
        $this->SetX($TopX + $EndNumCellW);
        $this->SetFont($this->FontStd, 'B', 9);
        $this->Cell($ArCellW * $NumArrow, $CellH, "Total ", 0, 0, 'R', 0);
        $this->Cell($TotalCellW * 0.75, $CellH, "", 1, 0, 'C', 0);
        $this->Cell($TotalCellW * 1.25, $CellH, "", 1, 0, 'C', 0);
        $this->Cell($XNineW, $CellH, "", 1, 0, 'C', 0);
        $this->Cell($XNineW, $CellH, "", 1, 1, 'C', 0);

        // Signatures
        $SignY = $TopY + $Height - 8;
        $this->SetXY($TopX + 2, $SignY);
        $this->SetFont($this->FontFix, 'BI', 6);
        $this->Cell($Width / 2 - 5, 3, "Tanda Tangan Atlet", 'T', 0, 'C');
        $this->SetX($TopX + $Width / 2 + 3);
        $this->Cell($Width / 2 - 5, 3, "Tanda Tangan Scorer", 'T', 1, 'C');
    }
}
?>
