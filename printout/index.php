<?php
$startTime = microtime(true);
require_once(__DIR__ . '/db.php');
require_once(__DIR__ . '/ScorePDF.php');
require_once(__DIR__ . '/ListPDF.php');

$requestUri = $_SERVER['REQUEST_URI'];
$method = $_SERVER['REQUEST_METHOD'];

function logExecutionTime($action) {
    global $startTime;
    $executionTime = round((microtime(true) - $startTime) * 1000, 2);
    error_log(sprintf("[Printout] %s completed in %s ms", $action, $executionTime));
}

/**
 * Acronym helper for bow divisions
 */
function getAcronym($name) {
    if (!$name) return '-';
    $map = [
        'Recurve' => 'R',
        'Compound' => 'C',
        'Barebow' => 'B',
        'Longbow' => 'L',
        'Standard' => 'S',
        'Traditional' => 'T',
        'National' => 'N'
    ];
    foreach($map as $fullName => $abbr) {
        if (stripos($name, $fullName) !== false) return $abbr;
    }
    return substr($name, 0, 1);
}

// ---------------------------------------------------------
// HEALTH CHECK
// ---------------------------------------------------------
if ($requestUri == '/' || $requestUri == '/index.php' || preg_match('/\/api\/v1\/printout\/?$/', $requestUri)) {
    header('Content-Type: text/html');
    echo "<h1>ArcheryHub Printout Server</h1>";
    echo "<p style='font-size: 1.2em; color: green;'>System working</p>";

    echo "<h3>Extensions Check:</h3><ul>";
    $extensions = ['pdo_mysql', 'mbstring', 'gd', 'curl', 'xml'];
    foreach ($extensions as $ext) {
        $loaded = extension_loaded($ext) ? "<strong style='color:green;'>Loaded</strong>" : "<strong style='color:red;'>Missing</strong>";
        echo "<li>{$ext}: {$loaded}</li>";
    }
    echo "</ul>";

    echo "<h3>Database Check:</h3>";
    try {
        // db.php is required at the very top, so $pdo should already exist
        if (isset($pdo) && $pdo) {
            echo "<p style='color:green;'>MySQL connection successful.</p>";
        } else {
            echo "<p style='color:red;'>MySQL connection variable not found.</p>";
        }
    } catch (Exception $e) {
        echo "<p style='color:red;'>MySQL connection error: " . htmlspecialchars($e->getMessage()) . "</p>";
    }
    exit;
}

