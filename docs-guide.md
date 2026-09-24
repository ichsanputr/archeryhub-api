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

### 3.3 Tone Bahasa Indonesia: Santai, Langsung, Relate

Docs ini dibaca panitia panahan Indonesia — bukan audiens korporat. Gunakan bahasa yang terasa seperti rekan yang tahu cara pakai aplikasinya, bukan manual teknis.

**Empat prinsip utama:**

1. **Mulai dari kebutuhan user, bukan dari definisi fitur**
2. **Kalimat pendek. Satu ide per kalimat.**
3. **"Kalau", "bisa", "pakai" — bukan "apabila", "dapat", "menggunakan"**
4. **Langsung ke poin — tidak perlu basa-basi pembuka**

---

#### Pembuka Artikel

Jangan buka artikel dengan mendefinisikan fitur. Mulai dari **konteks user** — kapan mereka butuh ini dan apa yang akan terjadi.

❌ **BURUK — mendefinisikan fitur:**
```html
<p>Halaman Buat Turnamen adalah formulir satu halaman yang diisi untuk mendaftarkan turnamen baru ke sistem. Pengisian membutuhkan sekitar 2–3 menit.</p>
```

❌ **BURUK — terlalu formal:**
```html
<p>Saat menyelenggarakan kejuaraan panahan di Archeris, menentukan Negara Tuan Rumah dan Mata Uang Resmi adalah langkah konfigurasi yang paling mendasar.</p>
```

✅ **BAIK — mulai dari konteks user:**
```html
<p>Di halaman ini Anda mengisi satu formulir untuk mendaftarkan turnamen baru ke sistem. Prosesnya butuh sekitar 2–3 menit.</p>
```

✅ **BAIK — langsung ke yang penting:**
```html
<p>Sebelum membuka pendaftaran, ada dua pengaturan penting yang perlu Anda isi dulu: <strong>Negara</strong> dan <strong>Mata Uang</strong>. Dua hal ini yang menentukan metode pembayaran apa yang muncul saat peserta checkout, dan bagaimana invoice pendaftaran mereka dicetak. Kalau diubah setelah peserta mulai daftar, bisa bikin data pembayaran jadi kacau.</p>
```

---

#### Penjelasan Fitur dalam List

Setiap bullet/list item harus terasa seperti penjelasan langsung, bukan entry kamus.

❌ **BURUK — kaku dan terjemahan langsung:**
```html
<li><strong>Routing Otomatis Gateway Pembayaran:</strong> Sistem secara otomatis mengarahkan checkout pembayaran peserta ke penyedia gateway yang sesuai dengan mata uang yang dipilih.</li>
```

✅ **BAIK — natural dan relate:**
```html
<li><strong>Metode Bayar:</strong> Menentukan metode pembayaran yang aktif. Pilih IDR dan Mayar (QRIS, transfer bank) akan muncul. Pilih USD dan PayPal yang aktif.</li>
```

---

#### Warning / Catatan Penting

Warning boleh tegas tapi tetap santai. Tidak perlu terdengar seperti dokumentasi hukum.

❌ **BURUK:**
```html
<p>Apabila peserta sudah memiliki invoice aktif, mengubah mata uang dapat menyebabkan ketidaksesuaian pada catatan pembayaran.</p>
```

✅ **BAIK:**
```html
<p><strong>Penting:</strong> Tentukan mata uang sebelum peserta mulai daftar. Kalau sudah ada yang punya invoice aktif lalu Anda ganti mata uang, data pembayaran mereka bisa tidak sinkron.</p>
```

---

#### Kalimat Penutup Section

Kalimat penutup harus ringkas — satu kalimat yang konfirmasi atau kasih konteks lanjutan. Tidak perlu panjang.

❌ **BURUK — terlalu panjang dan kaku:**
```html
<p>Peserta menerima invoice dengan format mata uang yang sama, sehingga jelas berapa yang harus dibayar dan dalam mata uang apa.</p>
```

✅ **BAIK — singkat dan natural:**
```html
<p>Peserta dapat invoice yang sudah jelas — berapa yang harus dibayar, pakai mata uang apa, tidak ada ambiguitas.</p>
```

