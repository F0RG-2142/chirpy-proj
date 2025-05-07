package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		expectError  bool
		validateHash bool
	}{
		{
			name:         "Valid Password",
			password:     "securepassword123",
			expectError:  false,
			validateHash: true,
		},
		{
			name:         "Empty Password",
			password:     "",
			expectError:  false,
			validateHash: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.validateHash && hash == "" {
				t.Error("expected non-empty hash, got empty")
			}
			if tt.validateHash {
				err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password))
				if err != nil {
					t.Errorf("hash validation failed: %v", err)
				}
			}
		})
	}
}

func TestCheckPasswordHash(t *testing.T) {
	// Generate a known hash for testing
	knownPassword := "testpassword"
	knownHash, err := bcrypt.GenerateFromPassword([]byte(knownPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate test hash: %v", err)
	}

	tests := []struct {
		name        string
		hash        string
		password    string
		expectError bool
	}{
		{
			name:        "Correct Password",
			hash:        string(knownHash),
			password:    knownPassword,
			expectError: false,
		},
		{
			name:        "Incorrect Password",
			hash:        string(knownHash),
			password:    "wrongpassword",
			expectError: true,
		},
		{
			name:        "Invalid Hash",
			hash:        "invalid-hash",
			password:    knownPassword,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPasswordHash(tt.hash, tt.password)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenSecret := "hrXkkQDZ5V5smyXvP46dBnVMUuAiAv7ZUGnKCUUx" // 64 bytes
	expiresIn := 1 * time.Hour

	tests := []struct {
		name          string
		userID        uuid.UUID
		tokenSecret   string
		expiresIn     time.Duration
		expectError   bool
		validateToken bool
	}{
		{
			name:          "Valid JWT",
			userID:        userID,
			tokenSecret:   tokenSecret,
			expiresIn:     expiresIn,
			expectError:   false,
			validateToken: true,
		},
		{
			name:          "Empty Secret",
			userID:        userID,
			tokenSecret:   "",
			expiresIn:     expiresIn,
			expectError:   true,
			validateToken: false,
		},
		{
			name:          "Zero Expiration",
			userID:        userID,
			tokenSecret:   tokenSecret,
			expiresIn:     0,
			expectError:   false,
			validateToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := MakeJWT(tt.userID, tt.tokenSecret, tt.expiresIn)
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v. Current token: %v", err, tokenSecret)
			}
			if tt.validateToken && token == "" {
				t.Error("expected non-empty token, got empty")
			}
			if tt.validateToken {
				// Validate the token using ValidateJWT
				parsedUserID, err := ValidateJWT(token, tt.tokenSecret)
				if err != nil {
					t.Errorf("token validation failed: %v", err)
				}
				if parsedUserID != tt.userID {
					t.Errorf("expected userID %v, got %v", tt.userID, parsedUserID)
				}
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenSecret := "test-secret-12345678901234567890123456789012" // 32 bytes
	expiresIn := 1 * time.Hour

	// Generate a valid token for testing
	validToken, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	// Generate an expired token
	expiredToken, err := MakeJWT(userID, tokenSecret, -1*time.Second)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	// Generate a token with wrong secret
	wrongSecret := "wrong-secret-1234567890123456789012345678"
	wrongSecretToken, err := MakeJWT(userID, wrongSecret, expiresIn)
	if err != nil {
		t.Fatalf("failed to generate wrong secret token: %v", err)
	}

	tests := []struct {
		name           string
		tokenString    string
		tokenSecret    string
		expectError    bool
		expectedUserID uuid.UUID
	}{
		{
			name:           "Valid Token",
			tokenString:    validToken,
			tokenSecret:    tokenSecret,
			expectError:    false,
			expectedUserID: userID,
		},
		{
			name:           "Expired Token",
			tokenString:    expiredToken,
			tokenSecret:    tokenSecret,
			expectError:    true,
			expectedUserID: uuid.Nil,
		},
		{
			name:           "Wrong Secret",
			tokenString:    wrongSecretToken,
			tokenSecret:    tokenSecret,
			expectError:    true,
			expectedUserID: uuid.Nil,
		},
		{
			name:           "Invalid Token Format",
			tokenString:    "invalid.token.format",
			tokenSecret:    tokenSecret,
			expectError:    true,
			expectedUserID: uuid.Nil,
		},
		{
			name:           "Empty Token",
			tokenString:    "",
			tokenSecret:    tokenSecret,
			expectError:    true,
			expectedUserID: uuid.Nil,
		},
		{
			name:           "Invalid Subject",
			tokenString:    createTokenWithInvalidSubject(t, tokenSecret, expiresIn),
			tokenSecret:    tokenSecret,
			expectError:    true,
			expectedUserID: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := ValidateJWT(tt.tokenString, tt.tokenSecret)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if userID != tt.expectedUserID {
				t.Errorf("expected userID %v, got %v", tt.expectedUserID, userID)
			}
		})
	}
}

// createTokenWithInvalidSubject creates JWT with invalid subject
func createTokenWithInvalidSubject(t *testing.T, tokenSecret string, expiresIn time.Duration) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "yappy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   "not-a-uuid",
	})
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		t.Fatalf("failed to create invalid subject token: %v", err)
	}
	return tokenString
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		headers       http.Header
		expectedToken string
		expectError   bool
	}{
		{
			name: "Valid Bearer Token",
			headers: http.Header{
				"Authorization": []string{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."},
			},
			expectedToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			expectError:   false,
		},
		{
			name:          "Missing Authorization Header",
			headers:       http.Header{},
			expectedToken: "",
			expectError:   true,
		},
		{
			name: "Empty Authorization Header",
			headers: http.Header{
				"Authorization": []string{""},
			},
			expectedToken: "",
			expectError:   true,
		},
		{
			name: "No Bearer Prefix",
			headers: http.Header{
				"Authorization": []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."},
			},
			expectedToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GetBearerToken(tt.headers)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if token != tt.expectedToken {
				t.Errorf("expected token %q, got %q", tt.expectedToken, token)
			}
		})
	}
}
