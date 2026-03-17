package mobile

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func MobileHello() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Selamat datang di API Mobile ArcheryHub",
			"status":  "active",
		})
	}
}
