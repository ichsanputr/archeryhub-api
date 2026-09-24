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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type RootDocUpsertRequest struct {
	Slug      string       `json:"slug" binding:"required"`
	Icon      string       `json:"icon"`
	Category  string       `json:"category" binding:"required"`
	Order     int          `json:"order"`
	ReadTime  string       `json:"readTime"`
	EN        DocLocalized `json:"en"`
	ID        DocLocalized `json:"id"`
	OldSlug   string       `json:"old_slug"`
}

// Helper to auto-generate Table of Contents from HTML content <h2> headings
func generateTOCFromHTML(htmlContent string) []DocTOCItem {
	var toc []DocTOCItem
	if strings.TrimSpace(htmlContent) == "" {
		return toc
	}

	// Regex for <h2> headings: matches <h2 ...>...</h2>
	re := regexp.MustCompile(`(?i)<h2([^>]*)>(.*?)<\/h2>`)
	idRe := regexp.MustCompile(`(?i)id=["']([^"']+)["']`)
	tagStripRe := regexp.MustCompile(`<[^>]*>`)

	matches := re.FindAllStringSubmatch(htmlContent, -1)
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}
		attrs := match[1]
		inner := match[2]

		cleanText := strings.TrimSpace(tagStripRe.ReplaceAllString(inner, ""))
		if cleanText == "" {
			continue
		}

		headingID := ""
		idMatch := idRe.FindStringSubmatch(attrs)
		if len(idMatch) > 1 {
			headingID = idMatch[1]
		} else {
			// Auto slugify text for ID
			headingID = strings.ToLower(cleanText)
			headingID = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(headingID, "-")
			headingID = strings.Trim(headingID, "-")
			if headingID == "" {
				headingID = fmt.Sprintf("heading-%d", i+1)
			}
		}

		toc = append(toc, DocTOCItem{
			ID:    headingID,
			Level: 2,
			Text:  cleanText,
		})
	}
	return toc
}

// Calculate approximate reading time based on word count
func calculateDocReadTime(contentEn, contentId string) string {
	combined := contentEn + " " + contentId
	clean := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(combined, " ")
	words := len(strings.Fields(clean))
	mins := (words / 2) / 180 // average between EN and ID
	if mins < 1 {
		mins = 1
	}
	return fmt.Sprintf("%d min", mins)
}

// RootListDocs returns full list of all docs files with metadata & translation status
func RootListDocs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		docsDir := "data/docs"
		files, err := os.ReadDir(docsDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read docs directory", "details": err.Error()})
			return
		}

		categoryOrder := map[string]int{
			"accounts":      1,
			"tournaments":   2,
			"scorekeeper":   3,
			"qualification": 4,
			"elimination":   5,
			"reporting":     6,
		}

		type RootDocItem struct {
			Slug        string `json:"slug"`
			BaseSlug    string `json:"base_slug"`
			Icon        string `json:"icon"`
			Category    string `json:"category"`
			Order       int    `json:"order"`
			ReadTime    string `json:"readTime"`
			TitleEN     string `json:"title_en"`
			TitleID     string `json:"title_id"`
			ExcerptEN   string `json:"excerpt_en"`
			ExcerptID   string `json:"excerpt_id"`
			HasEN       bool   `json:"has_en"`
			HasID       bool   `json:"has_id"`
			UpdatedAt   string `json:"updated_at"`
		}

		var list []RootDocItem
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
				baseFilename := strings.TrimSuffix(file.Name(), ".json")
				data, err := os.ReadFile(filepath.Join(docsDir, file.Name()))
				if err != nil {
					continue
				}

				var doc DocJSON
				if err := json.Unmarshal(data, &doc); err != nil {
					continue
				}

				slug := doc.Slug
				if slug == "" {
					slug = baseFilename
				}

				hasEN := strings.TrimSpace(doc.EN.Title) != ""
				hasID := doc.ID != nil && strings.TrimSpace(doc.ID.Title) != ""

				titleEN := doc.EN.Title
				excerptEN := doc.EN.Excerpt
				titleID := ""
				excerptID := ""
				if doc.ID != nil {
					titleID = doc.ID.Title
					excerptID = doc.ID.Excerpt
				}

				updatedAt := doc.UpdatedAt
				if updatedAt == "" {
					if fileInfo, err := file.Info(); err == nil {
						updatedAt = fileInfo.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00")
					}
				}

				list = append(list, RootDocItem{
					Slug:      nestedSlug(doc.Category, slug),
					BaseSlug:  slug,
					Icon:      doc.Icon,
					Category:  doc.Category,
					Order:     doc.Order,
					ReadTime:  doc.ReadTime,
					TitleEN:   titleEN,
					TitleID:   titleID,
					ExcerptEN: excerptEN,
					ExcerptID: excerptID,
					HasEN:     hasEN,
					HasID:     hasID,
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
			return list[i].BaseSlug < list[j].BaseSlug
		})

		c.JSON(http.StatusOK, gin.H{
			"docs":  list,
			"total": len(list),
		})
	}
}

// RootGetDoc retrieves raw DocJSON file content for editing
func RootGetDoc(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramSlug := normalizeDocSlugParam(c.Param("slug"))
		baseSlug := path.Base(paramSlug)
		if baseSlug == "." || baseSlug == "/" || baseSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid documentation slug"})
			return
		}

		filePath := filepath.Join("data/docs", baseSlug+".json")
		data, err := os.ReadFile(filePath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Documentation article not found", "details": err.Error()})
			return
		}

		var doc DocJSON
		if err := json.Unmarshal(data, &doc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse article JSON", "details": err.Error()})
			return
		}

		if doc.Slug == "" {
			doc.Slug = baseSlug
		}

		c.JSON(http.StatusOK, doc)
	}
}

