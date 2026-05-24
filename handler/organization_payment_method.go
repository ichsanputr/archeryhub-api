package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type OrganizationPaymentMethod struct {
	UUID           string     `json:"id" db:"uuid"`
	OrganizationID string     `json:"organization_id" db:"organization_id"`
	BankName       string     `json:"bank_name" db:"bank_name"`
	AccountNumber  string     `json:"account_number" db:"account_number"`
	AccountName    string     `json:"account_name" db:"account_name"`
	IsPrimary      bool       `json:"is_primary" db:"is_primary"`
	Status         string     `json:"status" db:"status"`
	CustomName     *string    `json:"custom_name" db:"custom_name"`
	Type           string     `json:"type" db:"type"`
	Instructions   *string    `json:"instructions" db:"instructions"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

func GetOrganizationPaymentMethods(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var methods []OrganizationPaymentMethod
		err := db.Select(&methods, "SELECT * FROM organization_payment_methods WHERE organization_id = ? ORDER BY is_primary DESC, created_at DESC", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data metode pembayaran"})
			return
		}

		c.JSON(http.StatusOK, methods)
	}
}

func CreateOrganizationPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			BankName      string  `json:"bank_name" binding:"required"`
			AccountNumber string  `json:"account_number" binding:"required"`
			AccountName   string  `json:"account_name" binding:"required"`
			IsPrimary     bool    `json:"is_primary"`
			CustomName    *string `json:"custom_name"`
			Type          string  `json:"type"`
			Instructions  *string `json:"instructions"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		methodID := uuid.New().String()

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		if req.IsPrimary {
			_, err = tx.Exec("UPDATE organization_payment_methods SET is_primary = FALSE WHERE organization_id = ?", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset status utama"})
				return
			}
		}

		if req.Type == "" {
			req.Type = "bank"
		}

		_, err = tx.Exec(`
			INSERT INTO organization_payment_methods (uuid, organization_id, bank_name, account_number, account_name, is_primary, custom_name, type, instructions)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, methodID, userID, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary, req.CustomName, req.Type, req.Instructions)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat metode pembayaran: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Metode pembayaran berhasil ditambahkan", "id": methodID})
	}
}

func UpdateOrganizationPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		methodID := c.Param("id")

		var req struct {
			BankName      string  `json:"bank_name" binding:"required"`
			AccountNumber string  `json:"account_number" binding:"required"`
			AccountName   string  `json:"account_name" binding:"required"`
			IsPrimary     bool    `json:"is_primary"`
			CustomName    *string `json:"custom_name"`
			Type          string  `json:"type"`
			Instructions  *string `json:"instructions"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		if req.IsPrimary {
			_, err = tx.Exec("UPDATE organization_payment_methods SET is_primary = FALSE WHERE organization_id = ?", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset status utama"})
				return
			}
		}

		if req.Type == "" {
			req.Type = "bank"
		}

		_, err = tx.Exec(`
			UPDATE organization_payment_methods 
			SET bank_name = ?, account_number = ?, account_name = ?, is_primary = ?, custom_name = ?, type = ?, instructions = ?, updated_at = NOW()
			WHERE uuid = ? AND organization_id = ?
		`, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary, req.CustomName, req.Type, req.Instructions, methodID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui metode pembayaran: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Metode pembayaran berhasil diperbarui"})
	}
}

func DeleteOrganizationPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		methodID := c.Param("id")

		_, err := db.Exec("DELETE FROM organization_payment_methods WHERE uuid = ? AND organization_id = ?", methodID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus metode pembayaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Metode pembayaran berhasil dihapus"})
	}
}

func SyncOrganizationPaymentMethods(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req []struct {
			UUID          string  `json:"id"`
			BankName      string  `json:"bank_name"`
			AccountNumber string  `json:"account_number"`
			AccountName   string  `json:"account_name"`
			IsPrimary     bool    `json:"is_primary"`
			CustomName    *string `json:"custom_name"`
			Type          string  `json:"type"`
			Instructions  *string `json:"instructions"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var existing []string
		err = tx.Select(&existing, "SELECT uuid FROM organization_payment_methods WHERE organization_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa metode pembayaran lama"})
			return
		}

		existingMap := make(map[string]bool)
		for _, uuidVal := range existing {
			existingMap[uuidVal] = true
		}

		processedIDs := make(map[string]bool)

		hasPrimary := false
		for _, item := range req {
			if item.IsPrimary {
				hasPrimary = true
				break
			}
		}
		if hasPrimary {
			_, err = tx.Exec("UPDATE organization_payment_methods SET is_primary = FALSE WHERE organization_id = ?", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset status utama"})
				return
			}
		}

		for _, item := range req {
			if item.BankName == "" || item.AccountNumber == "" || item.AccountName == "" {
				continue
			}

			itemType := item.Type
			if itemType == "" {
				itemType = "bank"
			}

			if item.UUID != "" && len(item.UUID) == 36 && existingMap[item.UUID] {
				_, err = tx.Exec(`
					UPDATE organization_payment_methods 
					SET bank_name = ?, account_number = ?, account_name = ?, is_primary = ?, custom_name = ?, type = ?, instructions = ?, updated_at = NOW()
					WHERE uuid = ? AND organization_id = ?
				`, item.BankName, item.AccountNumber, item.AccountName, item.IsPrimary, item.CustomName, itemType, item.Instructions, item.UUID, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui metode pembayaran: " + err.Error()})
					return
				}
				processedIDs[item.UUID] = true
			} else {
				newUUID := uuid.New().String()
				_, err = tx.Exec(`
					INSERT INTO organization_payment_methods (uuid, organization_id, bank_name, account_number, account_name, is_primary, custom_name, type, instructions)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, newUUID, userID, item.BankName, item.AccountNumber, item.AccountName, item.IsPrimary, item.CustomName, itemType, item.Instructions)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat metode pembayaran: " + err.Error()})
					return
				}
			}
		}

		for _, uuidVal := range existing {
			if !processedIDs[uuidVal] {
				_, err = tx.Exec("DELETE FROM organization_payment_methods WHERE uuid = ? AND organization_id = ?", uuidVal, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus metode pembayaran lama"})
					return
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Metode pembayaran berhasil disinkronisasi"})
	}
}
