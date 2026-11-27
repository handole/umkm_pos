package pos

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// Handler handles HTTP requests for the POS cashier domain.
type Handler struct {
	uc Usecase
}

// NewHandler creates a new POS Handler instance.
func NewHandler(uc Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterRoutes mounts all POS endpoints to the Gin router engine.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	pos := rg.Group("/pos")
	pos.Use(authMiddleware)
	{
		// Cashier Shift Endpoints
		pos.POST("/shifts/open", middleware.RequirePermissions("pos:shift"), h.OpenShift)
		pos.GET("/shifts/current", middleware.RequirePermissions("pos:shift"), h.GetCurrentShift)
		pos.POST("/shifts/cash-log", middleware.RequirePermissions("pos:shift"), h.AddCashLog)
		pos.POST("/shifts/close", middleware.RequirePermissions("pos:shift"), h.CloseShift)
		pos.GET("/shifts", middleware.RequirePermissions("pos:shift"), h.ListShifts)

		// Payment Methods
		pos.GET("/payment-methods", middleware.RequirePermissions("pos:checkout"), h.ListPaymentMethods)

		// Orders and Checkout
		pos.POST("/checkout", middleware.RequirePermissions("pos:checkout"), h.CheckoutOrder)
		pos.POST("/orders", middleware.RequirePermissions("pos:checkout"), h.CheckoutOrder)
		pos.GET("/orders", middleware.RequirePermissions("pos:checkout"), h.ListOrders)
		pos.GET("/orders/:uuid", middleware.RequirePermissions("pos:checkout"), h.GetOrder)
		pos.GET("/orders/:uuid/receipt", middleware.RequirePermissions("pos:checkout"), h.GetOrderReceipt)

		// Void Order (Manager Authorization)
		pos.POST("/orders/:uuid/void", middleware.RequirePermissions("pos:void"), h.VoidOrder)

		// Financial & Sales Reports
		pos.GET("/reports/summary", middleware.RequirePermissions("payroll:read"), h.GetSalesReport)
		pos.GET("/reports/export", middleware.RequirePermissions("payroll:read"), h.ExportSalesReportCSV)
	}
}

// OpenShift handles open shift.
// @Summary Open Shift
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body OpenShiftRequest true "OpenShiftRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/shifts/open [post]
func (h *Handler) OpenShift(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	cashierID, _ := middleware.GetUserID(c)

	var req OpenShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.OpenShift(c.Request.Context(), tenantID, cashierID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "Cashier shift opened successfully", res)
}

