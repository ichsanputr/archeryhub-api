package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// RosterAthleteInput represents a single athlete row submitted for verification
type RosterAthleteInput struct {
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
	CategoryName string `json:"category_name"`
	Phone        string `json:"phone"`
	DateOfBirth  string `json:"date_of_birth"`
	ClubName     string `json:"club_name"`
}

// VerifyRosterRequest is the input payload for batch verifying athletes
type VerifyRosterRequest struct {
	Athletes []RosterAthleteInput `json:"athletes"`
}

// RosterAthleteVerifiedResult represents verification result for a single athlete
type RosterAthleteVerifiedResult struct {
	RowIndex            int     `json:"row_index"`
	FullName            string  `json:"full_name"`
	Email               string  `json:"email"`
	Gender              string  `json:"gender"`
	Phone               string  `json:"phone"`
	DateOfBirth         string  `json:"date_of_birth"`
	ClubName            string  `json:"club_name"`
	AvatarURL           string  `json:"avatar_url"`
	IsExistingUser      bool    `json:"is_existing_user"`
	ArcherID            *string `json:"archer_id"`
	IsAlreadyRegistered bool    `json:"is_already_registered"`
	MatchedCategoryID   string  `json:"matched_category_id"`
	MatchedCategoryName string  `json:"matched_category_name"`
	Status              string  `json:"status"` // 'ready_existing', 'ready_new', 'category_needed', 'already_registered', 'invalid_email'
	Message             string  `json:"message"`
}

