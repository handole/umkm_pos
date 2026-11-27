package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/handokodenih/umkm_pos/backend/internal/auth"
	"github.com/handokodenih/umkm_pos/backend/internal/inventory"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/internal/payroll"
	"github.com/handokodenih/umkm_pos/backend/internal/platform/database"
	"github.com/handokodenih/umkm_pos/backend/internal/pos"
	"github.com/handokodenih/umkm_pos/backend/internal/product"
	"github.com/handokodenih/umkm_pos/backend/internal/promotion"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// @title UMKM POS & Integrated Backoffice API
// @version 1.0
// @description Sistem Terpadu POS Kasir, Manajemen Stok Gudang, Katalog Produk, Promosi, dan Absensi & Penggajian.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	log.Println("Starting UMKM POS Backend Service...")

	// 1. Initialize Database
	dbConfig := database.LoadConfigFromEnv()
	db, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL database: %v. Running in standalone mode.", err)
	} else {
		// Seed default roles and permissions
		if err := database.SeedDefaultRolesAndPermissions(db); err != nil {
			log.Printf("Warning: Failed to seed default roles and permissions: %v", err)
		}
	}

	// 2. Initialize JWT Service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "umkm-pos-super-secret-key-change-in-production"
	}
	jwtService := pkgjwt.NewService(jwtSecret, 2*time.Hour, 7*24*time.Hour)

	// 3. Initialize HTTP Engine
	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Global Middlewares
	r.Use(middleware.CORS())
	r.Use(middleware.ResolveTenantContext())

	// Health Check Probe
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "connected"
		if db == nil {
			dbStatus = "disconnected"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "umkm-pos-backend",
			"database": dbStatus,
			"time":     time.Now().Format(time.RFC3339),
		})
	})

	// Swagger / OpenAPI Docs
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!DOCTYPE html>
<html>
<head>
  <title>UMKM POS API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: '/api/v1/docs/swagger.json', dom_id: '#swagger-ui' });
  </script>
</body>
</html>`))
	})

	// API v1 Router Group
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/docs/swagger.json", func(c *gin.Context) {
			c.File("docs/swagger.json")
		})
		apiV1.GET("/ping", func(c *gin.Context) {
			response.Success(c, http.StatusOK, "pong", gin.H{
				"server_time": time.Now().UTC(),
			})
		})

		// Auth Domain Endpoints
		if db != nil {
			authRepo := auth.NewRepository(db)
			authUsecase := auth.NewUsecase(authRepo, jwtService)
			authHandler := auth.NewHandler(authUsecase)
			authHandler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

			// Product Domain Endpoints
			productRepo := product.NewRepository(db)
			productUsecase := product.NewUsecase(productRepo)
			productHandler := product.NewHandler(productUsecase)
			productHandler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

			// Inventory Domain Endpoints
			inventoryRepo := inventory.NewRepository(db)
			inventoryUsecase := inventory.NewUsecase(inventoryRepo)
			inventoryHandler := inventory.NewHandler(inventoryUsecase)
			inventoryHandler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

			// POS Cashier Domain Endpoints
			posRepo := pos.NewRepository(db)
			posUsecase := pos.NewUsecase(posRepo)
			posHandler := pos.NewHandler(posUsecase)
			posHandler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

			// Promotion Domain Endpoints
			promotionRepo := promotion.NewRepository(db)
			promotionUsecase := promotion.NewUsecase(promotionRepo)
			promotionHandler := promotion.NewHandler(promotionUsecase)
			promotionHandler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

			// Attendance & Payroll Domain Endpoints
			payrollRepo := payroll.NewRepository(db)
			payrollUsecase := payroll.NewUsecase(payrollRepo)
			payrollHandler := payroll.NewHandler(payrollUsecase)
			payrollHandler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))
		}

		// Example Protected Routes Demonstrating Role-Based Access Control (RBAC)
		protected := apiV1.Group("")
		protected.Use(middleware.Authenticate(jwtService))
		{
			// Admin only
			adminOnly := protected.Group("/admin")
			adminOnly.Use(middleware.RequireRoles("SUPERADMIN", "STORE_MANAGER"))
			{
				adminOnly.GET("/stats", func(c *gin.Context) {
					role, _ := middleware.GetRole(c)
					username, _ := c.Get(middleware.CtxKeyUsername)
					response.Success(c, http.StatusOK, "Admin stats access granted", gin.H{
						"role":     role,
						"username": username,
						"access":   "Management Backoffice",
					})
				})
			}

			// POS Cashier operation (Permission: pos:checkout)
			posRoutes := protected.Group("/pos")
			posRoutes.Use(middleware.RequirePermissions("pos:checkout"))
			{
				posRoutes.GET("/ping", func(c *gin.Context) {
					response.Success(c, http.StatusOK, "POS checkout access granted", nil)
				})
			}

			// Warehouse Staff operation (Permission: inventory:opname)
			warehouseRoutes := protected.Group("/inventory")
			warehouseRoutes.Use(middleware.RequirePermissions("inventory:opname"))
			{
				warehouseRoutes.GET("/ping", func(c *gin.Context) {
					response.Success(c, http.StatusOK, "Inventory opname access granted", nil)
				})
			}

			// HR / Finance operation (Permission: payroll:finalize)
			payrollRoutes := protected.Group("/payroll")
			payrollRoutes.Use(middleware.RequirePermissions("payroll:finalize"))
			{
				payrollRoutes.GET("/ping", func(c *gin.Context) {
					response.Success(c, http.StatusOK, "Payroll finalize access granted", nil)
				})
			}
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("UMKM POS API Server listening on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
