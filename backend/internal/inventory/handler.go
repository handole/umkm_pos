package inventory

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// Handler handles HTTP requests for the inventory domain.
type Handler struct {
	uc Usecase
}

// NewHandler creates a new inventory Handler instance.
func NewHandler(uc Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterRoutes mounts all inventory endpoints to the Gin router engine.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	inv := rg.Group("/inventory")
	inv.Use(authMiddleware)
	{
		// Locations
		inv.POST("/locations", middleware.RequirePermissions("inventory:transfer"), h.CreateLocation)
		inv.GET("/locations", middleware.RequirePermissions("inventory:read"), h.ListLocations)

		// Stock Balances and Ledger
		inv.GET("/stocks", middleware.RequirePermissions("inventory:read"), h.GetStocks)
		inv.GET("/movements", middleware.RequirePermissions("inventory:read"), h.GetStockMovements)

		// Stock Transfers
		inv.POST("/transfers", middleware.RequirePermissions("inventory:transfer"), h.CreateTransfer)
		inv.GET("/transfers", middleware.RequirePermissions("inventory:read"), h.ListTransfers)
		inv.GET("/transfers/:uuid", middleware.RequirePermissions("inventory:read"), h.GetTransfer)
		inv.POST("/transfers/:uuid/receive", middleware.RequirePermissions("inventory:transfer"), h.ReceiveTransfer)

		// Stock Opname
		inv.POST("/opnames", middleware.RequirePermissions("inventory:opname"), h.CreateOpname)
		inv.GET("/opnames", middleware.RequirePermissions("inventory:read"), h.ListOpnames)
		inv.GET("/opnames/:uuid", middleware.RequirePermissions("inventory:read"), h.GetOpname)
		inv.POST("/opnames/:uuid/scan", middleware.RequirePermissions("inventory:opname"), h.ScanOpnameBarcode)
		inv.POST("/opnames/:uuid/submit", middleware.RequirePermissions("inventory:opname"), h.SubmitOpname)
		inv.POST("/opnames/:uuid/approve", middleware.RequirePermissions("inventory:approve-opname"), h.ApproveOpname)

		// Manual Stock Adjustment
		inv.POST("/adjustments", middleware.RequirePermissions("inventory:approve-opname"), h.ManualAdjustment)

		// Suppliers & Purchase Orders
		inv.POST("/suppliers", middleware.RequirePermissions("inventory:transfer"), h.CreateSupplier)
		inv.GET("/suppliers", middleware.RequirePermissions("inventory:read"), h.ListSuppliers)
		inv.POST("/purchase-orders", middleware.RequirePermissions("inventory:transfer"), h.CreatePurchaseOrder)
		inv.GET("/purchase-orders", middleware.RequirePermissions("inventory:read"), h.ListPurchaseOrders)
		inv.GET("/purchase-orders/:uuid", middleware.RequirePermissions("inventory:read"), h.GetPurchaseOrder)
		inv.POST("/purchase-orders/:uuid/receive", middleware.RequirePermissions("inventory:transfer"), h.ReceivePurchaseOrder)
	}
}

