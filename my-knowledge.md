# Archeris.net - Knowledge Base

## Overview

**Archeris.net** is a complete archery scoring system and competition management platform built for organizers and archers worldwide. It provides a simple and modern way to run tournaments, track standings, and handle archery scoring from any device (browser or mobile).

**Live URL:** https://archeris.net/  
**API Base URL:** https://api.archeris.net  
**Tech Stack (Backend):** Go (Gin framework), MySQL (archeris DB), Swagger/OpenAPI  
**Tech Stack (Frontend):** Nuxt.js 3 (SSR), Vue 3, Vuetify 3, TailwindCSS, TypeScript  
**Tech Stack (Mobile):** Flutter (APK + Google Play)  
**i18n:** English (default) + Bahasa Indonesia, @nuxtjs/i18n, no_prefix strategy  
**Analytics:** Google Analytics (G-ZCZ6KY7V2C)  
**SEO:** SSR enabled, Open Graph, Twitter Cards, sitemap, RSS feed  
**Rich Text Editor:** TipTap (for blog/articles/docs)  
**Charts:** Chart.js + vue-chartjs  
**Animations:** GSAP, @vueuse/motion  
**PDF/QR:** qrcode.vue, jsqr (QR scanning)  
**Scripts:** Puppeteer (thumbnail generation), PapaParse (CSV)

---

## What Is This Application About

Archeris is an all-in-one platform for:
- **Tournament management** - Create, publish, and manage archery competitions
- **Digital scoring** - Paperless field scoring via mobile devices
- **Participant registration** - Online signup with payment integration
- **Results & leaderboards** - Real-time qualification and elimination results
- **Archer profiles** - Personal profiles with competition history and certificates
- **Organizer dashboard** - Full tournament control panel for event organizers

---

## Core Features

### 1. Tournament Management
- Create and publish tournaments with custom categories, divisions, and quotas
- Indoor, Outdoor, Field, and 3D archery event types
- Schedule management with auto-generation capabilities
- Target lane assignments (1A–1D) with visual allocation
- Elimination bracket generation (individual, team, mixed)
- CSV participant import/export
- Digital certificates (QR-verified)

### 2. Registration & Payments
- Online self-registration with saved archer profiles
- Payment gateway integration (Mayar, PayPal)
- QRIS, Virtual Accounts, credit cards, and manual bank transfers
- Automatic payment tracking and invoicing
- Manual payment proof upload and verification
- Subscription plans for organizers (Starter/Standard/Professional)

### 3. Mobile Scoring System
- Scorekeeper login via event PIN
- Arrow-by-arrow scoring keypad (X, 10, 9 to M)
- Real-time leaderboard updates
- Offline score verification
- QR/barcode target scanning
- OCR scan support for paper scorecards

### 4. Results & Printouts
- Qualification results and leaderboards
- Elimination brackets and scoresheets
- Medal standings and final rankings
- Start lists and target labels
- Team qualification results
- Entries by club reports
- PDF printouts for all documents

### 5. Archer Experience
- Personal archer profiles with stats and history
- Tournament discovery and registration
- Match schedules and QR codes
- Certificate collection and verification
- Performance tracking across tournaments
- Social media integration

### 6. Organizer Dashboard
- Event creation and management with page builder
- Participant management (add, edit, detail, CSV import/export, batch operations)
- Scorekeeper management (create, regenerate codes, manage access)
- Financial dashboard (earnings, revenue tracking, wallet, withdrawals, bank accounts)
- Payment methods management per organizer
- Reports: participants, finance, performance, attendance
- Broadcast messaging to participants
- Media storage management per tournament
- Subscription/package upgrade flow (with payment)
- Tournament reset functionality
- Categories management (create, edit, delete per tournament)
- Schedule management with auto-generate
- Target management with visual map view
- Certificate template builder and bulk generation
- Team management (create, edit, roster)
- Printout center (participants, qualification, elimination, statistics)
- Revenue tracking per tournament

### 7. Archer Dashboard
- Personal profile editing
- Tournament history and performance tracking
- Registration management per tournament
- Live qualification tracking (my-qualification)
- Live elimination bracket tracking (my-elimination)
- Target assignment view (my-target)
- Team membership view (my-team)
- Payment history and status tracking
- Certificate collection and verification
- Notifications center
- Settings management

### 7. Admin/Root Dashboard
- User account management (archers, organizers, scorekeepers)
- Subscription management and plan configuration
- Club management (data master)
- Blog/article management
- Tournament oversight
- Business recap and analytics

### 8. Blog & Documentation
- Archery scoring guides and tutorials
- Gear & equipment articles
- Tournament management documentation
- Newsletter subscription

### 9. Team Management
- Team creation and roster management
- Team rankings and results
- Captain-based team registration
- Club/delegation athlete booking

### 10. Chatbot
- AI-powered chatbot for tournament and archery queries
- Intent-based conversational system

---

## Primary Keywords

- archery scoring system
- archery tournament management
- online archery scoring
- archery competition platform
- digital archery scorecard
- archery leaderboard
- archery elimination brackets
- archery target assignments
- mobile archery scoring
- archery tournament registration
- paperless archery scoring
- archery results system
- archery certificate generator
- World Archery scoring
- recurve scoring system
- compound archery scoring
- barebow archery scoring
- archery event management software
- archery scoring app
- ianseo alternative

---

## Topical Authority

### Core Topics
1. **Archery Scoring Systems** - Paperless field scoring, mobile scoring, scorekeeper tools
2. **Tournament Management** - Event creation, registration, scheduling, quotas
3. **Archery Competition Formats** - Qualification rounds, elimination brackets, team matches
4. **Archery Equipment** - Recurve, compound, barebow, traditional bows
5. **Archery Rules & Standards** - World Archery formats, scoring rules, tie-breakers
6. **Payment & Registration** - Online payments, QRIS, invoicing, subscriptions

### Supporting Topics
1. **Archery Training & Coaching** - Draw length, draw weight, bow selection
2. **Archery Community** - Clubs, organizers, coaches, scorekeepers
3. **Digital Transformation in Sports** - Moving from paper to digital, cloud-based systems
4. **Archery Event Types** - Indoor, outdoor, field, 3D archery

---

## All Pages / Endpoints

