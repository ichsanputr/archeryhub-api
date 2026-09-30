package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type DocTOCItem struct {
	ID    string `json:"id"`
	Level int    `json:"level"`
	Text  string `json:"text"`
}

type DocLocalized struct {
	Title    string       `json:"title"`
	Excerpt  string       `json:"excerpt"`
	Content  string       `json:"content,omitempty"`
	ReadTime string       `json:"readTime"`
	TOC      []DocTOCItem `json:"toc,omitempty"`
}

type DocJSON struct {
	Slug      string        `json:"slug"`
	Icon      string        `json:"icon"`
	Category  string        `json:"category"`
	Order     int           `json:"order,omitempty"`
	ReadTime  string        `json:"readTime"`
	UpdatedAt string        `json:"updated_at,omitempty"`
	EN        DocLocalized  `json:"en"`
	ID        *DocLocalized `json:"id,omitempty"`
}

type DocResponse struct {
	Slug      string       `json:"slug"`
	Icon      string       `json:"icon"`
	Category  string       `json:"category"`
	Order     int          `json:"order,omitempty"`
	ReadTime  string       `json:"readTime"`
	Title     string       `json:"title"`
	Excerpt   string       `json:"excerpt"`
	Content   string       `json:"content,omitempty"`
	TOC       []DocTOCItem `json:"toc,omitempty"`
	UpdatedAt string       `json:"updated_at,omitempty"`
}

func normalizeDocSlugParam(param string) string {
	// Gin wildcard params (e.g. /*slug) include a leading slash.
	// Normalize both "/archer/archer-profile" and "archer/archer-profile" to "archer/archer-profile".
	slug := strings.TrimSpace(param)
	slug = strings.TrimPrefix(slug, "/")
	slug = strings.TrimSuffix(slug, "/")
	return slug
}

func nestedSlug(category, baseSlug string) string {
	category = strings.TrimSpace(category)
	baseSlug = strings.TrimSpace(baseSlug)
	if category == "" {
		return baseSlug
	}
	return category + "/" + baseSlug
}

// ListDocs returns metadata of all documentation articles sorted by category workflow and order
func ListDocs() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := strings.ToLower(c.DefaultQuery("lang", "en"))
		files, err := os.ReadDir("data/docs")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read docs directory", "details": err.Error()})
			return
		}

		categoryOrder := map[string]int{
			"about":         1,
			"accounts":      2,
			"tournaments":   3,
			"scorekeeper":   4,
			"qualification": 5,
			"elimination":   6,
			"reporting":     7,
		}

		var list []DocResponse
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
				data, err := os.ReadFile(filepath.Join("data/docs", file.Name()))
				if err != nil {
					continue
				}

				var doc DocJSON
				if err := json.Unmarshal(data, &doc); err != nil {
					continue
				}

				localized := doc.EN
				if lang == "id" && doc.ID != nil && doc.ID.Title != "" {
					localized = *doc.ID
				}

				updatedAt := doc.UpdatedAt
				if updatedAt == "" {
					if fileInfo, err := file.Info(); err == nil {
						updatedAt = fileInfo.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00")
					}
				}

				list = append(list, DocResponse{
					Slug:      nestedSlug(doc.Category, doc.Slug),
					Icon:      doc.Icon,
					Category:  doc.Category,
					Order:     doc.Order,
					ReadTime:  doc.ReadTime,
					Title:     localized.Title,
					Excerpt:   localized.Excerpt,
					UpdatedAt: updatedAt,
				})
			}
		}

		sort.Slice(list, func(i, j int) bool {
			catI := categoryOrder[list[i].Category]
			if catI == 0 {
				catI = 99
			}
			catJ := categoryOrder[list[j].Category]
			if catJ == 0 {
				catJ = 99
			}

			if catI != catJ {
				return catI < catJ
			}
			if list[i].Order != list[j].Order {
				return list[i].Order < list[j].Order
			}
			return list[i].Title < list[j].Title
		})

		c.JSON(http.StatusOK, list)
	}
}

