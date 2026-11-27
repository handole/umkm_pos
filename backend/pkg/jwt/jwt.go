package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired authentication token")
	ErrExpiredToken = errors.New("authentication token has expired")
)

// UserClaims holds user identity, tenancy, role, and permission data inside JWT.
type UserClaims struct {
	UserID      uint64    `json:"user_id"`
	UserUUID    uuid.UUID `json:"user_uuid"`
	TenantID    uint64    `json:"tenant_id"`
	TenantUUID  uuid.UUID `json:"tenant_uuid"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	OutletIDs   []uint64  `json:"outlet_ids"`
	jwt.RegisteredClaims
}

// TokenPair contains generated access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Service manages JWT creation and validation.
type Service struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewService instantiates a JWT service.
func NewService(secret string, accessExpiry, refreshExpiry time.Duration) *Service {
	if secret == "" {
		secret = "umkm-pos-super-secret-key-change-in-production"
	}
	if accessExpiry == 0 {
		accessExpiry = 2 * time.Hour
	}
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}
	return &Service{
		secretKey:     []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        "umkm-pos-backend",
	}
}

// GenerateTokenPair creates new Access and Refresh tokens for a user.
func (s *Service) GenerateTokenPair(
	userID uint64,
	userUUID uuid.UUID,
	tenantID uint64,
	tenantUUID uuid.UUID,
	username string,
	role string,
	permissions []string,
	outletIDs []uint64,
) (*TokenPair, error) {
	now := time.Now()
	accessExpiryTime := now.Add(s.accessExpiry)

	claims := &UserClaims{
		UserID:      userID,
		UserUUID:    userUUID,
		TenantID:    tenantID,
		TenantUUID:  tenantUUID,
		Username:    username,
		Role:        role,
		Permissions: permissions,
		OutletIDs:   outletIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiryTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userUUID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedAccessToken, err := accessToken.SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	// Refresh token claims (minimal data)
	refreshExpiryTime := now.Add(s.refreshExpiry)
	refreshClaims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(refreshExpiryTime),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    s.issuer,
		Subject:   userUUID.String(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
		ExpiresAt:    accessExpiryTime,
	}, nil
}

// ValidateToken parses and validates a signed JWT string.
func (s *Service) ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