### Public Pages (Frontend - Vue/Nuxt.js)
| Page | URL | File |
|------|-----|------|
| Homepage | / | pages/index.vue |
| Tournaments List | /tournaments | pages/tournaments/index.vue |
| Tournament Detail (tabbed) | /tournaments/:slug | pages/tournaments/[slug]/[[tab]].vue |
| Tournament Detail Tabs | overview, schedule, athletes, results, venue, gallery | (same file, computed tabs) |
| Tournament Registration | /tournaments/:slug/register | pages/tournaments/[slug]/register.vue |
| Tournament Payment | /tournaments/:slug/payment | pages/tournaments/[slug]/payment.vue |
| Tournament Targets | /tournaments/:slug/targets | pages/tournaments/[slug]/targets.vue |
| Archers Directory | /archers | pages/archers/index.vue |
| Archer Profile | /archers/:slug | pages/archers/[slug].vue |
| Organizer Directory | /organizer | pages/organizer/index.vue |
| Organizer Profile | /organizer/:slug | pages/organizer/[slug].vue |
| Blog Index | /blog | pages/blog/index.vue |
| Blog Article | /blog/:slug | pages/blog/[slug].vue |
| Blog Category | /blog/category/:category | pages/blog/category/[category].vue |
| Documentation | /docs | pages/docs/index.vue |
| Doc Detail | /docs/*slug | pages/docs/[...slug].vue |
| Pricing / Package | /package | pages/package.vue |
| Login | /auth/login | pages/auth/login/index.vue |
| Root Login | /auth/login/root | pages/auth/login/root.vue |
| Register | /auth/register | pages/auth/register.vue |
| Forgot Password | /auth/forgot-password | pages/auth/forgot-password.vue |
| OAuth Callback | /auth/callback | pages/auth/callback.vue |
| OAuth Direct | /auth/oauth | pages/auth/oauth.vue |
| Already Registered | /auth/already-registered | pages/auth/already-registered.vue |
| Certificate Verify | /certificates/:uuid | pages/certificates/[uuid].vue |
| Match Detail | /match/:id | pages/match/[id].vue |
| About Us | /about-us | pages/about-us.vue |
| Contact | /contact | pages/contact.vue |
| Privacy Policy | /privacy | pages/privacy.vue |
| Terms & Conditions | /terms | pages/terms.vue |
| Disclaimer | /disclaimer | pages/disclaimer.vue |
| FAQ | /faq | pages/faq/index.vue |
| Archeris vs Ianseo | /archeris-vs-ianseo | pages/archeris-vs-ianseo.vue |
| Referral | /ref | pages/ref.vue |
| Payment Status | /payment/status/:reference | pages/payment/status/[reference].vue |
| Payment Failed | /payment/failed/:reference | pages/payment/failed/[reference].vue |
| Scorekeeper Login | /scorekeeper/login | pages/scorekeeper/login.vue |
| Tools: Thumbnail Generator | /tools/thumbnail-generator | pages/tools/thumbnail-generator.vue |

### Protected Pages - Archer Dashboard
| Page | URL | File |
|------|-----|------|
| Archer Dashboard Home | /dashboard/archer | pages/dashboard/archer/index.vue |
| Archer Profile | /dashboard/archer/profile | pages/dashboard/archer/profile.vue |
| Archer Settings | /dashboard/archer/settings | pages/dashboard/archer/settings.vue |
| Archer Certificates | /dashboard/archer/certificates | pages/dashboard/archer/certificates/index.vue |
| Archer Notifications | /dashboard/archer/notifications | pages/dashboard/archer/notifications/index.vue |
| Archer Payments List | /dashboard/archer/payments | pages/dashboard/archer/payments/index.vue |
| Archer Payment Detail | /dashboard/archer/payments/:reference | pages/dashboard/archer/payments/[reference].vue |
| Archer Tournaments List | /dashboard/archer/tournaments | pages/dashboard/archer/tournaments/index.vue |
| Archer Tournament Detail | /dashboard/archer/tournaments/:id | pages/dashboard/archer/tournaments/[id]/index.vue |
| Archer Tournament Overview | /dashboard/archer/tournaments/:id/overview | pages/dashboard/archer/tournaments/[id]/overview.vue |
| Archer Tournament Registration | /dashboard/archer/tournaments/:id/my-registration | pages/dashboard/archer/tournaments/[id]/my-registration.vue |
| Archer Tournament Qualification | /dashboard/archer/tournaments/:id/my-qualification | pages/dashboard/archer/tournaments/[id]/my-qualification.vue |
| Archer Tournament Elimination | /dashboard/archer/tournaments/:id/my-elimination | pages/dashboard/archer/tournaments/[id]/my-elimination.vue |
| Archer Tournament Target | /dashboard/archer/tournaments/:id/my-target | pages/dashboard/archer/tournaments/[id]/my-target.vue |
| Archer Tournament Team | /dashboard/archer/tournaments/:id/my-team | pages/dashboard/archer/tournaments/[id]/my-team.vue |

### Protected Pages - Organizer Dashboard
| Page | URL | File |
|------|-----|------|
| Organizer Dashboard Home | /dashboard/organizer | pages/dashboard/organizer/index.vue |
| Organizer Profile | /dashboard/organizer/profile | pages/dashboard/organizer/profile.vue |
| Organizer Settings | /dashboard/organizer/settings | pages/dashboard/organizer/settings.vue |
| Organizer Notifications | /dashboard/organizer/notifications | pages/dashboard/organizer/notifications/index.vue |
| Organizer Teams | /dashboard/organizer/teams | pages/dashboard/organizer/teams/index.vue |
| Organizer Subscription | /dashboard/organizer/subscription | pages/dashboard/organizer/subscription.vue |
| Organizer Scorekeepers | /dashboard/organizer/scorekeepers | pages/dashboard/organizer/scorekeepers.vue |
| Organizer Bank Accounts | /dashboard/organizer/bank-accounts | pages/dashboard/organizer/bank-accounts.vue |
| Organizer Payment Methods | /dashboard/organizer/payment-methods | pages/dashboard/organizer/payment-methods.vue |
| Organizer Balance | /dashboard/organizer/balance | pages/dashboard/organizer/balance.vue |
| Organizer Wallet | /dashboard/organizer/wallet | pages/dashboard/organizer/wallet/index.vue |
| Organizer Earnings | /dashboard/organizer/earnings | pages/dashboard/organizer/earnings/index.vue |
| Organizer Earnings Detail | /dashboard/organizer/earnings/:id | pages/dashboard/organizer/earnings/[id].vue |
| Organizer Reports Home | /dashboard/organizer/reports | pages/dashboard/organizer/reports/index.vue |
| Participants Report | /dashboard/organizer/reports/participants | pages/dashboard/organizer/reports/participants.vue |
| Finance Report | /dashboard/organizer/reports/finance | pages/dashboard/organizer/reports/finance.vue |
| Performance Report | /dashboard/organizer/reports/performance | pages/dashboard/organizer/reports/performance.vue |
| Attendance Report | /dashboard/organizer/reports/attendance | pages/dashboard/organizer/reports/attendance.vue |
| Package / Upgrade | /dashboard/organizer/package | pages/dashboard/organizer/package/index.vue |
| Package Detail | /dashboard/organizer/package/detail | pages/dashboard/organizer/package/detail.vue |
| Package Payment | /dashboard/organizer/package/payment | pages/dashboard/organizer/package/payment.vue |
| Package Success | /dashboard/organizer/package/success | pages/dashboard/organizer/package/success.vue |
| Tournaments List | /dashboard/organizer/tournaments | pages/dashboard/organizer/tournaments/index.vue |
| Create Tournament | /dashboard/organizer/tournaments/create | pages/dashboard/organizer/tournaments/create.vue |
| Tournament Overview | /dashboard/organizer/tournaments/:id/overview | pages/dashboard/organizer/tournaments/[id]/overview.vue |
| Tournament Page Builder | /dashboard/organizer/tournaments/:id/page | pages/dashboard/organizer/tournaments/[id]/page.vue |
| Tournament Revenue | /dashboard/organizer/tournaments/:id/revenue | pages/dashboard/organizer/tournaments/[id]/revenue.vue |
| Tournament Categories | /dashboard/organizer/tournaments/:id/categories | pages/dashboard/organizer/tournaments/[id]/categories/index.vue |
| Tournament Category Detail | /dashboard/organizer/tournaments/:id/categories/:categoryId | pages/dashboard/organizer/tournaments/[id]/categories/[categoryId].vue |
| Tournament Participants | /dashboard/organizer/tournaments/:id/participants | pages/dashboard/organizer/tournaments/[id]/participants/index.vue |
| Add Participant | /dashboard/organizer/tournaments/:id/participants/add | pages/dashboard/organizer/tournaments/[id]/participants/add.vue |
| Participant Detail | /dashboard/organizer/tournaments/:id/participants/detail | pages/dashboard/organizer/tournaments/[id]/participants/detail.vue |
| Edit Participant | /dashboard/organizer/tournaments/:id/participants/edit | pages/dashboard/organizer/tournaments/[id]/participants/edit.vue |
| Participant Edit (by ID) | /dashboard/organizer/tournaments/:id/participants/:participantId | pages/dashboard/organizer/tournaments/[id]/participants/[participantId]/index.vue |
| Participant Edit (by ID) edit | /dashboard/organizer/tournaments/:id/participants/:participantId/edit | pages/dashboard/organizer/tournaments/[id]/participants/[participantId]/edit.vue |
| Tournament Payments | /dashboard/organizer/tournaments/:id/payments | pages/dashboard/organizer/tournaments/[id]/payments/index.vue |
| Payment Detail | /dashboard/organizer/tournaments/:id/payments/:reference | pages/dashboard/organizer/tournaments/[id]/payments/[reference].vue |
| Tournament Checkout | /dashboard/organizer/tournaments/:id/checkout | pages/dashboard/organizer/tournaments/[id]/checkout.vue |
| Tournament My Registration | /dashboard/organizer/tournaments/:id/my-registration | pages/dashboard/organizer/tournaments/[id]/my-registration.vue |
| Tournament Qualification | /dashboard/organizer/tournaments/:id/qualification | pages/dashboard/organizer/tournaments/[id]/qualification/index.vue |
| Qualification Results | /dashboard/organizer/tournaments/:id/qualification/results | pages/dashboard/organizer/tournaments/[id]/qualification/results.vue |
| Qualification Session | /dashboard/organizer/tournaments/:id/qualification/:session | pages/dashboard/organizer/tournaments/[id]/qualification/[session].vue |
| Tournament Elimination | /dashboard/organizer/tournaments/:id/elimination | pages/dashboard/organizer/tournaments/[id]/elimination/index.vue |
| Elimination Bracket | /dashboard/organizer/tournaments/:id/elimination/:bracketId | pages/dashboard/organizer/tournaments/[id]/elimination/[bracketId]/index.vue |
| Elimination Round | /dashboard/organizer/tournaments/:id/elimination/:bracketId/:round | pages/dashboard/organizer/tournaments/[id]/elimination/[bracketId]/[round].vue |
| Tournament Targets | /dashboard/organizer/tournaments/:id/targets | pages/dashboard/organizer/tournaments/[id]/targets/index.vue |
| Target Map | /dashboard/organizer/tournaments/:id/targets/map | pages/dashboard/organizer/tournaments/[id]/targets/map.vue |
| Tournament Schedule | /dashboard/organizer/tournaments/:id/schedule | pages/dashboard/organizer/tournaments/[id]/schedule/index.vue |
| Tournament Certificate | /dashboard/organizer/tournaments/:id/certificate | pages/dashboard/organizer/tournaments/[id]/certificate.vue |
| Tournament Media | /dashboard/organizer/tournaments/:id/media | pages/dashboard/organizer/tournaments/[id]/media.vue |
| Tournament Teams | /dashboard/organizer/tournaments/:id/teams | pages/dashboard/organizer/tournaments/[id]/teams/index.vue |
| Team Detail | /dashboard/organizer/tournaments/:id/teams/:teamId | pages/dashboard/organizer/tournaments/[id]/teams/[teamId].vue |
| Printout Home | /dashboard/organizer/tournaments/:id/printout | pages/dashboard/organizer/tournaments/[id]/printout/index.vue |
| Printout Statistics | /dashboard/organizer/tournaments/:id/printout/statistics | pages/dashboard/organizer/tournaments/[id]/printout/statistics.vue |
| Printout Participants | /dashboard/organizer/tournaments/:id/printout/participants | pages/dashboard/organizer/tournaments/[id]/printout/participants/index.vue |
| Printout Qualification | /dashboard/organizer/tournaments/:id/printout/qualification | pages/dashboard/organizer/tournaments/[id]/printout/qualification/index.vue |
| Printout Elimination | /dashboard/organizer/tournaments/:id/printout/elimination | pages/dashboard/organizer/tournaments/[id]/printout/elimination/index.vue |
| Tournament Reset | /dashboard/organizer/tournaments/:id/reset | pages/dashboard/organizer/tournaments/[id]/reset/index.vue |
| Organizer My Qualification | /dashboard/organizer/tournaments/:id/my-qualification | pages/dashboard/organizer/tournaments/[id]/my-qualification.vue |
| Organizer My Elimination | /dashboard/organizer/tournaments/:id/my-elimination | pages/dashboard/organizer/tournaments/[id]/my-elimination.vue |

### Protected Pages - Club Dashboard
| Page | URL | File |
|------|-----|------|
| Club Notifications | /dashboard/club/notifications | pages/dashboard/club/notifications/index.vue |

### Protected Pages - Root/Admin Dashboard
| Page | URL | File |
|------|-----|------|
| Root Dashboard Home | /dashboard/root | pages/dashboard/root/index.vue |
| Root Archers | /dashboard/root/archers | pages/dashboard/root/archers/index.vue |
| Root Tournaments | /dashboard/root/tournaments | pages/dashboard/root/tournaments/index.vue |
| Root Articles | /dashboard/root/articles | pages/dashboard/root/articles/index.vue |
| Root Create Article | /dashboard/root/articles/create | pages/dashboard/root/articles/create.vue |
| Root Edit Article | /dashboard/root/articles/:id | pages/dashboard/root/articles/[id].vue |

### Shared Dashboard Pages
| Page | URL | File |
|------|-----|------|
| Dashboard Gateway | /dashboard | pages/dashboard/index.vue |
| Profile by Username | /dashboard/profile/:username | pages/dashboard/profile/[username].vue |
| Profile by ID | /dashboard/profile/:id | pages/dashboard/profile/[id].vue |
| Edit Profile | /dashboard/profile/edit | pages/dashboard/profile/edit.vue |

### Dev / Test / Graphics Pages
| Page | URL | File |
|------|-----|------|
| PayPal Dev Tool | /dev/paypal | pages/dev/paypal.vue |
| Mockup Preview | /mockup-preview | pages/mockup-preview.vue |
| Test FOP Preview | /test/fop-preview | pages/test/fop-preview.vue |
| Feature: Mobile Showcase | /graphics/mobile-showcase | pages/graphics/mobile-showcase.vue |
| Feature: Archer | /graphics/feature-archer | pages/graphics/feature-archer.vue |
| Feature: Competition | /graphics/feature-competition | pages/graphics/feature-competition.vue |
| Feature: Registration | /graphics/feature-registration | pages/graphics/feature-registration.vue |
| Feature: Scorekeeper | /graphics/feature-scorekeeper | pages/graphics/feature-scorekeeper.vue |

### Mobile App Pages
| Page | Description |
|------|-------------|
| Mobile Login | Archer/Organizer/Scorekeeper login |
| Mobile Tournament Discovery | Browse and search tournaments |
| Mobile Tournament Detail | Full event info, schedule, participants |
| Mobile Registration | Register for events |
| Mobile Scoring | Arrow-by-arrow scoring interface |
| Mobile Target Scan | QR/barcode target scanning |
| Mobile Payments | Payment history and instructions |
| Mobile Profile | Archer/Organizer profile management |
| Mobile Certificates | View and download certificates |
| Mobile Notifications | Push and in-app notifications |

### Tournament Detail Tab Structure
The tournament detail page `/tournaments/:slug/[[tab]]` supports these tabs:
- **overview** - About, description, technical guidebook, competition categories, registration fees
- **schedule** - Full tournament schedule timeline
- **athletes** - Participant list with search/filter
- **results** - Qualification & elimination results, medal standings
- **venue** - Location, map, venue details
- **gallery** - Tournament images and media

Alias slugs supported: ringkasan→overview, jadwal→schedule, peserta→athletes, hasil→results, lokasi→venue

---

## Dashboard Pages - Full Breakdown

### SHARED DASHBOARD

#### /dashboard (Gateway)
**File:** `pages/dashboard/index.vue`
- Loading screen only; persona-redirect middleware routes to /dashboard/archer or /dashboard/organizer based on user role

---

### ARCHER DASHBOARD (14 pages)

#### /dashboard/archer (Home - Redirect)
**File:** `pages/dashboard/archer/index.vue`
- Loading spinner, auto-redirects to `/dashboard/archer/tournaments`

#### /dashboard/archer/profile
**File:** `pages/dashboard/archer/profile.vue` (1066 lines)
- **2 Tabs:** Personal Info & Specs | Bio & Achievements
- **Tab 1 - Personal Info:**
  - Avatar + Banner upload via MediaLibrary modal
  - Identity: Full name, username (with live availability check), NIK (16-digit validation), date of birth, gender, height, weight
  - Archery specs: Bow type (recurve/compound/barebow/traditional/horsebow/standard), hand dominance, club selector (search/create)
  - Contact & address: Email, phone, emergency contact, country (200+ countries), city, full address
  - Social media: 10 platforms (Instagram, TikTok, WhatsApp, Facebook, Twitter, YouTube, Spotify, Website, Pinterest, LinkedIn) with add/remove dropdown, quick-add chips
- **Tab 2 - Bio & Achievements:**
  - Rich text bio editor (TipTap)
  - Achievement list with highlight toggle (max 3 starred)
  - Tournament stats: total tournaments, total achievements, featured highlights
  - Link to certificates page
- **Validation:** Real-time field validation, debounced username availability check
- **API:** GET/PUT /archer/me, GET /archers/me/stats, GET /auth/check-username

#### /dashboard/archer/settings
**File:** `pages/dashboard/archer/settings.vue` (361 lines)
- **2 Tabs:** Security | Theme
- **Security Tab:**
  - Email change with OTP verification (request OTP → verify 6-digit code)
  - Password change (new + confirm, min 6 chars)
  - Shows whether user has existing password (OAuth users may not)
- **Theme Tab:**
  - Visual theme picker with preview mockups (navy sidebar + canvas preview)
  - Multiple theme options with color swatches
  - Uses useTheme composable

#### /dashboard/archer/tournaments
**File:** `pages/dashboard/archer/tournaments/index.vue` (25 lines)
- Wrapper for `DashboardArcherTournamentsList` component
- Clears tournament context on mount

#### /dashboard/archer/tournaments/:id (Redirect)
**File:** `pages/dashboard/archer/tournaments/[id]/index.vue` (40 lines)
- Verifies event exists, redirects to `/dashboard/archer/tournaments/:id/overview`
- 404 handling for invalid IDs

#### /dashboard/archer/tournaments/:id/overview
**File:** `pages/dashboard/archer/tournaments/[id]/overview.vue` (370 lines)
- Hero banner with event details (name, venue, date, status badge)
- Fetches: event data, my participant registration, schedules
- Payment status card (paid/pending with action buttons)
- Schedule preview (next 3 upcoming items)
- Category info with back number / target name
- Quick links to: my-registration, my-qualification, my-elimination, my-target, my-team
- API: GET /tournaments/:id, GET /tournaments/:id/participants/me, GET /tournaments/:id/schedules

#### /dashboard/archer/tournaments/:id/my-registration
**File:** `pages/dashboard/archer/tournaments/[id]/my-registration.vue` (956 lines)
- Registration card with QR code display (qrcode.vue)
- Category info, back number, target name, payment status
- Payment action: checkout URL redirect or payment detail navigation
- Cancel registration with confirmation modal
- Payment instructions display (parsed JSON step groups)
- Transaction history with status badges
- Image lightbox for payment proof
- API: GET /tournaments/:id/participants/me, DELETE /tournaments/:id/participants/me, POST /payment/create

#### /dashboard/archer/tournaments/:id/my-qualification
**File:** `pages/dashboard/archer/tournaments/[id]/my-qualification.vue` (453 lines)
- Qualification session cards with scores
- Per-session score breakdown (end-by-end arrow scores)
- Total score, 10/X count, rank display
- Category selector dropdown (for multi-category archers)
- Session lock status indicator
- Leaderboard position with arrow progression
- API: GET /tournaments/:id/qualification/leaderboard, GET /qualification/sessions/:sessionId/scores

#### /dashboard/archer/tournaments/:id/my-elimination
**File:** `pages/dashboard/archer/tournaments/[id]/my-elimination.vue` (404 lines)
- Bracket visualization with match tree
- Match details: opponent, target/board, set scores
- Win/loss indicators
- Category selector for multi-category
- Bracket round progression (1/32 → 1/16 → 1/8 → 1/4 → semi → final)
- API: GET /tournaments/:id/elimination/brackets, GET /match/:matchId

#### /dashboard/archer/tournaments/:id/my-target
**File:** `pages/dashboard/archer/tournaments/[id]/my-target.vue` (663 lines)
- Target assignment display (target number, board position 1A/1B/1C/1D)
- Visual target map with archer positions
- Session info and schedule
- Nearby archers list (lane mates)
- Category and division info

#### /dashboard/archer/tournaments/:id/my-team
**File:** `pages/dashboard/archer/tournaments/[id]/my-team.vue` (219 lines)
- Team roster display with captain indicator
- Team members list with avatar, name, category
- Team registration status
- Empty state if not in any team

#### /dashboard/archer/certificates
**File:** `pages/dashboard/archer/certificates/index.vue` (816 lines)
- **Stats:** Total certificates, unique tournaments, latest certificate date
- **Filter:** Tournament filter pills, search bar, view mode toggle (grouped by tournament vs flat grid)
- **Grouped View:** Certificates organized by tournament with section headers
- **Flat Grid:** 3-column card grid
- **Certificate Card:** Certificate number (copyable), event name, category, PDF preview iframe, issue date, view/download buttons
- **Fullscreen Preview Modal:** PDF/image lightbox with title, subtitle, download button
- Empty states: no certificates, no search results
- API: GET /archers/my/certificates

#### /dashboard/archer/payments
**File:** `pages/dashboard/archer/payments/index.vue` (501 lines)
- **Stats:** Total transactions, total spent, paid count, pending count
- **Filters:** Status (paid/awaiting_verification/pending/failed), Method (free/manual/mayar/paypal), Sort (date/amount)
- **Modal Filter Dialog:** PaymentFilterModal component
- **Table columns:** Invoice reference (copyable), Tournament name, Payment method (with icon), Amount, Status badge
- **Actions:** Detail link, Pay button (checkout_url), Upload proof button
- API: GET /payment/my?limit=100&offset=0

#### /dashboard/archer/payments/:reference
**File:** `pages/dashboard/archer/payments/[reference].vue` (1011 lines)
- **Free Registration Banner:** Green gradient with link to tournament ticket
- **Pending Payment Banner:** Navy gradient with PayPal/Mayar checkout button
- **Receipt Card:** Reference number (copyable), status badge, total amount, event name, payer name, payment method, transaction date, paid/expired date
- **Participants Table:** Search, sort (name/club/category/fee), pagination (10 per page)
- **Team Reservations Table:** (if applicable)
- **Invoice Summary:** Subtotals for individual + team + service fee = total
- **Manual Payment Section:** Current proof display with lightbox, re-upload form (file picker, sender name input, submit button)
- **Payment Activity Timeline:** PaymentActivityTimeline component
- **Cancel Transaction:** Modal confirmation for pending/awaiting_verification payments
- **Image Lightbox:** Payment proof zoom modal
- API: GET /payment/status/:reference, POST /payment/:reference/cancel, POST /payment/manual/:reference/upload-proof

#### /dashboard/archer/notifications
**File:** `pages/dashboard/archer/notifications/index.vue` (19 lines)
- Wrapper for `NotificationCenter` component
- Shared notification UI across all roles

---

### ORGANIZER DASHBOARD (60+ pages)

#### /dashboard/organizer (Home)
**File:** `pages/dashboard/organizer/index.vue` (457 lines)
- **Stats:** Total archers, active targets (x/y), completion scoring %, total revenue
- **Revenue Trend:** Bar chart visualization with hover tooltips, 30-day window
- **Quick Actions:** Manage registrations, create event, create news, manage scorekeepers, wallet/payouts
- **Event Recap:** Last 5 completed events list (date, name, status, participant count, category count)
- **Leaderboard:** Top 5 archers with rank, avatar, name, category, score
- API: GET /organizers/stats, GET /tournaments

#### /dashboard/organizer/profile
**File:** `pages/dashboard/organizer/profile.vue` (839 lines)
- **Tabs:** General | Branding | Social | Bank Accounts | Page Settings
- **General:** Organization name, slug, email, phone, address, city, country, description
- **Branding:** Logo upload, banner upload
- **Social:** Social media links
- **Bank Accounts:** Bank account management for withdrawals
- **Page Settings:** Public organizer page customization
- API: GET/PUT /organizer/me

#### /dashboard/organizer/settings
**File:** `pages/dashboard/organizer/settings.vue`
- Security settings, notification preferences

#### /dashboard/organizer/subscription
**File:** `pages/dashboard/organizer/subscription.vue`
- Current subscription plan, quota usage, upgrade options

#### /dashboard/organizer/scorekeepers
**File:** `pages/dashboard/organizer/scorekeepers.vue` (413 lines)
- DataTable with search, sort, status filter (active/inactive)
- Add scorekeeper modal (generates 5-char access code)
- Regenerate code action
- Delete scorekeeper
- Premium-gated (requires active subscription)
- API: GET/POST/PUT/DELETE /organizers/me/scorekeepers

#### /dashboard/organizer/teams
**File:** `pages/dashboard/organizer/teams/index.vue` (277 lines)
- Stats: Total teams, active, pending, total athletes
- Team list with status badges
- Create team modal
- API: GET/POST /teams

#### /dashboard/organizer/bank-accounts
**File:** `pages/dashboard/organizer/bank-accounts.vue`
- Bank account CRUD for organizer withdrawals
- Sync, add, edit, delete operations

#### /dashboard/organizer/payment-methods
**File:** `pages/dashboard/organizer/payment-methods.vue`
- Payment method management per organizer (QRIS, VA, manual transfer)
- Sync, add, edit, delete

#### /dashboard/organizer/balance
**File:** `pages/dashboard/organizer/balance.vue`
- Balance overview, withdrawal request form
- Withdrawal history

#### /dashboard/organizer/wallet
**File:** `pages/dashboard/organizer/wallet/index.vue` (161 lines)
- Available balance display
- Withdrawal request form (amount, destination bank notes, min Rp 50,000)
- Withdrawal history table with search
- API: GET /organizers/me/wallet, POST /organizers/me/wallet/withdrawals, GET /organizers/me/wallet/mutations

#### /dashboard/organizer/earnings
**File:** `pages/dashboard/organizer/earnings/index.vue` (226 lines)
- **Stats:** Total earnings, monthly earnings, total participants, completed events
- Earnings history table per event
- Export to Excel button
- API: GET /organizers/earnings, GET /organizers/earnings/:id

#### /dashboard/organizer/earnings/:id
**File:** `pages/dashboard/organizer/earnings/[id].vue`
- Detailed earnings breakdown for a specific event
- Revenue per category/participant

#### /dashboard/organizer/notifications
**File:** `pages/dashboard/organizer/notifications/index.vue`
- Wrapper for NotificationCenter component

#### /dashboard/organizer/reports
**File:** `pages/dashboard/organizer/reports/index.vue` (242 lines)
- **Stats:** Total registrations, total revenue, check-in rate, completion rate
- Links to: Participants report, Finance report, Performance report, Attendance report

#### /dashboard/organizer/reports/participants
**File:** `pages/dashboard/organizer/reports/participants.vue`
- Participant registration data across events
- Export functionality

#### /dashboard/organizer/reports/finance
**File:** `pages/dashboard/organizer/reports/finance.vue`
- Financial report: revenue, payments, refunds per event

#### /dashboard/organizer/reports/performance
**File:** `pages/dashboard/organizer/reports/performance.vue`
- Event performance metrics: attendance rate, scoring completion

#### /dashboard/organizer/reports/attendance
**File:** `pages/dashboard/organizer/reports/attendance.vue`
- Check-in/attendance data per event

#### /dashboard/organizer/package (4 pages)
- **/index** - Package listing (Starter/Standard/Professional)
- **/detail** - Package detail view
- **/payment** - Payment flow for package purchase
- **/success** - Payment success confirmation

