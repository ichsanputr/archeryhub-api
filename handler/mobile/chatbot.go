package mobile

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// ... moved to types.go ...

var chatbotIntents = []Intent{
	{
		Name:     "greeting",
		Examples: []string{"hai", "halo", "pagi", "hey"},
		Answer:   "Halo! Mau tanya apa?",
	},
	{
		Name:     "event_discovery",
		Examples: []string{"cari event", "event terdekat", "turnamen panahan"},
		Answer:   "Kamu bisa cek daftar event di menu Event lalu gunakan filter kota atau tanggal.",
	},
	{
		Name:     "membership_package",
		Examples: []string{"paket membership", "perpanjang paket", "fitur langganan"},
		Answer:   "Info paket ada di dashboard subscription, termasuk status aktif dan tanggal berakhir.",
	},
	{
		Name:     "event_schedule",
		Examples: []string{"jadwal event", "schedule pertandingan", "kapan lomba"},
		Answer:   "Kamu bisa cek jadwal event di menu Event. Kalau mau, kirim nama event yang ingin dicek.",
	},
	{
		Name:     "registration_help",
		Examples: []string{"cara daftar", "registrasi event", "bantu pendaftaran"},
		Answer:   "Untuk pendaftaran, buka detail event lalu klik Daftar. Kalau ada error, kirimkan nama event dan screenshot ya.",
	},
	{
		Name:     "score_tracking",
		Examples: []string{"cek skor", "lihat hasil", "track score"},
		Answer:   "Untuk cek skor, buka hasil event lalu pilih kategori. Kamu juga bisa kirim nama event biar aku bantu arahkan.",
	},
}

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
			if err == nil {
				fileBytes = b
				break
			}
		}

		if len(fileBytes) == 0 {
			return
		}

		var loaded []Intent
		if err := json.Unmarshal(fileBytes, &loaded); err != nil {
			return
		}
		if len(loaded) == 0 {
			return
		}

		chatbotIntents = loaded
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

			if normalizedExample != "" && strings.Contains(normalizedMessage, normalizedExample) {
				score += 0.35
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
	actions := []string{"Cek Jadwal Event", "Bantuan Pendaftaran", "Lihat Hasil Skor"}
	sort.Strings(actions)
	return actions
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

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pesan wajib diisi"})
			return
		}

		intent, confidence := bestIntent(req.Message)
		if intent.Name == "" || confidence < 0.18 {
			c.JSON(http.StatusOK, MobileChatbotResponse{
				Intent:       "fallback",
				Confidence:   confidence,
				Answer:       "Aku siap bantu customer service Archeryhub. Kamu bisa tanya tentang jadwal event, pendaftaran, hasil, atau membership.",
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
