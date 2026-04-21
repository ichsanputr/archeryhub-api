# 📱 ArcheryHub Mobile API Specification (v1.1)

This document provides the **Complete Request and Response bodies** for all Mobile API endpoints.

---

## 🔐 1. Authentication (Archer Login)
**Endpoint**: `POST /auth/archer/login`

### Full Request Body
```json
{
  "email": "archer@example.com",
  "password": "securepassword123",
  "device_id": "f83a-4921-b3b3-8888",
  "platform": "android"
}
```

### Full Response Body (200 OK)
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTM2OTY0MjksImlhdCI6MTcxMzY5MjgyOSwiaXNzIjoiYXJjaGVyeWh1YiIsIm5iZiI6MTcxMzY5MjgyOSwic3ViIjoiYXJjLWE0OWVlN2Q3LTlkN2ItNGJlNy04NjUyLTM0MmYyZmNhMjNmOSJ9.signature",
  "is_new_user": false,
  "user": {
    "uuid": "arc-a49ee7d7-9d7b-4be7-8652-342f2fca23f9",
    "id": "ARC-0042",
    "username": "rizky-pratama",
    "full_name": "Rizky Pratama",
    "email": "archer@example.com",
    "avatar_url": "https://cdn.archeryhub.id/media/archers/rizky.jpg",
    "role": "archer",
    "user_type": "archer"
  }
}
```

---

## 🎨 2. Event Gallery
**Endpoint**: `GET /mobile/events/:slug/gallery`

### Full Response Body (200 OK)
```json
{
  "gallery": [
    {
      "uuid": "img-001",
      "event_id": "evt-8f3c2a14",
      "url": "https://cdn.archeryhub.id/media/events/g1.jpg",
      "caption": "Opening Ceremony Stage",
      "alt_text": "Stage photo",
      "display_order": 1,
      "is_primary": true,
      "created_at": "2026-04-21T10:00:00Z"
    },
    {
      "uuid": "img-002",
      "event_id": "evt-8f3c2a14",
      "url": "https://cdn.archeryhub.id/media/events/g2.jpg",
      "caption": "Archer Practice Session",
      "alt_text": "Practice field",
      "display_order": 2,
      "is_primary": false,
      "created_at": "2026-04-21T10:05:00Z"
    }
  ]
}
```

---

## 📜 3. Event History
**Endpoint**: `GET /mobile/events/history`

### Full Response Body (200 OK)
```json
{
  "events": [
    {
      "uuid": "evt-8f3c2a14-2b73-4a7f-8f7f-2ef1e6c1159a",
      "slug": "jakarta-open-2025",
      "name": "ArcheryHub Jakarta Open 2025",
      "location": "Jakarta Selatan",
      "start_date": "2025-05-18T08:00:00Z",
      "end_date": "2025-05-21T17:00:00Z",
      "logo_url": "https://cdn.archeryhub.id/media/events/logo.png",
      "banner_url": "https://cdn.archeryhub.id/media/events/banner.jpg",
      "organizer_name": "ArcheryHub Jakarta",
      "organizer_avatar_url": "https://cdn.archeryhub.id/media/orgs/logo.png",
      "participant_count": 128
    }
  ],
  "total_count": 1
}
```

---

## 📅 4. Event Schedule
**Endpoint**: `GET /mobile/events/:slug/schedule`

### Full Response Body (200 OK)
```json
{
  "schedules": [
    {
      "uuid": "sch-001",
      "event_id": "evt-8f3c2a14",
      "title": "Technical Meeting",
      "description": "Final briefing for all participants",
      "start_time": "2026-05-17T19:00:00Z",
      "end_time": "2026-05-17T21:00:00Z",
      "day_order": 0,
      "sort_order": 1,
      "location": "Zoom Meeting / VIP Room",
      "created_at": "2026-04-21T09:00:00Z",
      "updated_at": "2026-04-21T09:00:00Z"
    }
  ]
}
```

---

## ✍️ 5. Event Registration
**Endpoint**: `POST /mobile/archer/events/register`

### Full Request Body (Complex Case)
```json
{
  "event_id": "national-open-2026",
  "athlete_id": "arc-a49ee7d7-9d7b-4be7-8652-342f2fca23f9",
  "event_category_ids": [
    "cat-recurve-adult-putra",
    "cat-recurve-adult-team"
  ],
  "payment_amount": 450000,
  "payment_type": "manual",
  "payment_proof_urls": [
    "https://cdn.archeryhub.id/media/proofs/receipt-001.jpg"
  ],
  "registration_source": "mobile_app"
}
```

### Full Response Body (201 Created)
```json
{
  "message": "Pendaftaran berhasil",
  "registration_id": "reg-6f0bf699-d807-4ad4-a50d-5d60f7f7ad5d",
  "registered_categories": [
    "cat-recurve-adult-putra",
    "cat-recurve-adult-team"
  ],
  "payment_status": "menunggu acc"
}
```

---

## 📊 6. Event Results
**Endpoint**: `GET /mobile/events/:slug/results`

### Real Case Example: POPDA Sleman
**Sample URL**: `http://api.archeryhub.id/api/v1/mobile/events/seleksi-popda-kabsleman-2026-7247378c/results`

