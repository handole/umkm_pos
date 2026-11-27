package jwt_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
)

func TestJWTGenerateAndValidate(t *testing.T) {
	service := pkgjwt.NewService("test-secret-key-123", time.Hour, 24*time.Hour)

	userUUID := uuid.New()
	tenantUUID := uuid.New()
	permissions := []string{"pos:checkout", "product:read"}
	outletIDs := []uint64{1, 2}

	tokens, err := service.GenerateTokenPair(
		10,
		userUUID,
		1,
		tenantUUID,
		"kasir1",
		"CASHIER",
		permissions,
		outletIDs,
	)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("Expected non-empty access and refresh tokens")
	}

	// Validate access token
	claims, err := service.ValidateToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("Expected token to be valid, got: %v", err)
	}

	if claims.UserID != 10 {
		t.Errorf("Expected UserID 10, got %d", claims.UserID)
	}
	if claims.UserUUID != userUUID {
		t.Errorf("Expected UserUUID %v, got %v", userUUID, claims.UserUUID)
	}
	if claims.TenantID != 1 {
		t.Errorf("Expected TenantID 1, got %d", claims.TenantID)
	}
	if claims.Role != "CASHIER" {
		t.Errorf("Expected Role CASHIER, got %s", claims.Role)
	}
	if len(claims.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(claims.Permissions))
	}
}

func TestJWTExpired(t *testing.T) {
	// 1 millisecond expiry
	service := pkgjwt.NewService("test-secret-key-123", time.Millisecond, time.Millisecond)

	tokens, err := service.GenerateTokenPair(
		1,
		uuid.New(),
		1,
		uuid.New(),
		"admin",
		"SUPERADMIN",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = service.ValidateToken(tokens.AccessToken)
	if err != pkgjwt.ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got %v", err)
	}
}
