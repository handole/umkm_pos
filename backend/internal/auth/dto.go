package auth

import (
	"time"

	"github.com/google/uuid"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
)

// RegisterRequest holds payloads to create a new user account.
type RegisterRequest struct {
	Username   string   `json:"username" binding:"required,min=3,max=50"`
	Email      string   `json:"email" binding:"omitempty,email"`
	Phone      string   `json:"phone" binding:"omitempty"`
	Password   string   `json:"password" binding:"required,min=6"`
	FullName   string   `json:"full_name" binding:"required"`
	RoleName   string   `json:"role_name" binding:"required"` // 'SUPERADMIN', 'STORE_MANAGER', 'CASHIER', 'WAREHOUSE', 'HR_FINANCE'
	Pin        string   `json:"pin" binding:"omitempty,len=6"` // 6-digit PIN for quick POS unlock / attendance
	TenantCode string   `json:"tenant_code"`                  // Optional in single-tenant; specifies target tenant
	TenantName string   `json:"tenant_name"`                  // Optional; used if auto-provisioning initial tenant
	OutletIDs  []uint64 `json:"outlet_ids"`                   // List of outlet IDs this user can access
}

// LoginRequest holds user credentials for authentication.
type LoginRequest struct {
	Username   string `json:"username" binding:"required"` // Accepts username or email
	Password   string `json:"password" binding:"required"`
	TenantCode string `json:"tenant_code"` // Optional in single-tenant; required in multi-tenant
}

// PinLoginRequest enables fast cashier switching and attendance clock-in on POS tablets.
type PinLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Pin      string `json:"pin" binding:"required,len=6"`
	OutletID uint64 `json:"outlet_id"`
}

// RefreshTokenRequest holds the refresh token string.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UserSummaryDTO represents basic user profile information.
type UserSummaryDTO struct {
	ID        uint64    `json:"id"`
	UUID      uuid.UUID `json:"uuid"`
	TenantID  uint64    `json:"tenant_id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantSummaryDTO represents organization data returned on login.
type TenantSummaryDTO struct {
	ID   uint64    `json:"id"`
	UUID uuid.UUID `json:"uuid"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

// OutletSummaryDTO represents accessible outlet branch.
type OutletSummaryDTO struct {
	ID           uint64 `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	IsMainBranch bool   `json:"is_main_branch"`
}

// AuthResponse is returned on successful authentication.
type AuthResponse struct {
	Token       pkgjwt.TokenPair   `json:"token"`
	User        UserSummaryDTO     `json:"user"`
	Tenant      TenantSummaryDTO   `json:"tenant"`
	Permissions []string           `json:"permissions"`
	Outlets     []OutletSummaryDTO `json:"outlets"`
}