// ---------------------------------------------------------
// QUALIFICATION SCORESHEET HANDLER
// ---------------------------------------------------------
if (preg_match('/\/api\/v1\/events\/([^\/]+)\/qualification\/sessions\/([^\/]+)\/scoresheet/', $requestUri, $matches)) {
    $eventIdentifier = $matches[1];
    $sessionCode = $matches[2];

    $categoryId = $_GET['category_id'] ?? null;
    $targetFrom = $_GET['target_from'] ?? null;
    $targetTo = $_GET['target_to'] ?? null;
    $blankMode = (isset($_GET['blank']) && $_GET['blank'] == '1');
    $autoPrint = (isset($_GET['autoprint']) && $_GET['autoprint'] == '1');

    $options = [
        'header' => ($_GET['header'] ?? '1') == '1',
        'images' => ($_GET['images'] ?? '1') == '1',
        'flags' => ($_GET['flags'] ?? '1') == '1',
        'detail_info' => ($_GET['detail_info'] ?? '0') == '1',
        'barcode' => ($_GET['barcode'] ?? '1') == '1',
    ];

    try {
        // 1. Fetch Event
        $stmt = $pdo->prepare("SELECT * FROM events WHERE uuid = ? OR slug = ? LIMIT 1");
        $stmt->execute([$eventIdentifier, $eventIdentifier]);
        $event = $stmt->fetch();
        if (!$event) {
            header('Content-Type: application/json');
            die(json_encode(['error' => 'Event not found']));
        }

        // 2. Fetch Session
        $stmt = $pdo->prepare("SELECT * FROM qualification_sessions WHERE event_uuid = ? AND session_code = ? LIMIT 1");
        $stmt->execute([$event['uuid'], $sessionCode]);
        $session = $stmt->fetch();
        if (!$session) {
            header('Content-Type: application/json');
            die(json_encode(['error' => 'Session not found']));
        }

        // 3. Fetch Assignments
        $query = "
            SELECT 
                qta.uuid, et.target_name, a.full_name as archer_name, a.email as archer_email,
                a.date_of_birth as archer_birthday, c.name as club_name,
                rag.name as category_name, rbt.name as division_name
            FROM qualification_target_assignments qta
            LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
            LEFT JOIN event_participants ep ON qta.participant_uuid = ep.uuid
            LEFT JOIN archers a ON ep.archer_id = a.uuid
            LEFT JOIN clubs c ON a.club_id = c.uuid
            LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
            LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
            LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
            WHERE qta.session_uuid = ?
        ";
        $args = [$session['uuid']];

        if ($categoryId) {
            $query .= " AND ep.category_id = ?";
            $args[] = $categoryId;
        }
        if ($targetFrom) {
            $query .= " AND et.target_name >= ?";
            $args[] = $targetFrom;
        }
        if ($targetTo) {
            $query .= " AND et.target_name <= ?";
            $args[] = $targetTo;
        }

        $query .= " ORDER BY et.target_name ASC";
        $stmt = $pdo->prepare($query);
        $stmt->execute($args);
        $assignments = $stmt->fetchAll();

        // 4. Generate PDF
        $pdf = new ScorePDF(true);
        $pdf->SetTitle('Lembar Skor - ' . $event['name']);
        
        $pdf->PrintHeader = $options['header'];
        $pdf->PrintLogo = $options['images'];
        $pdf->PrintBarcode = $options['barcode'];

        $defW = 90; $defH = 135; $margin = 10;
        $count = 0;

        foreach ($assignments as $row) {
            if ($count % 4 == 0) $pdf->AddPage();

            $pageIndex = $count % 4;
            $xPos = ($pageIndex % 2 == 0) ? $margin : ($margin + $defW + 10);
            $yPos = ($pageIndex < 2) ? $margin : ($margin + $defH + 5);

            $divAbbr = getAcronym($row['division_name']);
            $catName = $row['category_name'];

            $data = [
                'TournamentName' => $event['name'],
                'TournamentLocation' => $event['location'] . ', ' . $event['city'],
                'AthleteName' => $blankMode ? '' : $row['archer_name'],
                'TargetNo' => $row['target_name'],
                'Country' => $blankMode ? '' : ($row['club_name'] ?: '-'),
                'Category' => $divAbbr . ' ' . $catName,
                'NumEnds' => $session['total_ends'],
                'NumArrows' => $session['arrows_per_end'],
                'Email' => $options['detail_info'] ? $row['archer_email'] : '',
                'Birthday' => $options['detail_info'] ? $row['archer_birthday'] : '',
                'Session' => $session['session_code']
            ];

            $pdf->DrawScoreNew($xPos, $yPos, $defW, $defH, 0, $data);
            $count++;
        }

        if ($autoPrint) $pdf->IncludeJS("print(true);");
        logExecutionTime("Scoresheet: $sessionCode");
        $pdf->Output('scoresheet_' . $sessionCode . '.pdf', 'I');
        exit;

    } catch (Exception $e) {
        logExecutionTime("Scoresheet Error: " . rtrim(str_replace("\n", " ", $e->getMessage())));
        header('Content-Type: application/json');
        die(json_encode(['error' => 'Error: ' . $e->getMessage()]));
    }
}

