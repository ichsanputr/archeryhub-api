# Panduan Penulisan Dokumentasi Archeris

File ini berisi aturan dan standar penulisan untuk semua artikel dokumentasi di Archeris. Setiap artikel docs disimpan sebagai file JSON di `api/data/docs/` dan ditampilkan di halaman `/docs`.

---

## 1. Tujuan Dokumentasi

Dokumentasi Archeris dibaca oleh **pengguna baru dan pemula** yang sedang bingung atau ingin belajar cara menggunakan platform kita. Mereka mungkin:
- Atlet panahan yang baru pertama kali mendaftar turnamen
- Panitia event yang belum pernah membuat turnamen digital
- Petugas scoring yang dapat kode akses dan tidak tahu harus mulai dari mana

Tugas kita: **membuat mereka paham, bisa langsung praktek, dan merasa terbantu.**

---

## 2. Struktur File JSON

Setiap dokumen disimpan di `api/data/docs/{slug}.json` dengan format:

```json
{
  "slug": "nama-dokumen",
  "icon": "ph:icon-name-bold",
  "category": "accounts|tournaments|scorekeeper|qualification|elimination|finance|subscriptions",
  "order": 1,
  "readTime": "4 min",
  "en": {
    "title": "Judul dalam Bahasa Inggris",
    "excerpt": "Ringkasan singkat 1-2 kalimat",
    "toc": [
      { "id": "section-id", "level": 2, "text": "Judul Bagian" }
    ],
    "content": "<p>...</p><h2>...</h2>"
  },
  "id": {
    "title": "Judul dalam Bahasa Indonesia",
    "excerpt": "Ringkasan singkat",
    "toc": [...],
    "content": "<p>...</p><h2>...</h2>"
  }
}
```

**Urutan kategori (wajib):**
1. `accounts` — Akun & profil pengguna
2. `tournaments` — Pembuatan & pengelolaan turnamen
3. `scorekeeper` — Scoring lapangan
4. `qualification` — Babak kualifikasi
5. `elimination` — Babak eliminasi
6. `finance` — Pembayaran, wallet, pencairan dana
7. `subscriptions` — Paket & langganan

---

## 3. Aturan Penulisan Konten

### 3.1 Pembuka Harus Paragraf

**Setiap artikel HARUS dimulai dengan `<p>` (paragraf), bukan heading.** Jangan langsung masuk ke `<h2>`.

Paragraf pembuka harus:
- Menjelaskan **apa** yang sedang dibahas
- Menjelaskan **kenapa** ini penting bagi pembaca
- Menggunakan bahasa yang ramah dan tidak menggurui

✅ **CONTOH BAIK:**
```html
<p>Akun Pemanah di Archeris berfungsi sebagai paspor digital atlet panahan yang menyimpan riwayat skor pertandingan, peringkat kualifikasi dan eliminasi, serta e-sertifikat resmi dari seluruh kejuaraan yang pernah diikuti. Melengkapi profil dengan data yang valid memastikan pendaftaran turnamen berlangsung otomatis, cepat, dan akurat sesuai kelompok usia serta divisi busur Anda.</p>
```

❌ **CONTOH BURUK:**
```html
<h2>Cara Mengatur Profil</h2>
<p>Untuk mengatur profil, ikuti langkah berikut:</p>
```

### 3.2 Gunakan Bahasa yang Mudah Dipahami

**JANGAN pakai istilah teknis yang membingungkan.** Jika harus pakai istilah, jelaskan artinya.

| Jangan | Ganti Dengan |
|--------|-------------|
| "Konfigurasi parameter kualifikasi" | "Mengatur babak kualifikasi" |
| "Implementasi bracket eliminasi" | "Membuat bagan pertandingan eliminasi" |
| "Autentikasi via OTP" | "Verifikasi lewat kode yang dikirim ke email" |
| "Endpoint API payment gateway" | "Sistem pembayaran online" |
| "User role assignment" | "Menentukan tipe akun" |
| "Digital archery scoring system" | "Sistem pencatatan skor panahan digital" |

**Prinsip:** Bayangkan Anda menjelaskan ke teman yang baru pertama kali dengar tentang Archeris.

### 3.3 Gaya Penulisan: Profesional tapi Tidak Kaku

Gunakan bahasa yang:
- **Ramah** — Pakai kata "Anda" dan "kita" untuk terasa lebih dekat
- **Jelas** — Kalimat pendek, satu ide per kalimat
- **Tidak bertele-tele** — Langsung ke inti, jangan basa-basi
- **Fleksibel** — Boleh santai di bagian tips, tetap rapi di bagian langkah

