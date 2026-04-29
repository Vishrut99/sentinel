package jwtutil

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims are JWT claims used across the API.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// SignToken creates a signed JWT valid for 24 hours.
func SignToken(userID uuid.UUID, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("jwtutil.SignToken: JWT_SECRET is required")
	}

	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("jwtutil.SignToken: %w", err)
	}

	return signed, nil
}

// VerifyToken parses and validates a JWT token.
func VerifyToken(tokenStr string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("jwtutil.VerifyToken: JWT_SECRET is required")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwtutil.VerifyToken: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("jwtutil.VerifyToken: invalid token")
	}
	if claims.UserID == uuid.Nil {
		return nil, fmt.Errorf("jwtutil.VerifyToken: missing user_id claim")
	}
	if claims.Role == "" {
		return nil, fmt.Errorf("jwtutil.VerifyToken: missing role claim")
	}

	return claims, nil
}