// GetDocDetail returns full details of a specific documentation article
func GetDocDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := strings.ToLower(c.DefaultQuery("lang", "en"))
		paramSlug := normalizeDocSlugParam(c.Param("slug"))
		if strings.HasSuffix(paramSlug, "/id") {
			paramSlug = strings.TrimSuffix(paramSlug, "/id")
			if c.Query("lang") == "" {
				lang = "id"
			}
		}
		// We store docs as flat files using the base slug as the filename.
		// URL can be nested: /docs/{category}/{slug}
		baseSlug := path.Base(paramSlug)
		if baseSlug == "." || baseSlug == "/" || baseSlug == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Documentation article not found"})
			return
		}

		// Backward-compatible aliases for renamed docs.
		// Example: /docs/user-roles -> /docs/dashboard/account-types
		if baseSlug == "user-roles" {
			baseSlug = "account-types"
		}
		if baseSlug == "what-is-archers" {
			baseSlug = "what-is-archeris"
		}
		if baseSlug == "organization-profile" {
			baseSlug = "organizer-profile"
		}
		if baseSlug == "participant-reregistration-checkin" {
			baseSlug = "participant-reregistration"
		}
		if baseSlug == "tournament-reporting" {
			baseSlug = "tournament-reports"
		}
		if baseSlug == "shooting-line-scoring-input" {
			baseSlug = "scorekeeper-field-scoring"
		}
		if baseSlug == "qualification-printouts-scorecards" {
			baseSlug = "qualification-scorecards"
		}

		filePath := filepath.Join("data/docs", baseSlug+".json")
		fileInfo, statErr := os.Stat(filePath)
		if statErr != nil && os.IsNotExist(statErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Documentation article not found"})
			return
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read article", "details": err.Error()})
			return
		}

		var doc DocJSON
		if err := json.Unmarshal(data, &doc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse article", "details": err.Error()})
			return
		}

		localized := doc.EN
		if lang == "id" && doc.ID != nil && doc.ID.Title != "" {
			localized = *doc.ID
		}

		updatedAt := doc.UpdatedAt
		if updatedAt == "" && fileInfo != nil {
			updatedAt = fileInfo.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00")
		}

		res := DocResponse{
			Slug:      nestedSlug(doc.Category, doc.Slug),
			Icon:      doc.Icon,
			Category:  doc.Category,
			ReadTime:  doc.ReadTime,
			Title:     localized.Title,
			Excerpt:   localized.Excerpt,
			Content:   localized.Content,
			TOC:       localized.TOC,
			UpdatedAt: updatedAt,
		}

		c.JSON(http.StatusOK, res)
	}
}

// DeleteDoc deletes a specific documentation article and its related comments
func DeleteDoc(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramSlug := normalizeDocSlugParam(c.Param("slug"))
		baseSlug := path.Base(paramSlug)
		if baseSlug == "." || baseSlug == "/" || baseSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid documentation slug"})
			return
		}

		if baseSlug == "user-roles" {
			baseSlug = "account-types"
		}

		filePath := filepath.Join("data/docs", baseSlug+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Documentation article not found"})
			return
		}

		if err := os.Remove(filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete documentation article", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Documentation article deleted successfully",
			"slug":    baseSlug,
		})
	}
}

