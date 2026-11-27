package payroll

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// Handler handles HTTP requests for attendance, commissions, and payroll.
type Handler struct {
	uc Usecase
}

// NewHandler creates a new payroll Handler instance.
func NewHandler(uc Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterRoutes mounts payroll and attendance endpoints to the Gin router engine.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	pr := rg.Group("/payroll")
	pr.Use(authMiddleware)
	{
		// Employees & Salary Structure
		pr.POST("/employees", middleware.RequirePermissions("payroll:finalize"), h.CreateEmployee)
		pr.GET("/employees", middleware.RequirePermissions("payroll:read"), h.ListEmployees)
		pr.GET("/employees/:uuid", middleware.RequirePermissions("payroll:read"), h.GetEmployee)
		pr.PUT("/employees/:uuid", middleware.RequirePermissions("payroll:finalize"), h.UpdateEmployee)
		pr.PUT("/employees/:uuid/salary-structure", middleware.RequirePermissions("payroll:finalize"), h.UpsertSalaryStructure)

		// Attendance (Kiosk Clock-in/out + HR/Admin Management)
		pr.POST("/attendance/clock-in", h.ClockIn)
		pr.POST("/attendance/clock-out", h.ClockOut)
		pr.POST("/attendance", middleware.RequirePermissions("payroll:finalize"), h.RecordAttendance)
		pr.GET("/attendance", middleware.RequirePermissions("payroll:read"), h.ListAttendance)

		// Sales Commissions
		pr.POST("/commissions/calculate", middleware.RequirePermissions("payroll:finalize"), h.CalculateOrderCommission)
		pr.GET("/commissions", middleware.RequirePermissions("payroll:read"), h.ListCommissions)
		pr.POST("/commissions/:uuid/void", middleware.RequirePermissions("payroll:finalize"), h.VoidCommission)

		// Payroll Periods, Preview & Immutable Finalization
		pr.POST("/periods", middleware.RequirePermissions("payroll:finalize"), h.CreatePeriod)
		pr.GET("/periods", middleware.RequirePermissions("payroll:read"), h.ListPeriods)
		pr.GET("/periods/:uuid", middleware.RequirePermissions("payroll:read"), h.GetPeriod)
		pr.GET("/periods/:uuid/preview", middleware.RequirePermissions("payroll:read"), h.PreviewPeriodPayroll)
		pr.POST("/periods/:uuid/finalize", middleware.RequirePermissions("payroll:finalize"), h.FinalizePeriod)

		// Immutable Payslip Snapshots
		pr.GET("/slips", middleware.RequirePermissions("payroll:read"), h.ListSnapshots)
		pr.GET("/slips/:uuid", middleware.RequirePermissions("payroll:read"), h.GetSnapshot)
		pr.GET("/slips/:uuid/pdf", middleware.RequirePermissions("payroll:read"), h.DownloadSnapshotPDF)
	}
}

// CreateEmployee handles create employee.
// @Summary Create Employee
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateEmployeeRequest true "CreateEmployeeRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/employees [post]
func (h *Handler) CreateEmployee(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreateEmployee(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusCreated, "Employee registered successfully", res)
}

// ListEmployees handles list employees.
// @Summary List Employees
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/employees [get]
func (h *Handler) ListEmployees(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var query ListEmployeesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ListEmployees(c.Request.Context(), tenantID, query)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Employees retrieved", res)
}

// GetEmployee handles get employee.
// @Summary Get Employee
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/employees/{uuid} [get]
func (h *Handler) GetEmployee(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	res, err := h.uc.GetEmployee(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Employee not found", nil)
		return
	}
	response.Success(c, http.StatusOK, "Employee details retrieved", res)
}