// ---------------------------------------------------------
// PARTICIPANT LIST HANDLER
// ---------------------------------------------------------
if (preg_match('/\/api\/v1\/events\/([^\/]+)\/participants\/printout/', $requestUri, $matches)) {
    $eventIdentifier = $matches[1];
    $type = $_GET['type'] ?? 'alphabetical';

    try {
        // 1. Fetch Event
        $stmt = $pdo->prepare("SELECT * FROM events WHERE uuid = ? OR slug = ? LIMIT 1");
        $stmt->execute([$eventIdentifier, $eventIdentifier]);
        $event = $stmt->fetch();
        if (!$event) {
            header('Content-Type: application/json');
            die(json_encode(['error' => 'Event not found']));
        }

        // 2. Query Participants
        $query = "
            SELECT 
                a.full_name as athlete_name, c.name as club_name,
                rag.name as category_name, rbt.name as division_name,
                qs.session_code, et.target_name
            FROM event_participants ep
            LEFT JOIN archers a ON ep.archer_id = a.uuid
            LEFT JOIN clubs c ON a.club_id = c.uuid
            LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
            LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
            LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
            LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
            LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
            LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
            WHERE ep.event_id = ?
        ";

        if ($type == 'alphabetical') {
            $query .= " ORDER BY a.full_name ASC";
            $title = 'Daftar Peserta - Berdasarkan abjad';
        } else {
            $query .= " ORDER BY c.name ASC, a.full_name ASC";
            $title = 'Daftar Peserta - Per klub / organisasi';
        }

        $stmt = $pdo->prepare($query);
        $stmt->execute([$event['uuid']]);
        $participants = $stmt->fetchAll();

        // 3. Generate PDF
        $pdf = new ListPDF($title);
        $pdf->TournamentName = $event['name'];
        $pdf->TournamentLocation = $event['location'] . ', ' . $event['city'];
        $pdf->AddPage();

        // Header Table
        $pdf->SetFont('helvetica', 'B', 8);
        $pdf->SetFillColor(230, 230, 230);
        $pdf->Cell(10, 7, 'No', 1, 0, 'C', 1);
        $pdf->Cell(50, 7, 'Nama Atlet', 1, 0, 'L', 1);
        $pdf->Cell(45, 7, 'Klub / Organisasi', 1, 0, 'L', 1);
        $pdf->Cell(15, 7, 'Ses.', 1, 0, 'C', 1);
        $pdf->Cell(15, 7, 'Target', 1, 0, 'C', 1);
        $pdf->Cell(45, 7, 'Kategori', 1, 1, 'L', 1);

        $pdf->SetFont('helvetica', '', 8);
        $count = 1;
        $currentClub = '___INIT___';

        foreach ($participants as $row) {
            if ($pdf->GetY() > 265) {
                $pdf->AddPage();
                $pdf->SetFont('helvetica', 'B', 8);
                $pdf->SetFillColor(230, 230, 230);
                $pdf->Cell(10, 7, 'No', 1, 0, 'C', 1);
                $pdf->Cell(50, 7, 'Nama Atlet', 1, 0, 'L', 1);
                $pdf->Cell(45, 7, 'Klub / Organisasi', 1, 0, 'L', 1);
                $pdf->Cell(15, 7, 'Ses.', 1, 0, 'C', 1);
                $pdf->Cell(15, 7, 'Target', 1, 0, 'C', 1);
                $pdf->Cell(45, 7, 'Kategori', 1, 1, 'L', 1);
                $pdf->SetFont('helvetica', '', 8);
            }

            if ($type == 'by-club' && $row['club_name'] !== $currentClub) {
                $currentClub = $row['club_name'];
                $pdf->SetFont('helvetica', 'B', 8);
                $pdf->SetFillColor(245, 245, 245);
                $pdf->Cell(180, 6, ($currentClub ?: 'Individu / Tanpa Klub'), 1, 1, 'L', 1);
                $pdf->SetFont('helvetica', '', 8);
            }

            $divAbbr = getAcronym($row['division_name']);
            $catName = $row['category_name'];

            $pdf->Cell(10, 6, $count++, 1, 0, 'C');
            $pdf->Cell(50, 6, $row['athlete_name'], 1, 0, 'L');
            $pdf->Cell(45, 6, ($row['club_name'] ?: '-'), 1, 0, 'L');
            $pdf->Cell(15, 6, ($row['session_code'] ?: '-'), 1, 0, 'C');
            $pdf->Cell(15, 6, ($row['target_name'] ?: '-'), 1, 0, 'C');
            $pdf->Cell(45, 6, ($divAbbr . ' ' . $catName), 1, 1, 'L');
        }

        if (isset($_GET['autoprint']) && $_GET['autoprint'] == '1') $pdf->IncludeJS("print(true);");
        logExecutionTime("Participant List: $type");
        $pdf->Output('participants_list.pdf', 'I');
        exit;

    } catch (Exception $e) {
        logExecutionTime("Participant List Error: " . rtrim(str_replace("\n", " ", $e->getMessage())));
        header('Content-Type: application/json');
        die(json_encode(['error' => 'Error: ' . $e->getMessage()]));
    }
}