// ReplaceDocImage handles smart image replacement for documentation in dev / localhost mode.
// Route: POST /docs/replace-image
func ReplaceDocImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded", "details": err.Error()})
			return
		}

		rawSlug := c.PostForm("slug")
		baseSlug := path.Base(normalizeDocSlugParam(rawSlug))
		if baseSlug == "." || baseSlug == "/" || baseSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid documentation slug"})
			return
		}

		if baseSlug == "user-roles" {
			baseSlug = "account-types"
		}

		oldSrc := strings.TrimSpace(c.PostForm("old_src"))

		fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if fileExt == "" {
			fileExt = ".webp"
		}

		// Generate clean filename
		cleanBaseName := strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
		cleanBaseName = strings.ReplaceAll(cleanBaseName, " ", "-")
		cleanBaseName = strings.ToLower(cleanBaseName)
		cleanBaseName = regexp.MustCompile(`[^a-z0-9\-_]+`).ReplaceAllString(cleanBaseName, "-")
		cleanBaseName = regexp.MustCompile(`-+`).ReplaceAllString(cleanBaseName, "-")
		cleanBaseName = strings.Trim(cleanBaseName, "-")
		if cleanBaseName == "" || cleanBaseName == "image" {
			cleanBaseName = baseSlug
		}

		newFilename := fmt.Sprintf("%s-%d%s", cleanBaseName, time.Now().Unix(), fileExt)

		// Read uploaded bytes
		srcFile, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file", "details": err.Error()})
			return
		}
		defer srcFile.Close()

		fileBytes, err := io.ReadAll(srcFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file bytes", "details": err.Error()})
			return
		}

		// Destination folders: app/public/docs, api/public/docs, and api/media
		targetDirs := []string{
			filepath.Join("..", "app", "public", "docs"),
			filepath.Join("app", "public", "docs"),
			filepath.Join("public", "docs"),
			"media",
			filepath.Join("api", "media"),
		}

		for _, dir := range targetDirs {
			_ = os.MkdirAll(dir, 0755)
			destPath := filepath.Join(dir, newFilename)
			_ = os.WriteFile(destPath, fileBytes, 0644)
		}

		newSrc := "/docs/" + newFilename

		// Optional occurrence index (0-based) for the specific image instance
		occurrenceIndex := -1
		if occStr := c.PostForm("occurrence_index"); occStr != "" {
			if parsed, err := strconv.Atoi(occStr); err == nil {
				occurrenceIndex = parsed
			}
		}

		replaceNthOccurrence := func(content, oldStr, newStr string, occ int) string {
			if occ < 0 {
				return strings.ReplaceAll(content, oldStr, newStr)
			}
			count := 0
			start := 0
			for {
				idx := strings.Index(content[start:], oldStr)
				if idx == -1 {
					break
				}
				actualIdx := start + idx
				if count == occ {
					return content[:actualIdx] + newStr + content[actualIdx+len(oldStr):]
				}
				count++
				start = actualIdx + len(oldStr)
			}
			return strings.Replace(content, oldStr, newStr, 1)
		}

		// Find data/docs/{baseSlug}.json across possible directories
		possibleDocPaths := []string{
			filepath.Join("data", "docs", baseSlug+".json"),
			filepath.Join("api", "data", "docs", baseSlug+".json"),
			filepath.Join("..", "api", "data", "docs", baseSlug+".json"),
		}

		for _, jsonFilePath := range possibleDocPaths {
			if _, statErr := os.Stat(jsonFilePath); statErr == nil {
				jsonData, readErr := os.ReadFile(jsonFilePath)
				if readErr == nil {
					var rawDoc map[string]interface{}
					if jsonErr := json.Unmarshal(jsonData, &rawDoc); jsonErr == nil {
						// Update UpdatedAt
						rawDoc["updated_at"] = time.Now().UTC().Format("2006-01-02T15:04:05Z07:00")

						// Helper to replace oldSrc in a content string
						replaceInContent := func(fieldMap map[string]interface{}, key string) {
							if val, ok := fieldMap[key].(string); ok && val != "" {
								if oldSrc != "" {
									fieldMap[key] = replaceNthOccurrence(val, oldSrc, newSrc, occurrenceIndex)
								}
							}
						}

						if enMap, ok := rawDoc["en"].(map[string]interface{}); ok {
							replaceInContent(enMap, "content")
						}
						if idMap, ok := rawDoc["id"].(map[string]interface{}); ok {
							replaceInContent(idMap, "content")
						}

						updatedBytes, marshalErr := json.MarshalIndent(rawDoc, "", "  ")
						if marshalErr == nil {
							_ = os.WriteFile(jsonFilePath, updatedBytes, 0644)
						}
					}
				}
				break
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "success",
			"message":  "Gambar dokumentasi berhasil diganti dan disimpan ke data/docs",
			"new_src":  newSrc,
			"old_src":  oldSrc,
			"slug":     baseSlug,
			"filename": newFilename,
		})
	}
}

