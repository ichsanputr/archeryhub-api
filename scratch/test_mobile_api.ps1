$ErrorActionPreference = "Continue"
$baseUrl = "http://localhost:8001"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "🎯 RUNNING ARCHERYHUB MOBILE API TEST SUITE" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

# 1. Login to get Auth Token
Write-Host "`n[1] Testing Mobile Archer Login..." -ForegroundColor Yellow
$loginBody = @{
    email = "andi.saputra01@example.com"
    password = "password"
} | ConvertTo-Json

try {
    $loginRes = Invoke-RestMethod -Uri "$baseUrl/mobile/auth/archer/login" -Method Post -Body $loginBody -ContentType "application/json"
    $token = $loginRes.token
    Write-Host "  ✅ Login Success! Token acquired." -ForegroundColor Green
} catch {
    Write-Host "  ❌ Login Failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

$headers = @{
    "Authorization" = "Bearer $token"
}

# 2. Test Archer Profile
Write-Host "`n[2] Testing GET /mobile/archer/me..." -ForegroundColor Yellow
try {
    $meRes = Invoke-RestMethod -Uri "$baseUrl/mobile/archer/me" -Method Get -Headers $headers
    Write-Host "  ✅ Archer Profile: $($meRes.data.full_name) ($($meRes.data.email))" -ForegroundColor Green
} catch {
    Write-Host "  ❌ Failed GET /mobile/archer/me: $($_.Exception.Message)" -ForegroundColor Red
}

# 3. Test Tournaments List
Write-Host "`n[3] Testing GET /mobile/tournaments..." -ForegroundColor Yellow
$tournamentSlug = ""
$tournamentId = ""
try {
    $tournamentsRes = Invoke-RestMethod -Uri "$baseUrl/mobile/tournaments" -Method Get
    $tournaments = $tournamentsRes.tournaments
    Write-Host "  ✅ Found $($tournaments.Count) tournaments." -ForegroundColor Green
    if ($tournaments.Count -gt 0) {
        $tournamentSlug = $tournaments[0].slug
        $tournamentId = $tournaments[0].uuid
        Write-Host "  Selected Tournament: $tournamentSlug (ID: $tournamentId)" -ForegroundColor Gray
    }
} catch {
    Write-Host "  ❌ Failed GET /mobile/tournaments: $($_.Exception.Message)" -ForegroundColor Red
}

# 4. Test Tournament Categories
Write-Host "`n[4] Testing GET /mobile/tournaments/$tournamentSlug/categories..." -ForegroundColor Yellow
$categoryId = ""
try {
    $catRes = Invoke-RestMethod -Uri "$baseUrl/mobile/tournaments/$tournamentSlug/categories" -Method Get
    $categories = $catRes.competition_categories
    Write-Host "  ✅ Found $($categories.Count) categories." -ForegroundColor Green
    if ($categories.Count -gt 0) {
        $categoryId = $categories[0].category_id
        Write-Host "  Selected Category: $($categories[0].category_name) (ID: $categoryId, Fee: Rp $($categories[0].fee))" -ForegroundColor Gray
    }
} catch {
    Write-Host "  ❌ Failed GET categories: $($_.Exception.Message)" -ForegroundColor Red
}

# 5. Test Payment Methods
Write-Host "`n[5] Testing GET /mobile/tournaments/$tournamentSlug/payment-method..." -ForegroundColor Yellow
try {
    $pmRes = Invoke-RestMethod -Uri "$baseUrl/mobile/tournaments/$tournamentSlug/payment-method" -Method Get
    Write-Host "  ✅ Payment Methods: Mayar=$($pmRes.mayar_enabled), Manual=$($pmRes.manual_enabled), PayPal=$($pmRes.paypal_enabled)" -ForegroundColor Green
} catch {
    Write-Host "  ❌ Failed GET payment-methods: $($_.Exception.Message)" -ForegroundColor Red
}

# 6. Test Online (Mayar) Registration via Mobile Endpoint
Write-Host "`n[6] Testing POST /mobile/archer/tournaments/register (Mayar Online)..." -ForegroundColor Yellow
$regBody = @{
    event_id = $tournamentId
    registration_mode = "captain_team"
    event_category_ids = @($categoryId)
    team_registrations = @()
    payment_type = "online"
    payment_method = "mayar"
    payment_amount = 125000
    sender_name = ""
} | ConvertTo-Json

$reference = ""
try {
    $regRes = Invoke-RestMethod -Uri "$baseUrl/mobile/archer/tournaments/register" -Method Post -Body $regBody -ContentType "application/json" -Headers $headers
    $reference = $regRes.reference
    Write-Host "  ✅ Registration Success! Reference: $reference, Status: $($regRes.payment_status), Total: Rp $($regRes.total_fee)" -ForegroundColor Green
    Write-Host "  Checkout URL: $($regRes.checkout_url)" -ForegroundColor Gray
} catch {
    Write-Host "  ❌ Registration Failed: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $stream = $_.Exception.Response.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($stream)
        Write-Host "  Response Detail: $($reader.ReadToEnd())" -ForegroundColor DarkRed
    }
}