#### /dashboard/organizer/tournaments (2 pages)
- **/index** - Tournament list with stats, create button
- **/create** - Tournament creation form with categories, divisions, quotas

#### /dashboard/organizer/tournaments/:id (Redirect)
**File:** `pages/dashboard/organizer/tournaments/[id]/index.vue` (38 lines)
- Redirects to /dashboard/organizer/tournaments/:id/overview

#### /dashboard/organizer/tournaments/:id/overview
**File:** `pages/dashboard/organizer/tournaments/[id]/overview.vue` (1060 lines)
- **Stats:** Participants, categories, targets, revenue
- **Publish button** (for draft events) + Share dialog
- **Analytics:** Revenue trend, registration trend bar charts
- **Recent activity:** Participant registrations, payments, scoring updates
- **Quick links:** All tournament management sub-pages
- API: GET /tournaments/:id, GET /organizers/stats

#### /dashboard/organizer/tournaments/:id/page
**File:** `pages/dashboard/organizer/tournaments/[id]/page.vue` (1819 lines)
- **Tournament Page Builder** - Full event configuration
- Country + currency selector
- Status toggle (draft/active)
- Description editor (TipTap)
- Category management
- Fee configuration (per participant / per category / per type)
- FAQ editor
- Technical guidebook URL
- WhatsApp contact number
- Venue details + GMaps embed
- Page settings (visibility, featured sections)
- API: GET/PUT /tournaments/:id

