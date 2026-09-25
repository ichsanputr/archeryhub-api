package handler

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

type Intent struct {
	Name        string   `json:"name"`
	Examples    []string `json:"examples"`
	ExamplesEn  []string `json:"examples_en,omitempty"`
	Answer      string   `json:"answer"`
	AnswerEn    string   `json:"answer_en,omitempty"`
}

var chatbotIntents = []Intent{}

var loadIntentsOnce sync.Once

func loadChatbotIntentsFromFile() {
	loadIntentsOnce.Do(func() {
		candidatePaths := []string{
			filepath.Join("data", "chatbot_intents.json"),
			filepath.Join("api", "data", "chatbot_intents.json"),
			filepath.Join("..", "api", "data", "chatbot_intents.json"),
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
		if err := json.Unmarshal(fileBytes, &loaded); err != nil {
			return
		}
		if len(loaded) > 0 {
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

// detectLanguage checks if a message is primarily English based on English tokens/keywords
func detectLanguage(message string, explicitLang string) string {
	explicitLang = strings.ToLower(strings.TrimSpace(explicitLang))
	if explicitLang == "en" || explicitLang == "en-us" || explicitLang == "en-gb" {
		return "en"
	}
	if explicitLang == "id" || explicitLang == "id-id" {
		return "id"
	}

	norm := normalizeText(message)
	tokens := strings.Fields(norm)
	if len(tokens) == 0 {
		return "id"
	}

	englishKeywords := map[string]bool{
		"what": true, "how": true, "why": true, "when": true, "where": true,
		"who": true, "can": true, "could": true, "would": true, "should": true,
		"is": true, "are": true, "the": true, "and": true, "for": true,
		"with": true, "to": true, "in": true, "of": true, "about": true,
		"tournament": true, "tournaments": true, "scoring": true, "score": true,
		"archer": true, "archers": true, "archery": true, "arrow": true,
		"arrows": true, "round": true, "rules": true, "package": true,
		"pricing": true, "price": true, "register": true, "registration": true,
		"certificate": true, "certificates": true, "ticket": true, "tickets": true,
		"hello": true, "hi": true, "hey": true, "thanks": true, "thank": true,
		"help": true, "support": true, "organizer": true, "organizers": true,
	}

	indonesianKeywords := map[string]bool{
		"apa": true, "bagaimana": true, "kenapa": true, "kapan": true, "di": true,
		"siapa": true, "bisa": true, "apakah": true, "ini": true, "itu": true,
		"dan": true, "untuk": true, "dengan": true, "ke": true, "dari": true,
		"soal": true, "tentang": true, "turnamen": true, "lomba": true, "panahan": true,
		"skor": true, "pemanah": true, "atlet": true, "anak": true, "panah": true,
		"aturan": true, "paket": true, "harga": true, "biaya": true, "daftar": true,
		"pendaftaran": true, "sertifikat": true, "tiket": true, "halo": true,
		"hai": true, "makasih": true, "terima": true, "kasih": true, "tolong": true,
		"bantuan": true, "penyelenggara": true, "panitia": true, "gimana": true,
	}

	enCount := 0
	idCount := 0

	for _, t := range tokens {
		if englishKeywords[t] {
			enCount++
		}
		if indonesianKeywords[t] {
			idCount++
		}
	}

	if enCount > idCount {
		return "en"
	}
	return "id"
}

func bestIntent(message string, lang string) (Intent, float64, string) {
	normalizedMessage := normalizeText(message)
	messageTokens := toTokenSet(normalizedMessage)

	bestScore := 0.0
	best := Intent{}
	matchedLang := lang

	for _, intent := range chatbotIntents {
		// Test Indonesian examples
		for _, ex := range intent.Examples {
			normalizedExample := normalizeText(ex)
			exampleTokens := toTokenSet(normalizedExample)
			score := jaccardScore(messageTokens, exampleTokens)

			if normalizedExample != "" && strings.Contains(normalizedMessage, normalizedExample) {
				score += 0.40
			}

			if score > bestScore {
				bestScore = score
				best = intent
				if lang == "" {
					matchedLang = "id"
				}
			}
		}

		// Test English examples
		for _, ex := range intent.ExamplesEn {
			normalizedExample := normalizeText(ex)
			exampleTokens := toTokenSet(normalizedExample)
			score := jaccardScore(messageTokens, exampleTokens)

			if normalizedExample != "" && strings.Contains(normalizedMessage, normalizedExample) {
				score += 0.40
			}

			if score > bestScore {
				bestScore = score
				best = intent
				if lang == "" {
					matchedLang = "en"
				}
			}
		}
	}

	return best, bestScore, matchedLang
}

func recommendedQuickActions(lang string) []string {
	if lang == "en" {
		actions := []string{"Tournament Schedule", "Registration Help", "Live Scoring & Leaderboard", "Organizer Packages"}
		sort.Strings(actions)
		return actions
	}
	actions := []string{"Cek Jadwal Turnamen", "Bantuan Pendaftaran", "Lihat Hasil Skor", "Paket Penyelenggara"}
	sort.Strings(actions)
	return actions
}

func ChatbotMessage() gin.HandlerFunc {
	return func(c *gin.Context) {
		loadChatbotIntentsFromFile()

		var req struct {
			Message string `json:"message" binding:"required"`
			Lang    string `json:"lang"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
			return
		}

		queryLang := c.Query("lang")
		if queryLang != "" && req.Lang == "" {
			req.Lang = queryLang
		}

		lang := detectLanguage(req.Message, req.Lang)

		intent, confidence, matchedLang := bestIntent(req.Message, lang)
		if matchedLang != "" {
			lang = matchedLang
		}

		if intent.Name == "" || confidence < 0.16 {
			fallbackAnswer := "Halo! Saya asisten Archeris. Saya bisa membantu Anda seputar jadwal turnamen, scoring digital, aturan World Archery, profil atlet, e-sertifikat, pendaftaran peserta, dan paket EO."
			if lang == "en" {
				fallbackAnswer = "Hello! I am the Archeris AI Assistant. I can assist you with tournament discovery, live digital scoring, World Archery rules, athlete profiles, certificates, registration, and organizer packages. How can I help you today?"
			}

			c.JSON(http.StatusOK, gin.H{
				"intent":        "fallback",
				"confidence":    confidence,
				"lang":          lang,
				"answer":        fallbackAnswer,
				"quick_actions": recommendedQuickActions(lang),
			})
			return
		}

		answer := intent.Answer
		if lang == "en" && intent.AnswerEn != "" {
			answer = intent.AnswerEn
		}

		c.JSON(http.StatusOK, gin.H{
			"intent":        intent.Name,
			"confidence":    confidence,
			"lang":          lang,
			"answer":        answer,
			"quick_actions": recommendedQuickActions(lang),
		})
	}
}

func ChatbotIntents() gin.HandlerFunc {
	return func(c *gin.Context) {
		loadChatbotIntentsFromFile()
		c.JSON(http.StatusOK, gin.H{"intents": chatbotIntents})
	}
}
