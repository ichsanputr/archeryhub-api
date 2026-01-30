package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// GetCities returns a list of Indonesian cities from local JSON file
func GetCities() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get path to cities.json
		// Assuming the executable is running from the api directory or root
		// We'll try relative to the current working directory first
		path := filepath.Join("data", "cities.json")
		
		file, err := os.ReadFile(path)
		if err != nil {
			// Try one level up if not found (for different run configurations)
			path = filepath.Join("api", "data", "cities.json")
			file, err = os.ReadFile(path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load cities data", "details": err.Error()})
				return
			}
		}

		var cities []interface{}
		if err := json.Unmarshal(file, &cities); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse cities data"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": cities,
		})
	}
}
