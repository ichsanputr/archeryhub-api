package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MobileHello godoc
// @Summary Welcome to Mobile API
// @Description Provides a welcome message for mobile clients
// @Tags mobile
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string
// @Router /mobile/hello [get]
func MobileHello() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to ArcheryHub Mobile API",
			"status":  "active",
		})
	}
}
