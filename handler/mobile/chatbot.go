package mobile

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var defaultChatbotIntents = []Intent{
	{
		Name:     "greeting",
		Examples: []string{"hai", "halo", "pagi", "siang", "sore", "malam", "assalamualaikum", "hey", "hi", "selamat pagi"},
		Answer:   "Halo pemanah hebat! Ada yang bisa saya bantu terkait turnamen panahan, pendaftaran event, skor pertandingan, atau belanja peralatan panahan?",
	},
	{
		Name:     "event_discovery",
		Examples: []string{"cari event", "event terdekat", "turnamen panahan", "kejuaraan panahan", "lomba panahan", "daftar event", "event aktif"},
		Answer:   "Kamu bisa melihat semua jadwal turnamen panahan yang sedang dan akan datang pada menu 'Event'. Gunakan filter lokasi kota atau divisi busur untuk menemukan turnamen yang cocok!",
	},
	{
		Name:     "event_schedule",
		Examples: []string{"jadwal event", "schedule pertandingan", "kapan lomba", "tanggal turnamen", "rundown acara", "jadwal sesi"},
		Answer:   "Jadwal lengkap dan rundown sesi turnamen dapat dilihat pada halaman detail event masing-masing. Silakan pilih event yang kamu ikuti untuk melihat jadwal sesi dan target.",
	},
	{
		Name:     "registration_help",
		Examples: []string{"cara daftar", "registrasi event", "bantu pendaftaran", "daftar turnamen", "cara bayar", "metode pembayaran", "tiket lomba"},
		Answer:   "Untuk mendaftar turnamen:\n1. Buka halaman Event dan pilih kategori yang sesuai.\n2. Klik tombol 'Daftar Sekarang'.\n3. Isi data pemanah dan klub.\n4. Lakukan pembayaran tiket sesuai instruksi. Tiket & QR Pass akan langsung aktif!",
	},
	{
		Name:     "score_tracking",
		Examples: []string{"cek skor", "lihat hasil", "track score", "scorecard", "hasil kualifikasi", "skor eliminasi", "leaderboard"},
		Answer:   "Kamu bisa melihat skor kualifikasi dan bagan eliminasi secara langsung di menu 'Hasil Lomba' atau buka 'Event Saya' > 'Laporan Penampilan' untuk memantau performa panahanmu.",
	},
	{
		Name:     "target_info",
		Examples: []string{"nomor target", "bantalan", "posisi bantalan", "cek target saya", "target tembak"},
		Answer:   "Nomor target dan posisi bantalan kamu tercantum di tiket digital dan halaman 'Event Saya' > 'My Target'. Penyelenggara akan mengumumkan alokasi target sebelum sesi kualifikasi dimulai.",
	},
	{
		Name:     "marketplace_equipment",
		Examples: []string{"beli busur", "toko panahan", "marketplace", "jual panah", "beli anak panah", "arrow", "aksesoris panah", "recurve", "compound", "barebow"},
		Answer:   "Kamu bisa membeli busur, anak panah, stabilizer, finger tab, dan perlengkapan panahan lainnya di menu 'Marketplace'. Semua produk dijual oleh penjual panahan terpercaya!",
	},
	{
		Name:     "certificate_info",
		Examples: []string{"sertifikat", "unduh sertifikat", "piagam", "e-certificate", "sertifikat juara", "sertifikat peserta"},
		Answer:   "E-Sertifikat resmi akan otomatis tersedia di menu 'Profil' > 'Sertifikat Saya' atau di halaman 'Event Saya' > 'My Certificate' setelah penyelenggara menyelesaikan turnamen.",
	},
	{
		Name:     "membership_package",
		Examples: []string{"paket membership", "perpanjang paket", "fitur langganan", "upgrade akun", "paket organizer"},
		Answer:   "Info paket membership dan upgrade kuota event dapat diakses pada dashboard pengaturan akun atau dashboard organizer.",
	},
	{
		Name:     "contact_support",
		Examples: []string{"hubungi admin", "customer service", "bantuan cs", "kontak panitia", "nomor wa admin", "kendala teknis"},
		Answer:   "Jika memerlukan bantuan lebih lanjut atau mengalami kendala, silakan hubungi tim Customer Care Archeris melalui WhatsApp Support di menu Profil > Bantuan & Dukungan.",
	},
}

var chatbotIntents = defaultChatbotIntents
var loadIntentsOnce sync.Once