✅ **CONTOH BAIK:**
```html
<p>Sebelum membuat turnamen, pastikan profil organisasi Anda sudah lengkap. Panitia lain dan peserta turnamen akan melihat info ini, jadi isi dengan data yang benar.</p>
<p>Untuk mulai membuat turnamen, ikuti langkah-langkah di bawah ini:</p>
<ol>
  <li>Masuk ke <strong>Dashboard Organizer</strong> lalu klik menu <strong>Turnamen</strong>.</li>
  <li>Klik tombol <strong>+ Buat Turnamen</strong> di pojok kanan atas.</li>
</ol>
```

❌ **CONTOH BURUK (terlalu kaku):**
```html
<p>Proses inisiasi turnamen memerlukan prasyarat validasi entitas organisasi. Pengguna wajib memenuhi kriteria kelengkapan data sebelum menginisiasi workflow pembuatan event.</p>
```

### 3.4 Urutan Penjelasan yang Logis

Susun artikel dengan urutan yang masuk akal bagi pembaca baru:

1. **Pembuka** — Apa ini? Kenapa penting?
2. **Langkah dasar** — Cara mulai dari nol
3. **Penjelasan fitur** — Detail setiap bagian
4. **Tips & hal penting** — Info tambahan yang berguna
5. **FAQ (opsional)** — Pertanyaan yang sering muncul

**Contoh alur untuk "Membuat Turnamen":**
1. Apa itu turnamen di Archeris?
2. Langkah membuat turnamen baru
3. Mengisi detail event (nama, lokasi, tanggal)
4. Mengatur kategori lomba
5. Mengatur pembayaran
6. Tips sebelum mempublikasikan

### 3.5 Berdasarkan Fitur yang Ada di Aplikasi

**Semua penjelasan HARUS sesuai dengan fitur yang benar-benar ada di aplikasi.** Jangan mengarang fitur yang belum ada.

Sebelum menulis:
- Cek halaman Vue yang relevan di folder `app/pages/`
- Cek handler API di folder `api/handler/`
- Cek model data di folder `api/models/`
- Kalau ragu, tanya developer

**Contoh:** Di halaman profil pemanah (`pages/dashboard/archer/profile.vue`), tab yang tersedia adalah:
- Tab 1: "Personal Info & Specs" — nama, username, NIK, tanggal lahir, gender, tinggi, berat, jenis busur, dominansi tangan, klub, email, telepon, alamat, avatar, banner, sosial media
- Tab 2: "Bio & Achievements" — bio TipTap, daftar prestasi (maks 3 starred), statistik turnamen

**Jangan menulis:** "Di tab Equipment Anda bisa mengatur spesifikasi busur" — karena tab Equipment tidak ada di aplikasi.

### 3.6 Gunakan Navigasi yang Jelas

Sebutkan lokasi menu dengan format:

```html
<p>Buka menu <strong>Dashboard Organizer &gt; Profil Organisasi</strong>.</p>
```

Gunakan tanda panah `>` untuk menunjukkan urutan navigasi.

### 3.7 Hindari Heading "Low Value" (Terlalu Singkat)

**Setiap heading `<h2>` HARUS memiliki penjelasan yang cukup substansial — bukan cuma 1 paragraf pendek lalu pindah ke heading berikutnya.**

Heading yang hanya diisi 1-2 kalimat tidak memberikan nilai bagi pembaca dan membuat artikel terasa dangkal. Jika sebuah heading hanya punya penjelasan singkat, pertimbangkan untuk menggabungkannya dengan heading lain atau mengembangkan penjelasannya.

✅ **CONTOH BAIK** (heading dengan penjelasan lengkap + langkah + tips):
```html
<h2 id="mengisi-identitas">2. Mengisi Data Identitas dan Foto</h2>
<p>Untuk melengkapi data profil atlet:</p>
<ol>
  <li>Masuk ke akun Anda dan buka menu <strong>Dashboard Archer &gt; Profil</strong>.</li>
  <li>Pada tab <strong>Informasi Pribadi</strong>, lengkapi kolom berikut:
    <ul>
      <li><strong>Foto Profil:</strong> Unggah foto diri yang jelas agar panitia pelaksana dan juri garis dapat mengenali Anda saat verifikasi di lapangan.</li>
      <li><strong>Nama Lengkap:</strong> Masukkan nama lengkap sesuai identitas resmi (KTP/KIA/Paspor). Nama ini yang akan dicetak pada lembar skor resmi dan e-sertifikat kejuaraan.</li>
      <li><strong>Username Profil:</strong> Tentukan tautan unik untuk alamat portofolio profil publik atlet Anda.</li>
    </ul>
  </li>
</ol>
<p><strong>Tips:</strong> Gunakan foto terbaru yang menampilkan wajah Anda dengan jelas.</p>
```

