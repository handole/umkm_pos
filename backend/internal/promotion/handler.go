package promotion

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// Handler handles HTTP requests for marketing promotions and customer loyalty.
type Handler struct {
	uc Usecase
}

// NewHandler creates a new promotion Handler instance.
func NewHandler(uc Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterRoutes mounts customer and promotion endpoints to the Gin router engine.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Customers Group
	customers := rg.Group("/customers")
	customers.Use(authMiddleware)
	{
		customers.POST("", middleware.RequirePermissions("pos:checkout"), h.CreateCustomer)
		customers.GET("", middleware.RequirePermissions("pos:checkout"), h.ListCustomers)
		customers.GET("/:uuid", middleware.RequirePermissions("pos:checkout"), h.GetCustomer)
		customers.PUT("/:uuid", middleware.RequirePermissions("pos:checkout"), h.UpdateCustomer)
	}

	// Promotions Group
	promos := rg.Group("/promotions")
	promos.Use(authMiddleware)
	{
		promos.POST("", middleware.RequirePermissions("promotion:manage"), h.CreatePromotion)
		promos.GET("", middleware.RequirePermissions("promotion:manage"), h.ListPromotions)
		promos.GET("/:uuid", middleware.RequirePermissions("promotion:manage"), h.GetPromotion)
		promos.PUT("/:uuid", middleware.RequirePermissions("promotion:manage"), h.UpdatePromotion)
		promos.DELETE("/:uuid", middleware.RequirePermissions("promotion:manage"), h.DeletePromotion)

		// Evaluation Endpoint for POS and E-Commerce carts
		promos.POST("/evaluate", middleware.RequirePermissions("pos:checkout"), h.EvaluateCart)
	}
}

// CreateCustomer handles create customer.
// @Summary Create Customer
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateCustomerRequest true "CreateCustomerRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/customers [post]
func (h *Handler) CreateCustomer(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreateCustomer(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "Customer registered successfully", res)
}

// GetCustomer handles get customer.
// @Summary Get Customer
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/customers/{uuid} [get]
func (h *Handler) GetCustomer(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	customer, err := h.uc.GetCustomer(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Customer not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Customer details retrieved", customer)
}

// UpdateCustomer handles update customer.
// @Summary Update Customer
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body UpdateCustomerRequest true "UpdateCustomerRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/customers/{uuid} [put]
func (h *Handler) UpdateCustomer(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.UpdateCustomer(c.Request.Context(), tenantID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Customer updated successfully", res)
}

// ListCustomers handles list customers.
// @Summary List Customers
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/customers [get]
func (h *Handler) ListCustomers(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	search := c.Query("search")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	customers, total, err := h.uc.ListCustomers(c.Request.Context(), tenantID, search, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Customers retrieved",
		"data":    customers,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// CreatePromotion handles create promotion.
// @Summary Create Promotion
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePromotionRequest true "CreatePromotionRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/promotions [post]
func (h *Handler) CreatePromotion(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req CreatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreatePromotion(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "Promotion campaign created successfully", res)
}

// ListPromotions handles list promotions.
// @Summary List Promotions
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/promotions [get]
func (h *Handler) ListPromotions(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var promoType *string
	if pt := c.Query("promo_type"); pt != "" {
		promoType = &pt
	}

	var isActive *bool
	if actStr := c.Query("is_active"); actStr != "" {
		val := actStr == "true"
		isActive = &val
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	promos, total, err := h.uc.ListPromotions(c.Request.Context(), tenantID, promoType, isActive, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Promotions retrieved",
		"data":    promos,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetPromotion handles get promotion.
// @Summary Get Promotion
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/promotions/{uuid} [get]
func (h *Handler) GetPromotion(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	promo, err := h.uc.GetPromotion(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Promotion not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Promotion details retrieved", promo)
}

// UpdatePromotion handles update promotion.
// @Summary Update Promotion
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body UpdatePromotionRequest true "UpdatePromotionRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/promotions/{uuid} [put]
func (h *Handler) UpdatePromotion(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req UpdatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.UpdatePromotion(c.Request.Context(), tenantID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Promotion updated successfully", res)
}

// DeletePromotion handles delete promotion.
// @Summary Delete Promotion
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/promotions/{uuid} [delete]
func (h *Handler) DeletePromotion(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	if err := h.uc.DeletePromotion(c.Request.Context(), tenantID, u); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Promotion deleted successfully", nil)
}

// EvaluateCart handles evaluate cart.
// @Summary Evaluate Cart
// @Tags Promotion
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body EvaluatePromotionRequest true "EvaluatePromotionRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/promotions/evaluate [post]
func (h *Handler) EvaluateCart(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req EvaluatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.EvaluateCart(c.Request.Context(), tenantID, req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Promotion cart evaluation completed", res)
}