---

#### Kata-Kata yang Perlu Dihindari

| ❌ Hindari | ✅ Ganti dengan |
|-----------|----------------|
| `menggunakan` | `pakai` |
| `apabila` | `kalau`, `jika` |
| `dapat` (= bisa) | `bisa` |
| `mengarahkan`, `menentukan` (untuk sistem) | `otomatis tampilkan`, `langsung` |
| `diselenggarakan` | `diadakan`, `digelar` |
| `memastikan` (pembuka kalimat) | `supaya`, `biar` |
| `menginput` | `isi`, `masukkan`, `ketik` |
| `melakukan pengaturan` | `atur`, `setting` |
| `terdapat` | `ada` |
| `merupakan` | `adalah`, atau hapus saja |



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

**Wajib cek nama tab/halaman yang sebenarnya di kode sebelum menulis.** Jangan mengarang label yang tidak ada di UI. Contoh tab yang benar di halaman pengaturan turnamen (`/dashboard/organizer/tournaments/[id]`):

| Tab | Isi |
|-----|-----|
| **Informasi** | Nama turnamen, deskripsi, tanggal, banner, logo |
| **Pendaftaran** | Batas pendaftaran, negara, mata uang, biaya kategori |
| **Lokasi** | Nama venue, alamat, link Google Maps |
| **Media** | Galeri foto event |
| **Hasil** | Upload hasil manual atau kelola hasil system-generated |
| **FAQ** | Pertanyaan umum turnamen |

❌ **JANGAN pakai nama yang tidak ada di UI:**
```
"Go to the Page Builder tab"       ← tidak ada tab bernama ini
"Open the Settings section"        ← terlalu generik
"In the Tournament Details tab"    ← tidak ada tab ini
```

✅ **Pakai nama tab yang sebenarnya:**
```html
<p>Klik tab <strong>Pendaftaran</strong>, lalu scroll ke bagian <strong>Negara &amp; Mata Uang Turnamen</strong>.</p>
<p>Klik tab <strong>Lokasi</strong> untuk mengisi nama venue dan alamat.</p>
```

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

**⚠️ WAJIB: Image TIDAK boleh jadi elemen terakhir dalam satu section.**

Setiap `<img>` harus selalu diikuti minimal satu paragraf `<p>` setelahnya. Image yang diletakkan di posisi paling bawah section — tepat sebelum `<h2>` berikutnya atau sebelum konten habis — membuat pembaca bingung karena tidak ada konteks lanjutan. Ini juga buruk untuk SEO karena search engine tidak dapat menghubungkan gambar dengan teks yang relevan.

**⚠️ WAJIB: Image TIDAK boleh langsung sebelum atau sesudah `<ol>`, `<ul>`.**

Image harus selalu dikelilingi oleh paragraf, bukan list. Jangan letakkan image langsung sebelum atau sesudah elemen list karena akan terasa disjointed (terpotong konteksnya). Image dan list harus dipisahkan oleh minimal satu paragraf penjelasan atau transisi.

❌ **CONTOH BURUK** (image langsung setelah list):
```html
<ol>
  <li>Buka tab <strong>Rekening Bank</strong>.</li>
  <li>Masukkan nomor rekening dan nama pemilik.</li>
  <li>Klik simpan.</li>
</ol>
<img src="https://picsum.photos/800/400" alt="Tab Rekening Bank" class="w-full rounded-2xl border border-gray-200 shadow-sm my-6" />

<h2 id="halaman-publik">5. Halaman Publik Organisasi</h2>
```

❌ **CONTOH BURUK** (image langsung sebelum list):
```html
<p>Ikuti langkah berikut untuk membuat turnamen:</p>
<img src="https://picsum.photos/800/400" alt="Halaman pembuatan turnamen" class="w-full rounded-2xl border border-gray-200 shadow-sm my-6" />

<ol>
  <li>Klik tombol <strong>+ Buat Turnamen</strong>.</li>
  <li>Isi nama turnamen dan tanggal.</li>
</ol>
```