// VerifyTournamentRoster checks batch athlete emails, account status, duplicate tournament entries, and matches categories
func VerifyTournamentRoster(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID turnamen tidak boleh kosong"})
			return
		}

		var req VerifyRosterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload data atlet tidak valid", "details": err.Error()})
			return
		}

		if len(req.Athletes) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"total":       0,
				"valid_count": 0,
				"results":     []RosterAthleteVerifiedResult{},
			})
			return
		}

		// Resolve actual tournament UUID
		var tournamentUUID string
		err := db.Get(&tournamentUUID, "SELECT uuid FROM events WHERE id = ? OR uuid = ? LIMIT 1", eventID, eventID)
		if err != nil {
			tournamentUUID = eventID
		}

		// Fetch tournament individual categories
		type EventCat struct {
			UUID               string  `db:"uuid"`
			CustomName         *string `db:"category_name_custom"`
			DivisionName       string  `db:"division_name"`
			CategoryName       string  `db:"category_name"`
			GenderDivisionName string  `db:"gender_division_name"`
			EventType          string  `db:"event_type_name"`
		}
		var categories []EventCat
		_ = db.Select(&categories, `
			SELECT 
				ec.uuid,
				ec.category_name_custom,
				COALESCE(d.name, '') as division_name,
				COALESCE(ag.name, '') as category_name,
				COALESCE(gd.name, '') as gender_division_name,
				COALESCE(et.name, '') as event_type_name
			FROM tournament_categories ec
			LEFT JOIN ref_bow_types d ON ec.division_uuid = d.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN ref_gender_divisions gd ON ec.gender_division_uuid = gd.uuid
			LEFT JOIN ref_tournament_types et ON ec.tournament_type_uuid = et.uuid
			WHERE ec.tournament_id = ?
		`, tournamentUUID)

		// Build normalized category lookup map
		type CategoryLookup struct {
			UUID   string
			Name   string
			Gender string // 'male', 'female', 'mixed'
		}
		var lookupList []CategoryLookup
		for _, cat := range categories {
			fullName := ""
			if cat.CustomName != nil && strings.TrimSpace(*cat.CustomName) != "" {
				fullName = strings.TrimSpace(*cat.CustomName)
			} else {
				parts := []string{}
				if cat.DivisionName != "" {
					parts = append(parts, cat.DivisionName)
				}
				if cat.CategoryName != "" {
					parts = append(parts, cat.CategoryName)
				}
				if cat.GenderDivisionName != "" {
					parts = append(parts, cat.GenderDivisionName)
				}
				if cat.EventType != "" && !strings.EqualFold(cat.EventType, "Individual") {
					parts = append(parts, cat.EventType)
				}
				fullName = strings.Join(parts, " ")
			}

			catGender := "mixed"
			gLower := strings.ToLower(fullName + " " + cat.GenderDivisionName)
			if strings.Contains(gLower, "putri") || strings.Contains(gLower, "women") || strings.Contains(gLower, "female") {
				catGender = "female"
			} else if strings.Contains(gLower, "putra") || strings.Contains(gLower, "men") || strings.Contains(gLower, "male") {
				catGender = "male"
			}

			lookupList = append(lookupList, CategoryLookup{
				UUID:   cat.UUID,
				Name:   fullName,
				Gender: catGender,
			})
		}

		results := make([]RosterAthleteVerifiedResult, 0, len(req.Athletes))
		validCount := 0
		seenEmails := make(map[string]int)

		for idx, ath := range req.Athletes {
			email := strings.ToLower(strings.TrimSpace(ath.Email))
			fullName := strings.TrimSpace(ath.FullName)
			phone := strings.TrimSpace(ath.Phone)
			dob := strings.TrimSpace(ath.DateOfBirth)
			club := strings.TrimSpace(ath.ClubName)

			// Normalize gender
			gender := strings.ToLower(strings.TrimSpace(ath.Gender))
			if gender == "m" || gender == "l" || gender == "male" || gender == "men" || gender == "man" || gender == "boy" || gender == "putra" || gender == "laki-laki" || gender == "laki" || gender == "pria" {
				gender = "male"
			} else if gender == "f" || gender == "w" || gender == "female" || gender == "women" || gender == "woman" || gender == "girl" || gender == "putri" || gender == "perempuan" || gender == "wanita" || (gender == "p" && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(ath.Gender)), "pri")) {
				gender = "female"
			} else {
				gender = "male" // default fallback
			}

			res := RosterAthleteVerifiedResult{
				RowIndex:    idx + 1,
				FullName:    fullName,
				Email:       email,
				Gender:      gender,
				Phone:       phone,
				DateOfBirth: dob,
				ClubName:    club,
			}

			// Validate Email format basic check
			if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
				res.Status = "invalid_email"
				res.Message = "Format email tidak valid"
				results = append(results, res)
				continue
			}

			// Check Duplicate in this batch
			if prevRow, exists := seenEmails[email]; exists {
				res.Status = "duplicate_batch"
				res.Message = fmt.Sprintf("Email duplikat di baris %d dan %d", prevRow, idx+1)
				results = append(results, res)
				continue
			}
			seenEmails[email] = idx + 1

			// 1. Check existing user in archers or users table
			type ExistingArcher struct {
				UUID        string  `db:"uuid"`
				FullName    string  `db:"full_name"`
				Gender      *string `db:"gender"`
				DateOfBirth *string `db:"date_of_birth"`
				Phone       *string `db:"phone"`
				ClubName    *string `db:"club_name"`
				AvatarURL   *string `db:"avatar_url"`
			}
			var existing ExistingArcher
			err := db.Get(&existing, `
				SELECT 
					uuid, 
					full_name, 
					gender, 
					date_of_birth, 
					phone, 
					club_name, 
					avatar_url 
				FROM archers 
				WHERE LOWER(email) = ? 
				LIMIT 1
			`, email)

			if err == nil && existing.UUID != "" {
				res.IsExistingUser = true
				res.ArcherID = &existing.UUID
				if existing.AvatarURL != nil {
					res.AvatarURL = *existing.AvatarURL
				}
				// If CSV row left fields empty, backfill with user's profile
				if res.FullName == "" && existing.FullName != "" {
					res.FullName = existing.FullName
				}
				if existing.Gender != nil && *existing.Gender != "" && ath.Gender == "" {
					res.Gender = *existing.Gender
				}
				if res.Phone == "" && existing.Phone != nil {
					res.Phone = *existing.Phone
				}
				if res.DateOfBirth == "" && existing.DateOfBirth != nil {
					res.DateOfBirth = *existing.DateOfBirth
				}
				if res.ClubName == "" && existing.ClubName != nil {
					res.ClubName = *existing.ClubName
				}
			} else {
				res.IsExistingUser = false
			}

			// 2. Check if already registered in this tournament
			var alreadyRegisteredCount int
			_ = db.Get(&alreadyRegisteredCount, `
				SELECT COUNT(*) 
				FROM tournament_participants 
				WHERE tournament_id = ? 
				  AND (LOWER(email) = ? OR (archer_id IS NOT NULL AND archer_id = ?))
			`, tournamentUUID, email, res.ArcherID)

			if alreadyRegisteredCount > 0 {
				res.IsAlreadyRegistered = true
				res.Status = "already_registered"
				res.Message = "Atlet dengan email ini sudah terdaftar di turnamen ini"
				results = append(results, res)
				continue
			}

			// 3. Match Category
			rawCatName := strings.ToLower(strings.TrimSpace(ath.CategoryName))
			if rawCatName != "" {
				// Exact or sub-match
				for _, cat := range lookupList {
					catNameLower := strings.ToLower(cat.Name)
					if catNameLower == rawCatName || strings.Contains(catNameLower, rawCatName) || strings.Contains(rawCatName, catNameLower) {
						// Check gender compatibility
						if cat.Gender == "mixed" || cat.Gender == res.Gender {
							res.MatchedCategoryID = cat.UUID
							res.MatchedCategoryName = cat.Name
							break
						}
					}
				}
			}

			if res.MatchedCategoryID == "" {
				res.Status = "category_needed"
				res.Message = "Kategori belum cocok, silakan pilih kategori di form"
			} else {
				if res.IsExistingUser {
					res.Status = "ready_existing"
					res.Message = "Akun terdaftar ditemukan dan siap didaftarkan"
				} else {
					res.Status = "ready_new"
					res.Message = "Akun baru akan dibuat saat pendaftaran"
				}
				validCount++
			}

			results = append(results, res)
		}

		c.JSON(http.StatusOK, gin.H{
			"total":       len(results),
			"valid_count": validCount,
			"results":     results,
		})
	}
}