// GetCurrentShift handles get current shift.
// @Summary Get Current Shift
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/shifts/current [get]
func (h *Handler) GetCurrentShift(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	cashierID, _ := middleware.GetUserID(c)

	shift, err := h.uc.GetCurrentShift(c.Request.Context(), tenantID, cashierID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	if shift == nil {
		response.Success(c, http.StatusOK, "No active open shift found for current cashier", nil)
		return
	}

	response.Success(c, http.StatusOK, "Active cashier shift details", shift)
}

// AddCashLog handles add cash log.
// @Summary Add Cash Log
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CashLogRequest true "CashLogRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/shifts/cash-log [post]
func (h *Handler) AddCashLog(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	cashierID, _ := middleware.GetUserID(c)

	var req CashLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.AddCashLog(c.Request.Context(), tenantID, cashierID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "Cash log recorded successfully", res)
}

// CloseShift handles close shift.
// @Summary Close Shift
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CloseShiftRequest true "CloseShiftRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/shifts/close [post]
func (h *Handler) CloseShift(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	cashierID, _ := middleware.GetUserID(c)

	var req CloseShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CloseShift(c.Request.Context(), tenantID, cashierID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Cashier shift closed and reconciled successfully", res)
}

// ListShifts handles list shifts.
// @Summary List Shifts
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/shifts [get]
func (h *Handler) ListShifts(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var outletID *uint64
	if oidStr := c.Query("outlet_id"); oidStr != "" {
		if oid, err := strconv.ParseUint(oidStr, 10, 64); err == nil {
			outletID = &oid
		}
	}

	var cashierID *uint64
	if cidStr := c.Query("cashier_user_id"); cidStr != "" {
		if cid, err := strconv.ParseUint(cidStr, 10, 64); err == nil {
			cashierID = &cid
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	shifts, total, err := h.uc.ListShifts(c.Request.Context(), tenantID, outletID, cashierID, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cashier shifts retrieved",
		"data":    shifts,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// ListPaymentMethods handles list payment methods.
// @Summary List Payment Methods
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/payment-methods [get]
func (h *Handler) ListPaymentMethods(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	methods, err := h.uc.ListPaymentMethods(c.Request.Context(), tenantID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Payment methods retrieved", methods)
}

// CheckoutOrder handles checkout order.
// @Summary Checkout Order
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CheckoutOrderRequest true "CheckoutOrderRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/checkout [post]
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/orders [post]
func (h *Handler) CheckoutOrder(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	cashierID, _ := middleware.GetUserID(c)

	var req CheckoutOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	// Support client idempotency key via header if not supplied in body
	if req.IdempotencyKey == uuid.Nil {
		headerKey := c.GetHeader("Idempotency-Key")
		if parsedKey, errParse := uuid.Parse(headerKey); errParse == nil {
			req.IdempotencyKey = parsedKey
		} else {
			response.BadRequest(c, "Valid idempotency_key is required in body or Idempotency-Key header", nil)
			return
		}
	}

	res, err := h.uc.CheckoutOrder(c.Request.Context(), tenantID, cashierID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "POS checkout transaction completed successfully", res)
}

// GetOrder handles get order.
// @Summary Get Order
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/orders/{uuid} [get]
func (h *Handler) GetOrder(c *gin.Context) {
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

	order, err := h.uc.GetOrder(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Order not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Order details retrieved", order)
}

// ListOrders handles list orders.
// @Summary List Orders
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/orders [get]
func (h *Handler) ListOrders(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var outletID *uint64
	if oidStr := c.Query("outlet_id"); oidStr != "" {
		if oid, err := strconv.ParseUint(oidStr, 10, 64); err == nil {
			outletID = &oid
		}
	}

	var cashierID *uint64
	if cidStr := c.Query("cashier_user_id"); cidStr != "" {
		if cid, err := strconv.ParseUint(cidStr, 10, 64); err == nil {
			cashierID = &cid
		}
	}

	var status *string
	if st := c.Query("status"); st != "" {
		status = &st
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orders, total, err := h.uc.ListOrders(c.Request.Context(), tenantID, outletID, cashierID, status, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "POS orders retrieved",
		"data":    orders,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetOrderReceipt handles get order receipt.
// @Summary Get Order Receipt
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/orders/{uuid}/receipt [get]
func (h *Handler) GetOrderReceipt(c *gin.Context) {
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

	receipt, err := h.uc.GetOrderReceipt(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Order not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Receipt data ready for ESC/POS thermal printing", receipt)
}

// VoidOrder handles void order.
// @Summary Void Order
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body VoidOrderRequest true "VoidOrderRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/orders/{uuid}/void [post]
func (h *Handler) VoidOrder(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	managerUserID, _ := middleware.GetUserID(c)
	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req VoidOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.VoidOrder(c.Request.Context(), tenantID, managerUserID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Order voided and inventory stock restored", res)
}

// GetSalesReport handles get sales report.
// @Summary Get Sales Report
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/reports/summary [get]
func (h *Handler) GetSalesReport(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var query SalesReportQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.GetSalesReport(c.Request.Context(), tenantID, query)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Sales report retrieved", res)
}

// ExportSalesReportCSV handles export sales report csv.
// @Summary Export Sales Report CSV
// @Tags POS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/pos/reports/export [get]
func (h *Handler) ExportSalesReportCSV(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var query SalesReportQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	csvBytes, err := h.uc.ExportSalesReportCSV(c.Request.Context(), tenantID, query)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="sales-report.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", csvBytes)
}
