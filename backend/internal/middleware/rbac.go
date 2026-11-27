package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// RequireRoles restricts endpoint access to specified user roles.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := GetRole(c)
		if !exists {
			response.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}

		// Superadmin always has universal access
		if role == "SUPERADMIN" {
			c.Next()
			return
		}

		for _, allowed := range allowedRoles {
			if role == allowed {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "Your role ("+role+") does not have permission to access this resource")
		c.Abort()
	}
}

// RequirePermissions ensures the authenticated user possesses all specified permissions.
func RequirePermissions(requiredPerms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := GetRole(c)
		if role == "SUPERADMIN" {
			c.Next()
			return
		}

		userPerms := GetPermissions(c)
		permMap := make(map[string]bool, len(userPerms))
		for _, p := range userPerms {
			permMap[p] = true
		}

		for _, req := range requiredPerms {
			if !permMap[req] {
				response.Forbidden(c, "Missing required permission: "+req)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireAnyPermission ensures the authenticated user possesses at least one of the specified permissions.
func RequireAnyPermission(candidatePerms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := GetRole(c)
		if role == "SUPERADMIN" {
			c.Next()
			return
		}

		userPerms := GetPermissions(c)
		permMap := make(map[string]bool, len(userPerms))
		for _, p := range userPerms {
			permMap[p] = true
		}

		for _, cand := range candidatePerms {
			if permMap[cand] {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "You do not have any of the required permissions to perform this action")
		c.Abort()
	}
}