func loadChatbotIntentsFromFile() {
	loadIntentsOnce.Do(func() {
		candidatePaths := []string{
			filepath.Join("data", "chatbot_intents.json"),
			filepath.Join("api", "data", "chatbot_intents.json"),
		}

		var fileBytes []byte
		for _, path := range candidatePaths {
			b, err := os.ReadFile(path)
			if err == nil && len(b) > 0 {
				fileBytes = b
				break
			}
		}

		if len(fileBytes) == 0 {
			return
		}

		var loaded []Intent
		if err := json.Unmarshal(fileBytes, &loaded); err == nil && len(loaded) > 0 {
			chatbotIntents = loaded
		}
	})
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9\s]+`)

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonAlphaNum.ReplaceAllString(value, " ")
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func toTokenSet(value string) map[string]struct{} {
	tokens := strings.Fields(value)
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		set[token] = struct{}{}
	}
	return set
}

func jaccardScore(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersect := 0
	for key := range a {
		if _, ok := b[key]; ok {
			intersect++
		}
	}
	union := len(a) + len(b) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}

func bestIntent(message string) (Intent, float64) {
	normalizedMessage := normalizeText(message)
	messageTokens := toTokenSet(normalizedMessage)

	bestScore := 0.0
	best := Intent{}

	for _, intent := range chatbotIntents {
		intentScore := 0.0
		for _, ex := range intent.Examples {
			normalizedExample := normalizeText(ex)
			exampleTokens := toTokenSet(normalizedExample)
			score := jaccardScore(messageTokens, exampleTokens)

			if normalizedExample != "" && (strings.Contains(normalizedMessage, normalizedExample) || strings.Contains(normalizedExample, normalizedMessage)) {
				score += 0.40
			}

			if score > intentScore {
				intentScore = score
			}
		}

		if intentScore > bestScore {
			bestScore = intentScore
			best = intent
		}
	}

	return best, bestScore
}

func recommendedQuickActions() []string {
	return []string{
		"Jadwal Turnamen",
		"Bantuan Pendaftaran",
		"Cek Skor & Target",
		"Marketplace Panahan",
		"Hubungi Admin",
	}
}

// MobileChatbotMessage handles chatbot interaction
// @Summary Send Message to Chatbot
// @Description Send a natural language message and get an automated answer
// @Tags Mobile - Chatbot
// @Accept json
// @Produce json
// @Param request body MobileChatbotMessageRequest true "User Message"
// @Success 200 {object} MobileChatbotResponse
// @Router /mobile/chatbot/message [post]
func MobileChatbotMessage() gin.HandlerFunc {
	return func(c *gin.Context) {
		loadChatbotIntentsFromFile()

		var req MobileChatbotMessageRequest

		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
			c.JSON(http.StatusOK, MobileChatbotResponse{
				Intent:       "greeting",
				Confidence:   1.0,
				Answer:       "Halo! Ada yang bisa saya bantu seputar kejuaraan panahan atau marketplace?",
				QuickActions: recommendedQuickActions(),
			})
			return
		}

		intent, confidence := bestIntent(req.Message)
		if intent.Name == "" || confidence < 0.15 {
			c.JSON(http.StatusOK, MobileChatbotResponse{
				Intent:       "fallback",
				Confidence:   confidence,
				Answer:       "Maaf, saya belum memahami pertanyaan tersebut secara spesifik. Namun saya dapat membantu menjawab seputar jadwal event panahan, cara pendaftaran, cek nomor bantalan/skor, sertifikat, atau toko peralatan panahan.",
				QuickActions: recommendedQuickActions(),
			})
			return
		}

		c.JSON(http.StatusOK, MobileChatbotResponse{
			Intent:       intent.Name,
			Confidence:   confidence,
			Answer:       intent.Answer,
			QuickActions: recommendedQuickActions(),
		})
	}
}

// MobileChatbotIntents returns list of supported intents
// @Summary List Chatbot Intents
// @Description Get a list of supported topics the chatbot can answer
// @Tags Mobile - Chatbot
// @Produce json
// @Success 200 {object} MobileChatbotIntentsResponse
// @Router /mobile/chatbot/intents [get]
func MobileChatbotIntents() gin.HandlerFunc {
	return func(c *gin.Context) {
		loadChatbotIntentsFromFile()
		c.JSON(http.StatusOK, MobileChatbotIntentsResponse{Intents: chatbotIntents})
	}
}
