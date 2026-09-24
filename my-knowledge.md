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


---

## Flutter Mobile App (Archery Hub)

### Overview
**Project Name:** Archery Hub (archeris.net Mobile)  
**Repository:** `C:\E\ichsan\startup\archeryhub.id\Archery-Hub`  
**Framework:** Flutter (Dart ^3.9.0)  
**Version:** 1.0.0+3  
**Architecture:** Clean Architecture with BLoC pattern  
**API Base URL:** https://api.archeris.net (same as web platform)  

### Tech Stack

| Component | Library/Tool |
|-----------|--------------|
| **State Management** | flutter_bloc (^9.1.1) |
| **Dependency Injection** | get_it (^8.0.3) |
| **Routing** | go_router (^17.1.0) |
| **HTTP Client** | dio (^5.9.2) |
| **Secure Storage** | flutter_secure_storage (^10.0.0) |
| **QR Scanning** | mobile_scanner (^7.2.0) |
| **QR Generation** | qr_flutter (^4.1.0) |
| **Google Sign-In** | google_sign_in (^6.2.1) |
| **Image Picker** | image_picker (^1.1.2) |
| **PDF Viewer** | flutter_pdfview (^1.4.4) |
| **WebView** | webview_flutter (^4.10.0) |
| **Sharing** | share_plus (^13.3.0) |
| **UI/Fonts** | google_fonts (^8.0.2) |
| **HTML Rendering** | flutter_widget_from_html_core (^0.17.0) |
| **Loading States** | shimmer (^3.0.0) |
| **Internationalization** | intl (^0.20.2) |
| **Firebase** | firebase_core (^4.13.0) |

### Architecture

```
lib/
├── core/                  # Shared utilities, widgets, theme, routes
│   ├── constants/         # API constants, colors, routes
│   ├── network/           # Dio client, interceptors
│   ├── theme/             # App theme, colors, text styles
│   ├── widgets/           # Reusable widgets (DsImage, buttons, cards)
│   └── routes/            # GoRouter configuration
├── features/              # Feature modules (Clean Architecture)
│   ├── auth/              # Authentication & onboarding
│   ├── home/              # Landing screen & search
│   ├── tournaments/       # Tournament browsing & registration
│   ├── profile/           # User profile & settings
│   ├── news/              # News/blog articles
│   ├── shop/              # Marketplace products
│   ├── cart/              # Shopping cart
│   ├── scoring/           # Scorekeeper scoring interface
│   ├── organization/      # Event organizer portal
│   ├── seller/            # Product seller dashboard
│   ├── notifications/     # Notification center
│   └── chat/              # Chat & chatbot
├── presentation/          # Legacy seller shell screens
├── l10n/                  # Localization files
└── main.dart              # App entry point
```