✅ **CONTOH BAIK** (image diikuti paragraf, baru kemudian list):
```html
<ol>
  <li>Buka tab <strong>Rekening Bank</strong>.</li>
  <li>Masukkan nomor rekening dan nama pemilik.</li>
  <li>Klik simpan.</li>
</ol>
<img src="https://picsum.photos/800/400" alt="Tab Rekening Bank dengan kolom nomor rekening" class="w-full rounded-2xl border border-gray-200 shadow-sm my-6" />
<p><strong>Tips:</strong> Gunakan rekening atas nama organisasi agar proses verifikasi pencairan dana berlangsung lebih cepat.</p>
```

❌ **CONTOH BURUK** (image jadi elemen terakhir lalu langsung heading baru):
```html
<ol>
  <li>Buka tab <strong>Rekening Bank</strong>.</li>
  <li>Masukkan nomor rekening dan nama pemilik.</li>
  <li>Klik simpan.</li>
</ol>
<img src="https://picsum.photos/800/400" alt="Tab Rekening Bank" class="w-full rounded-2xl border border-gray-200 shadow-sm my-6" />

<h2 id="halaman-publik">5. Halaman Publik Organisasi</h2>
```

**Cara mudah memilih kalimat penutup setelah image:**
- Tambahkan tips atau catatan penting yang relevan dengan isi screenshot
- Tulis kalimat transisi ke langkah atau topik berikutnya
- Konfirmasi ulang poin kunci dari section tersebut dengan 1 kalimat singkat

### 3.9 Mermaid Diagram Flow

Platform Archeris mendukung diagram interaktif menggunakan **Mermaid** untuk membantu pembaca memahami alur yang kompleks secara visual. Gunakan `<pre class="mermaid">` di dalam content HTML.

#### Kapan Pakai Diagram

Gunakan diagram Mermaid ketika ada:
- **Alur status** (lifecycle) — misalnya status sesi kualifikasi: Draft → Locked → In Progress → Completed
- **Alur proses multi-langkah** — misalnya flow registrasi, flow pembayaran, flow eliminasi
- **Hubungan antar entitas** — misalnya bagaimana skor kualifikasi menghasilkan bracket eliminasi
- **Keputusan bercabang** — misalnya pilihan bracket size berdasarkan jumlah peserta

Jangan gunakan diagram untuk hal yang sudah cukup jelas dengan teks atau tabel.

#### Aturan Wajib: Gunakan `TD` (Top-Down), Bukan `LR` (Left-Right)

**SELALU gunakan arah vertikal `TD` atau `TB`** agar diagram tampil besar dan mudah dibaca pada layar sempit. Diagram horizontal (`LR`) menyebabkan semua node berdesakan menjadi satu baris kecil yang sulit dibaca.

❌ **BURUK — terlalu sempit dan kecil:**
```
flowchart LR
  A --> B --> C --> D --> E
```

✅ **BAIK — besar, mudah dibaca, scroll-friendly:**
```
flowchart TD
  A --> B
  B --> C
  C --> D
  D --> E
```

Untuk `stateDiagram-v2` gunakan `direction TB`:
```
stateDiagram-v2
  direction TB
  [*] --> Draft
  Draft --> Locked
```

#### Aturan Wajib: Ukuran Node Harus Jelas

Tambahkan label panjang dan deskriptif pada setiap node agar kotak terlihat besar:

❌ **Terlalu singkat, node kecil:**
```
A([Draft]) --> B([Lock]) --> C([Done])
```

✅ **Label informatif, node besar:**
```
A([🔧 Draft — Session setup\nin progress]) --> B([🔒 Locked — Ready,\nawaiting start command])
```

Gunakan `\n` untuk baris baru di dalam label node agar node jadi lebih tinggi dan tidak terlalu sempit.

#### Aturan Wajib: Bridge Teks Sebelum dan Sesudah Diagram

**Diagram TIDAK BOLEH muncul tiba-tiba** tanpa konteks. Selalu ada paragraf pengantar sebelumnya yang menjelaskan apa yang akan ditunjukkan oleh diagram, dan paragraf penutup setelahnya yang merangkum atau menarik kesimpulan dari diagram tersebut.

