package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"archeryhub/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Device struct {
	ID               string  `db:"id" json:"id"`
	TournamentID     string  `db:"tournament_id" json:"tournament_id"`
	DeviceCode       string  `db:"device_code" json:"device_code"`
	DeviceName       *string `db:"device_name" json:"device_name"`
	DeviceType       string  `db:"device_type" json:"device_type"`
	PIN              *string `db:"pin" json:"pin"`
	QRPayload        *string `db:"qr_payload" json:"qr_payload"`
	TargetAssignment *string `db:"target_assignment" json:"target_assignment"`
	Session          *int    `db:"session" json:"session"`
	LastSync         *string `db:"last_sync" json:"last_sync"`
	Status           string  `db:"status" json:"status"`
	CreatedAt        string  `db:"created_at" json:"created_at"`
}

// generatePIN generates a random 6-digit PIN
func generatePIN() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2])%1000000)
}

// generateDeviceCode generates a unique device code
func generateDeviceCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}

// RegisterDevice registers a new scoring device
func RegisterDevice(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TournamentID     string  `json:"tournament_id" binding:"required"`
			DeviceName       *string `json:"device_name"`
			DeviceType       string  `json:"device_type" binding:"required"`
			TargetAssignment *string `json:"target_assignment"`
			Session          *int    `json:"session"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		deviceID := uuid.New().String()
		deviceCode := generateDeviceCode()
		pin := generatePIN()

		// Generate QR payload (format: TOURNAMENT_ID|DEVICE_CODE|PIN)
		qrPayload := fmt.Sprintf("%s|%s|%s", req.TournamentID, deviceCode, pin)

		_, err := db.Exec(`
			INSERT INTO devices 
			(id, tournament_id, device_code, device_name, device_type, pin, qr_payload, 
			 target_assignment, session, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active')
		`, deviceID, req.TournamentID, deviceCode, req.DeviceName, req.DeviceType,
			pin, qrPayload, req.TargetAssignment, req.Session)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), req.TournamentID, "device_registered", "device", deviceID, "Registered scoring device: "+deviceCode, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":          deviceID,
			"device_code": deviceCode,
			"pin":         pin,
			"qr_payload":  qrPayload,
			"message":     "Device registered successfully",
		})
	}
}

// GetDevices returns all devices for a tournament
func GetDevices(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")

		var devices []Device
		err := db.Select(&devices, `
			SELECT * FROM devices 
			WHERE tournament_id = ? 
			ORDER BY created_at DESC
		`, tournamentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch devices"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"devices": devices,
			"total":   len(devices),
		})
	}
}

// GetDeviceConfig returns device configuration (for mobile apps)
func GetDeviceConfig(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceCode := c.Param("deviceCode")
		pin := c.Query("pin")

		var device Device
		err := db.Get(&device, `
			SELECT * FROM devices 
			WHERE device_code = ? AND pin = ? AND status = 'active'
		`, deviceCode, pin)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid device code or PIN"})
			return
		}

		c.JSON(http.StatusOK, device)
	}
}

// UpdateDeviceStatus updates device status
func UpdateDeviceStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")

		var req struct {
			Status string `json:"status" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec("UPDATE devices SET status = ? WHERE id = ?", req.Status, deviceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device status"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "device_status_updated", "device", deviceID, "Updated device status to: "+req.Status, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Device status updated successfully"})
	}
}

// SyncDevice updates last sync timestamp
func SyncDevice(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceCode := c.Param("deviceCode")

		_, err := db.Exec("UPDATE devices SET last_sync = NOW() WHERE device_code = ?", deviceCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync device"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Device synced successfully"})
	}
}

// GetDeviceQRCode returns a QR code image for the device
func GetDeviceQRCode(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")

		var device struct {
			QRPayload *string `db:"qr_payload"`
		}
		err := db.Get(&device, "SELECT qr_payload FROM devices WHERE id = ?", deviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}

		if device.QRPayload == nil || *device.QRPayload == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Device has no QR payload"})
			return
		}

		png, err := utils.GenerateQRCode(*device.QRPayload, 256)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
			return
		}

		c.Data(http.StatusOK, "image/png", png)
	}
}