// ---------------------------------------------------------
// STATISTICS - CLASSES AND DIVISIONS
// ---------------------------------------------------------
if (preg_match('/\/api\/v1\/events\/([^\/]+)\/participants\/statistics\-classes/', $requestUri, $matches)) {
    $eventIdentifier = $matches[1];
    require_once(__DIR__ . '/StatPDF.php');

    try {
        // 1. Fetch Event
        $stmt = $pdo->prepare("SELECT * FROM events WHERE uuid = ? OR slug = ? LIMIT 1");
        $stmt->execute([$eventIdentifier, $eventIdentifier]);
        $event = $stmt->fetch();
        if (!$event) {
            header('Content-Type: application/json');
            die(json_encode(['error' => 'Event not found']));
        }

        // 2. Fetch Data
        $query = "
            SELECT 
                rag.name as age_group,
                rbt.name as bow_type,
                count(ep.uuid) as total
            FROM event_participants ep
            INNER JOIN event_categories ec ON ep.category_id = ec.uuid
            INNER JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
            INNER JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
            WHERE ep.event_id = ?
            GROUP BY rag.name, rbt.name
            ORDER BY rag.name, rbt.name
        ";
        $stmt = $pdo->prepare($query);
        $stmt->execute([$event['uuid']]);
        $rows = $stmt->fetchAll();

        // 3. Process Matrix
        $bowTypes = [];
        $ageGroups = [];
        $matrix = [];
        foreach ($rows as $row) {
            $ageGroups[$row['age_group']] = true;
            $bowTypes[$row['bow_type']] = true;
            $matrix[$row['age_group']][$row['bow_type']] = $row['total'];
        }
        $bowTypesList = array_keys($bowTypes);
        sort($bowTypesList);
        $ageGroupsList = array_keys($ageGroups);
        sort($ageGroupsList);

        // 4. Generate PDF
        $pdf = new StatPDF('Statistics (Classes and Divisions)');
        $pdf->TournamentName = $event['name'];
        $pdf->TournamentLocation = $event['location'] . ', ' . $event['city'];
        $pdf->AddPage();

        // Prepare table
        $header = array_merge(['Age Group / Bow Type'], $bowTypesList, ['Total']);
        $colWidths = [50];
        $remainingWidth = 180 - 50;
        $cellCount = count($bowTypesList) + 1;
        $cellWidth = $cellCount > 0 ? round($remainingWidth / $cellCount) : $remainingWidth;
        for ($i = 0; $i < $cellCount; $i++) {
            $colWidths[] = $cellWidth;
        }

        $data = [];
        $colTotals = array_fill(0, count($bowTypesList), 0);
        foreach ($ageGroupsList as $ag) {
            $row = [$ag];
            $rowTot = 0;
            foreach ($bowTypesList as $idx => $bt) {
                $val = $matrix[$ag][$bt] ?? 0;
                $row[] = $val ?: '-';
                $rowTot += $val;
                $colTotals[$idx] += $val;
            }
            $row[] = $rowTot;
            $data[] = $row;
        }

        // Add Footer Total Row
        if ($ageGroupsList) {
            $footer = ['Total'];
            $grandTotal = 0;
            foreach ($colTotals as $ct) {
                $footer[] = $ct;
                $grandTotal += $ct;
            }
            $footer[] = $grandTotal;
            $data[] = $footer;
        }

        $pdf->StyledTable($header, $data, $colWidths, array_fill(1, count($header)-1, 'C'));
        logExecutionTime("Stat Classes");
        $pdf->Output('statistics_classes.pdf', 'I');
        exit;

    } catch (Exception $e) {
        logExecutionTime("Stat Classes Error: " . rtrim(str_replace("\n", " ", $e->getMessage())));
        header('Content-Type: application/json');
        die(json_encode(['error' => 'Error: ' . $e->getMessage()]));
    }
}

