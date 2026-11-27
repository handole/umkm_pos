package middleware

import (
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ResolveTenantContext extracts or defaults the tenant context before authentication or on public endpoints.
func ResolveTenantContext() gin.HandlerFunc {
	appMode := os.Getenv("APP_MODE")
	if appMode == "" {
		appMode = "single-tenant"
	}

	defaultTenantIDStr := os.Getenv("DEFAULT_TENANT_ID")
	defaultTenantID, _ := strconv.ParseUint(defaultTenantIDStr, 10, 64)
	if defaultTenantID == 0 {
		defaultTenantID = 1
	}

	return func(c *gin.Context) {
		// 1. If already set by Auth JWT middleware, keep it
		if _, exists := c.Get(CtxKeyTenantID); exists {
			c.Next()
			return
		}

		// 2. Check X-Tenant-ID header (for client apps / external calls)
		headerTenantID := c.GetHeader("X-Tenant-ID")
		if headerTenantID != "" {
			if tid, err := strconv.ParseUint(headerTenantID, 10, 64); err == nil && tid > 0 {
				c.Set(CtxKeyTenantID, tid)
				c.Next()
				return
			}
		}

		// 3. In single-tenant mode, fallback to default tenant ID
		if appMode == "single-tenant" {
			c.Set(CtxKeyTenantID, defaultTenantID)
		}

		c.Next()
	}
}