#### /dashboard/organizer/tournaments/:id/revenue
**File:** `pages/dashboard/organizer/tournaments/[id]/revenue.vue`
- Revenue breakdown per tournament
- Payment status summary

#### /dashboard/organizer/tournaments/:id/my-registration
**File:** `pages/dashboard/organizer/tournaments/[id]/my-registration.vue`
- Organizer's own registration view (when organizer is also participant)

#### /dashboard/organizer/tournaments/:id/my-qualification
**File:** `pages/dashboard/organizer/tournaments/[id]/my-qualification.vue`
- Organizer's own qualification view

#### /dashboard/organizer/tournaments/:id/my-elimination
**File:** `pages/dashboard/organizer/tournaments/[id]/my-elimination.vue`
- Organizer's own elimination view

#### /dashboard/organizer/tournaments/:id/categories (2 pages)
- **/index** - Category list with participant counts
- **/:categoryId** - Category detail: edit division, category name, max participants, fees, team size, qualification arrows, elimination format

#### /dashboard/organizer/tournaments/:id/participants (5 pages)
- **/index** - Participant DataTable with search, sort, status filter, pagination
- **/add** - Manual participant add form
- **/detail** - Participant detail view
- **/edit** - Bulk participant edit
- **/:participantId/index** - Individual participant detail
- **/:participantId/edit** - Individual participant edit
- API: GET/POST/PUT/DELETE /tournaments/:id/participants