// ---------------------------------------------------------
// STATISTICS - CLUBS
// ---------------------------------------------------------
if (preg_match('/\/api\/v1\/events\/([^\/]+)\/participants\/statistics\-clubs/', $requestUri, $matches)) {
    $eventIdentifier = $matches[1];
    require_once(__DIR__ . '/StatPDF.php');

    try {
        // 1. Fetch Event
        $stmt = $pdo->prepare("SELECT * FROM events WHERE uuid = ? OR slug = ? LIMIT 1");
        $stmt->execute([$eventIdentifier, $eventIdentifier]);
        $event = $stmt->fetch();
        if (!$event) {
            header('Content-Type: application/json');
            die(json_encode(['error' => 'Event not found']));
        }

        // 2. Fetch Data
        $query = "
            SELECT 
                COALESCE(c.name, 'Individu / Tanpa Klub') as club_name,
                count(ep.uuid) as total
            FROM event_participants ep
            LEFT JOIN archers a ON ep.archer_id = a.uuid
            LEFT JOIN clubs c ON a.club_id = c.uuid
            WHERE ep.event_id = ?
            GROUP BY club_name
            ORDER BY total DESC, club_name ASC
        ";
        $stmt = $pdo->prepare($query);
        $stmt->execute([$event['uuid']]);
        $rows = $stmt->fetchAll();

        // 3. Generate PDF
        $pdf = new StatPDF('Statistics (Clubs)');
        $pdf->TournamentName = $event['name'];
        $pdf->TournamentLocation = $event['location'] . ', ' . $event['city'];
        $pdf->AddPage();

        $header = ['No', 'Club / Organization', 'Total Participants'];
        $colWidths = [15, 120, 45];
        $data = [];
        $count = 1;
        $grandTotal = 0;
        foreach ($rows as $row) {
            $data[] = [$count++, $row['club_name'], $row['total']];
            $grandTotal += $row['total'];
        }
        if ($rows) {
            $data[] = ['', 'Grand Total', $grandTotal];
        }

        $pdf->StyledTable($header, $data, $colWidths, ['C', 'L', 'C']);
        logExecutionTime("Stat Clubs");
        $pdf->Output('statistics_clubs.pdf', 'I');
        exit;

    } catch (Exception $e) {
        logExecutionTime("Stat Clubs Error: " . rtrim(str_replace("\n", " ", $e->getMessage())));
        header('Content-Type: application/json');
        die(json_encode(['error' => 'Error: ' . $e->getMessage()]));
    }
}

