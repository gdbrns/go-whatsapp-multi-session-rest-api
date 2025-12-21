package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
)

// JWTSecretKey for signing device tokens
// REQUIRED: Application will panic if not set
var JWTSecretKey string

func init() {
	// JWT_SECRET_KEY is REQUIRED (min 32 chars) - app will panic if not configured
	JWTSecretKey = env.MustGetEnvString("JWT_SECRET_KEY")
}

// DeviceTokenClaims represents the claims in a device JWT
type DeviceTokenClaims struct {
	DeviceID   string `json:"device_id"`
	APIKeyID   int64  `json:"api_key_id"`
	JID        string `json:"jid,omitempty"` // WhatsApp JID, may be empty initially
	JWTVersion int    `json:"version"`       // For token invalidation
	jwt.RegisteredClaims
}

// GenerateDeviceToken creates a long-lived JWT for a device
// The token does not expire, but can be invalidated by incrementing jwt_version
func GenerateDeviceToken(deviceID string, apiKeyID int64, jid string, jwtVersion int) (string, error) {
	if JWTSecretKey == "" {
		return "", errors.New("JWT_SECRET_KEY not configured")
	}

	claims := DeviceTokenClaims{
		DeviceID:   deviceID,
		APIKeyID:   apiKeyID,
		JID:        jid,
		JWTVersion: jwtVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   deviceID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			// No ExpiresAt - token never expires
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecretKey))
}

// ValidateDeviceToken validates a device JWT and returns the claims
func ValidateDeviceToken(tokenString string) (*DeviceTokenClaims, error) {
	if JWTSecretKey == "" {
		return nil, errors.New("JWT_SECRET_KEY not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &DeviceTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(JWTSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*DeviceTokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}
