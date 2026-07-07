# API Gap Analysis — Flutter Mobile vs Go Backend

## Ringkasan

Analisis endpoint yang dipanggil Flutter mobile app vs yang tersedia di Go backend (`main.go`).
Ditemukan beberapa **path mismatch**, **endpoint missing**, dan **handler stubs (belum diimplementasikan)**.

---

## 🔴 Critical Issues

### 1. Path Mismatch: `organization` vs `organizer`

Flutter mobile app panggil `/mobile/organization/...` tapi backend daftarkan route sebagai `/mobile/organizer/...`.

| Flutter (api_constants.dart) | Backend (main.go) | Status |
|------------------------------|-------------------|--------|
| `POST /mobile/auth/organization/login` | `POST /mobile/auth/organizer/login` | ❌ **MISMATCH** |
| `GET /mobile/organization/me` | `GET /mobile/organizer/me` | ❌ **MISMATCH** |
| `PUT /mobile/organization/me` | `PUT /mobile/organizer/me` | ❌ **MISMATCH** |
| `GET /mobile/organization/events` | `GET /mobile/organizer/events` | ❌ **MISMATCH** |
| `GET /mobile/organization/events/:id/participants` | `GET /mobile/organizer/events/:id/participants` | ❌ **MISMATCH** |
| `DELETE /mobile/organization/events/:id/participants/:user_id` | `DELETE /mobile/organizer/events/:id/participants/:user_id` | ❌ **MISMATCH** |
| `POST /mobile/organization/scan-registration` | `POST /mobile/organizer/scan-registration` | ❌ **MISMATCH** |
| `GET /mobile/organization/dashboard` | `GET /mobile/organizer/dashboard` | ❌ **MISMATCH** |
| `GET /mobile/organization/finance/earnings` | `GET /mobile/organizer/finance/earnings` | ❌ **MISMATCH** |
| `GET /mobile/organization/finance/balance` | `GET /mobile/organizer/finance/balance` | ❌ **MISMATCH** |
| `GET /mobile/organization/finance/bank-accounts` | `GET /mobile/organizer/finance/bank-accounts` | ❌ **MISMATCH** |
| `POST /mobile/organization/finance/bank-accounts` | `POST /mobile/organizer/finance/bank-accounts` | ❌ **MISMATCH** |
| `PUT /mobile/organization/finance/bank-accounts/:id` | `PUT /mobile/organizer/finance/bank-accounts/:id` | ❌ **MISMATCH** |
| `DELETE /mobile/organization/finance/bank-accounts/:id` | `DELETE /mobile/organizer/finance/bank-accounts/:id` | ❌ **MISMATCH** |

**Solusi**: Ubah route di `main.go` dari `/organizer` jadi `/organization`, ATAU ubah `api_constants.dart` di Flutter menjadi `/organizer`.

---

### 2. Missing Mobile Endpoint: Create Payment for Participant

Flutter panggil `POST /mobile/archer/participants/:participantId/payment`, tapi endpoint ini **tidak ada** di backend.

| Flutter | Backend | Status |
|---------|---------|--------|
| `POST /mobile/archer/participants/:participantId/payment` | ❌ Tidak ada | 🔴 **MISSING** |
| (sebagai gantinya) `POST /events/participants/:participantId/payment` | ✅ Ada (non-mobile) | Bisa dipakai, tapi beda path |

**Solusi**: Tambahkan route `POST /mobile/archer/participants/:participantId/payment` di grup mobile, atau alihkan Flutter ke `/events/participants/:participantId/payment`.

---

### 3. Auth Register Path Mismatch

Flutter panggil `https://api.archeris.net/v1/auth/register` (full URL dengan `/v1/`), tapi backend hanya punya `POST /auth/register` (tanpa `/v1`).

| Flutter | Backend | Status |
|---------|---------|--------|
| `POST /v1/auth/register` | `POST /auth/register` | ❌ **MISMATCH** |

**Solusi**: Ubah `api_constants.dart` dari full URL jadi `'/auth/register'`, atau tambahkan `/v1` prefix di backend.

---

### 4. Cart Endpoints — Semua Stub (Belum Diimplementasi)