❌ **BURUK — diagram tiba-tiba tanpa konteks:**
```html
<ol>
  <li>Sesi dibuat...</li>
  <li>Sesi dikunci...</li>
  <li>Sesi dibuka...</li>
</ol>
<pre class="mermaid">
flowchart TD ...
</pre>
<h2>Section berikutnya</h2>
```

✅ **BAIK — ada pengantar + penutup:**
```html
<ol>
  <li>Sesi dibuat...</li>
  <li>Sesi dikunci...</li>
  <li>Sesi dibuka...</li>
</ol>
<p>Diagram berikut menunjukkan alur lengkap perubahan status sesi dari awal dibuat hingga hasil resmi diterbitkan.</p>
<pre class="mermaid">
flowchart TD ...
</pre>
<p>Perlu diingat bahwa transisi dari <strong>In Progress</strong> kembali ke <strong>Locked</strong> bisa dilakukan kapan saja oleh panitia jika perlu menghentikan input skor sementara — misalnya saat ada protes nilai di lapangan.</p>
```

#### Template Diagram yang Baik

```html
<!-- Paragraf pengantar wajib ada sebelum diagram -->
<p>Diagram berikut menggambarkan [apa yang ditunjukkan]. Perhatikan [hal penting].</p>

<pre class="mermaid">
flowchart TD
  A(["🔧 Node A<br/>Deskripsi panjang"]) --> B(["Node B<br/>Deskripsi panjang"])
  B --> C{"Apakah kondisi X?"}
  C -->|Ya| D(["✅ Hasil Positif<br/>Penjelasan singkat"])
  C -->|Tidak| E(["❌ Hasil Lain<br/>Penjelasan singkat"])
</pre>

<!-- Paragraf penutup wajib ada setelah diagram -->
<p>Dari diagram di atas terlihat bahwa [kesimpulan]. [Tindak lanjut yang perlu pembaca lakukan].</p>
```

**⚠️ Penting: Newline dalam label node**

- Flowchart node: Gunakan `<br/>` di dalam tanda kutip untuk baris baru — `A(["Baris 1<br/>Baris 2"])`
- StateDiagram state names: Gunakan `state "Nama State" as id` — **jangan** gunakan `id : deskripsi` dengan newline
- Transition labels: Harus **satu baris** — tidak ada newline di dalam label transition

```
❌ SALAH — newline dalam transition label akan error:
  Draft --> Locked : Parameter disimpan,
siap untuk hari lomba

✅ BENAR — transition label satu baris:
  Draft --> Locked : Parameter disimpan

✅ BENAR — state dengan label multi-kata pakai state keyword:
  state "🔧 Draft Setup" as Draft
```

#### Checklist Diagram

- [ ] Arah diagram `TD` atau `TB`, **bukan `LR`**
- [ ] Label node panjang dan deskriptif — minimal 4–5 kata per node
- [ ] Ada `\n` di label untuk membuat node lebih tinggi jika perlu
- [ ] Ada paragraf **sebelum** diagram yang menjelaskan apa yang akan ditampilkan
- [ ] Ada paragraf **setelah** diagram yang merangkum atau memberi konteks lanjutan
- [ ] Diagram tidak langsung diikuti `<h2>` tanpa penutup paragraf

### 3.10 FAQ Tidak Wajib di Setiap Dokumen

**Bagian FAQ (Pertanyaan Umum) bersifat opsional.** Hanya tambahkan jika:
- Ada pertanyaan yang memang sering muncul dari pengguna
- Ada hal yang perlu diklarifikasi lebih lanjut
- Konteks dokumen memang kompleks dan butuh penjelasan tambahan

**Jangan memaksakan FAQ** jika topiknya sederhana dan sudah jelas dari penjelasan utama. Lebih baik tidak ada FAQ daripada FAQ yang tidak relevan.

✅ **Cocok ada FAQ:** Dokumen tentang pembayaran, verifikasi akun, pengaturan skor — topik yang sering menimbulkan pertanyaan.

