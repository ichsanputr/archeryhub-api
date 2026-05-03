# ArcheryHub.id Mobile API Specification (v1)

This document provides the technical specification for mobile-exclusive API endpoints.

---

## 🔐 1. Authentication
Mobile apps use specific login endpoints that return a flat `Token` and `User` object.

### Archer Login
`POST /mobile/auth/archer/login`

### Google Login
`POST /mobile/auth/google/login`

### Bind Google Account
`POST /mobile/auth/google/bind` (Requires Auth)

Links the current logged-in account with a Google ID.

### Scorekeeper Login
`POST /mobile/auth/scorekeeper/login`

---

## 🎯 2. Target Scanning
Scan a target face QR code or Barcode to start scoring.

`GET /mobile/scan?code=TARGET_UUID`

---

## 📝 3. Scoring Endpoints

### Get Scoring Cards
`GET /mobile/qualification/scoring/cards`

### Update Qualification Score
`POST /mobile/qualification/scoring/scores/:assignmentId`

---

## 🏟️ 4. Events
Public event listing and details.

`GET /mobile/events`
`GET /mobile/events/:slug`

---

## 🛍️ 5. Marketplace
Public marketplace listing for archers to buy equipment.

`GET /mobile/marketplace/products`
`GET /mobile/marketplace/products/:id`

---

## 💬 6. Chat
Direct messaging between Archers and Sellers.

`GET /mobile/chat/conversations`
`POST /mobile/chat/conversations/:id/messages`

---

## 💳 7. Payment Detail
Enriched payment status for mobile checkout screens.

`GET /mobile/archer/payments/:reference`

---

## 🗞️ 8. News & Comments
Public news feed with community comments.

`GET /mobile/news`
`POST /mobile/news/:id/comments`

---

## 📊 9. Detailed Qualification Ranking
**Endpoint**: `GET /events/:id/results/qualification`

This endpoint provides the complete leaderboard for a specific category, including scores for every "End" and every "Session".

### Request Parameters
| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `category_id` | `uuid` | **Yes** | The category UUID to fetch rankings for |

### Response Body (200 OK)
```json
{
  "total_ends": 12,
  "results": [
    {
      "rank": 1,
      "participant_id": "reg-uuid-...",
      "archer_uuid": "arc-uuid-...",
      "archer_name": "Rizky Pratama",
      "avatar_url": "https://cdn.archeryhub.id/media/archers/rizky.jpg",
      "club_name": "Jakarta Archery",
      "total_score": 675,
      "total_10x": 18,
      "total_x": 8,
      "ends_completed": 12,
      "sessions": [
        {
          "assignment_id": "asgn-uuid-...",
          "session_code": "S1",
          "session_name": "Qualification Phase 1",
          "end_scores": "58,55,59,60,57,58",
          "total_ends": 6,
          "total_score": 347,
          "total_10x": 10,
          "total_x": 4
        }
      ]
    }
  ]
}
```

---

## 🔐 10. Auth Logout
**Endpoint**: `POST /mobile/auth/logout`

Clears the session on the server (for cookie-based sessions) and returns success.

---

## 💳 11. List Payment Channels
**Endpoint**: `GET /mobile/payment/channels`

Returns a list of available Tripay payment channels.

---

## 🛍️ 12. Seller Product Management
**Base Path**: `/mobile/seller/products` (Requires Auth)

CRUD for seller products. Supports `POST`, `PUT`, `DELETE`.

---

## ⚙️ 13. Mobile Options (Public)
**Base Path**: `/mobile/options`

Endpoints for dropdown selections: 
- `/clubs`
- `/organizations`
- `/disciplines`
- `/bow-types`
- `/age-groups`
- `/gender-divisions`
- `/cities`
- `/event-types`

---

## 🏛️ 14. Mobile Organization Dashboard
**Endpoint**: `GET /mobile/organization/dashboard` (Requires Auth)

Returns statistics, upcoming deadlines, and recent activities for the organization.

---

## 🏬 15. Mobile Seller Dashboard
**Endpoint**: `GET /mobile/seller/dashboard` (Requires Auth)

Returns statistics and recent orders for the seller.

---

## 📁 16. Mobile Media Upload
**Base Path**: `/mobile/media` (Requires Auth)

### Upload File
**Endpoint**: `POST /mobile/media/upload`
**Body**: `multipart/form-data` with `file` field.

### List My Media
**Endpoint**: `GET /mobile/media`

### Delete Media
**Endpoint**: `DELETE /mobile/media/:id`