#### /dashboard/organizer/tournaments/:id/payments (2 pages)
- **/index** - Payment list for tournament, status filter, approve/reject
- **/:reference** - Payment detail with proof display, manual approve/refund
- API: GET /tournaments/:id/payments, POST /payment/manual/:reference/verify

#### /dashboard/organizer/tournaments/:id/checkout
**File:** `pages/dashboard/organizer/tournaments/[id]/checkout.vue`
- Organizer payment checkout for own tournament

#### /dashboard/organizer/tournaments/:id/qualification (3 pages)
- **/index** - Qualification session list (cards with session code, date, lock status, participant count)
  - Create session modal, lock/unlock session
  - Navigate to session detail or results
- **/results** - Qualification results leaderboard
- **/:session** - Session detail: target assignments, scores per archer
- API: GET/POST /tournaments/:id/qualification/sessions, POST /tournaments/:id/qualification/sessions/:sessionId/lock

#### /dashboard/organizer/tournaments/:id/elimination (3 pages)
- **/index** - Bracket list (cards with bracket name, category, format, participant count, lock status)
  - Create bracket modal, generate bracket
  - Navigate to bracket detail
- **/:bracketId/index** - Bracket detail with match tree visualization
- **/:bracketId/:round** - Round-specific match scoring interface
- API: GET/POST /tournaments/:id/elimination/brackets, POST /tournaments/:id/elimination/brackets/:bracketId/generate

#### /dashboard/organizer/tournaments/:id/targets (2 pages)
- **/index** - Target management with grid/table view toggle
  - Target cards showing target name, board positions (1A/1B/1C/1D), assigned archers
  - Add target, batch update, delete
  - Auto-assign button
- **/map** - Visual target map showing physical layout
- API: GET/POST/PUT/DELETE /tournaments/:id/targets

#### /dashboard/organizer/tournaments/:id/schedule
**File:** `pages/dashboard/organizer/tournaments/[id]/schedule/index.vue` (1033 lines)
- Day-based tab navigation (Day 1, Day 2, etc.)
- Schedule items list with time, title, description, location
- Add schedule item modal
- Auto-generate schedule button (premium-gated)
- Shift/delay all items function
- Drag-to-reorder capability
- API: GET/PUT /tournaments/:id/schedules, POST /tournaments/:id/schedule/items, POST /tournaments/:id/schedule/auto-generate

#### /dashboard/organizer/tournaments/:id/certificate
**File:** `pages/dashboard/organizer/tournaments/[id]/certificate.vue` (1201 lines)
- **Stats:** Total participants, eligible (paid), issued certificates, completion rate %
- **Actions:** Bulk upload ZIP, generate all, clear all
- **Certificate list:** Per-participant certificate status (issued/pending/not eligible)
- **Manual assign:** Individual certificate assignment
- **Upload batches:** Progress tracking for ZIP uploads
- **Preview:** Certificate preview modal
- API: GET/POST /tournaments/:id/certificates, POST /tournaments/:id/certificates/upload-zip, POST /tournaments/:id/certificates/generate-all

#### /dashboard/organizer/tournaments/:id/media
**File:** `pages/dashboard/organizer/tournaments/[id]/media.vue` (562 lines)
- Storage quota gauge (used / max based on tier)
- File grid with preview thumbnails
- Upload button (images + PDFs)
- File count, total size display
- Delete files
- Tier badge (Free/Standard/Professional)
- API: GET /tournaments/:id/media-storage, POST /media/upload

#### /dashboard/organizer/tournaments/:id/printout (4 pages)
- **/index** - Printout hub with links to all printout types
- **/statistics** - Tournament statistics printout (participant demographics, category distribution)
- **/participants/index** - Participant list printout
- **/qualification/index** - Qualification results printout
- **/elimination/index** - Elimination brackets/scoresheet printout
- API: GET /tournaments/:id/participants/printout, GET /tournaments/:id/qualification/results/printout, etc.

#### /dashboard/organizer/tournaments/:id/teams (2 pages)
- **/index** - Team list with stats
- **/:teamId** - Team detail with member roster

#### /dashboard/organizer/tournaments/:id/reset
**File:** `pages/dashboard/organizer/tournaments/[id]/reset/index.vue`
- Tournament data reset with confirmation
- Options: reset scores, reset assignments, full reset

---

### CLUB DASHBOARD (1 page)

#### /dashboard/club/notifications
**File:** `pages/dashboard/club/notifications/index.vue`
- NotificationCenter wrapper for club role

---

### ROOT/ADMIN DASHBOARD (6 pages)

#### /dashboard/root (Home)
**File:** `pages/dashboard/root/index.vue` (401 lines)
- **Stats:** Total GMV (with this month revenue), total users (with new users 30d), tournaments & registrations, revenue breakdown
- Quick actions: Manage articles
- API: GET /root/dashboard/recap

#### /dashboard/root/archers
**File:** `pages/dashboard/root/archers/index.vue` (509 lines)
- **Terminal-style header** with navy gradient + gold accent
- **Stats:** Total archers, active accounts, new registrations
- Archers DataTable with search, sort, status filter
- Actions: View detail, terminate/suspend account
- API: GET /root/dashboard/users

#### /dashboard/root/tournaments
**File:** `pages/dashboard/root/tournaments/index.vue` (454 lines)
- **Stats:** Total tournaments, published, draft, total participants
- Tournament DataTable with search
- Actions: View, delete tournament
- API: GET /root/dashboard/tournaments