#### Full Response Body (200 OK)
```json
{
  "results": [
    {
      "participant_id": "dace4cbe-545e-449b-8bfb-19170e799f00",
      "full_name": "Arya Dimas Wijarnako",
      "category_name": "U-13",
      "rank": 1,
      "score": 163,
      "x_count": 2
    },
    {
      "participant_id": "169f7470-fa6d-4761-9e9a-e3ecdd4730d3",
      "full_name": "Angger Raka Sanjaya 1",
      "category_name": "U-13",
      "rank": 2,
      "score": 155,
      "x_count": 1
    }
  ],
  "page_settings": {
    "pdf_result_url": "https://cdn.archeryhub.id/media/results/popda-sleman-2026.pdf",
    "show_live_score": true,
    "theme_color": "#2D5A27"
  }
}
```

---

## ℹ️ 7. My Registered Events (Archer)
**Endpoint**: `GET /mobile/archer/events`

### Full Response Body (200 OK)
```json
{
  "events": [
    {
      "event_uuid": "evt-8f3c2a14",
      "event_name": "National Open 2026",
      "event_slug": "national-open-2026",
      "location": "Senayan, Jakarta",
      "start_date": "2026-05-18",
      "end_date": "2026-05-21",
      "logo_url": "https://cdn.archeryhub.id/media/events/logo.png",
      "qr_raw": "EVT2026-ARC-0042",
      "qr_code_data_url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
      "category_name": "Recurve Senior Putra",
      "payment_status": "lunas",
      "registration_date": "2026-04-01T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

## 🖼️ 8. WebView Embed: Pure Bracket (Bagan)
**URL**: `http://api.archeryhub.id/embed/results/:slug`

This is a **Pure Bracket (Bagan)** view designed specifically for mobile apps. It excludes all headers, footers, and other UI noise.

### Features:
*   **Pure View**: No main header/logo/menu.
*   **Category Filter**: Includes a compact horizontal filter to switch categories (Barebow, Recurve, etc.).
*   **No Chatbot**: Clean view without any third-party widgets.
*   **Deep Linking**: Pass `?category_id=UUID` to open a specific category bracket directly.

### Integration (Android/Swift):
```html
<WebView
  source={{ uri: 'http://api.archeryhub.id/embed/results/seleksi-popda-kabsleman-2026-7247378c?category_id=Optional-ID' }}
  style={{ flex: 1 }}
/>
```

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
      "archer_name": "Rizky Pratama",
      "club_name": "Jakarta Archery",
      "total_score": 675,
      "total_10x": 18,
      "total_x": 8,
      "ends_completed": 12,
      "sessions": [
        {
          "session_code": "S1",
          "session_name": "Qualification Phase 1",
          "end_scores": "58,55,59,60,57,58",
          "total_score": 347
        },
        {
          "session_code": "S2",
          "session_name": "Qualification Phase 2",
          "end_scores": "55,54,58,55,51,55",
          "total_score": 328
        }
      ]
    }
  ]
}
```
