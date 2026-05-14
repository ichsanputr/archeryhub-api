package mobile

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func generateJWT(userID, email, role, userType, name, avatar, orgUUID string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("Archeris-secret-key-change-in-production")
	}

	claims := jwt.MapClaims{
		"user_id":   userID,
		"email":     email,
		"name":      name,
		"avatar":    avatar,
		"role":      role,
		"user_type": userType,
		"org_id":    orgUUID,
		"exp":       time.Now().Add(time.Hour * 24 * 60).Unix(), // 60 days
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