// CreateLocation handles create location.
// @Summary Create Location
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateLocationRequest true "CreateLocationRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/locations [post]
func (h *Handler) CreateLocation(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req CreateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreateLocation(c.Request.Context(), tenantID, req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Storage location created successfully", res)
}

// ListLocations handles list locations.
// @Summary List Locations
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/locations [get]
func (h *Handler) ListLocations(c *gin.Context) {
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

	locations, err := h.uc.ListLocations(c.Request.Context(), tenantID, outletID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Storage locations retrieved", locations)
}

// GetStocks handles get stocks.
// @Summary Get Stocks
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/stocks [get]
func (h *Handler) GetStocks(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var locID *uint64
	if locStr := c.Query("location_id"); locStr != "" {
		if lid, err := strconv.ParseUint(locStr, 10, 64); err == nil {
			locID = &lid
		}
	}

	lowStockOnly := c.Query("low_stock") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	stocks, total, err := h.uc.GetStocks(c.Request.Context(), tenantID, locID, lowStockOnly, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stock balances retrieved",
		"data":    stocks,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetStockMovements handles get stock movements.
// @Summary Get Stock Movements
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/movements [get]
func (h *Handler) GetStockMovements(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var filter StockMovementFilterQuery
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	movements, total, err := h.uc.GetStockMovements(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stock movement ledger retrieved",
		"data":    movements,
		"pagination": gin.H{
			"total":  total,
			"limit":  filter.Limit,
			"offset": filter.Offset,
		},
	})
}

// CreateTransfer handles create transfer.
// @Summary Create Transfer
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateStockTransferRequest true "CreateStockTransferRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/transfers [post]
func (h *Handler) CreateTransfer(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	userID, _ := middleware.GetUserID(c)

	var req CreateStockTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreateTransfer(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "Stock transfer initiated and dispatched", res)
}

// ListTransfers handles list transfers.
// @Summary List Transfers
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/transfers [get]
func (h *Handler) ListTransfers(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var status *string
	if st := c.Query("status"); st != "" {
		status = &st
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	transfers, total, err := h.uc.ListTransfers(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stock transfers retrieved",
		"data":    transfers,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetTransfer handles get transfer.
// @Summary Get Transfer
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/transfers/{uuid} [get]
func (h *Handler) GetTransfer(c *gin.Context) {
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

	transfer, err := h.uc.GetTransfer(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Stock transfer not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Stock transfer details", transfer)
}

// ReceiveTransfer handles receive transfer.
// @Summary Receive Transfer
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body ReceiveTransferRequest true "ReceiveTransferRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/transfers/{uuid}/receive [post]
func (h *Handler) ReceiveTransfer(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	userID, _ := middleware.GetUserID(c)
	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req ReceiveTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ReceiveTransfer(c.Request.Context(), tenantID, userID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Stock transfer received successfully", res)
}

// CreateOpname handles create opname.
// @Summary Create Opname
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateStockOpnameRequest true "CreateStockOpnameRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/opnames [post]
func (h *Handler) CreateOpname(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	userID, _ := middleware.GetUserID(c)

	var req CreateStockOpnameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreateOpname(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "Stock opname session initialized", res)
}

// ListOpnames handles list opnames.
// @Summary List Opnames
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/opnames [get]
func (h *Handler) ListOpnames(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var status *string
	if st := c.Query("status"); st != "" {
		status = &st
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	opnames, total, err := h.uc.ListOpnames(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stock opname sessions retrieved",
		"data":    opnames,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetOpname handles get opname.
// @Summary Get Opname
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/opnames/{uuid} [get]
func (h *Handler) GetOpname(c *gin.Context) {
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

	opname, err := h.uc.GetOpname(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Stock opname not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Stock opname session details", opname)
}

// ScanOpnameBarcode handles scan opname barcode.
// @Summary Scan Opname Barcode
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body ScanOpnameBarcodeRequest true "ScanOpnameBarcodeRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/opnames/{uuid}/scan [post]
func (h *Handler) ScanOpnameBarcode(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	userID, _ := middleware.GetUserID(c)
	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req ScanOpnameBarcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ScanOpnameBarcode(c.Request.Context(), tenantID, userID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Barcode scanned and stock recorded", res)
}

// SubmitOpname handles submit opname.
// @Summary Submit Opname
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/opnames/{uuid}/submit [post]
func (h *Handler) SubmitOpname(c *gin.Context) {
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

	res, err := h.uc.SubmitOpname(c.Request.Context(), tenantID, u)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Stock opname submitted for supervisor approval", res)
}

// ApproveOpname handles approve opname.
// @Summary Approve Opname
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/opnames/{uuid}/approve [post]
func (h *Handler) ApproveOpname(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	userID, _ := middleware.GetUserID(c)
	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	res, err := h.uc.ApproveOpname(c.Request.Context(), tenantID, userID, u)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Stock opname approved and ledger adjustments applied", res)
}

// ManualAdjustment handles manual adjustment.
// @Summary Manual Adjustment
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ManualAdjustmentRequest true "ManualAdjustmentRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/adjustments [post]
func (h *Handler) ManualAdjustment(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	userID, _ := middleware.GetUserID(c)

	var req ManualAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ManualAdjustment(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Manual stock adjustment recorded", res)
}

// CreateSupplier handles create supplier.
// @Summary Create Supplier
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateSupplierRequest true "CreateSupplierRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/suppliers [post]
func (h *Handler) CreateSupplier(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreateSupplier(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusCreated, "Supplier created successfully", res)
}

// ListSuppliers handles list suppliers.
// @Summary List Suppliers
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/suppliers [get]
func (h *Handler) ListSuppliers(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	res, err := h.uc.ListSuppliers(c.Request.Context(), tenantID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Suppliers retrieved", res)
}

// CreatePurchaseOrder handles create purchase order.
// @Summary Create Purchase Order
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePurchaseOrderRequest true "CreatePurchaseOrderRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/purchase-orders [post]
func (h *Handler) CreatePurchaseOrder(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}
	userID, _ := middleware.GetUserID(c)

	var req CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreatePurchaseOrder(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusCreated, "Purchase order created successfully", res)
}

// ListPurchaseOrders handles list purchase orders.
// @Summary List Purchase Orders
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/purchase-orders [get]
func (h *Handler) ListPurchaseOrders(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	var statusPtr *string
	if s := c.Query("status"); s != "" {
		statusPtr = &s
	}

	items, total, err := h.uc.ListPurchaseOrders(c.Request.Context(), tenantID, statusPtr, limit, offset)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Purchase orders retrieved", gin.H{
		"items": items,
		"total": total,
	})
}

// GetPurchaseOrder handles get purchase order.
// @Summary Get Purchase Order
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/purchase-orders/{uuid} [get]
func (h *Handler) GetPurchaseOrder(c *gin.Context) {
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

	res, err := h.uc.GetPurchaseOrder(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Purchase order not found", nil)
		return
	}
	response.Success(c, http.StatusOK, "Purchase order retrieved", res)
}

// ReceivePurchaseOrder handles receive purchase order.
// @Summary Receive Purchase Order
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body ReceivePurchaseOrderRequest true "ReceivePurchaseOrderRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/inventory/purchase-orders/{uuid}/receive [post]
func (h *Handler) ReceivePurchaseOrder(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c)
	if !exists {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}
	userID, _ := middleware.GetUserID(c)

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req ReceivePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ReceivePurchaseOrder(c.Request.Context(), tenantID, userID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Purchase order received and stock updated", res)
}