❌ **CONTOH BURUK** (heading "low value" — cuma 1 paragraf singkat):
```html
<h2 id="cara-mengakses">1. Cara Mengakses Profil Pemanah</h2>
<p>Setelah login, buka menu <strong>Dashboard Archer &gt; Profil</strong>. Halaman profil Anda memiliki dua tab di bagian atas: Informasi & Spesifikasi dan Biografi & Prestasi. Anda akan mengisi data di kedua bagian ini.</p>
```

**Solusi:** Jika bagian "Cara Mengakses" terlalu singkat, gabungkan dengan heading berikutnya seperti "Mengisi Data Identitas" agar jadi satu section yang lebih padat.

### 3.8 Menambahkan Screenshot (Placeholder)

**Sangat disarankan untuk menyertakan screenshot pada setiap bagian penting agar pembaca bisa melihat langsung tampilan aplikasi.**

Karena screenshot belum tersedia saat penulisan, gunakan **placeholder** dari `https://picsum.photos/200` dengan ukuran yang sesuai konteks:

```html
<!-- Screenshot halaman profil -->
<img src="https://picsum.photos/800/400" alt="Halaman Profil Pemanah dengan tab Informasi Pribadi" class="w-full rounded-2xl border border-gray-200 shadow-sm my-6" />

<!-- Screenshot form pendaftaran -->
<img src="https://picsum.photos/800/500" alt="Formulir Pendaftaran Turnamen Step 2" class="w-full rounded-2xl border border-gray-200 shadow-sm my-6" />
```

**Ukuran yang disarankan:**
- `800/400` — untuk screenshot halaman dashboard lebar
- `800/500` — untuk screenshot form atau halaman panjang
- `800/300` — untuk screenshot bagian kecil (modal, popup, card)
- `400/300` — untuk screenshot elemen kecil (tombol, badge, notifikasi)

**Catatan:** Nanti developer akan mengganti URL `https://picsum.photos/...` dengan screenshot asli sesuai halaman yang dimaksud. Pastikan **alt text** deskriptif agar mudah diganti.

### 3.9 FAQ Tidak Wajib di Setiap Dokumen

**Bagian FAQ (Pertanyaan Umum) bersifat opsional.** Hanya tambahkan jika:
- Ada pertanyaan yang memang sering muncul dari pengguna
- Ada hal yang perlu diklarifikasi lebih lanjut
- Konteks dokumen memang kompleks dan butuh penjelasan tambahan

**Jangan memaksakan FAQ** jika topiknya sederhana dan sudah jelas dari penjelasan utama. Lebih baik tidak ada FAQ daripada FAQ yang tidak relevan.

✅ **Cocok ada FAQ:** Dokumen tentang pembayaran, verifikasi akun, pengaturan skor — topik yang sering menimbulkan pertanyaan.

❌ **Tidak perlu FAQ:** Dokumen singkat tentang cara mengakses halaman tertentu yang sudah jelas alurnya.

### 3.10 Heading & TOC

- Gunakan `<h2>` untuk setiap bagian utama
- Setiap `<h2>` harus punya `id` yang sama dengan `toc` di JSON
- Heading harus deskriptif dan bernomor

```html
<h2 id="langkah-membuat-turnamen">1. Langkah Membuat Turnamen</h2>
<h2 id="mengatur-kategori">2. Mengatur Kategori Lomba</h2>
```

---

## 4. Template Artikel

Gunakan template ini sebagai starting point:

```json
{
  "slug": "nama-dokumen",
  "icon": "ph:icon-name-bold",
  "category": "category-slug",
  "order": 1,
  "readTime": "4 min",
  "en": {
    "title": "Document Title",
    "excerpt": "One or two sentence summary of what this doc covers and why it matters.",
    "toc": [
      { "id": "section-one", "level": 2, "text": "1. Section Title" },
      { "id": "section-two", "level": 2, "text": "2. Section Title" },
      { "id": "common-questions", "level": 2, "text": "3. Frequently Asked Questions" }
    ],
    "content": "<p>Opening paragraph that explains what this feature is and why it's useful for the reader.</p>\n\n<h2 id=\"section-one\">1. Section Title</h2>\n<p>Content here...</p>\n\n<h2 id=\"section-two\">2. Section Title</h2>\n<p>Content here...</p>\n\n<h2 id=\"common-questions\">3. Frequently Asked Questions</h2>\n<p>Content here...</p>"
  },
  "id": {
    "title": "Judul Dokumen",
    "excerpt": "Ringkasan satu atau dua kalimat tentang isi dokumen ini.",
    "toc": [
      { "id": "bagian-satu", "level": 2, "text": "1. Judul Bagian" },
      { "id": "bagian-dua", "level": 2, "text": "2. Judul Bagian" },
      { "id": "tanya-jawab", "level": 2, "text": "3. Pertanyaan Umum" }
    ],
    "content": "<p>Paragraf pembuka yang menjelaskan fitur ini dan kenapa penting bagi pembaca.</p>\n\n<h2 id=\"bagian-satu\">1. Judul Bagian</h2>\n<p>Konten di sini...</p>\n\n<h2 id=\"bagian-dua\">2. Judul Bagian</h2>\n<p>Konten di sini...</p>\n\n<h2 id=\"tanya-jawab\">3. Pertanyaan Umum</h2>\n<p>Konten di sini...</p>"
  }
}
```