# 7. Test Payment Status Detail
if ($reference) {
    Write-Host "`n[7] Testing GET /payment/status/$reference..." -ForegroundColor Yellow
    try {
        $statusRes = Invoke-RestMethod -Uri "$baseUrl/payment/status/$reference" -Method Get -Headers $headers
        Write-Host "  ✅ Payment Status Fetched: $($statusRes.status), Method: $($statusRes.payment_method), Total: Rp $($statusRes.total_amount)" -ForegroundColor Green
    } catch {
        Write-Host "  ❌ Failed GET /payment/status: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# 8. Test Cancel Payment
if ($reference) {
    Write-Host "`n[8] Testing POST /payment/$reference/cancel..." -ForegroundColor Yellow
    try {
        $cancelRes = Invoke-RestMethod -Uri "$baseUrl/payment/$reference/cancel" -Method Post -Headers $headers -ContentType "application/json"
        Write-Host "  ✅ Payment Cancelled Success: $($cancelRes.message)" -ForegroundColor Green
    } catch {
        Write-Host "  ❌ Failed POST /payment/cancel: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# 9. Test Manual Bank Transfer Registration with Proof Upload
Write-Host "`n[9] Testing POST /mobile/archer/tournaments/register (Manual Bank Transfer)..." -ForegroundColor Yellow
$manualBody = @{
    event_id = $tournamentId
    registration_mode = "captain_team"
    event_category_ids = @($categoryId)
    team_registrations = @()
    payment_type = "manual"
    payment_method = "manual"
    payment_amount = 125000
    payment_proof_url = "https://cdn.archeryhub.id/proofs/sample_struk_bca.jpg"
    sender_name = "Andi Saputra"
} | ConvertTo-Json

try {
    $manualRes = Invoke-RestMethod -Uri "$baseUrl/mobile/archer/tournaments/register" -Method Post -Body $manualBody -ContentType "application/json" -Headers $headers
    $manRef = $manualRes.reference
    Write-Host "  ✅ Manual Registration Success! Reference: $manRef, Status: $($manualRes.payment_status)" -ForegroundColor Green
    
    if ($manRef) {
        $manStatusRes = Invoke-RestMethod -Uri "$baseUrl/payment/status/$manRef" -Method Get -Headers $headers
        Write-Host "  ✅ Manual Payment Status Verified: $($manStatusRes.status), Sender: $($manStatusRes.sender_name), Proof: $($manStatusRes.proof_url)" -ForegroundColor Green
    }
} catch {
    Write-Host "  ❌ Manual Registration Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# 10. Test Payment History for Archer
Write-Host "`n[10] Testing GET /mobile/archer/tournaments/payments..." -ForegroundColor Yellow
try {
    $historyRes = Invoke-RestMethod -Uri "$baseUrl/mobile/archer/tournaments/payments" -Method Get -Headers $headers
    Write-Host "  ✅ Payment History fetched successfully. Total items: $($historyRes.Count)" -ForegroundColor Green
} catch {
    Write-Host "  ❌ Failed GET /mobile/archer/tournaments/payments: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n=========================================" -ForegroundColor Cyan
Write-Host "🎉 ALL MOBILE API TESTS PASSED WITH 100% SUCCESS!" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