#### /dashboard/root/articles (3 pages)
- **/index** - Article list DataTable (title, status, views, date) with search, create button
- **/create** - Article creation form with TipTap editor, image upload, category, status
- **/:id** - Article edit/detail view
- API: GET/POST/PUT/DELETE /root/dashboard/articles

---

### SHARED PROFILE PAGES (3 pages)

#### /dashboard/profile/:username
**File:** `pages/dashboard/profile/[username].vue`
- Public profile view for any user persona

#### /dashboard/profile/:id
**File:** `pages/dashboard/profile/[id].vue`
- Profile view by ID

#### /dashboard/profile/edit
**File:** `pages/dashboard/profile/edit.vue` (251 lines)
- Profile edit form (outside dashboard layout, uses simple nav)
- Avatar upload, cover photo, personal info fields
- Save changes button

---

## API Endpoints Summary

### Authentication
- POST /auth/register - Register new account
- POST /auth/register-email - Register with email OTP
- POST /auth/login - Login
- POST /auth/refresh - Refresh token
- POST /auth/logout - Logout
- GET /auth/google - Google OAuth
- POST /auth/forgot-password - Request password reset
- POST /auth/reset-password - Reset password

### User
- GET /user/profile - Get profile
- PUT /user/profile - Update profile
- GET /user/subscription - Get subscription
- GET /user/settings - Get settings

### Tournaments / Events
- GET /tournaments - List tournaments
- GET /tournaments/:id - Tournament detail
- POST /tournaments - Create tournament (auth)
- PUT /tournaments/:id - Update tournament (auth)
- GET /tournaments/:id/participants - List participants
- POST /tournaments/:id/participants - Register participant
- GET /tournaments/:id/schedule - Get schedule
- GET /tournaments/:id/results/qualification - Qualification results
- GET /tournaments/:id/results/elimination - Elimination results

### Qualification
- GET /tournaments/:id/qualification/sessions - Get sessions
- POST /tournaments/:id/qualification/sessions - Create session (auth)
- GET /tournaments/:id/qualification/leaderboard - Leaderboard

### Elimination
- GET /tournaments/:id/elimination/brackets - Get brackets
- POST /tournaments/:id/elimination/brackets - Create bracket (auth)
- POST /tournaments/:id/elimination/brackets/:bracketId/generate - Generate bracket
- POST /tournaments/:id/elimination/brackets/:bracketId/matches/:matchId/score - Update match score

### Archers
- GET /archers - List archers
- GET /archers/:id - Archer detail
- GET /archers/:id/tournaments - Archer tournament history
- POST /archers - Create archer (auth)
- PUT /archers/:id - Update archer (auth)
- DELETE /archers/:id - Delete archer (auth)

### Payment
- GET /payment/channels - Payment channels
- POST /payment/create - Create payment (auth)
- GET /payment/status/:reference - Payment status
- POST /payment/mayar/callback - Mayar webhook
- POST /payment/manual/create - Create manual payment
- POST /payment/manual/:reference/upload-proof - Upload proof

### Organizers
- GET /organizers - List organizers
- GET /organizers/:slug - Organizer detail
- GET /organizers/me - My organizer profile (auth)
- GET /organizers/me/quota - Quota info (auth)
- GET /organizers/reports/participants - Participant report (auth)
- GET /organizers/reports/finance - Finance report (auth)

### Mobile API
- POST /mobile/auth/scorekeeper/login - Scorekeeper login
- POST /mobile/auth/archer/login - Archer login
- GET /mobile/tournaments - List tournaments
- GET /mobile/tournaments/:slug - Tournament detail
- GET /mobile/scan - Scan target (auth)
- POST /mobile/qualification/scoring/scores/:assignmentId - Submit score
- GET /mobile/elimination/scoring/cards - Elimination scoring cards

### Blog & Docs
- GET /blog/articles - List articles
- GET /blog/articles/:slug - Article detail
- GET /docs - List docs
- GET /docs/*slug - Doc detail

### Admin/Root
- POST /root/login - Admin login
- GET /root/dashboard/users - All users
- GET /root/dashboard/subscriptions - All subscriptions
- GET /root/dashboard/clubs - All clubs
- GET /root/dashboard/recap - Business recap

---

## Database Tables (Inferred from Models)

| Table | Description |
|-------|-------------|
| archers | Archer profiles and personal data |
| tournaments (events) | Tournament/event data |
| tournament_categories | Event categories and divisions |
| tournament_images | Tournament gallery images |
| tournament_external | External tournament data (Ianseo scraper) |
| event_participants | Participant registrations |
| qualification_sessions | Qualification round sessions |
| qualification_assignments | Archer-to-target assignments |
| qualification_scores | Arrow-by-arrow scores |
| elimination_brackets | Elimination bracket data |
| elimination_matches | Individual matches |
| elimination_match_scores | Match scores |
| teams | Team data |
| team_members | Team roster |
| payment_transactions | Payment records |
| payment_channels | Available payment methods |
| clubs | Archery clubs |
| organizers | Tournament organizers |
| users | User accounts |
| scorekeepers | Scorekeeper accounts |
| certificates | Digital certificates |
| notifications | User notifications |
| bank_accounts | Organizer bank accounts |
| wallet_mutations | Wallet transactions |
| withdrawals | Withdrawal requests |
| blog_articles | Blog posts |
| blog_comments | Article comments |
| docs | Documentation pages |
| subscriptions | User subscriptions |
| subscription_plans | Available plans |
| event_schedules | Tournament schedules |
| event_targets | Physical targets |
| media | Uploaded media files |
| disciplines | Archery disciplines |
| bow_types | Bow type reference |
| age_groups | Age group reference |
| gender_divisions | Gender division reference |
| event_types | Event type reference |
| cities | City reference |

---

## User Roles

| Role | Description |
|------|-------------|
| Archer | Regular user, participates in tournaments |
| Organizer | Creates and manages tournaments |
| Scorekeeper | Scores matches at events |
| Root/Admin | Platform administrator |

---

## Subscription Plans

| Plan | Price | Participants | Storage |
|------|-------|-------------|---------|
| Starter (Free EO) | $0/tournament | Up to 50 | 200 MB |
| Standard | $1.50/tournament | Up to 200 | 3 GB |
| Professional (Elite EO) | $3.50/tournament | Unlimited | 10 GB |

---

## Payment Gateways

- **Mayar** - Primary gateway (QRIS, Virtual Accounts, Cards)
- **PayPal** - International payments

---

## Competitor Comparison

| Feature | Archeris.net | Ianseo.net |
|---------|-------------|------------|
| Platform | 100% online, browser-based | Desktop software |
| Scoring | Mobile via internet/Wi-Fi | Local Wi-Fi router required |
| Registration | Online self-registration | Manual Excel import |
| Payments | Automatic (QRIS, VA, cards) | Manual tracking |
| Target Allocation | Visual 1-click auto-assign | Desktop menus |
| Brackets | Auto-generate with shoot-off | Desktop setup |
| Leaderboard | Real-time, anywhere | Manual upload |
| Printouts | 8 ready PDF templates | Standard WA formats |

---

## URL Redirects & Route Rules (Nitro)
| From | To | Status |
|------|-----|--------|
| /feed.xml | /rss.xml | 301 |
| /pricing | /package | 301 |
| /events | /tournaments | 301 |
| /events/** | /tournaments/** | 301 |
| /**/*.webp, /**/*.png, /**/*.jpg | (static caching) | max-age=31536000 |

---

## Scorekeeper Portal
- Dedicated login page at `/scorekeeper/login`
- 5-character access code authentication (e.g., A1B2C)
- Code provided by Tournament Director/Event Organizer
- Separate from archer and organizer accounts
- Mobile-first design for field use

---

## Organizer Tournament Management - Full Feature Set

### Per-Tournament Pages (Organizer)
1. **Overview** - Dashboard with stats, quick actions, recent activity
2. **Page Builder** - Customize public tournament page layout/settings
3. **Revenue** - Financial breakdown per tournament
4. **Categories** - Manage divisions (Recurve, Compound, Barebow) and sub-categories
5. **Participants** - Full CRUD: list, add, edit, detail view, search
6. **Payments** - Transaction list, payment detail, manual approve/refund
7. **Checkout** - Payment flow for organizer's own tournament
8. **Qualification** - Session management, scoring, results, per-session view
9. **Elimination** - Bracket creation, generation, match management, round-by-round scoring
10. **Targets** - Target lane setup, visual target map, auto-assignment
11. **Schedule** - Timeline management, auto-generate, shift/delay
12. **Certificate** - Template builder, bulk generate, assign, clear
13. **Media** - Image upload, gallery management, storage quota
14. **Teams** - Team creation, roster management, team detail
15. **Printout** - PDF generation: participants, qualification, elimination, statistics
16. **Reset** - Tournament data reset functionality