| Endpoint | Status | Keterangan |
|----------|--------|------------|
| `GET /mobile/archer/cart` | ⚠️ **Stub** | Return `{"data": []}` |
| `POST /mobile/archer/cart` | ⚠️ **Stub** | Return `403 Forbidden` |
| `PUT /mobile/archer/cart/:id` | ⚠️ **Stub** | Return `403 Forbidden` |
| `DELETE /mobile/archer/cart/:id` | ⚠️ **Stub** | Return `403 Forbidden` |
| `DELETE /mobile/archer/cart` | ⚠️ **Stub** | Return `{"message": "..."}` |
| `POST /mobile/archer/cart/checkout` | ⚠️ **Stub** | Return `403 Forbidden` |

**Solusi**: Implementasi penuh cart dengan database operations.

### 5. Order History — Return Data Kosong

| Endpoint | Status | Keterangan |
|----------|--------|------------|
| `GET /mobile/archer/orders` | ⚠️ **Stub** | Return `{"data": []}` |

**Solusi**: Implementasi query orders dari database.

---

## 🟡 Minor Issues

### 6. Mobile Register Endpoint — Field Name Mismatch

Backend struct `MobileRegisterEventRequest` pakai field `athlete_id`, `event_id`, `event_category_ids`.  
Flutter pakai field sama ✅ **cocok**.

Backend response `MobileRegisterEventResponse`:
```go
json:"message"
json:"registration_id"
json:"registered_categories"
json:"payment_status"
```
Flutter model:
```dart
json['message']
json['registration_id']
json['registered_categories']
json['payment_status']
```
✅ **Sudah cocok**.

### 7. GetMyRegistration Path Mismatch

Flutter panggil `GET /mobile/archer/events/:eventId/registration`.  
Backend punya `GET /mobile/archer/events/:id/registration`. ✅ **cocok**.

### 8. Event Payment Methods

Flutter panggil `GET /mobile/events/:slug/payment-method`.  
Backend punya `GET /mobile/events/:slug/payment-method` via `MobileGetEventPaymentMethods`. ✅ **Ada**.

---

## ✅ Endpoints yang Sudah Lengkap

| Feature | Status |
|---------|--------|
| Auth (login archer/seller/org/scorekeeper, register, forgot/reset, OTP, logout, Google) | ✅ Lengkap |
| Events (list, detail, participants, schedule, categories, gallery, FAQ, fees, rewards, location, results) | ✅ Lengkap |
| News (list, detail, comments, related) | ✅ Lengkap |
| Marketplace (products list, detail) | ✅ Lengkap |
| Chat (conversations, messages, unread, last-active) | ✅ Lengkap |
| Chatbot (intents, message) | ✅ Lengkap |
| Scoring qualification & elimination | ✅ Lengkap |
| Scan (target QR) | ✅ Lengkap |
| Session boards & assignment detail | ✅ Lengkap |
| Payment channels & instructions | ✅ Lengkap |
| Options (clubs, organizers, disciplines, bow-types, age-groups, gender-divisions, cities, event-types, banks) | ✅ Lengkap |
| Media upload | ✅ Lengkap |
| Seller (me, products CRUD, dashboard, page) | ✅ Lengkap |
| Archer profile (me, update) | ✅ Lengkap |
| Archer events (my events, detail, registration, QR, payments) | ✅ Lengkap |
| Scorekeeper (me, events) | ✅ Lengkap |
| Organization dashboard & finance (me, events, participants, dashboard, earnings, wallet, bank-accounts) | ✅ Ada (tapi path mismatch) |

---

## 📋 Priority Action Plan

| Priority | Action | Dampak |
|----------|--------|--------|
| 🔴 P1 | Fix `organization` → `organizer` path mismatch di salah satu sisi (backend atau Flutter) | Organization flow完全不工作 |
| 🔴 P1 | Tambah `POST /mobile/archer/participants/:participantId/payment` | Payment flow setelah registrasi event tidak bisa |
| 🔴 P1 | Fix `register` path (v1 prefix) | Register archer dari mobile gagal |
| 🟡 P2 | Implementasi cart endpoints (CRUD + checkout) | Fitur keranjang tidak bisa dipakai |
| 🟡 P2 | Implementasi order history | Fitur riwayat order tidak bisa dipakai |
| 🔵 P3 | Validasi field response untuk semua endpoint mobile | Memastikan format response cocok dengan Flutter model |
