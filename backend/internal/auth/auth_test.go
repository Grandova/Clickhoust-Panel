package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTSecretAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jwt-secret")
	if err := InitJWTSecret(path); err != nil {
		t.Fatal(err)
	}
	token, err := GenerateToken(1, "admin", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := InitJWTSecret(path); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token); err != nil {
		t.Fatalf("persisted key cannot verify token: %v", err)
	}
	claims := &Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{Issuer: "clickhouse-manager", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	forged, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("clickhouse-manager-secret-key-2026-safe-production"))
	if _, err := ParseToken(forged); err == nil {
		t.Fatal("accepted old public signing key")
	}
	otherAlgorithm, _ := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString(jwtSecret)
	if _, err := ParseToken(otherAlgorithm); err == nil {
		t.Fatal("accepted unexpected algorithm")
	}
	claims.ExpiresAt = nil
	noExpiry, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	if _, err := ParseToken(noExpiry); err == nil {
		t.Fatal("accepted token without expiration")
	}
	if err := InitJWTSecret(filepath.Join(t.TempDir(), "jwt-secret")); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Fatal("another installation accepted token")
	}
	if err := os.WriteFile(path, []byte("short"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InitJWTSecret(path); err == nil {
		t.Fatal("accepted corrupt secret")
	}
}
