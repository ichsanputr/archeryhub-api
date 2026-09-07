package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ApiResponse represents the standardized API response envelope
type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ApiError   `json:"error,omitempty"`
	Meta    *ApiMeta    `json:"meta,omitempty"`
}

// ApiError represents structured error details
type ApiError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ApiMeta represents pagination and metadata details
type ApiMeta struct {
	Total       int `json:"total,omitempty"`
	Page        int `json:"page,omitempty"`
	Limit       int `json:"limit,omitempty"`
	UnreadCount int `json:"unread_count,omitempty"`
}

// Success sends a standardized success response
func Success(c *gin.Context, statusCode int, data interface{}, message string, meta ...*ApiMeta) {
	resp := ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	if len(meta) > 0 && meta[0] != nil {
		resp.Meta = meta[0]
	}
	c.JSON(statusCode, resp)
}

// Error sends a standardized error response
func Error(c *gin.Context, statusCode int, code string, message string, details ...interface{}) {
	var detailVal interface{}
	if len(details) > 0 {
		detailVal = details[0]
	}

	c.JSON(statusCode, ApiResponse{
		Success: false,
		Error: &ApiError{
			Code:    code,
			Message: message,
			Details: detailVal,
		},
	})
}

// GlobalRecoveryMiddleware recovers from panics, logs stack trace, and returns standard error
func GlobalRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logrus.WithFields(logrus.Fields{
					"error":  err,
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
					"ip":     c.ClientIP(),
				}).Error("[PANIC RECOVERED] Unhandled runtime panic in request handler")

				Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Terjadi kesalahan internal pada server.")
				c.Abort()
			}
		}()
		c.Next()
	}
}
