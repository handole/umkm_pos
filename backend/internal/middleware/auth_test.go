package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
)

func setupTestRouter(jwtService *pkgjwt.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	api := r.Group("/api")
	api.Use(middleware.Authenticate(jwtService))
	{
		api.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		adminOnly := api.Group("/admin")
		adminOnly.Use(middleware.RequireRoles("SUPERADMIN", "STORE_MANAGER"))
		{
			adminOnly.GET("/dashboard", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"admin": true})
			})
		}

		posOnly := api.Group("/pos")
		posOnly.Use(middleware.RequirePermissions("pos:checkout"))
		{
			posOnly.GET("/checkout", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"pos": true})
			})
		}
	}

	return r
}

func TestAuthAndRBACMiddleware(t *testing.T) {
	jwtService := pkgjwt.NewService("test-secret", time.Hour, 24*time.Hour)
	router := setupTestRouter(jwtService)

	// Create tokens for a cashier user
	cashierTokens, _ := jwtService.GenerateTokenPair(
		1,
		uuid.New(),
		1,
		uuid.New(),
		"kasir_budi",
		"CASHIER",
		[]string{"pos:checkout", "pos:shift"},
		[]uint64{1},
	)

	// Test 1: Access without token -> should fail 401
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/protected", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}

	// Test 2: Access with valid cashier token to general protected route -> should succeed 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+cashierTokens.AccessToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	// Test 3: Cashier tries to access admin-only route -> should fail 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+cashierTokens.AccessToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for Cashier on Admin route, got %d", w.Code)
	}

	// Test 4: Cashier accesses POS route requiring 'pos:checkout' -> should succeed 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/pos/checkout", nil)
	req.Header.Set("Authorization", "Bearer "+cashierTokens.AccessToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for Cashier on POS route, got %d", w.Code)
	}

	// Create tokens for a Superadmin user
	adminTokens, _ := jwtService.GenerateTokenPair(
		2,
		uuid.New(),
		1,
		uuid.New(),
		"owner_agus",
		"SUPERADMIN",
		nil, // Superadmin bypasses permission checks
		[]uint64{1},
	)

	// Test 5: Superadmin accesses admin route -> should succeed 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for Superadmin on Admin route, got %d", w.Code)
	}

	// Test 6: Superadmin accesses POS route -> should succeed 200 because Superadmin has universal access
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/pos/checkout", nil)
	req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for Superadmin on POS route, got %d", w.Code)
	}
}
