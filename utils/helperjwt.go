package utils

import (
	"MonCR/models"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type Claims struct {
	UserID       uint   `json:"user_id"`
    Email        string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateJWT(user *models.Users) (string, error) {
	expirationTime := time.Now().Add(24*time.Hour)

	claims := &Claims{
		UserID: user.ID,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expirationTime),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "Avocycle",
            Subject:   user.Email,
        },
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT validates and parses JWT token
func ValidateJWT(tokenString string) (*Claims, error) {
    // Check if JWT secret is set
    if len(tokenString) == 0 {
        return nil, errors.New("JWT secret key is not set")
    }

    // Parse token
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Validate signing method
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("invalid signing method")
        }
        return jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    // Extract claims
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, errors.New("invalid token")
}

// RefreshJWT generates a new JWT token with extended expiration
func RefreshJWT(oldTokenString string) (string, error) {
    // Validate old token
    claims, err := ValidateJWT(oldTokenString)
    if err != nil {
        return "", err
    }

    // Check if token is expired
    if time.Until(claims.ExpiresAt.Time) < 0 {
        return "", errors.New("token has expired")
    }

    // Create new token with extended expiration
    newExpirationTime := time.Now().Add(24 * time.Hour)
    
    newClaims := &Claims{
        UserID:       claims.UserID,
        Email:        claims.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(newExpirationTime),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "Avocycle",
            Subject:   claims.Email,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
    return token.SignedString(jwtSecret)
}