---

## 5. Checklist Sebelum Menyimpan

Sebelum menyimpan file JSON docs, pastikan:

- [ ] Dimulai dengan `<p>` (paragraf pembuka)
- [ ] Tidak ada istilah teknis tanpa penjelasan
- [ ] Bahasa ramah, profesional, tidak kaku
- [ ] Semua fitur yang disebutkan benar-benar ada di aplikasi
- [ ] Urutan penjelasan logis dan mudah diikuti
- [ ] Navigasi menu ditulis dengan format yang jelas
- [ ] Ada versi Bahasa Inggris (`en`) dan Bahasa Indonesia (`id`)
- [ ] TOC (`toc`) sesuai dengan `<h2 id="...">` di content
- [ ] `slug` menggunakan huruf kecil dan dash (kebab-case)
- [ ] `icon` menggunakan format `ph:icon-name-bold`
- [ ] `category` sesuai dengan salah satu dari 7 kategori yang ada
- [ ] `order` ditentukan sesuai urutan dalam kategori
- [ ] Tidak ada heading "low value" (1 paragraf saja tanpa penjelasan cukup)
- [ ] Screenshot placeholder `https://picsum.photos/...` ditambahkan di bagian penting dengan alt text deskriptif
- [ ] FAQ hanya ditambahkan jika memang relevan dengan konteks (tidak dipaksakan)

---

## 6. Icon yang Tersedia

Gunakan icon dari [Iconify Phosphor](https://icon-sets.iconify.design/ph/) dengan format `ph:nama-icon-bold`.

Icon yang sudah dipakai di docs existing:
- `ph:users-three-bold` — Account types
- `ph:buildings-bold` — Organization profile
- `ph:user-circle-gear-bold` — Archer profile
- `ph:trophy-bold` — Tournaments
- `ph:user-focus-bold` — Scorekeeper
- `ph:crosshair-bold` — Qualification
- `ph:tree-structure-bold` — Elimination
- `ph:wallet-bold` — Finance
- `ph:credit-card-bold` — Payment methods
- `ph:receipt-bold` — Payment status
- `ph:certificate-bold` — Certificates
- `ph:file-text-bold` — Reports

---

## 7. Referensi Fitur Aplikasi

Untuk memastikan keakuratan konten, referensi file aplikasi utama:

### Profil Pemanah
- **File:** `app/pages/dashboard/archer/profile.vue`
- **Tab:** Personal Info & Specs | Bio & Achievements
- **Field:** full_name, username, nik, date_of_birth, gender, height_cm, weight_kg, bow_type, hand_dominance, club_id, email, phone, emergency_contact_name, city, country, address, avatar_url, banner_url, bio, achievements (array), social media (10 platforms)
- **API:** GET/PUT /archer/me, GET /archers/me/stats, GET /auth/check-username

### Profil Organisasi
- **File:** `app/pages/dashboard/organizer/profile.vue`
- **Tab:** General | Branding | Social | Bank Accounts | Page Settings
- **API:** GET/PUT /organizer/me

### Pendaftaran Turnamen
- **File:** `app/pages/tournaments/[slug]/register.vue` (3323 lines)
- **3 Langkah:** Registration Type → Select Categories → Payment
- **Mode:** Captain/Team | Club Delegation
- **Fitur:** Draft restore, gender toggle, TeamSlotBuilder, CSV import, multi-category

### Pembayaran
- **File:** `app/pages/tournaments/[slug]/payment.vue`
- **Metode:** Mayar (QRIS/VA/Card), PayPal, Manual Transfer
- **API:** GET /payment/channels, GET /payment/instruction, POST /payment/create

### Dashboard Organizer
- **File:** `app/pages/dashboard/organizer/index.vue`
- **Menu:** Overview, Tournaments, Profile, Settings, Scorekeepers, Teams, Wallet, Earnings, Reports, Notifications, Bank Accounts, Payment Methods, Subscription

### Dashboard Archer
- **File:** `app/pages/dashboard/archer/tournaments/[id]/overview.vue`
- **Menu:** Tournaments, Profile, Settings, Certificates, Payments, Notifications
- **Per-Tournament:** Overview, My Registration, My Qualification, My Elimination, My Target, My Team
