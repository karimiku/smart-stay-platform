package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// jwtSecret is the secret key for signing JWT tokens
	// In production, this should be loaded from environment variables or secrets manager
	jwtSecret = []byte(getJWTSecret())
)

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token with user information
func GenerateToken(userID, role, email string, expiresIn time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiresIn)
	
	claims := &Claims{
		UserID: userID,
		Role:   role,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "smart-stay-platform",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// getJWTSecret retrieves JWT secret from environment variable
// JWT_SECRET MUST be set via environment variable
// For local development, create .env file from .env.example
// Generate a secure secret: openssl rand -base64 32
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is required. Please set it in .env file or environment.")
	}
	return secret
}

