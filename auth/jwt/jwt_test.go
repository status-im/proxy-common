package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerate(t *testing.T) {
	secret := "test-secret"
	challenge := "test-challenge-123"
	expMinutes := 10
	requestLimit := 100

	tokenString, expiresAt, err := Generate(secret, challenge, expMinutes, requestLimit)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if tokenString == "" {
		t.Error("expected non-empty token string")
	}

	// Check that expiration time is approximately correct
	expectedExp := time.Now().Add(time.Duration(expMinutes) * time.Minute)
	diff := expiresAt.Sub(expectedExp).Abs()
	if diff > 2*time.Second {
		t.Errorf("expiration time differs by %v, expected ~%v, got %v", diff, expectedExp, expiresAt)
	}

	// Verify the token to check claims
	claims, err := Verify(tokenString, secret)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if claims.Challenge != challenge {
		t.Errorf("expected challenge %s, got %s", challenge, claims.Challenge)
	}
	if claims.RequestLimit != requestLimit {
		t.Errorf("expected request limit %d, got %d", requestLimit, claims.RequestLimit)
	}
	if claims.ID != challenge {
		t.Errorf("expected ID %s, got %s", challenge, claims.ID)
	}
}

func TestVerify(t *testing.T) {
	secret := "test-secret"
	challenge := "test-challenge-456"
	expMinutes := 10
	requestLimit := 50

	tokenString, _, err := Generate(secret, challenge, expMinutes, requestLimit)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Test valid token
	claims, err := Verify(tokenString, secret)
	if err != nil {
		t.Fatalf("Verify failed for valid token: %v", err)
	}

	if claims.Challenge != challenge {
		t.Errorf("expected challenge %s, got %s", challenge, claims.Challenge)
	}
	if claims.RequestLimit != requestLimit {
		t.Errorf("expected request limit %d, got %d", requestLimit, claims.RequestLimit)
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	secret := "test-secret"
	challenge := "test-challenge-789"
	expMinutes := 10
	requestLimit := 100

	tokenString, _, err := Generate(secret, challenge, expMinutes, requestLimit)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Try to verify with different secret
	_, err = Verify(tokenString, "wrong-secret")
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	secret := "test-secret"
	challenge := "test-challenge-exp"
	requestLimit := 100

	// Create token with -1 minute expiry (already expired)
	exp := time.Now().Add(-1 * time.Minute)
	claims := Claims{
		RequestLimit: requestLimit,
		Challenge:    challenge,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
			ID:        challenge,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to create expired token: %v", err)
	}

	// Try to verify expired token
	_, err = Verify(tokenString, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	secret := "test-secret"

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "malformed token",
			token: "not.a.valid.jwt",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "random string",
			token: "random-string-not-jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Verify(tt.token, secret)
			if err == nil {
				t.Error("expected error for invalid token")
			}
		})
	}
}

func TestClaimsStructure(t *testing.T) {
	secret := "test-secret"
	challenge := "test-challenge-structure"
	expMinutes := 15
	requestLimit := 200

	tokenString, _, err := Generate(secret, challenge, expMinutes, requestLimit)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Parse token manually to check structure
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("ParseWithClaims failed: %v", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		t.Fatal("failed to cast claims")
	}

	// Check all fields
	if claims.Challenge != challenge {
		t.Errorf("expected challenge %s, got %s", challenge, claims.Challenge)
	}
	if claims.RequestLimit != requestLimit {
		t.Errorf("expected request limit %d, got %d", requestLimit, claims.RequestLimit)
	}
	if claims.ID != challenge {
		t.Errorf("expected ID %s, got %s", challenge, claims.ID)
	}
	if claims.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
	if claims.IssuedAt == nil {
		t.Error("expected IssuedAt to be set")
	}
}

func TestGenerateWithDifferentDurations(t *testing.T) {
	secret := "test-secret"
	challenge := "test-challenge"
	requestLimit := 100

	tests := []struct {
		name       string
		expMinutes int
	}{
		{"1 minute", 1},
		{"5 minutes", 5},
		{"30 minutes", 30},
		{"60 minutes", 60},
		{"120 minutes", 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString, expiresAt, err := Generate(secret, challenge, tt.expMinutes, requestLimit)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Verify token
			claims, err := Verify(tokenString, secret)
			if err != nil {
				t.Fatalf("Verify failed: %v", err)
			}

			// Check expiration is reasonable
			expectedExp := time.Now().Add(time.Duration(tt.expMinutes) * time.Minute)
			diff := expiresAt.Sub(expectedExp).Abs()
			if diff > 2*time.Second {
				t.Errorf("expiration time differs by %v", diff)
			}

			// Check claims expiration matches
			claimsDiff := claims.ExpiresAt.Time.Sub(expiresAt).Abs()
			if claimsDiff > 1*time.Second {
				t.Errorf("claims expiration differs from returned expiration by %v", claimsDiff)
			}
		})
	}
}
