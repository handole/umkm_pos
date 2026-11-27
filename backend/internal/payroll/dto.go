package payroll

import (
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
)

// CreateEmployeeRequest represents payload to register a new employee.
type CreateEmployeeRequest struct {
	UserID            *uint64                 `json:"user_id,omitempty"`
	OutletID          uint64                  `json:"outlet_id" binding:"required"`
	EmployeeCode      string                  `json:"employee_code" binding:"required,max=50"`
	FullName          string                  `json:"full_name" binding:"required,max=255"`
	JobTitle          string                  `json:"job_title" binding:"required,max=100"`
	Phone             string                  `json:"phone" binding:"max=30"`
	BankName          string                  `json:"bank_name" binding:"max=100"`
	BankAccountNumber string                  `json:"bank_account_number" binding:"max=100"`
	BankAccountHolder string                  `json:"bank_account_holder" binding:"max=255"`
	JoinedDate        string                  `json:"joined_date" binding:"required"` // YYYY-MM-DD
	EmploymentStatus  string                  `json:"employment_status" binding:"omitempty,oneof=PERMANENT CONTRACT PROBATION"`
	SalaryStructure   *SalaryStructureRequest `json:"salary_structure,omitempty"`
}

// UpdateEmployeeRequest represents payload to update employee details.
type UpdateEmployeeRequest struct {
	UserID            *uint64 `json:"user_id,omitempty"`
	OutletID          *uint64 `json:"outlet_id,omitempty"`
	FullName          *string `json:"full_name,omitempty" binding:"omitempty,max=255"`
	JobTitle          *string `json:"job_title,omitempty" binding:"omitempty,max=100"`
	Phone             *string `json:"phone,omitempty" binding:"omitempty,max=30"`
	BankName          *string `json:"bank_name,omitempty" binding:"omitempty,max=100"`
	BankAccountNumber *string `json:"bank_account_number,omitempty" binding:"omitempty,max=100"`
	BankAccountHolder *string `json:"bank_account_holder,omitempty" binding:"omitempty,max=255"`
	EmploymentStatus  *string `json:"employment_status,omitempty" binding:"omitempty,oneof=PERMANENT CONTRACT PROBATION"`
	IsActive          *bool   `json:"is_active,omitempty"`
}

// SalaryStructureRequest represents payload to create or update an employee's salary structure.
type SalaryStructureRequest struct {
	BasicSalary             int64   `json:"basic_salary" binding:"gte=0"`
	FixedAllowance          int64   `json:"fixed_allowance" binding:"gte=0"`
	MealTransportAllowance  int64   `json:"meal_transport_allowance" binding:"gte=0"`
	CommissionRatePercent   float64 `json:"commission_rate_percent" binding:"gte=0,lte=100"`
	CommissionFixedPerItem  int64   `json:"commission_fixed_per_item" binding:"gte=0"`
	BPJSHealthDeduction     int64   `json:"bpjs_health_deduction" binding:"gte=0"`
	BPJSEmploymentDeduction int64   `json:"bpjs_employment_deduction" binding:"gte=0"`
}

// ListEmployeesQuery represents query filters for listing employees.
type ListEmployeesQuery struct {
	OutletID         uint64 `form:"outlet_id"`
	IsActive         *bool  `form:"is_active"`
	EmploymentStatus string `form:"employment_status"`
	Search           string `form:"search"`
	Page             int    `form:"page"`
	Limit            int    `form:"limit"`
}

// ClockInRequest represents kiosk attendance clock-in payload.
type ClockInRequest struct {
	EmployeeUUID uuid.UUID `json:"employee_uuid" binding:"required"`
	OutletID     uint64    `json:"outlet_id" binding:"required"`
	Pin          string    `json:"pin"` // Verified if employee is linked to a user with PinHash
	PhotoURL     string    `json:"photo_url" binding:"max=500"`
	LogDate      string    `json:"log_date"`     // Optional YYYY-MM-DD; defaults to today
	ClockInTime  string    `json:"clock_in_at"`  // Optional RFC3339; defaults to now
	LateMinutes  int       `json:"late_minutes" binding:"gte=0"`
}

// ClockOutRequest represents kiosk attendance clock-out payload.
type ClockOutRequest struct {
	EmployeeUUID    uuid.UUID `json:"employee_uuid" binding:"required"`
	Pin             string    `json:"pin"`
	PhotoURL        string    `json:"photo_url" binding:"max=500"`
	LogDate         string    `json:"log_date"`     // Optional YYYY-MM-DD; defaults to today
	ClockOutTime    string    `json:"clock_out_at"` // Optional RFC3339; defaults to now
	OvertimeMinutes int       `json:"overtime_minutes" binding:"gte=0"`
}

// RecordAttendanceRequest represents HR/Admin manual attendance entry or correction.
type RecordAttendanceRequest struct {
	EmployeeUUID    uuid.UUID `json:"employee_uuid" binding:"required"`
	OutletID        uint64    `json:"outlet_id" binding:"required"`
	LogDate         string    `json:"log_date" binding:"required"` // YYYY-MM-DD
	Status          string    `json:"status" binding:"required,oneof=PRESENT LATE ABSENT LEAVE SICK"`
	LateMinutes     int       `json:"late_minutes" binding:"gte=0"`
	OvertimeMinutes int       `json:"overtime_minutes" binding:"gte=0"`
}