// UpdateEmployee handles update employee.
// @Summary Update Employee
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body UpdateEmployeeRequest true "UpdateEmployeeRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/employees/{uuid} [put]
func (h *Handler) UpdateEmployee(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.UpdateEmployee(c.Request.Context(), tenantID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Employee updated successfully", res)
}

// UpsertSalaryStructure handles upsert salary structure.
// @Summary Upsert Salary Structure
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body SalaryStructureRequest true "SalaryStructureRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/employees/{uuid}/salary-structure [put]
func (h *Handler) UpsertSalaryStructure(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req SalaryStructureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.UpsertSalaryStructure(c.Request.Context(), tenantID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Salary structure saved successfully", res)
}

// ClockIn handles kiosk attendance clock-in.
// @Summary Clock In
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ClockInRequest true "Clock In Payload"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/attendance/clock-in [post]
func (h *Handler) ClockIn(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req ClockInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ClockIn(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusCreated, "Clock-in recorded successfully", res)
}

// ClockOut handles kiosk attendance clock-out.
// @Summary Clock Out
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ClockOutRequest true "Clock Out Payload"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/attendance/clock-out [post]
func (h *Handler) ClockOut(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req ClockOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ClockOut(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Clock-out recorded successfully", res)
}

// RecordAttendance handles record attendance.
// @Summary Record Attendance
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body RecordAttendanceRequest true "RecordAttendanceRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/attendance [post]
func (h *Handler) RecordAttendance(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req RecordAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.RecordAttendance(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Attendance recorded successfully", res)
}

// ListAttendance handles list attendance.
// @Summary List Attendance
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/attendance [get]
func (h *Handler) ListAttendance(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var query ListAttendanceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ListAttendance(c.Request.Context(), tenantID, query)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Attendance logs retrieved", res)
}

// CalculateOrderCommission handles calculate order commission.
// @Summary Calculate Order Commission
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CalculateCommissionRequest true "CalculateCommissionRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/commissions/calculate [post]
func (h *Handler) CalculateOrderCommission(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req CalculateCommissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CalculateOrderCommission(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusCreated, "Commission calculated successfully", res)
}

// ListCommissions handles list commissions.
// @Summary List Commissions
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/commissions [get]
func (h *Handler) ListCommissions(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var query ListCommissionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ListCommissions(c.Request.Context(), tenantID, query)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Commissions retrieved", res)
}

// VoidCommission handles void commission.
// @Summary Void Commission
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/commissions/{uuid}/void [post]
func (h *Handler) VoidCommission(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	res, err := h.uc.VoidCommission(c.Request.Context(), tenantID, u)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Commission voided successfully", res)
}

// CreatePeriod handles create period.
// @Summary Create Period
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePeriodRequest true "CreatePeriodRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/periods [post]
func (h *Handler) CreatePeriod(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var req CreatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.CreatePeriod(c.Request.Context(), tenantID, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusCreated, "Payroll period created successfully", res)
}

// ListPeriods handles list periods.
// @Summary List Periods
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/periods [get]
func (h *Handler) ListPeriods(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var query ListPeriodsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ListPeriods(c.Request.Context(), tenantID, query)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Payroll periods retrieved", res)
}

// GetPeriod handles get period.
// @Summary Get Period
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/periods/{uuid} [get]
func (h *Handler) GetPeriod(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	res, err := h.uc.GetPeriod(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payroll period not found", nil)
		return
	}
	response.Success(c, http.StatusOK, "Payroll period retrieved", res)
}

// PreviewPeriodPayroll handles preview period payroll.
// @Summary Preview Period Payroll
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/periods/{uuid}/preview [get]
func (h *Handler) PreviewPeriodPayroll(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var query PreviewPeriodQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.PreviewPeriodPayroll(c.Request.Context(), tenantID, u, query)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Payroll preview calculated", res)
}

// FinalizePeriod handles finalize period.
// @Summary Finalize Period
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Param request body FinalizePeriodRequest true "FinalizePeriodRequest"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/periods/{uuid}/finalize [post]
func (h *Handler) FinalizePeriod(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "Missing user identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	var req FinalizePeriodRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error(), nil)
			return
		}
	}

	res, err := h.uc.FinalizePeriod(c.Request.Context(), tenantID, userID, u, req)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Payroll period finalized and snapshots locked", res)
}

// ListSnapshots handles list snapshots.
// @Summary List Snapshots
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/slips [get]
func (h *Handler) ListSnapshots(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	var query SnapshotListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	res, err := h.uc.ListSnapshots(c.Request.Context(), tenantID, query)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Payslip snapshots retrieved", res)
}

// GetSnapshot handles get snapshot.
// @Summary Get Snapshot
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/slips/{uuid} [get]
func (h *Handler) GetSnapshot(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	res, err := h.uc.GetSnapshot(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payslip snapshot not found", nil)
		return
	}
	response.Success(c, http.StatusOK, "Payslip snapshot retrieved", res)
}

// DownloadSnapshotPDF handles download snapshot pdf.
// @Summary Download Snapshot PDF
// @Tags Payroll
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uuid path string true "uuid"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/payroll/slips/{uuid}/pdf [get]
func (h *Handler) DownloadSnapshotPDF(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c)
	if !ok {
		response.Unauthorized(c, "Missing tenant identification context")
		return
	}

	u, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.BadRequest(c, "Invalid UUID parameter format", nil)
		return
	}

	snap, err := h.uc.GetSnapshot(c.Request.Context(), tenantID, u)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payslip snapshot not found", nil)
		return
	}

	pdfBytes := GeneratePayslipPDF(snap)
	filename := "slip-gaji-" + snap.EmployeeCode + ".pdf"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