// RootCreateDoc creates a new documentation JSON file
func RootCreateDoc(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RootDocUpsertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document data: " + err.Error()})
			return
		}

		cleanSlug := strings.ToLower(strings.TrimSpace(req.Slug))
		cleanSlug = regexp.MustCompile(`[^a-z0-9\-_]+`).ReplaceAllString(cleanSlug, "-")
		cleanSlug = strings.Trim(cleanSlug, "-")

		if cleanSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Slug cannot be empty"})
			return
		}

		filePath := filepath.Join("data/docs", cleanSlug+".json")
		if _, err := os.Stat(filePath); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "A document with this slug already exists"})
			return
		}

		// Ensure TOC is populated from H2 headings if empty
		if len(req.EN.TOC) == 0 && req.EN.Content != "" {
			req.EN.TOC = generateTOCFromHTML(req.EN.Content)
		}
		if len(req.ID.TOC) == 0 && req.ID.Content != "" {
			req.ID.TOC = generateTOCFromHTML(req.ID.Content)
		}

		// Calculate readTime if empty
		if strings.TrimSpace(req.ReadTime) == "" {
			req.ReadTime = calculateDocReadTime(req.EN.Content, req.ID.Content)
		}

		doc := DocJSON{
			Slug:      cleanSlug,
			Icon:      req.Icon,
			Category:  req.Category,
			Order:     req.Order,
			ReadTime:  req.ReadTime,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			EN:        req.EN,
			ID:        &req.ID,
		}

		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to format JSON", "details": err.Error()})
			return
		}

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document file", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Documentation created successfully",
			"slug":    cleanSlug,
			"doc":      doc,
		})
	}
}

// RootUpdateDoc updates an existing documentation JSON file
func RootUpdateDoc(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramSlug := normalizeDocSlugParam(c.Param("slug"))
		currentBaseSlug := path.Base(paramSlug)

		var req RootDocUpsertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document data: " + err.Error()})
			return
		}

		newSlug := strings.ToLower(strings.TrimSpace(req.Slug))
		newSlug = regexp.MustCompile(`[^a-z0-9\-_]+`).ReplaceAllString(newSlug, "-")
		newSlug = strings.Trim(newSlug, "-")
		if newSlug == "" {
			newSlug = currentBaseSlug
		}

		currentFilePath := filepath.Join("data/docs", currentBaseSlug+".json")
		if _, err := os.Stat(currentFilePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found to update"})
			return
		}

		// If slug was renamed, check if target already exists
		newFilePath := filepath.Join("data/docs", newSlug+".json")
		if newSlug != currentBaseSlug {
			if _, err := os.Stat(newFilePath); err == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Target slug already exists"})
				return
			}
		}

		// Auto-generate TOC from headings
		req.EN.TOC = generateTOCFromHTML(req.EN.Content)
		req.ID.TOC = generateTOCFromHTML(req.ID.Content)

		if strings.TrimSpace(req.ReadTime) == "" {
			req.ReadTime = calculateDocReadTime(req.EN.Content, req.ID.Content)
		}

		doc := DocJSON{
			Slug:      newSlug,
			Icon:      req.Icon,
			Category:  req.Category,
			Order:     req.Order,
			ReadTime:  req.ReadTime,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			EN:        req.EN,
			ID:        &req.ID,
		}

		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to format JSON", "details": err.Error()})
			return
		}

		if err := os.WriteFile(newFilePath, data, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write document file", "details": err.Error()})
			return
		}

		// If slug changed, remove the old file
		if newSlug != currentBaseSlug {
			_ = os.Remove(currentFilePath)
			// Update comment references in DB
			if db != nil {
				_, _ = db.Exec("UPDATE docs_comments SET doc_slug = ? WHERE doc_slug = ?", newSlug, currentBaseSlug)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Documentation updated successfully",
			"slug":    newSlug,
			"doc":      doc,
		})
	}
}

// RootUploadDocImage handles uploading screenshots / illustrations for documentation
func RootUploadDocImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		file, header, err = c.Request.FormFile("file")
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided in request"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" && ext != ".svg" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supported image formats: JPG, PNG, WEBP, GIF, SVG"})
		return
	}

	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image file size exceeds maximum limit of 10MB"})
		return
	}

	uploadDir := "public/docs"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	rawFilename := strings.TrimSuffix(filepath.Base(header.Filename), ext)
	cleanName := regexp.MustCompile(`[^a-z0-9\-_]+`).ReplaceAllString(strings.ToLower(rawFilename), "-")
	cleanName = strings.Trim(cleanName, "-")
	if cleanName == "" {
		cleanName = "doc-asset"
	}

	newFilename := fmt.Sprintf("%s-%d%s", cleanName, time.Now().Unix(), ext)
	dstPath := filepath.Join(uploadDir, newFilename)

	dst, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create target file on server"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file content"})
		return
	}

	imageURL := "/docs/" + newFilename
	c.JSON(http.StatusOK, gin.H{
		"url":      imageURL,
		"filename": newFilename,
		"message":  "Image uploaded successfully",
	})
}