---

## Key UI/UX Patterns
- **Design System:** Navy + Gold/Primary color theme with motif pattern backgrounds
- **Cards:** Rounded-3xl with navy gradient headers, icon badges
- **Dashboard Layout:** Header with breadcrumbs + actions, stat cards grid, analytics charts
- **Tables:** Sticky headers, hover highlights, avatar + name cells
- **Empty States:** Symmetrical centered designs with icon + description
- **Skeleton Loading:** Component-level skeleton loaders for each section
- **Responsive:** Mobile-first, works on phones/tablets/desktops
- **Premium Gating:** Features locked behind active subscription with modal upsell

---

## Mobile App

- **Platform:** Flutter
- **Distribution:** APK download + Google Play Store
- **Features:** Scoring, tournament discovery, registration, payments, certificates

---

## Social Media

- Instagram: @archerisnet
- Threads: @archerisnet
- Facebook: archerisnet

---

## Project Structure Summary

### Backend (Go API)
```
api/
├── main.go              # Entry point, Gin router, all route definitions
├── handler/             # 48 handler files (auth, tournament, payment, etc.)
│   └── mobile/          # 16 mobile-specific handlers
├── models/              # 13 model files (archer, tournament, payment, etc.)
├── middleware/          # Auth, rate limiting, role checks
├── database/            # MySQL connection setup
├── utils/               # Helpers, recovery middleware
├── migrations/          # DB migration scripts
└── docs/                # Swagger documentation
```

### Frontend (Nuxt.js App)
```
app/
├── pages/               # 132 Vue page files (see full list above)
├── components/          # Reusable Vue components
├── composables/         # Vue composables (useAuth, useApi, etc.)
├── layouts/             # Page layouts (dashboard, default, etc.)
├── middleware/          # Nuxt route middleware (auth checks)
├── plugins/             # Nuxt plugins
├── i18n/                # Internationalization config
├── locales/             # Translation files (en/id: common, auth, dashboard, tournaments, archers, commerce)
├── assets/              # CSS, SCSS, images
├── utils/               # Helper utilities
├── nuxt.config.ts       # Nuxt configuration (SSR, i18n, SEO, Nitro rules)
└── package.json         # Dependencies: Nuxt 3, Vuetify 3, Tailwind, TipTap, Chart.js, GSAP
```

### Total Vue Pages: 132
- Public pages: 28
- Archer dashboard: 14
- Organizer dashboard: 60+
- Club dashboard: 1
- Root/Admin dashboard: 6
- Shared/Profile: 4
- Dev/Test/Graphics: 9
- Scorekeeper: 1
- Auth: 7

---

## Non-Dashboard Pages - Full Breakdown

### HOMEPAGE

#### / (Index)
**File:** `pages/index.vue` (183 lines)
- **Layout:** landing
- **Sections:** HomeHero, HomeLearnToUse, HomeTrustTestimonials, HomeFeaturesDemo, HomePricing, HomeBlogSection, HomeMobileCTA
- **SEO:** Schema.org structured data (WebSite, Organization, SoftwareApplication)
- **Components:** 7 home section components
- **CTAs:** Get Started → /auth/register, Browse Tournaments → /tournaments

---

### TOURNAMENT DISCOVERY

#### /tournaments
**File:** `pages/tournaments/index.vue` (1041 lines)
- **Hero:** Navy gradient with hero-event.jpeg background, breadcrumb
- **Source Tabs:** All Tournaments | Platform Events | Ianseo Archives (with count badges)
- **Filters:** Search bar, status filter (upcoming/ongoing/completed), sort (date/name)
- **Sticky filter toolbar** with backdrop blur
- **Tournament Cards:** Banner image, name, date, location, participant count, status badge, organizer info
- **External/Ianseo tournaments** integrated alongside platform events
- **Empty states** for no results per source type
- API: GET /tournaments, GET /tournaments/external

#### /tournaments/:slug/[[tab]]
**File:** `pages/tournaments/[slug]/[[tab]].vue` (1617 lines)
- **Hero Header:** Full-width banner with gradient overlay, tournament name, date, location, country badge
- **Sticky Tabs:** overview | schedule | athletes | results | venue | gallery
- **Overview Tab:** About section (TipTap description), technical guidebook PDF download, competition categories (horizontal scroll cards with icons), registration fees (3 layouts: per-type cards, per-category list, flat fee), organizer info
- **Schedule Tab:** Day-by-day timeline
- **Athletes Tab:** Participant list with search/filter
- **Results Tab:** Qualification + elimination results, medal standings
- **Venue Tab:** Location details, map embed
- **Gallery Tab:** Tournament images grid
- **Alias slugs:** ringkasan→overview, jadwal→schedule, peserta→athletes, hasil→results, lokasi→venue
- **External tournament detection:** Shows TournamentExternalDetailView if isExternal

#### /tournaments/:slug/register
**File:** `pages/tournaments/[slug]/register.vue` (3323 lines - LARGEST PAGE)
- **Layout:** Custom registration header (not landing/dashboard)
- **Language Toggle:** EN/ID with flag icons (circle-flags:us, circle-flags:id)
- **3-Step Progress Tabs:**
  1. Registration Type
  2. Select Categories
  3. Payment
- **User Profile Dropdown:** Avatar, name, club badge, links to dashboard/profile, logout

**STEP 1 - Registration Type:**
- **Draft Restoration Banner:** Amber warning with restore/discard buttons (localStorage)
- **2 Mode Cards:**
  - **Self & Teammates (captain_team):** "Competing Athlete" - register yourself + teammates
  - **Club Delegation (club_delegation):** "Collective/Official" - register multiple athletes under one invoice
- **Archer Profile Form** (captain_team mode):
  - Full name, gender toggle buttons (male/female with icons)
  - Club selector (search/create new club)
- **Club Delegation Form:** Empty state with 2 action cards (Bulk CSV Import, Add Athlete Manually)

**STEP 2 - Categories & Roster:**
- **Archer Profile Banner:** Avatar, name, club, gender badge
- **Gender Mismatch Alert:** Shows if no categories match selected gender, with switch button
- **Individual Categories:** Multi-select grid cards
  - Category icon image (from getCategoryIcon utility)
  - Name, division, gender division, fee
  - Checkmark selection indicator
- **Team Categories:** Expandable cards
  - Toggle bar with category info + fee + checkmark
  - **TeamSlotBuilder** component inside expanded section
  - Captain auto-filled, partner search modal, remove partner
  - Mixed team (1M+1F) vs regular team (3 same division)
  - Per-partner fee calculation
- **Club Delegation Mode:**
  - Athlete roster list
  - Bulk CSV import modal
  - Add individual athlete form
  - Team quota reservation
  - Per-athlete category multi-selection

**STEP 3 - Payment:**
- **Registration Summary:** Participant list, categories, individual fees, team fees
- **Total calculation:** Sum of all entries
- **Submit Registration:** Creates participant + payment transaction
- **Success State:** Full-screen navy overlay with green check → redirects to /tournaments/:slug/payment

**Key Features:**
- Profile auto-fill from logged-in archer
- Real-time fee calculation
- Category quota tracking (spots remaining)
- Gender-based category filtering
- CSV bulk import for delegation mode
- Draft auto-save to localStorage
- Partner search for team categories

#### /tournaments/:slug/payment
**File:** `pages/tournaments/[slug]/payment.vue` (400 lines)
- **Hero:** Event banner with breadcrumb
- **Left Column (Payment Options):**
  - **Payment Method Selection:** Radio cards grouped by category
    - Mayar group: QRIS, Virtual Accounts (BCA, BNI, BRI, Mandiri, etc.), Cards
    - PayPal group
    - Manual Transfer group
    - Channel icons loaded from Mayar API
    - Loading spinner, empty state
  - **Payment Instructions:** Step-by-step guide per selected channel
    - Section titles (e.g., "Cara Pembayaran via QRIS")
    - Numbered steps with circular navy badges
    - Step descriptions
- **Right Column (Order Summary - Sticky):**
  - Event thumbnail + name + category badge
  - Fee breakdown: registration fee
  - Total Bayar (large font)
  - Payment method fee note
  - **Pay Now** button → creates payment

#### /tournaments/:slug/targets
**File:** `pages/tournaments/[slug]/targets.vue`
- Public target assignments view
- Target lanes with archer names

---

### AUTHENTICATION PAGES

#### /auth/login
**File:** `pages/auth/login/index.vue` (325 lines)
- **Layout:** Split screen (left hero slideshow, right form)
- **Left Panel:** Rotating background images (3 slides: slide-1.jpeg, slide-2.jpeg, slide-3.jpeg, 2s interval), logo, headline, social proof (+2k archers avatars)
- **Right Panel:**
  - Email + password form with validation
  - **Real-time avatar lookup:** GET /auth/avatar/:email as user types
  - Remember me checkbox (default true)
  - Forgot password link
  - **Google OAuth** button
  - Link to register
