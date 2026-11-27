package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

const (
	CtxKeyUserID      = "user_id"
	CtxKeyUserUUID    = "user_uuid"
	CtxKeyTenantID    = "tenant_id"
	CtxKeyTenantUUID  = "tenant_uuid"
	CtxKeyUsername    = "username"
	CtxKeyRole        = "role"
	CtxKeyPermissions = "permissions"
	CtxKeyOutletIDs   = "outlet_ids"
)

// Authenticate verifies the incoming Bearer JWT token and extracts claims.
func Authenticate(jwtService *pkgjwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Unauthorized(c, "Authorization header format must be: Bearer <token>")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			response.Unauthorized(c, err.Error())
			c.Abort()
			return
		}

		// Inject user and tenant identity into Gin Context
		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyUserUUID, claims.UserUUID)
		c.Set(CtxKeyTenantID, claims.TenantID)
		c.Set(CtxKeyTenantUUID, claims.TenantUUID)
		c.Set(CtxKeyUsername, claims.Username)
		c.Set(CtxKeyRole, claims.Role)
		c.Set(CtxKeyPermissions, claims.Permissions)
		c.Set(CtxKeyOutletIDs, claims.OutletIDs)

		c.Next()
	}
}

// GetUserID retrieves the authenticated user ID from context.
func GetUserID(c *gin.Context) (uint64, bool) {
	val, exists := c.Get(CtxKeyUserID)
	if !exists {
		return 0, false
	}
	id, ok := val.(uint64)
	return id, ok
}

// GetUserUUID retrieves the authenticated user UUID from context.
func GetUserUUID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get(CtxKeyUserUUID)
	if !exists {
		return uuid.Nil, false
	}
	u, ok := val.(uuid.UUID)
	return u, ok
}

// GetTenantID retrieves the resolved tenant ID from context.
func GetTenantID(c *gin.Context) (uint64, bool) {
	val, exists := c.Get(CtxKeyTenantID)
	if !exists {
		return 0, false
	}
	id, ok := val.(uint64)
	return id, ok
}

// GetRole retrieves the authenticated user role from context.
func GetRole(c *gin.Context) (string, bool) {
	val, exists := c.Get(CtxKeyRole)
	if !exists {
		return "", false
	}
	r, ok := val.(string)
	return r, ok
}

// GetPermissions retrieves the slice of permissions from context.
func GetPermissions(c *gin.Context) []string {
	val, exists := c.Get(CtxKeyPermissions)
	if !exists {
		return nil
	}
	perms, ok := val.([]string)
	if !ok {
		return nil
	}
	return perms
}