// ListAttendanceQuery represents query filters for attendance logs.
type ListAttendanceQuery struct {
	EmployeeUUID string `form:"employee_uuid"`
	OutletID     uint64 `form:"outlet_id"`
	StartDate    string `form:"start_date"`
	EndDate      string `form:"end_date"`
	Status       string `form:"status"`
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
}

// CalculateCommissionRequest represents a request to compute and queue sales commission for a POS order.
type CalculateCommissionRequest struct {
	EmployeeUUID uuid.UUID `json:"employee_uuid" binding:"required"`
	OrderID      uint64    `json:"order_id" binding:"required"`
	OrderTotal   int64     `json:"order_total" binding:"gte=0"`
	TotalItems   int       `json:"total_items" binding:"gte=0"`
}

// ListCommissionsQuery represents query filters for pending/processed commissions.
type ListCommissionsQuery struct {
	EmployeeUUID string `form:"employee_uuid"`
	Status       string `form:"status"`
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
}

// CreatePeriodRequest represents payload to open a new monthly payroll period.
type CreatePeriodRequest struct {
	PeriodName  string `json:"period_name" binding:"required,max=100"`
	StartDate   string `json:"start_date" binding:"required"`   // YYYY-MM-DD
	EndDate     string `json:"end_date" binding:"required"`     // YYYY-MM-DD
	PaymentDate string `json:"payment_date" binding:"required"` // YYYY-MM-DD
}

// ListPeriodsQuery represents query filters for payroll periods.
type ListPeriodsQuery struct {
	Status string `form:"status"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

// PayrollAdjustmentInput allows HR/Admin to add ad-hoc bonus or deduction (e.g. kasbon) during finalization.
type PayrollAdjustmentInput struct {
	EmployeeUUID  uuid.UUID `json:"employee_uuid" binding:"required"`
	ItemType      string    `json:"item_type" binding:"required,oneof=EARNING DEDUCTION"`
	ComponentName string    `json:"component_name" binding:"required,max=100"`
	Amount        int64     `json:"amount" binding:"gt=0"`
	Notes         string    `json:"notes" binding:"max=255"`
}

// FinalizePeriodRequest represents payload to finalize and lock a payroll period.
type FinalizePeriodRequest struct {
	OvertimeRatePerHour int64                    `json:"overtime_rate_per_hour" binding:"gte=0"`
	LatePenaltyPerMin   int64                    `json:"late_penalty_per_minute" binding:"gte=0"`
	Adjustments         []PayrollAdjustmentInput `json:"adjustments,omitempty" binding:"dive"`
}

// PreviewPeriodQuery represents optional calculation rates for live payroll preview.
type PreviewPeriodQuery struct {
	OvertimeRatePerHour int64 `form:"overtime_rate_per_hour"`
	LatePenaltyPerMin   int64 `form:"late_penalty_per_minute"`
}

// EmployeePayrollPreview represents a single employee's calculated payroll breakdown before finalization.
type EmployeePayrollPreview struct {
	EmployeeUUID      uuid.UUID                   `json:"employee_uuid"`
	EmployeeCode      string                      `json:"employee_code"`
	EmployeeName      string                      `json:"employee_name"`
	JobTitle          string                      `json:"job_title"`
	PresentDays       int                         `json:"present_days"`
	LateDays          int                         `json:"late_days"`
	AbsentDays        int                         `json:"absent_days"`
	TotalLateMinutes  int                         `json:"total_late_minutes"`
	TotalOvertimeMins int                         `json:"total_overtime_minutes"`
	BasicSalary       int64                       `json:"basic_salary"`
	TotalAllowances   int64                       `json:"total_allowances"`
	TotalOvertimePay  int64                       `json:"total_overtime_pay"`
	TotalCommissions  int64                       `json:"total_commissions"`
	TotalDeductions   int64                       `json:"total_deductions"`
	NetSalary         int64                       `json:"net_salary"`
	Items             []model.PayrollItemSnapshot `json:"items"`
}

// PeriodPayrollPreviewResponse represents the complete preview for a payroll period.
type PeriodPayrollPreviewResponse struct {
	Period      model.PayrollPeriod      `json:"period"`
	TotalPayout int64                    `json:"total_payout"`
	Employees   []EmployeePayrollPreview `json:"employees"`
}

// PagedResponse wraps paginated list responses.
type PagedResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int64 `json:"total_pages"`
}

// PeriodSummaryResponse wraps period with its finalized snapshots when requested.
type PeriodDetailResponse struct {
	Period    model.PayrollPeriod     `json:"period"`
	Snapshots []model.PayrollSnapshot `json:"snapshots,omitempty"`
}

// SnapshotListQuery represents query filters for finalized payslip snapshots.
type SnapshotListQuery struct {
	PeriodUUID   string `form:"period_uuid"`
	EmployeeUUID string `form:"employee_uuid"`
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
}

// AttendanceSummary aggregates attendance counts and minutes for a period.
type AttendanceSummary struct {
	PresentDays       int
	LateDays          int
	AbsentDays        int
	TotalLateMinutes  int
	TotalOvertimeMins int
}

// parseDate parses YYYY-MM-DD in UTC date format.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