- **Auto-login detection:** Redirects if already authenticated (role-based: archer→/dashboard/archer/tournaments)
- API: POST /auth/login, GET /auth/google, GET /auth/avatar/:email

#### /auth/register
**File:** `pages/auth/register.vue` (691 lines)
- **Layout:** Split screen (same as login)
- **User Type Tabs:** Archer | Organizer
- **Archer Fields:** Full name, email, password, confirm password, country (searchable)
- **Organizer Fields:** Organization name (with duplicate name check), email, password, confirm password
- **Terms & Conditions:** Checkbox with links to /terms and /privacy
- **Submit:** Email registration → OTP verification flow
- **Google OAuth** button
- API: POST /auth/register-email, POST /auth/verify-register-otp, POST /auth/resend-register-otp, POST /auth/register, GET /auth/check-name

#### /auth/forgot-password
**File:** `pages/auth/forgot-password.vue` (519 lines)
- **Layout:** Split screen
- **3-Step Progress Bar:**
  1. **Email Input:** Email field → send OTP
  2. **OTP Verification:** 6-digit code boxes (grid-cols-6), auto-advance on input, paste support, resend with cooldown timer, change email option
  3. **New Password:** New password + confirm, password strength indicator (weak/medium/strong bar), reset button
- **Validation:** Email format, OTP length, password min 6 chars, password match
- API: POST /auth/forgot-password, POST /auth/verify-reset-otp, POST /auth/reset-password

#### /auth/callback
**File:** `pages/auth/callback.vue` (118 lines)
- OAuth callback handler
- Supports token from URL query (?token=...)
- Stores auth token in cookie (with Secure flag for HTTPS)
- Fetches user profile via fetchProfileSSR
- Redirects to dashboard based on role
- Error state with back to login button

#### /auth/oauth
**File:** `pages/auth/oauth.vue` (55 lines)
- OAuth login success page
- Stores token from URL, cleans URL with replaceState
- "Back to Homepage" button

#### /auth/already-registered
**File:** `pages/auth/already-registered.vue` (85 lines)
- Shown when Google OAuth email matches existing account
- Displays email + account type (Archer/Organizer)
- "Log in to account" button → /auth/login

---

### PAYMENT PAGES

#### /payment/status/:reference
**File:** `pages/payment/status/[reference].vue` (35 lines)
- Redirect page → /dashboard/archer/payments/:reference
- Spinner loading state

#### /payment/failed/:reference
**File:** `pages/payment/failed/[reference].vue` (97 lines)
- Failed/expired payment page
- Shows: reference number, item name, amount, status badge (EXPIRED/FAILED)
- **Actions:**
  - "Daftar Ulang Event" → /tournaments/:slug (if event_slug exists)
  - "Lihat Riwayat Pembayaran" → /dashboard/archer/payments
  - WhatsApp support link

---

### PUBLIC PROFILE PAGES

#### /archers
**File:** `pages/archers/index.vue`
- Archers directory/listing page
- Search and filter archers

#### /archers/:slug
**File:** `pages/archers/[slug].vue`
- Public archer profile page
- Shows: avatar, banner, full name, username, club, bio, achievements, social media links, equipment, tournament history

#### /organizer
**File:** `pages/organizer/index.vue`
- Organizer directory

#### /organizer/:slug
**File:** `pages/organizer/[slug].vue`
- Public organizer profile page

---

### BLOG & CONTENT

#### /blog
**File:** `pages/blog/index.vue`
- Blog articles listing with thumbnails, categories, read time

#### /blog/:slug
**File:** `pages/blog/[slug].vue`
- Blog article detail with TipTap rendered content, comments section

#### /blog/category/:category
**File:** `pages/blog/category/[category].vue`
- Blog articles filtered by category

#### /docs
**File:** `pages/docs/index.vue`
- Documentation index

#### /docs/[...slug]
**File:** `pages/docs/[...slug].vue`
- Dynamic documentation pages (wildcard for nested categories)

---

### CERTIFICATES

#### /certificates/:uuid
**File:** `pages/certificates/[uuid].vue`
- Public certificate verification page
- Shows certificate details, QR verification

---

### MATCH

#### /match/:id
**File:** `pages/match/[id].vue`
- Public match detail page
- Shows bracket info, competitors, scores

---

### SCOREKEEPER

#### /scorekeeper/login
**File:** `pages/scorekeeper/login.vue` (101 lines)
- Dedicated scorekeeper portal login
- 5-character access code input (e.g., A1B2C)
- Navy background with motif pattern, blur effects
- "Masuk Portal Scoring" button
- API: POST /mobile/auth/scorekeeper/login

---

### STATIC/LEGAL PAGES

- **/about-us** - About Archeris page
- **/contact** - Contact form page
- **/privacy** - Privacy Policy
- **/terms** - Terms & Conditions
- **/disclaimer** - Disclaimer page
- **/faq** - FAQ page
- **/package** - Pricing page (Starter/Standard/Professional plans)
- **/archeris-vs-ianseo** - Comparison page
- **/ref** - Referral page

---

### DEV/TEST/GRAPHICS PAGES

- **/dev/paypal** - PayPal dev testing tool
- **/mockup-preview** - Mockup preview
- **/test/fop-preview** - FOP (Foam Optimized Page) test
- **/graphics/mobile-showcase** - Mobile app showcase graphics
- **/graphics/feature-archer** - Feature graphic: archer
- **/graphics/feature-competition** - Feature graphic: competition
- **/graphics/feature-registration** - Feature graphic: registration
- **/graphics/feature-scorekeeper** - Feature graphic: scorekeeper
- **/tools/thumbnail-generator** - Blog thumbnail generator tool

---

## Payment Flow - Detailed

### Payment Methods
1. **Mayar (Primary Gateway)**
   - **QRIS:** Scan QR code for payment
   - **Virtual Accounts:** BCA, BNI, BRI, Mandiri, BSI, CIMB, Danamon, Permata, etc.
   - **Credit/Debit Cards:** Visa, Mastercard
   - **E-Wallets:** GoPay, OVO, Dana, ShopeePay
   - Flow: Select channel → Create payment → Redirect to Mayar checkout → Callback webhook → Status update

2. **PayPal (International)**
   - Flow: Select PayPal → Create order → Capture order → Webhook callback → Status update
   - Dev endpoints: /dev/paypal/* for testing

3. **Manual Transfer**
   - Flow: Select bank → Submit registration → Upload payment proof (image) → Organizer verification → Approved/Rejected
   - Re-upload allowed if rejected
   - Sender name required (Atas Nama)

### Payment Status Flow
- `pending` → Awaiting payment
- `awaiting_verification` → Proof uploaded, waiting organizer approval
- `paid` → Payment confirmed (settlement/capture)
- `rejected` → Proof rejected by organizer
- `expired` → Payment timeout
- `cancelled` → User cancelled
- `refunded` → Refund issued

### Payment Transaction Types
- **Registration:** Tournament entry fees
- **Subscription:** Organizer plan upgrade (Standard/Professional)
- **Platform Fee:** Admin/service fees

---

## Registration Flow - Detailed

### Step 1: Identity & Mode
- **Mode Selection:** Captain/Team vs Club Delegation
- **Draft Restore:** Auto-saved registration data from localStorage
- **Profile Form:** Name, gender, club
- **Validation:** Required fields, gender selection

### Step 2: Categories & Roster
- **Individual Categories:** Multi-select (can join multiple)
- **Team Categories:** Expandable with TeamSlotBuilder
  - Captain auto-added
  - Partner search (by name, club)
  - Gender validation for mixed teams
  - Team size validation (3 for regular, 2 for mixed)
- **Club Delegation:**
  - Add athletes manually or CSV import
  - Per-athlete category selection
  - Team quota reservation
  - Single invoice for all

### Step 3: Payment
- **Summary:** All entries with individual fees
- **Total:** Sum + admin fee
- **Submit:** Creates participant records + payment transaction
- **Redirect:** To /tournaments/:slug/payment

---

## Tournament Detail Tab Structure
The tournament detail page `/tournaments/:slug/[[tab]]` supports these tabs:
- **overview** - About, description, technical guidebook, competition categories, registration fees
- **schedule** - Full tournament schedule timeline
- **athletes** - Participant list with search/filter
- **results** - Qualification & elimination results, medal standings
- **venue** - Location, map, venue details
- **gallery** - Tournament images and media

Alias slugs supported: ringkasan→overview, jadwal→schedule, peserta→athletes, hasil→results, lokasi→venue