❌ **Tidak perlu FAQ:** Dokumen singkat tentang cara mengakses halaman tertentu yang sudah jelas alurnya.

### 3.11 Pola Bahasa Indonesia yang Harus Dihindari

Penulisan docs dalam Bahasa Indonesia sering jatuh ke pola yang terasa kaku, terjemahan langsung, atau tidak natural. Berikut daftar pola spesifik yang **dilarang** beserta alternatifnya:

#### Pola Kalimat Pembuka yang Kaku

❌ **JANGAN** mulai artikel dengan konstruksi `"[Nama halaman/fitur] adalah..."`:
```
Halaman Buat Turnamen adalah formulir satu halaman yang diisi untuk...
Profil Pemanah adalah identitas digital yang berisi...
Fitur Export adalah tool yang digunakan untuk...
```

✅ **GANTI** dengan kalimat yang langsung menjelaskan konteks atau manfaatnya:
```
Di halaman ini Anda mengisi satu formulir untuk mendaftarkan turnamen baru...
Di sini Anda menyimpan informasi pribadi, spesifikasi busur, dan catatan prestasi...
Gunakan fitur ini untuk mengekspor data ke format PDF atau Excel...
```

#### Kata Teknis / Terjemahan Kaku

Selain tabel di 3.2, hindari juga pola-pola ini:

| ❌ Jangan | ✅ Ganti Dengan |
|-----------|----------------|
| `mengkonsumsi 1 slot kuota` | `memakai 1 slot kuota` |
| `dikonsumsi` (untuk sumber daya) | `dipakai`, `terpakai`, `habis` |
| `record turnamen` | `data turnamen` |
| `Pengisian membutuhkan sekitar X menit` | `Prosesnya butuh sekitar X menit` |
| `Aturan pengembalian sebagian berlaku tergantung apakah...` | `Ada aturan pengembalian, tergantung apakah...` |
| `server menolak pembuatan Anda` | `terjadi error saat proses pembuatan` |
| `sparring internal klub` | `latihan internal`, `event internal klub` |
| `dua kontrol` (untuk pilihan UI) | `dua pilihan`, `dua opsi` |
| `Tetapkan ke sesuatu yang...` | `Tentukan...`, `Pilih...`, `Isi dengan...` |
| `Pelajarannya:` | `Intinya:`, `Kesimpulannya:` |
| `terlepas dari apakah` | `tidak peduli apakah`, `meskipun` |

#### Pola Judul Terjemahan Langsung

❌ **JANGAN** terjemahkan judul secara harfiah jika hasilnya terdengar aneh:
```
"Formulir Pembuatan Turnamen"   ← terlalu formal/kaku
"Pengaturan Kategori Lomba"     ← ok
"Alat Generasi Kode Scorekeeper" ← aneh
```

✅ **GANTI** dengan judul yang natural dan mudah dipahami:
```
"Form Buat Turnamen"            ← singkat, natural
"Membuat Kategori Turnamen"     ← action-oriented
"Kode Akses Scorekeeper"        ← langsung ke pointnya
```

#### Tabel Tambahan Kata/Frasa yang Perlu Dihindari

Tambahkan ke tabel di section 3.2 jika ditemukan pola baru. Panduan singkatnya: **jika kalimat tersebut terasa seperti terjemahan Google Translate, tulis ulang dari nol dalam Bahasa Indonesia yang natural.**

---

### 3.12 Heading & TOC

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
- [ ] Setiap `<img>` selalu diikuti minimal satu `<p>` setelahnya — tidak ada image di posisi terakhir section
- [ ] Jika ada diagram Mermaid: arah `TD`/`TB`, label node panjang, ada paragraf sebelum dan sesudah diagram, tidak langsung diikuti `<h2>`
- [ ] FAQ hanya ditambahkan jika memang relevan dengan konteks (tidak dipaksakan)
- [ ] Versi Indonesia: tidak ada pola kalimat kaku seperti `"[Nama halaman] adalah..."`, `"mengkonsumsi"`, `"record"`, `"server menolak pembuatan Anda"`, atau terjemahan harfiah lainnya (lihat 3.11)

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
