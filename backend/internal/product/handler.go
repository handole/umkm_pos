package product

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// Handler handles HTTP requests for Category and Product management.
type Handler struct {
	usecase Usecase
}

// NewHandler instantiates a product HTTP handler.
func NewHandler(usecase Usecase) *Handler {
	return &Handler{usecase: usecase}
}

// RegisterRoutes registers product catalog endpoints with authentication and permission checks.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Categories group
	categories := r.Group("/categories")
	categories.Use(authMiddleware)
	{
		categories.GET("", middleware.RequirePermissions("product:read"), h.ListCategories)
		categories.POST("", middleware.RequirePermissions("product:create"), h.CreateCategory)
		categories.PUT("/:uuid", middleware.RequirePermissions("product:update"), h.UpdateCategory)
		categories.DELETE("/:uuid", middleware.RequirePermissions("product:delete"), h.DeleteCategory)
	}

	// Products group
	products := r.Group("/products")
	products.Use(authMiddleware)
	{
		products.GET("", middleware.RequirePermissions("product:read"), h.ListProducts)
		products.GET("/barcode/:code", middleware.RequirePermissions("product:read"), h.ScanBarcode)
		products.GET("/:uuid", middleware.RequirePermissions("product:read"), h.GetProductByUUID)
		products.POST("", middleware.RequirePermissions("product:create"), h.CreateProduct)
		products.PUT("/:uuid", middleware.RequirePermissions("product:update"), h.UpdateProduct)
		products.DELETE("/:uuid", middleware.RequirePermissions("product:delete"), h.DeleteProduct)
	}
}

// ListCategories handles list categories.
// @Summary List Categories
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/categories [get]
func (h *Handler) ListCategories(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	results, err := h.usecase.ListCategories(c.Request.Context(), tenantID)
	if err != nil {
		response.InternalServerError(c, "Failed to load categories")
		return
	}

	response.Success(c, http.StatusOK, "Categories loaded successfully", results)
}

// CreateCategory handles create category.
// @Summary Create Category
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateCategoryRequest true "CreateCategoryRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/categories [post]
func (h *Handler) CreateCategory(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid category input", err.Error())
		return
	}

	result, err := h.usecase.CreateCategory(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "Category created successfully", result)
}

// UpdateCategory handles update category.
// @Summary Update Category
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body UpdateCategoryRequest true "UpdateCategoryRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/categories/{uuid} [put]
func (h *Handler) UpdateCategory(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	catUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid category UUID format", nil)
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid category update input", err.Error())
		return
	}

	result, err := h.usecase.UpdateCategory(c.Request.Context(), tenantID, catUUID, req)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Failed to update category")
		return
	}

	response.Success(c, http.StatusOK, "Category updated successfully", result)
}

// DeleteCategory handles delete category.
// @Summary Delete Category
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/categories/{uuid} [delete]
func (h *Handler) DeleteCategory(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	catUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid category UUID format", nil)
		return
	}

	if err := h.usecase.DeleteCategory(c.Request.Context(), tenantID, catUUID); err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Failed to delete category")
		return
	}

	response.Success(c, http.StatusOK, "Category deleted successfully", nil)
}

// ListProducts handles list products.
// @Summary List Products
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/products [get]
func (h *Handler) ListProducts(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	var filter ProductQueryFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	products, total, err := h.usecase.ListProducts(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.InternalServerError(c, "Failed to fetch products")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"code":    http.StatusOK,
		"message": "Products retrieved successfully",
		"data":    products,
		"pagination": gin.H{
			"total": total,
			"page":  filter.Page,
			"limit": filter.Limit,
		},
	})
}

// GetProductByUUID handles get product by uuid.
// @Summary Get Product By UUID
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/products/{uuid} [get]
func (h *Handler) GetProductByUUID(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	productUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid product UUID format", nil)
		return
	}

	product, err := h.usecase.GetProductByUUID(c.Request.Context(), tenantID, productUUID)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Failed to get product")
		return
	}

	response.Success(c, http.StatusOK, "Product retrieved successfully", product)
}

// CreateProduct handles create product.
// @Summary Create Product
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateProductRequest true "CreateProductRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/products [post]
func (h *Handler) CreateProduct(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid product payload", err.Error())
		return
	}

	product, err := h.usecase.CreateProduct(c.Request.Context(), tenantID, req)
	if err != nil {
		if errors.Is(err, ErrSKUAlreadyExists) {
			response.BadRequest(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Product created successfully", product)
}

// UpdateProduct handles update product.
// @Summary Update Product
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body UpdateProductRequest true "UpdateProductRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/products/{uuid} [put]
func (h *Handler) UpdateProduct(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	productUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid product UUID format", nil)
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid product update payload", err.Error())
		return
	}

	product, err := h.usecase.UpdateProduct(c.Request.Context(), tenantID, productUUID, req)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Product updated successfully", product)
}

// DeleteProduct handles delete product.
// @Summary Delete Product
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/products/{uuid} [delete]
func (h *Handler) DeleteProduct(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	productUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid product UUID format", nil)
		return
	}

	if err := h.usecase.DeleteProduct(c.Request.Context(), tenantID, productUUID); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Failed to delete product")
		return
	}

	response.Success(c, http.StatusOK, "Product deleted successfully", nil)
}

// ScanBarcode handles instant barcode scanner requests from POS and Mobile Warehouse.
// ScanBarcode handles scan barcode.
// @Summary Scan Barcode
// @Tags Product
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param code path string true "code"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/products/barcode/{code} [get]
func (h *Handler) ScanBarcode(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Tenant identification missing")
		return
	}

	barcode := c.Param("code")
	if barcode == "" {
		response.BadRequest(c, "Barcode parameter is required", nil)
		return
	}

	result, err := h.usecase.ScanBarcode(c.Request.Context(), tenantID, barcode)
	if err != nil {
		if errors.Is(err, ErrBarcodeNotFound) {
			response.Error(c, http.StatusNotFound, "Barcode not found in catalog", nil)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Barcode matched successfully", result)
}