// ---------------------------------------------------------
// ACCREDITATION / CHECK-IN PRINTOUT
// Modeled after ianseo Accreditation > PrintOut.php
//
// Endpoints:
//   GET /api/v1/events/{id}/accreditation/printout?type=alphabetical
//   GET /api/v1/events/{id}/accreditation/printout?type=by-club
//   GET /api/v1/events/{id}/accreditation/printout?type=session[&session=A]
// ---------------------------------------------------------
if (preg_match('/\/api\/v1\/events\/([^\/]+)\/accreditation\/printout/', $requestUri, $matches)) {
    $eventIdentifier = $matches[1];
    $type            = in_array($_GET['type'] ?? '', ['alphabetical', 'by-club', 'session'])
                       ? $_GET['type']
                       : 'alphabetical';
    $filterSession   = isset($_GET['session']) ? trim($_GET['session']) : null;
    $autoPrint       = (isset($_GET['autoprint']) && $_GET['autoprint'] == '1');

    require_once(__DIR__ . '/AccreditationPDF.php');

    try {
        // 1. Fetch event
        $stmt = $pdo->prepare("SELECT * FROM events WHERE uuid = ? OR slug = ? LIMIT 1");
        $stmt->execute([$eventIdentifier, $eventIdentifier]);
        $event = $stmt->fetch();
        if (!$event) {
            header('Content-Type: application/json');
            die(json_encode(['error' => 'Event not found']));
        }

        // 2. Build participant query
        // Joins: archer → club → category → age_group + bow_type → target assignment → session
        $query = "
            SELECT
                a.full_name                                      AS athlete_name,
                COALESCE(c.name, 'Individu / Tanpa Klub')        AS club_name,
                COALESCE(rag.name, '-')                          AS age_group,
                COALESCE(rbt.name, '-')                          AS division_name,
                COALESCE(qs.session_code, '-')                   AS session_code,
                COALESCE(et.target_name, '-')                    AS target_name,
                ep.uuid                                          AS participant_uuid
            FROM event_participants ep
            LEFT JOIN archers a              ON ep.archer_id         = a.uuid
            LEFT JOIN clubs c                ON a.club_id             = c.uuid
            LEFT JOIN event_categories ec    ON ep.category_id        = ec.uuid
            LEFT JOIN ref_age_groups rag     ON ec.category_uuid      = rag.uuid
            LEFT JOIN ref_bow_types rbt      ON ec.division_uuid      = rbt.uuid
            LEFT JOIN qualification_target_assignments qta
                                             ON ep.uuid               = qta.participant_uuid
            LEFT JOIN qualification_sessions qs
                                             ON qta.session_uuid      = qs.uuid
            LEFT JOIN event_targets et       ON qta.target_uuid       = et.uuid
            WHERE ep.event_id = ?
        ";

        $params = [$event['uuid']];

        if ($type === 'session' && $filterSession) {
            $query  .= " AND qs.session_code = ?";
            $params[] = $filterSession;
        }

        if ($type === 'alphabetical') {
            $query .= " ORDER BY a.full_name ASC";
        } elseif ($type === 'by-club') {
            $query .= " ORDER BY c.name ASC, a.full_name ASC";
        } else { // session
            $query .= " ORDER BY qs.session_code ASC, et.target_name ASC, a.full_name ASC";
        }

        $stmt = $pdo->prepare($query);
        $stmt->execute($params);
        $participants = $stmt->fetchAll();

        // 3. Build PDF title
        $typeLabels = [
            'alphabetical' => 'Daftar Peserta – Urutan Abjad',
            'by-club'      => 'Daftar Peserta – Per Klub / Organisasi',
            'session'      => 'Daftar Akreditasi – Per Sesi',
        ];
        $docTitle = $typeLabels[$type] ?? 'Daftar Peserta';

        // 4. Initialise PDF
        $pdf = new AccreditationPDF($docTitle);
        $pdf->TournamentName     = $event['name'] ?? '';
        $pdf->TournamentLocation = trim(($event['location'] ?? '') . ', ' . ($event['city'] ?? ''), ', ');
        // Format event date if available
        if (!empty($event['start_date'])) {
            $ts = strtotime($event['start_date']);
            if ($ts !== false) {
                $months = [
                    1=>'Januari',2=>'Februari',3=>'Maret',4=>'April',5=>'Mei',6=>'Juni',
                    7=>'Juli',8=>'Agustus',9=>'September',10=>'Oktober',11=>'November',12=>'Desember'
                ];
                $pdf->TournamentDate = (int)date('j', $ts) . ' ' . $months[(int)date('n', $ts)] . ' ' . date('Y', $ts);
            }
        }

        $pdf->AddPage();

        // -------------------------------------------------------
        // 5A. ALPHABETICAL – grouped by first initial (ianseo style)
        // -------------------------------------------------------
        if ($type === 'alphabetical') {
            $currentInitial = null;
            $groupCount     = 0;
            $rowNum         = 0;
            $fill           = false;

            foreach ($participants as $row) {
                $initial = strtoupper(mb_substr($row['athlete_name'], 0, 1));

                // New letter group
                if ($initial !== $currentInitial) {
                    // Print previous group total
                    if ($currentInitial !== null && $groupCount > 0) {
                        $pdf->DrawGroupTotal($groupCount);
                    }

                    // If remaining page space is tight, start new page
                    if ($currentInitial !== null && !$pdf->CanFit(4)) {
                        $pdf->AddPage();
                    }

                    $pdf->DrawSectionHeader($initial, false);
                    $pdf->DrawAlphaTableHeader();

                    $currentInitial = $initial;
                    $groupCount     = 0;
                    $fill           = false;
                } elseif (!$pdf->CanFit(1)) {
                    // Page break mid-group – reprint header (ianseo "Segue" behaviour)
                    $pdf->RepeatHeaderOnNewPage('alphabetical');
                    $fill = false;
                }

                $pdf->DrawAlphaRow(
                    ++$rowNum,
                    $row['athlete_name'],
                    $row['club_name'],
                    $row['session_code'],
                    $row['target_name'],
                    $row['age_group'],
                    $row['division_name'],
                    $fill
                );
                $fill = !$fill;
                $groupCount++;
            }

            // Final group total
            if ($groupCount > 0) {
                $pdf->DrawGroupTotal($groupCount);
            }

        // -------------------------------------------------------
        // 5B. BY-CLUB – grouped by club name (ianseo PrnCountry style)
        // -------------------------------------------------------
        } elseif ($type === 'by-club') {
            $currentClub = null;
            $groupCount  = 0;
            $rowNum      = 0;
            $fill        = false;

            foreach ($participants as $row) {
                $club = $row['club_name'];

                // New club group
                if ($club !== $currentClub) {
                    if ($currentClub !== null && $groupCount > 0) {
                        $pdf->DrawGroupTotal($groupCount);
                    }

                    if ($currentClub !== null && !$pdf->CanFit(4)) {
                        $pdf->AddPage();
                    }

                    $pdf->DrawSectionHeader($club, false);
                    $pdf->DrawAlphaTableHeader();

                    $currentClub = $club;
                    $groupCount  = 0;
                    $rowNum      = 0;
                    $fill        = false;
                } elseif (!$pdf->CanFit(1)) {
                    $pdf->RepeatHeaderOnNewPage('by-club');
                    $fill = false;
                }

                $pdf->DrawAlphaRow(
                    ++$rowNum,
                    $row['athlete_name'],
                    $row['club_name'],
                    $row['session_code'],
                    $row['target_name'],
                    $row['age_group'],
                    $row['division_name'],
                    $fill
                );
                $fill = !$fill;
                $groupCount++;
            }

            if ($groupCount > 0) {
                $pdf->DrawGroupTotal($groupCount);
            }

        // -------------------------------------------------------
        // 5C. SESSION – grouped by session, then by target (ianseo PrnSession style)
        // -------------------------------------------------------
        } else {
            $currentSession = null;
            $sessionCount   = 0;
            $fill           = false;

            foreach ($participants as $row) {
                $ses = $row['session_code'];

                // New session group
                if ($ses !== $currentSession) {
                    if ($currentSession !== null && $sessionCount > 0) {
                        $pdf->DrawGroupTotal($sessionCount);
                    }

                    // Always start each session on a fresh page (ianseo behaviour)
                    if ($currentSession !== null) {
                        $pdf->AddPage();
                    }

                    $sesLabel = ($ses === '-') ? 'Sesi (Tidak Ditentukan)' : 'Sesi ' . $ses;
                    $pdf->DrawSectionHeader($sesLabel, false);
                    $pdf->DrawSessionTableHeader();

                    $currentSession = $ses;
                    $sessionCount   = 0;
                    $fill           = false;
                } elseif (!$pdf->CanFit(1)) {
                    $pdf->RepeatHeaderOnNewPage('session');
                    $fill = false;
                }

                $pdf->DrawSessionRow(
                    $row['target_name'],
                    $row['athlete_name'],
                    $row['club_name'],
                    $row['age_group'],
                    $row['division_name'],
                    $fill
                );
                $fill = !$fill;
                $sessionCount++;
            }

            if ($sessionCount > 0) {
                $pdf->DrawGroupTotal($sessionCount);
            }
        }

        // 6. Grand total at the end of the document
        $total = count($participants);
        $pdf->Ln(3);
        $pdf->SetFont('helvetica', 'B', 8);
        $pdf->SetFillColor(40, 40, 40);
        $pdf->SetTextColor(255, 255, 255);
        $pdf->Cell(AccreditationPDF::BODY_W, 6, 'Total Keseluruhan Peserta: ' . $total . ' orang', 0, 1, 'R', true);
        $pdf->SetTextColor(0, 0, 0);

        if ($autoPrint) $pdf->IncludeJS("print(true);");

        logExecutionTime("Accreditation/$type");
        $filename = 'akreditasi_' . $type . '_' . preg_replace('/[^a-z0-9_-]/i', '_', $event['slug'] ?? $eventIdentifier) . '.pdf';
        $pdf->Output($filename, 'I');
        exit;

    } catch (Exception $e) {
        logExecutionTime("Accreditation Error: " . rtrim(str_replace("\n", " ", $e->getMessage())));
        header('Content-Type: application/json');
        die(json_encode(['error' => 'Error: ' . $e->getMessage()]));
    }
}

header('Content-Type: application/json');
echo json_encode(['error' => 'Endpoint not found', 'uri' => $requestUri]);
?>