Each feature follows Clean Architecture:
- **data/** — API data sources, models (fromJson/toJson), repository implementations
- **domain/** — Business entities, repository contracts, use cases
- **presentation/** — BLoC state management, screens, widgets

### User Roles

The app supports 4 distinct user roles with different navigation flows:

| Role | Description | Main Entry Point |
|------|-------------|-----------------|
| **Archer** | Athletes/archers who register for tournaments, view results, manage tickets | MainHub (bottom nav) |
| **Scorekeeper** | Field judges who scan targets and input arrow scores | ScorekeeperLandingScreen |
| **Organizer** | Event organizers who manage tournaments, participants, check-ins | OrganizationEventSelectorScreen |
| **Seller** | Marketplace vendors who sell archery equipment | SellerMainScaffold |

### Navigation Flow

```
SplashScreen → OnboardingScreen → LoginScreen → Role-Based Hub
                                                    ↓
                ┌────────────────────────────────────────────────────┐
                │                                                    │
       ┌────────▼────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐
       │ Archer MainHub  │  │ Scorekeeper  │  │ Organizer   │  │ Seller Hub   │
       │ (5 bottom tabs) │  │ Landing      │  │ Event Picker│  │ (4 tabs)     │
       └─────────────────┘  └──────────────┘  └─────────────┘  └──────────────┘
```

### Complete Screen List (77 Total)

#### 🔐 Auth & Onboarding (8 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 1 | SplashScreen | splash_screen.dart | /splash | App initialization with logo animation |
| 2 | OnboardingScreen | onboarding_screen.dart | /onboarding | Feature introduction for new users |
| 3 | LoginScreen | login_screen.dart | /login | Multi-role login (Archer/Organizer/Scorekeeper) |
| 4 | RegisterScreen | register_screen.dart | /register | Account registration (Archer/Organizer) |
| 5 | ForgotPasswordScreen | forgot_password_screen.dart | /forgot-password | Password reset request |
| 6 | ResetPasswordScreen | reset_password_screen.dart | /reset-password | New password input with OTP |
| 7 | VerifyEmailScreen | verify_email_screen.dart | /verify-email | Email OTP verification |
| 8 | ScorekeeperLoginScreen | scorekeeper_login_screen.dart | /scorekeeper-login | 5-character access code login for scorekeepers |

#### 🏹 Archer/User Portal (29 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 9 | MainHub | main_hub.dart | — | Bottom navigation shell (Home/Events/Tickets/News/Profile) |
| 10 | LandingScreen | landing_screen.dart | / | Home screen with featured events, news, quick actions |
| 11 | SearchScreen | search_screen.dart | /search | Global search for tournaments, clubs, locations |
| 12 | TournamentsScreen | tournaments_screen.dart | /events | Tournament directory with filters |
| 13 | TournamentDetailScreen | tournament_detail_screen.dart | /events/detail | Event details, categories, schedule, registration |
| 14 | TicketConfirmationScreen | ticket_confirmation_screen.dart | /ticket-confirmation | Registration summary before payment |
| 15 | PaymentMethodScreen | payment_method_screen.dart | /payment-method | Payment channel selection (QRIS/VA/Cards/Manual) |
| 16 | PaymentStatusScreen | payment_status_screen.dart | /payment-status | Payment instructions with countdown timer |
| 17 | PaymentSuccessScreen | payment_success_screen.dart | /payment-success | Payment confirmation & ticket issuance |
| 18 | ArcherPaymentHistoryScreen | archer_payment_history_screen.dart | /payment-history | Payment transaction history |
| 19 | PaymentDetailScreen | payment_detail_screen.dart | /payment-detail | Invoice detail view |
| 20 | MyTicketsScreen | my_tickets_screen.dart | /my-tickets | Active tournament tickets with QR codes |
| 21 | TicketDetailScreen | ticket_detail_screen.dart | /ticket-detail | Digital ticket with QR for check-in |
| 22 | MyTournamentsScreen | my_tournaments_screen.dart | /my-events | Tournaments archer is registered for |
| 23 | TournamentResultScreen | tournament_result_screen.dart | /event-result | Live qualification & elimination results |
| 24 | TournamentMyScoreScreen | tournament_my_score_screen.dart | /event-my-score | Personal scorecard per end |
| 25 | ArcherTournamentPerformanceScreen | archer_tournament_performance_screen.dart | /event-performance | Accuracy stats & performance analytics |
| 26 | SavedTournamentsScreen | saved_tournaments_screen.dart | /saved-events | Bookmarked/favorite tournaments |
| 27 | TournamentPdfPreviewScreen | tournament_pdf_preview_screen.dart | /event-pdf-preview | Technical handbook PDF viewer |
| 28 | ImagePreviewScreen | image_preview_screen.dart | /image-preview | Full-screen image viewer with zoom |
| 29 | WebViewScreen | webview_screen.dart | /webview | In-app web browser (payment gateway, etc.) |
| 30 | ProfileScreen | profile_screen.dart | /profile | User profile with stats & settings |
| 31 | EditProfileScreen | edit_profile_screen.dart | /profile/edit | Edit name, photo, club, contact info |
| 32 | ArcherInfoScreen | archer_info_screen.dart | /profile/archer-info | Technical archery data (bow specs, draw weight) |
| 33 | MyCertificatesScreen | my_certificates_screen.dart | /my-certificates | Digital certificates gallery |
| 34 | CertificatePreviewScreen | certificate_preview_screen.dart | /certificate-preview | Certificate viewer & download |
| 35 | NotificationScreen | notification_screen.dart | /notifications | Activity notifications center |
| 36 | SecurityScreen | security_screen.dart | /security | Password change & security settings |
| 37 | HelpSupportScreen | help_support_screen.dart | /help-support | FAQ & customer support |
| 38 | AboutAppScreen | about_app_screen.dart | /about-app | App version, credits, licenses |
| 39 | TermsConditionsScreen | terms_conditions_screen.dart | /terms-conditions | Terms of service & privacy policy |

#### 📰 News (2 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 40 | NewsScreen | news_screen.dart | /news | News articles listing |
| 41 | NewsDetailScreen | news_detail_screen.dart | /news/detail | Article detail with comments |

#### 🛒 Marketplace & Cart (9 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 42 | MarketplaceScreen | marketplace_screen.dart | /marketplace | Product browsing with categories |
| 43 | ShopDetailScreen | shop_detail_screen.dart | /shop/detail | Product detail page |
| 44 | ShopReviewOrderScreen | shop_review_order_screen.dart | /shop/review-order | Order review before checkout |
| 45 | ShopCommentListScreen | shop_comment_list_screen.dart | /shop/comments | Product reviews & ratings |
| 46 | OrderDetailScreen | order_detail_screen.dart | /order/detail | Order detail & tracking |
| 47 | OrderHistoryScreen | order_history_screen.dart | /order/history | Past orders list |
| 48 | CartScreen | cart_screen.dart | /cart | Shopping cart with item management |
| 49 | PaymentSuccessScreen | payment_success_screen.dart | /shop/payment-success | Shop order payment confirmation |

#### 💬 Chat & Support (3 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 50 | ChatListScreen | chat_list_screen.dart | /chat | Conversation list |
| 51 | ChatDetailScreen | chat_detail_screen.dart | /chat/detail | Individual chat thread |
| 52 | ChatbotScreen | chatbot_screen.dart | /chatbot | AI chatbot for tournament/archery queries |

#### 🎯 Scorekeeper (8 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 53 | ScorekeeperLandingScreen | scorekeeper_landing_screen.dart | /scorekeeper | Scorekeeper hub: target selection |
| 54 | ListScoreScreen | list_score_screen.dart | /list-score | List of archers on assigned target |
| 55 | TargetBarcodeScreen | target_barcode_screen.dart | /barcode | QR code scanner for target verification |
| 56 | ManualCodeInputScreen | manual_code_input_screen.dart | /manual-code-input | Manual ticket code entry (fallback) |
| 57 | ManualScoreEntryScreen | manual_score_entry_screen.dart | /manual-score | Per-end score input keypad (X,10,9..M) |
| 58 | TargetScoreEntryScreen | target_score_entry_screen.dart | /target-score | Quick scoring for all archers on target |
| 59 | DetailScoreScreen | detail_score_screen.dart | /detail-score | Complete scorecard with signature |
| 60 | ScorekeeperHistoryScreen | scorekeeper_history_screen.dart | /scorekeeper/history | Past scored sessions |

#### 🏢 Event Organizer (23 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 61 | OrganizationHomeScreen | organization_home_screen.dart | /organization | Organizer portal main hub |
| 62 | OrganizationEventSelectorScreen | organization_event_selector_screen.dart | /organization/event-selector | Tournament switcher |
| 63 | OrganizationEventShellScreen | organization_event_shell_screen.dart | /organization/event-shell | 3-tab shell: Dashboard/Check-in/Broadcast |
| 64 | OrganizationDashboardScreen | organization_dashboard_screen.dart | (Tab 0) | Event stats, attendance, activity log |
| 65 | OrganizationCheckinScreen | organization_checkin_screen.dart | (Tab 1) | QR ticket scanner for athlete check-in |
| 66 | OrganizationAnnouncementsScreen | organization_announcements_screen.dart | (Tab 2) | Broadcast composer (Push/Email/WhatsApp) |
| 67 | OrganizationBroadcastHistoryScreen | organization_broadcast_history_screen.dart | /organization/broadcast-history | Past broadcast messages |
| 68 | OrganizationBroadcastDetailScreen | organization_broadcast_detail_screen.dart | /organization/broadcast-detail | Broadcast message detail & read status |
| 69 | OrganizationEventDetailScreen | organization_event_detail_screen.dart | /organization/event-detail | Event settings & configuration |
| 70 | OrganizationTicketDetailScreen | organization_ticket_detail_screen.dart | /organization/ticket-detail | Participant ticket validation |
| 71 | OrganizationParticipantsScreen | organization_participants_screen.dart | /organization/participants | Full participant table with filters |
| 72 | OrganizationParticipantDetailScreen | organization_participant_detail_screen.dart | /organization/participants/detail | Individual participant profile |
| 73 | OrganizationPaymentListScreen | organization_payment_list_screen.dart | /organization/payments | Payment transactions list |
| 74 | OrganizationInvoiceDetailScreen | organization_invoice_detail_screen.dart | /organization/payments/invoice | Invoice detail & payment proof |
| 75 | OrganizationSettingsScreen | organization_settings_screen.dart | /organization/settings | Organizer preferences & scorekeepers |
| 76 | OrganizationWithdrawScreen | organization_withdraw_screen.dart | /organization/withdraw | Payout/withdrawal request form |
| 77 | OrganizationWithdrawStatusScreen | organization_withdraw_status_screen.dart | /organization/withdraw-status | Withdrawal status tracking |
| 78 | OrganizationTransactionHistoryScreen | organization_transaction_history_screen.dart | /organization/transaction-history | Financial ledger & mutations |
| 79 | OrganizationTransactionDetailScreen | organization_transaction_detail_screen.dart | /organization/transaction-detail | Transaction proof detail |
| 80 | OrganizationNotificationsScreen | organization_notifications_screen.dart | /organization/notifications | EO notification center |
| 81 | OrganizationEditProfileScreen | organization_edit_profile_screen.dart | /organization/profile/edit | Organization profile editor |
| 82 | OrganizationGlobalSearchScreen | organization_global_search_screen.dart | /organization/search | Search participants, tickets, transactions |
| 83 | OrganizationHelpScreen | organization_help_screen.dart | /organization/help | EO-specific help center |

#### 🏪 Seller/Marketplace Vendor (8 screens)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 84 | SellerMainScaffold | seller_shell_screen.dart | /seller | Seller hub with 4 bottom tabs |
| 85 | SellerHomeScreen | seller_home_screen.dart | /seller/home | Seller dashboard overview |
| 86 | SellerProductFormScreen | seller_product_form_screen.dart | /seller/product/form | Add/edit product listing |
| 87 | SellerDashboardScreen | seller_dashboard_screen.dart | /seller/dashboard | Sales analytics & metrics |
| 88 | SellerInventoryScreen | seller_inventory_screen.dart | /seller/inventory | Product inventory management |
| 89 | SellerOrdersScreen | seller_orders_screen.dart | /seller/orders | Incoming orders & fulfillment |
| 90 | SellerStoreProfileScreen | seller_store_profile_screen.dart | /seller/profile | Store branding & settings |
| 91 | SellerChatScreen | seller_chat_screen.dart | /seller/chat | Customer chat support |

#### 🛠️ Developer Tools (1 screen)

| # | Screen Name | File | Route | Description |
|---|-------------|------|-------|-------------|
| 92 | DesignSystemShowcasePage | desain_system_showcase_page.dart | /design-system | UI component catalog for developers |

### Total Screens: **92**

### Screen Statistics by Module

| Module | Screen Count |
|--------|:------------:|
| 🔐 Auth & Onboarding | 8 |
| 🏹 Archer/User Portal | 29 |
| 📰 News | 2 |
| 🛒 Marketplace & Cart | 9 |
| 💬 Chat & Support | 3 |
| 🎯 Scorekeeper | 8 |
| 🏢 Event Organizer | 23 |
| 🏪 Seller/Vendor | 8 |
| 🛠️ Developer Tools | 1 |
| **TOTAL** | **92** |

### API Integration Status

| Feature | API Status | Data Source | BLoC/State |
|---------|-----------|-------------|------------|
| **Auth** (login/register/forgot/reset/Google OAuth) | ✅ Complete | AuthRemoteDataSource | AuthBloc |
| **Events/Tournaments** (list/detail/register/payment) | ✅ Complete | EventsRemoteDataSource | EventsBloc, TicketBloc |
| **News/Blog** (list/detail/comments) | ✅ Complete | NewsRemoteDataSource | NewsBloc |
| **Shop/Products** (list/detail/categories) | ✅ Complete | ShopRemoteDataSource | ShopBloc |
| **Cart** (CRUD/checkout) | ✅ Complete | CartRemoteDataSource | CartBloc |
| **QR Scanning** (target/ticket scan) | ✅ Complete | ScoringRemoteDataSource | ScanBloc |
| **Scoring** (assignments/submit scores) | ✅ Complete | ScoringRemoteDataSource | ScoringBloc |
| **Chat/Chatbot** (conversations/messages) | ✅ Complete | ChatRemoteDataSource | ChatBloc, ChatbotBloc |
| **Organization** (dashboard/participants/finance) | ⚠️ Partial | OrgRemoteDataSource | Uses mock via AppConfig |
| **Seller** (products/inventory/orders) | ⚠️ Partial | SellerRemoteDataSource | Uses mock via AppConfig |
| **Profile** (view/edit) | ⚠️ Partial | Profile edit hits API, others UI only | — |
| **Notifications** | ❌ UI Only | No data source | — |
| **Certificates** | ❌ UI Only | No data source | — |
| **Order History** | ❌ UI Only | No data source | — |

**Note:** `AppConfig.useStaticData = true` (default) means repositories use mock data from MockData class. When set to `false`, all repositories switch to real API calls.

### Key Features

#### Multi-Role Authentication
- Archer login (email/password or Google OAuth)
- Organizer login (email/password)
- Scorekeeper login (5-character access code)
- Auto-redirect to role-specific hub after login

#### Tournament Discovery & Registration
- Browse tournaments with filters (status, type, location)
- View detailed event information with categories & schedules
- Multi-step registration flow (ticket confirmation → payment method → payment status)
- Digital ticket generation with QR code for check-in

#### Payment Integration
- Mayar payment gateway (QRIS, Virtual Accounts, E-Wallets, Cards)
- PayPal for international payments
- Manual bank transfer with proof upload
- Payment status tracking with countdown timer
- Invoice history & receipt viewing

#### Digital Scoring (Scorekeeper)
- QR code scanning for target verification
- Arrow-by-arrow score entry (X, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, M)
- Quick scoring for multiple archers on same target
- Digital scorecard with signature
- Real-time leaderboard updates

#### Event Organizer Portal
- Multi-tournament management (event switcher)
- Dashboard with real-time stats (participants, check-ins, revenue)
- QR ticket scanner for athlete check-in at venue
- Participant management (search, filter, detail view)
- Broadcast messaging (push notifications, email, WhatsApp)
- Financial tracking (revenue, withdrawals, transaction history)
- Payment proof verification & approval

#### Marketplace
- Product browsing by category
- Product detail with images & reviews
- Shopping cart with quantity management
- Order checkout flow
- Order history & tracking
- Seller dashboard with inventory & order management

#### Profile & Certificates
- Archer profile with statistics & tournament history
- Digital certificate gallery with QR verification
- Profile photo & banner upload
- Security settings (password change)
- Help & support center

### Custom UI Components

#### DsImage Widget
Premium image component with:
- **Loading State:** CircularProgressIndicator while downloading
- **Error Fallback:** Gray container with icon if image fails to load
- **Empty URL Handling:** Automatic placeholder for missing URLs
- **URL Sanitization:** Auto-replaces localhost/127.0.0.1 with host IP

Used throughout the app to ensure consistent image handling and prevent broken image errors.

### Build & Release

#### Development
```bash
flutter pub get
flutter run
```

#### Production Build
```bash
flutter build apk --release --split-per-abi --obfuscate --split-debug-info=build/app/outputs/symbols
```

#### CI/CD
- **Platform:** GitHub Actions
- **Workflow:** Automated APK build on version tag push
- **Release:** Automatic GitHub Releases with changelog
- **Artifacts:** Split APK per ABI (arm64-v8a, armeabi-v7a, x86_64)
- **Retention:** 90 days

### Release Workflow
1. Update version in pubspec.yaml
2. Commit and create git tag (e.g., `v1.0.3`)
3. Push tag to GitHub: `git push origin main --tags`
4. GitHub Actions automatically builds and releases APK

### Configuration

#### App Config
Located in `lib/core/constants/api_constants.dart`:
- **API Base URL:** `https://api.archeris.net`
- **Static Data Mode:** `AppConfig.useStaticData = true` (toggles mock vs real API)
- **Image URL Sanitization:** Auto-fixes localhost URLs to host IP

#### Routing
GoRouter configuration with:
- Nested ShellRoutes for multi-tab navigation
- Role-based route guards
- Deep linking support
- Named routes for type-safe navigation

#### State Management
- BLoC for business logic & state
- GetIt for dependency injection
- Equatable for value comparison
- Stream-based reactive updates

### Testing
```bash
flutter test
flutter analyze
```

Test accounts and API endpoints: See `docs/API_REFERENCE.md` in project root.

### Distribution
- **Google Play Store:** Published
- **Direct APK Download:** Available from website
- **Minimum Android SDK:** 21 (Android 5.0 Lollipop)

### Future Development
Planned features (based on UI-only screens):
- Full certificate API integration
- Complete notification system with push
- Enhanced order tracking with real-time updates
- Seller analytics dashboard with API
- Organization financial reports with API



---

## Detailed Flow Documentation

### 1. Tournament Creation Flow

#### API Endpoint
```
POST /api/v1/tournaments
Authorization: Bearer token (Organizer/Club only)
```

#### Prerequisites & Access Control
- **User Role Check**: Only `organizer` or `club` roles can create tournaments
- **Archers are rejected** with error code `only_organizer_allowed`
- **Subscription Check**: Organizer's subscription status must be `active`

#### Auto-Generated Fields
1. **Tournament Code**: Auto-generated as `EVT-XXXX` (e.g., `EVT-0001`, `EVT-0002`)
   - System queries last code: `SELECT code FROM tournaments WHERE code LIKE 'EVT-%' ORDER BY code DESC LIMIT 1`
   - Increments number sequentially
   - Can be manually provided in request

2. **Slug Generation**:
   - Uses user-provided `slug` field OR falls back to `name` field
   - Sanitization: lowercase, spaces→dashes, only `a-z`, `0-9`, `-` allowed
   - **Uniqueness**: Appends numeric suffix if slug exists (e.g., `event`, `event-2`, `event-3`)
   - No random strings - deterministic suffixes only

3. **UUID**: Generated via `uuid.New().String()`

#### Quota System (Per-Tournament Tier)
The platform has **3 tournament tiers** with different quota types:

| Tier | quota_type | Participant Limit | Storage Limit | Price (IDR/USD) | Notes |
|------|-----------|-------------------|---------------|-----------------|-------|
| **Free Starter** | `free` | 50 | 200 MB | Rp 0 / $0 | 20 free quotas per organizer, lifetime |
| **Standard EO** | `standard` | 200 | 3 GB | Rp 49,999 / $3.00 | Promo: 50% off (Rp 24,999 / $1.50) |
| **Elite EO** | `elite` | Unlimited | 10 GB | Rp 79,999 / $7.00 | Promo: 50% off (Rp 39,999 / $3.50) |

#### Quota Deduction Logic
**Tournament quota is deducted at CREATION time**, NOT at publish time:

```go
// Standard Quota Check
SELECT quota_standard FROM organizers WHERE uuid = ? FOR UPDATE
UPDATE organizers SET quota_standard = quota_standard - 1 WHERE uuid = ? AND quota_standard > 0

// Elite Quota Check  
SELECT quota_elite FROM organizers WHERE uuid = ? FOR UPDATE
UPDATE organizers SET quota_elite = quota_elite - 1 WHERE uuid = ? AND quota_elite > 0
```

**Error Response if Insufficient**:
```json
{
  "error": "Quota Standard tidak mencukupi. Silakan beli kuota terlebih dahulu.",
  "code": "quota_insufficient"
}
```

#### Date Fields (Nullable)
- `start_date`: Can be NULL if not set
- `end_date`: Can be NULL if not set
- `registration_deadline`: Can be NULL (open registration)

#### Transaction Safety
- Entire creation wrapped in SQL transaction (`db.Beginx()`)
- Auto-rollback on any error (`defer tx.Rollback()`)
- Commits only if all steps succeed

#### Response
```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "code": "EVT-0042",
  "slug": "national-championship-2026",
  "name": "National Championship 2026",
  "quota_type": "standard",
  "status": "draft",
  "created_at": "2026-09-23T10:30:00Z"
}
```

#### Status Lifecycle
1. **draft** (initial) → Organizer can edit everything
2. **published** → Public, participants can register
3. **ongoing** → Event has started
4. **completed** → Event finished, results finalized

---

### 2. Tournament Registration Flow

#### API Endpoint
```
POST /api/v1/tournaments/:id/participants
Authorization: Bearer token OR organizer privilege
```

#### Registration Modes
The system supports **3 distinct registration modes**:

| Mode | Code | Who Can Use | Use Case |
|------|------|-------------|----------|
| **Individual/Captain** | `captain_team` | Archer themselves | Self-register + optionally add teammates |
| **Club Delegation** | `club_delegation` | Archer/Organizer | Mass registration under single invoice |
| **Organizer Manual** | `organizer_added` | Organizer only | Add participant directly (no self-register) |

#### Gate Checking System

##### Gate A: Organizer Subscription Status Check
```go
SELECT COALESCE(subscription_status, 'active') 
FROM organizers 
WHERE uuid = ?
```
**If NOT `active`:**
```json
{
  "error": "Pendaftaran ditutup sementara",
  "code": "organizer_subscription_expired",
  "message": "Pendaftaran peserta untuk event ini ditutup sementara oleh sistem karena masa berlaku layanan penyelenggara telah berakhir."
}
```

##### Gate B: Registration Deadline Check
```go
if !isPrivileged && event.RegistrationDeadline != nil && time.Now().After(*event.RegistrationDeadline)
```
**If deadline passed:**
```json
{
  "error": "Pendaftaran untuk turnamen ini telah ditutup",
  "code": "registration_closed"
}
```

**Bypass**: Admins and tournament organizers can bypass this (`isPrivileged = true`)

##### Gate C: Overall Event Quota Check
```go
SELECT COUNT(*) FROM tournament_participants 
WHERE tournament_id = ? AND payment_status != 'cancelled'
```
**If `current_total >= quota_max_participants`:**
```json
{
  "error": "Kuota keseluruhan turnamen ini telah penuh",
  "code": "event_quota_exceeded"
}
```

#### Category-Level Validation

##### Eligibility Validation
**Gender Check**:
```go
func validateEligibility(catMeta *CategoryMeta, archerGender string, archerDOB *time.Time) (error, code) {
  catGender := catMeta.GenderCode // "men" | "women" | "mixed"
  
  if catGender == "men" && archerGender != "male":
    return "Atlet perempuan tidak dapat mendaftar di kategori putra (Men)", "gender_ineligible"
  
  if catGender == "women" && archerGender != "male":
    return "Atlet laki-laki tidak dapat mendaftar di kategori putri (Women)", "gender_ineligible"
}
```

**Age Validation**: Currently not enforced (future feature based on `date_of_birth` vs `age_code`)

##### Quota Check Per Category
```go
SELECT COUNT(*) FROM tournament_participants 
WHERE category_id = ? AND payment_status != 'cancelled'
```
**Bypass**: Organizers and admins can exceed category quotas

#### Registration Modes in Detail

##### Mode 1: Individual/Captain Registration (`captain_team`)
**Request Structure**:
```json
{
  "athlete_id": "archer-uuid-or-username",
  "registration_mode": "captain_team",
  "event_category_ids": ["cat-uuid-1", "cat-uuid-2"],
  "team_registration": {
    "team_category_id": "team-cat-uuid",
    "partner_id": "partner-uuid",
    "team_name": "Team Awesome"
  }
}
```

**Flow**:
1. Fetch captain archer data: `SELECT uuid, gender, date_of_birth FROM archers WHERE uuid = ?`
2. For each **individual category**:
   - Validate eligibility (gender, age)
   - Check category quota
   - Calculate fee from category `entry_fee`
   - Create `tournament_participants` record
3. For **team categories**:
   - Fetch partner archer data
   - Validate BOTH captain and partner eligibility
   - Check team quota
   - Calculate fee = captain_fee + partner_fee
   - Create 2 participant records (captain + partner) linked by `team_uuid`
   - Create `teams` record with `team_name`

##### Mode 2: Club Delegation (`club_delegation`)
**Request Structure**:
```json
{
  "registration_mode": "club_delegation",
  "delegation_athletes": [
    {
      "athlete_id": "archer-1-uuid",
      "category_ids": ["cat-1", "cat-2"]
    },
    {
      "athlete_id": "archer-2-uuid",
      "category_ids": ["cat-3"]
    }
  ],
  "delegation_teams": [
    {
      "team_category_id": "team-cat-uuid",
      "team_name": "Club Team Alpha",
      "member_ids": ["archer-1-uuid", "archer-2-uuid", "archer-3-uuid"]
    }
  ]
}
```

**Flow**:
1. Loop through each athlete in `delegation_athletes`
2. For each athlete:
   - Fetch archer data
   - Loop through their `category_ids`
   - Validate eligibility per category
   - Check quota
   - Create participant record
3. Loop through `delegation_teams`
4. For each team:
   - Validate all team members
   - Create team record
   - Create participant records for each member
5. **Single invoice** for entire delegation

##### Mode 3: Organizer Manual Add
**Auto-detected when**:
- Request comes from organizer/admin (`isPrivileged = true`)
- `registration_source` defaults to `"organizer_added"`
- Can set `payment_status` directly (e.g., `"paid"` for free entry)

#### Payment Status Assignment
```go
paymentStatus := "unpaid"  // Default

// Organizers can override
if req.PaymentStatus != "" && isPrivileged {
  paymentStatus = req.PaymentStatus  // "paid", "free", "unpaid"
}
```

#### Response
```json
{
  "success": true,
  "participants": [
    {
      "uuid": "participant-uuid-1",
      "athlete_id": "archer-uuid",
      "category_id": "cat-uuid",
      "payment_amount": 150000,
      "payment_status": "unpaid",
      "registration_source": "self_register"
    }
  ],
  "teams": [
    {
      "uuid": "team-uuid",
      "team_name": "Dream Team",
      "members": ["archer-1", "archer-2"]
    }
  ],
  "total_amount": 450000
}
```

---

### 3. Tournament Payment Flow

#### API Endpoint
```
POST /api/v1/payment/create
Authorization: Bearer token
```

#### Payment Types
| Type | When | Amount Source |
|------|------|---------------|
| `registration` | Participant registration | Sum of all `participant.payment_amount` |
| `subscription` | Organizer buys package | `subscription_plan.price * months` |
| `platform_fee` | Organizer publishes event | Fixed Rp 50,000 (env: `PLATFORM_FEE_AMOUNT`) |

#### Currency-Gateway Matching (Gate Check)
**Query event currency**:
```go
SELECT page_settings FROM tournaments WHERE uuid = ?
// Parse JSON: {"currency": "IDR" | "USD"}
```

**Rules**:
- **IDR events** → Cannot use PayPal
  ```json
  {
    "error": "Turnamen dengan mata uang IDR tidak mendukung pembayaran via PayPal",
    "code": "gateway_currency_mismatch"
  }
  ```

- **USD events** → MUST use PayPal
  ```json
  {
    "error": "Turnamen internasional dengan mata uang USD hanya mendukung pembayaran via PayPal",
    "code": "gateway_currency_mismatch"
  }
  ```

#### Payment Methods
| Method Code | Gateway | Supported Channels |
|-------------|---------|-------------------|
| `mayar` | Mayar | QRIS, Virtual Account (BCA, BNI, BRI, Mandiri, BSI, CIMB, Danamon, Permata), E-Wallet (GoPay, OVO, Dana, ShopeePay), Cards (Visa, Mastercard) |
| `paypal` | PayPal | International credit/debit cards |
| `manual` | Manual Transfer | Bank transfer with proof upload |

#### Payment Transaction Creation

##### Step 1: Validate No Pending Payment
```go
// For platform_fee
SELECT COUNT(*) FROM payment_transactions 
WHERE tournament_id = ? AND subscription_plan_id IS NULL 
  AND registration_id IS NULL AND status = 'pending' 
  AND expired_at > NOW()

// For subscription
SELECT COUNT(*) FROM payment_transactions 
WHERE user_id = ? AND subscription_plan_id IS NOT NULL 
  AND status = 'pending' AND expired_at > NOW()
```

**If exists:**
```json
{
  "error": "Anda memiliki pembayaran yang masih tertunda. Silakan selesaikan pembayaran tersebut atau tunggu hingga kedaluwarsa.",
  "code": "pending_subscription_exists"
}
```

##### Step 2: Calculate Total Amount
**For Registration**:
```go
SELECT payment_amount FROM tournament_participants WHERE uuid IN (?)
// Sum all amounts
```

**For Subscription**:
```go
SELECT price FROM subscription_plans WHERE id = ?
totalPrice = price * months  // months: 1-12
```

##### Step 3: Generate Reference Number
```go
// Pattern: TXN-YYYYMMDD-RANDOM6
reference := fmt.Sprintf("TXN-%s-%s", time.Now().Format("20060102"), randomString(6))
```

##### Step 4: Create Payment Transaction Record
```sql
INSERT INTO payment_transactions (
  uuid, user_id, tournament_id, subscription_plan_id, registration_id,
  amount, payment_method, status, reference, expired_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?, DATE_ADD(NOW(), INTERVAL 24 HOUR), NOW())
```

**Expiry**: Default 24 hours from creation

##### Step 5: Gateway-Specific Processing

###### Mayar Gateway
```go
// Call Mayar API
POST https://api.mayar.id/v1/payments
{
  "amount": 150000,
  "currency": "IDR",
  "channel": "qris", // or "va_bca", "gopay", etc.
  "reference": "TXN-20260923-ABC123",
  "customer": {
    "name": "John Archer",
    "email": "john@example.com",
    "phone": "08123456789"
  },
  "redirect_url": "https://archeris.net/payment/status/TXN-20260923-ABC123",
  "webhook_url": "https://api.archeris.net/api/v1/payment/mayar/webhook"
}

// Response
{
  "checkout_url": "https://checkout.mayar.id/pay/xyz",
  "qr_string": "00020101...",  // For QRIS
  "va_number": "880812345678"  // For VA
}
```

**Update transaction**:
```sql
UPDATE payment_transactions 
SET checkout_url = ?, instructions = ? 
WHERE uuid = ?
```

###### PayPal Gateway
```go
// Create PayPal Order
POST https://api.paypal.com/v2/checkout/orders
{
  "intent": "CAPTURE",
  "purchase_units": [{
    "amount": {
      "currency_code": "USD",
      "value": "3.50"
    },
    "reference_id": "TXN-20260923-ABC123"
  }],
  "application_context": {
    "return_url": "https://archeris.net/payment/status/TXN-20260923-ABC123",
    "cancel_url": "https://archeris.net/payment/failed/TXN-20260923-ABC123"
  }
}

// Response
{
  "id": "PAYPAL-ORDER-ID",
  "links": [{
    "rel": "approve",
    "href": "https://www.paypal.com/checkoutnow?token=..."
  }]
}
```

###### Manual Transfer
- No external API call
- Returns bank account details in `instructions` JSON
- User uploads proof later via `/payment/manual/:reference/upload-proof`

##### Step 6: Update Participant Payment Status Link
```sql
UPDATE tournament_participants 
SET payment_transaction_id = ? 
WHERE uuid IN (?)
```

#### Webhook Handling (Mayar)
```
POST /api/v1/payment/mayar/webhook
X-Signature: mayar-signature-hmac
```

**Verification**:
```go
expectedSignature := hmac.New(sha256.New, []byte(MAYAR_SECRET_KEY))
expectedSignature.Write([]byte(requestBody))
if hmac.Equal(expectedSignature.Sum(nil), receivedSignature) {
  // Process webhook
}
```

**Status Mapping**:
| Mayar Status | Our Status | Action |
|--------------|-----------|--------|
| `settlement` | `paid` | Mark participants as `paid`, trigger certificate generation |
| `expired` | `expired` | Mark transaction expired |
| `failed` | `failed` | Allow retry |
| `cancelled` | `cancelled` | Release quotas |

**Payment Confirmation Actions**:
1. Update `payment_transactions.status = 'paid'`
2. Update all linked participants: `tournament_participants.payment_status = 'paid'`
3. For subscription payments: Update `organizers.quota_standard += qty` or `quota_elite += qty`
4. Send email confirmation (async)
5. Generate digital ticket QR code

#### Manual Payment Verification (Organizer)
```
POST /api/v1/payment/manual/:reference/verify
Authorization: Bearer token (Organizer)
```

**Request**:
```json
{
  "action": "approve",  // or "reject"
  "notes": "Payment verified via BCA transfer"
}
```

**If approved**: Same actions as webhook settlement
**If rejected**: Participant can re-upload proof

---

### 4. Organizer Package/Subscription System

#### Package Tiers (Unified Pricing)
**Source of Truth**: `handler/subscription.go` → `buildUnifiedPricing()`

| ID | Tier Key | Name | Badge | Price IDR (Promo) | Price USD (Promo) | Discount |
|----|----------|------|-------|-------------------|-------------------|----------|
| 0 | `free` | Free Starter | Starter | Rp 0 | $0 | - |
| 7 | `standard` | Standard EO | Most Popular | Rp 49,999 (Rp 24,999) | $3.00 ($1.50) | 50% |
| 8 | `elite` | Elite EO | Professional Tier | Rp 79,999 (Rp 39,999) | $7.00 ($3.50) | 50% |

#### Feature Comparison Matrix

| Feature | Free | Standard | Elite |
|---------|------|----------|-------|
| **Participant Limit** | 50 | 200 | Unlimited |
| **Media Storage** | 200 MB | 3 GB | 10 GB |
| **Competition Categories** | Unlimited | Unlimited | Unlimited |
| **Scorekeeper Accounts** | Unlimited | Unlimited | Unlimited |
| **Access Period** | Lifetime | Lifetime | Lifetime |
| **Tournament Setup** | ✅ | ✅ | ✅ |
| **Online Registration** | ✅ | ✅ | ✅ |
| **Scoring & Match Play** | ✅ | ✅ | ✅ |
| **Printouts & Certificates** | ✅ | ✅ | ✅ |
| **Data Export (CSV/Excel)** | ✅ | ✅ | ✅ |
| **Customer Support** | 24/7 | 24/7 | 24/7 |

#### Bundle Discount Rules
**Apply to Standard & Elite purchases**:

| Quantity | Discount | Label |
|----------|----------|-------|
| 1 | 0% | Single Package |
| 3-4 | 7% | Save 7% |
| 5-9 | 12% | Save 12% |
| 10+ | 20% | Save 20% |

**Example**:
- Buy 5 Standard EO: `5 × Rp 24,999 × (1 - 0.12) = Rp 109,995`
- Buy 10 Elite EO: `10 × Rp 39,999 × (1 - 0.20) = Rp 319,992`

#### Quota Storage (Organizer Account)
```sql
SELECT quota_free, quota_standard, quota_elite, subscription_status 
FROM organizers 
WHERE user_id = ?
```

**Fields**:
- `quota_free`: Integer (starts at 20, never expires, non-refundable)
- `quota_standard`: Integer (purchased quota count)
- `quota_elite`: Integer (purchased quota count)
- `subscription_status`: `'active'` | `'expired'` | `'suspended'`

#### Purchasing Flow
1. **Browse Packages**: `GET /api/v1/subscription/pricing`
2. **Create Payment**: `POST /api/v1/payment/create` with `type: "subscription"`, `plan_id: 7`, `months: 3`
3. **Payment Gateway**: Redirect to Mayar/PayPal checkout
4. **Webhook Callback**: On success, increment quota
5. **Quota Update**:
   ```sql
   UPDATE organizers 
   SET quota_standard = quota_standard + 3 
   WHERE user_id = ?
   ```

#### Quota Consumption
**Consumed when**:
- Creating a tournament (see Tournament Creation Flow above)

**NOT consumed when**:
- Saving tournament as draft
- Editing tournament settings
- Deleting tournament (quota is NOT refunded)

#### Auto-Expiry (Not Implemented)
**Note**: Current system has lifetime quotas. No automatic expiration logic in codebase.

---

### 5. Scoring Rules & Systems

#### A. Scorekeeper Role & Access

##### Scorekeeper Account Creation
```
POST /api/v1/organizers/me/scorekeepers
Authorization: Bearer token (Organizer)
```

**Request**:
```json
{
  "name": "John Scorekeeper",
  "email": "scorekeeper@example.com"
}
```

**Auto-Generated**:
- **Access Code**: 5-character alphanumeric (e.g., `A7B2X`)
- Unique per scorekeeper
- Used for mobile app login

**Response**:
```json
{
  "uuid": "scorekeeper-uuid",
  "name": "John Scorekeeper",
  "code": "A7B2X",
  "status": "active"
}
```

##### Scorekeeper Login (Mobile App)
```
POST /api/v1/mobile/auth/scorekeeper/login
```

**Request**:
```json
{
  "code": "A7B2X"
}
```

**Returns**: JWT token with scorekeeper role

---

#### B. Qualification Scoring System

##### Qualification Session Structure
**Each tournament can have multiple qualification sessions**:
- Session Name (e.g., "Morning Session Day 1")
- `total_ends`: Default 12 (World Archery outdoor standard)
- `arrows_per_end`: Default 6
- `is_locked`: Boolean (prevents score changes when true)

##### Target Board & Assignment
**Target Board Qualification**:
```sql
CREATE TABLE target_board_qualification (
  uuid VARCHAR(36) PRIMARY KEY,
  session_uuid VARCHAR(36),
  category_uuid VARCHAR(36),
  board_number INT,
  code VARCHAR(10) UNIQUE  -- e.g., "001JFR", "042JFR"
)
```

**Target Assignment**:
```sql
CREATE TABLE qualification_target_assignments (
  uuid VARCHAR(36) PRIMARY KEY,
  session_uuid VARCHAR(36),
  participant_uuid VARCHAR(36),
  target_uuid VARCHAR(36),
  target_board_id VARCHAR(36),
  position VARCHAR(2)  -- "A", "B", "C", "D" (World Archery 4 archers per target)
)
```

**Positions**:
- Each target (e.g., `5`) has 4 board positions: `5A`, `5B`, `5C`, `5D`
- Stored in `tournament_targets.target_name`

##### Scoring Input (Mobile App)

###### Scan Target Board
```
GET /api/v1/mobile/scan?code=001JFR
Authorization: Bearer token (Scorekeeper)
```

**Response**:
```json
{
  "type": "qualification",
  "board": {
    "uuid": "board-uuid",
    "board_number": 1,
    "code": "001JFR",
    "session_uuid": "session-uuid",
    "session_name": "Morning Session"
  },
  "archers": [
    {
      "assignment_uuid": "assign-uuid",
      "participant_uuid": "part-uuid",
      "position": "A",
      "target_name": "1A",
      "name": "John Doe",
      "avatar_url": "...",
      "division": "Recurve Men",
      "current_score": 320,
      "ends_completed": 8,
      "total_ends": 12
    }
  ]
}
```

###### Submit Scores
```
PUT /api/v1/qualification/scoring/scores/:assignmentId
Authorization: Bearer token (Scorekeeper)
```

**Request (Batch Mode)**:
```json
{
  "ends": [
    {
      "end_number": 1,
      "arrows": ["X", "10", "9", "9", "8", "7"]
    },
    {
      "end_number": 2,
      "arrows": ["10", "10", "9", "8", "8", "M"]
    }
  ]
}
```

**Request (Single End Mode)**:
```json
{
  "end_number": 3,
  "arrows": ["X", "X", "10", "9", "9", "8"]
}
```

##### Arrow Value Calculation
```go
func calculateArrowValue(arrow string) (val int, x int, ten int) {
  arrow = strings.ToUpper(strings.TrimSpace(arrow))
  
  switch arrow {
  case "X":
    return 10, 1, 1  // X counts as 10 + X + 10 for tiebreak
  case "10":
    return 10, 0, 1
  case "M", "":
    return 0, 0, 0   // Miss
  default:
    v, _ := strconv.Atoi(arrow)  // "9", "8", "7"..."1"
    return v, 0, 0
  }
}
```

**Allowed Values**: `X`, `10`, `9`, `8`, `7`, `6`, `5`, `4`, `3`, `2`, `1`, `M` (miss)

##### Validation Rules
1. **End Number**: Must be 1 to `total_ends` (default 12)
2. **Arrows Per End**: Max `arrows_per_end` (default 6)
3. **Locked Session**: Cannot submit scores if `is_locked = true`
   ```json
   {
     "error": "Qualification session is locked. Score changes are not permitted."
   }
   ```

##### Score Storage Structure
```sql
-- Per-end totals
CREATE TABLE qualification_end_scores (
  uuid VARCHAR(36) PRIMARY KEY,
  session_uuid VARCHAR(36),
  participant_uuid VARCHAR(36),
  end_number INT,
  total_score_end INT,   -- Sum of 6 arrows
  x_count_end INT,       -- Count of X's
  ten_count_end INT      -- Count of 10's (including X's)
)

-- Individual arrows (for detailed scorecard)
CREATE TABLE qualification_arrow_scores (
  uuid VARCHAR(36) PRIMARY KEY,
  end_score_uuid VARCHAR(36),
  arrow_number INT,      -- 1-6
  value INT,             -- 0-10
  is_x TINYINT(1)        -- 1 if X, 0 otherwise
)
```

##### Leaderboard Calculation
```
GET /api/v1/tournaments/:id/qualification/leaderboard
```

**Query**:
```sql
SELECT 
  ep.uuid, a.full_name, a.avatar_url,
  ec.category_name_custom,
  SUM(qes.total_score_end) as total_score,
  SUM(qes.x_count_end) as total_x,
  SUM(qes.ten_count_end) as total_ten,
  COUNT(DISTINCT qes.end_number) as ends_completed,
  qs.total_ends
FROM tournament_participants ep
LEFT JOIN qualification_end_scores qes 
  ON qes.participant_uuid = ep.uuid AND qes.session_uuid = ?
GROUP BY ep.uuid
ORDER BY total_score DESC, total_x DESC, total_ten DESC
```

**Tie-Breaking Order** (World Archery standard):
1. Total Score (highest wins)
2. Total X count (most X's wins)
3. Total 10 count (most 10's wins)
4. If still tied: Shoot-off (not automated)

---

#### C. Elimination Bracket System

##### Bracket Creation
```
POST /api/v1/tournaments/:id/elimination/brackets
Authorization: Bearer token (Organizer)
```

**Request**:
```json
{
  "name": "Men Recurve Elimination",
  "category_id": "cat-uuid",
  "bracket_size": 16,
  "format": "set_system",  // or "cumulative"
  "match_type": "individual"  // or "team", "mixed_team"
}
```

**Bracket Sizes** (Power of 2):
- 4, 8, 16, 32, 64, 128

**Formats**:
1. **Set System** (World Archery Olympic):
   - Best of 5 sets
   - Each set: 3 arrows per archer
   - Win set: 2 points, Draw: 1 point each
   - First to 6 points wins

2. **Cumulative** (Total score):
   - Fixed number of ends (e.g., 5 ends × 3 arrows)
   - Highest total score wins

##### Bracket Generation (Seeding)
```
POST /api/v1/tournaments/:id/elimination/brackets/:bracketId/generate
```

**Flow**:
1. Fetch top N qualifiers: `SELECT * FROM qualification_leaderboard ORDER BY total_score DESC LIMIT :bracket_size`
2. Apply World Archery seeding pattern
3. Create matches for Round 1

**Seeding Example (16-person bracket)**:
```
Match 1: Seed 1 vs Seed 16
Match 2: Seed 8 vs Seed 9
Match 3: Seed 5 vs Seed 12
Match 4: Seed 4 vs Seed 13
Match 5: Seed 3 vs Seed 14
Match 6: Seed 6 vs Seed 11
Match 7: Seed 7 vs Seed 10
Match 8: Seed 2 vs Seed 15
```

**Round Structure**:
- 32-person: 1/32, 1/16, 1/8, 1/4, Semi, Final
- 16-person: 1/16, 1/8, 1/4, Semi, Final
- 8-person: 1/8, 1/4, Semi, Final

##### Match Structure
```sql
CREATE TABLE elimination_matches (
  uuid VARCHAR(36) PRIMARY KEY,
  bracket_uuid VARCHAR(36),
  round_no INT,          -- 1, 2, 3, 4... (1=earliest, higher=closer to final)
  match_no INT,          -- Within round
  match_id VARCHAR(10),  -- e.g., "1/8-1", "1/4-2", "SF-1", "F"
  entry_a_uuid VARCHAR(36),  -- Link to elimination_entries
  entry_b_uuid VARCHAR(36),
  target_uuid VARCHAR(36),
  board_uuid VARCHAR(36),
  status VARCHAR(20),    -- "pending", "ongoing", "completed"
  winner_side VARCHAR(1), -- "A" or "B"
  
  -- Set System scoring
  sets_a INT,            -- Number of sets won by A
  sets_b INT,            -- Number of sets won by B
  total_points_a INT,    -- Total set points (max 10)
  total_points_b INT,
  
  -- Cumulative scoring
  total_score_a INT,     -- Sum of all arrows
  total_score_b INT,
  
  next_match_uuid VARCHAR(36)  -- Winner advances to this match
)
```

##### Elimination Scoring (Mobile App)

###### Scan Elimination Board
```
GET /api/v1/mobile/scan?code=E001JFR
```

**Response**:
```json
{
  "type": "elimination",
  "board": {
    "uuid": "board-uuid",
    "board_number": 1,
    "code": "E001JFR",
    "bracket_uuid": "bracket-uuid",
    "category_name": "Men Recurve"
  },
  "archers": [
    {
      "match_uuid": "match-uuid",
      "match_id": "1/8-1",
      "side": "A",
      "target_name": "1A",
      "name": "John Doe",
      "club": "Archery Club",
      "score": 4,
      "status": "ongoing"
    },
    {
      "match_uuid": "match-uuid",
      "match_id": "1/8-1",
      "side": "B",
      "target_name": "1B",
      "name": "Jane Smith",
      "club": "Elite Archers",
      "score": 2,
      "status": "ongoing"
    }
  ]
}
```

###### Submit Match Score
```
POST /api/v1/elimination/matches/:matchId/score
Authorization: Bearer token (Scorekeeper)
```

**Set System Format**:
```json
{
  "sets": [
    {
      "set_number": 1,
      "arrows_a": ["X", "10", "9"],
      "arrows_b": ["10", "9", "8"]
    },
    {
      "set_number": 2,
      "arrows_a": ["10", "10", "9"],
      "arrows_b": ["X", "10", "10"]
    }
  ]
}
```

**Set Winner Calculation**:
```go
totalA := sum(arrows_a)
totalB := sum(arrows_b)

if totalA > totalB {
  points_a += 2
} else if totalB > totalA {
  points_b += 2
} else {
  points_a += 1
  points_b += 1
}
```

**Match Winner**:
- First to **6 set points** wins
- If 5-5: Shoot-off (1 arrow closest to center)

##### Shoot-Off Handling
```json
{
  "shoot_off": {
    "round": 1,
    "arrow_a": {
      "score": 10,
      "distance_mm": 45  // Distance from center in mm
    },
    "arrow_b": {
      "score": 10,
      "distance_mm": 52
    }
  }
}
```

**Winner**: Closest to center (smallest `distance_mm`)

##### Match Completion & Advancement
```
POST /api/v1/elimination/matches/:matchId/finish
```

**Actions**:
1. Set `winner_side` = "A" or "B"
2. Set `status` = "completed"
3. Find `next_match_uuid`
4. Update next match:
   ```sql
   UPDATE elimination_matches 
   SET entry_a_uuid = :winner_entry_uuid
   WHERE uuid = :next_match_uuid AND entry_a_uuid IS NULL
   
   -- OR if entry_a already filled:
   UPDATE elimination_matches 
   SET entry_b_uuid = :winner_entry_uuid
   WHERE uuid = :next_match_uuid
   ```
5. If next match is FINAL and completed → Award medals

##### Medal Assignment
**Automatic on final completion**:
```sql
UPDATE elimination_entries 
SET medal = 'gold' 
WHERE uuid = :winner_entry_uuid

UPDATE elimination_entries 
SET medal = 'silver' 
WHERE uuid = :loser_entry_uuid

-- Bronze (both semi-final losers)
UPDATE elimination_entries 
SET medal = 'bronze' 
WHERE match_uuid IN (
  SELECT uuid FROM elimination_matches 
  WHERE next_match_uuid = :final_match_uuid
) AND winner = false
```

---

### 6. Gate Checking Summary (All Critical Gates)

#### Tournament Creation Gates
✅ **G1.1**: User role must be `organizer` or `club` (NOT `archer`)  
✅ **G1.2**: Quota check (`quota_free` OR `quota_standard` OR `quota_elite` > 0)  
✅ **G1.3**: Slug uniqueness check (auto-append `-2`, `-3` if duplicate)  
✅ **G1.4**: Transaction rollback on any error  

#### Tournament Registration Gates
✅ **G2.1**: Organizer subscription status = `active`  
✅ **G2.2**: Registration deadline not passed (bypass for organizers)  
✅ **G2.3**: Event overall quota not exceeded  
✅ **G2.4**: Category quota not exceeded (per category)  
✅ **G2.5**: Gender eligibility (`men` → male only, `women` → female only)  
✅ **G2.6**: No duplicate registration (same archer + category)  

#### Payment Gates
✅ **G3.1**: Currency-gateway matching (IDR ≠ PayPal, USD = PayPal only)  
✅ **G3.2**: No pending payment for same type (subscription/platform_fee)  
✅ **G3.3**: Payment expiry (24 hours default)  
✅ **G3.4**: Webhook signature verification (HMAC SHA256)  

#### Scoring Gates
✅ **G4.1**: Scorekeeper must have valid access code  
✅ **G4.2**: Session must NOT be locked (`is_locked = false`)  
✅ **G4.3**: End number within valid range (1 to `total_ends`)  
✅ **G4.4**: Arrow count ≤ `arrows_per_end`  
✅ **G4.5**: Arrow values must be valid (X, 10, 9...1, M)  
✅ **G4.6**: Match must be `ongoing` (not `completed`)  

---

### 7. Board Code System

#### Qualification Board Codes
**Format**: `{board_number}JFR` with zero-padding
- Board 1 → `001JFR`
- Board 42 → `042JFR`
- Board 128 → `128JFR`

**Alternative Scan Formats**:
- Raw number: `1`, `42`
- With position: `1A`, `42C`
- Full code: `001JFR`

**Generation**:
```go
code := fmt.Sprintf("%03dJFR", board_number)
```

#### Elimination Board Codes
**Format**: `E{board_number}JFR`
- Board 1 → `E001JFR`
- Board 10 → `E010JFR`

**Purpose**: Distinguish elimination boards from qualification boards when scanning

#### QR Code Generation
**Mobile App Display**:
- Archer ticket: Contains `participant_uuid`
- Board code: Contains board `code` (e.g., `001JFR`)
- Scorekeeper scans using `mobile_scanner` package (Flutter)

**Backend Decode**:
```go
// Flexible matching in SQL
WHERE code = ? 
   OR CAST(board_number AS CHAR) = ?
   OR CONCAT(CAST(board_number AS CHAR), 'A') = ?
   OR code = CONCAT(LPAD(?, 3, '0'), 'JFR')
```

---

This comprehensive flow documentation covers every gate, validation, calculation, and business rule in the Archeris platform's tournament creation, registration, payment, package subscription, and scoring systems.

