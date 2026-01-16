package utils

import (
	"github.com/skip2/go-qrcode"
)

// GenerateQRCode generates a QR code as a byte slice (PNG)
func GenerateQRCode(content string, size int) ([]byte, error) {
	var png []byte
	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return nil, err
	}
	return png, nil
}
