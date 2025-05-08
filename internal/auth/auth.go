package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("no auth token provided")
	}
	apiKey := strings.TrimPrefix(authHeader, "ApiKey ")
	return apiKey, nil
}

func HashPassword(password string) (string, error) {
	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashPass), nil
}

func CheckPasswordHash(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	if tokenSecret == "" {
		return "", fmt.Errorf("no secret entered")
	}
	if len(tokenSecret) < 32 {
		return "", fmt.Errorf("enter a secure secret(minimum 32 bytes)")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "yappy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	})
	wt, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return wt, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := parsedToken.Claims.(*jwt.RegisteredClaims)
	if !ok || !parsedToken.Valid {
		return uuid.Nil, jwt.ErrTokenMalformed
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("no auth token provided")
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	return tokenString, nil
}

func MakeRefreshToken() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	token := hex.EncodeToString(randomBytes)
	return token, nil
}
