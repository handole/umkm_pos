package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Employee represents staff members employed by an outlet/tenant.
type Employee struct {
	ID                 uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID               uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_employees_tenant_uuid,priority:2" json:"uuid"`
	TenantID           uint64         `gorm:"not null;uniqueIndex:idx_employees_tenant_uuid,priority:1;uniqueIndex:idx_employees_tenant_code,priority:1" json:"tenant_id"`
	UserID             *uint64        `gorm:"index" json:"user_id,omitempty"` // Relasi ke akun login jika staf berhak login
	OutletID           uint64         `gorm:"not null;index:idx_employees_tenant_active,priority:2" json:"outlet_id"`
	EmployeeCode       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_employees_tenant_code,priority:2" json:"employee_code"`
	FullName           string         `gorm:"type:varchar(255);not null" json:"full_name"`
	JobTitle           string         `gorm:"type:varchar(100);not null" json:"job_title"`
	Phone              string         `gorm:"type:varchar(30)" json:"phone"`
	BankName           string         `gorm:"type:varchar(100)" json:"bank_name"`
	BankAccountNumber  string         `gorm:"type:varchar(100)" json:"bank_account_number"`
	BankAccountHolder  string         `gorm:"type:varchar(255)" json:"bank_account_holder"`
	JoinedDate         time.Time      `gorm:"type:date;not null" json:"joined_date"`
	EmploymentStatus   string         `gorm:"type:varchar(30);not null;default:'PERMANENT'" json:"employment_status"` // 'PERMANENT', 'CONTRACT', 'PROBATION'
	IsActive           bool           `gorm:"not null;default:true;index:idx_employees_tenant_active,priority:3" json:"is_active"`
	CreatedAt          time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant          *Tenant                   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	User            *User                     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Outlet          *Outlet                   `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
	SalaryStructure *SalaryStructure          `gorm:"foreignKey:EmployeeID" json:"salary_structure,omitempty"`
	AttendanceLogs  []AttendanceLog           `gorm:"foreignKey:EmployeeID" json:"attendance_logs,omitempty"`
	Commissions     []PayrollCommissionPending `gorm:"foreignKey:EmployeeID" json:"commissions,omitempty"`
	Snapshots       []PayrollSnapshot         `gorm:"foreignKey:EmployeeID" json:"snapshots,omitempty"`
}

func (Employee) TableName() string {
	return "employees"
}

// SalaryStructure defines baseline compensation, allowances, commission formulas, and deductions.
type SalaryStructure struct {
	ID                       uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_salary_struct_tenant_uuid,priority:2" json:"uuid"`
	TenantID                 uint64    `gorm:"not null;uniqueIndex:idx_salary_struct_tenant_uuid,priority:1" json:"tenant_id"`
	EmployeeID               uint64    `gorm:"not null;uniqueIndex" json:"employee_id"`
	BasicSalary              int64     `gorm:"not null;default:0" json:"basic_salary"`
	FixedAllowance           int64     `gorm:"not null;default:0" json:"fixed_allowance"`
	MealTransportAllowance   int64     `gorm:"not null;default:0" json:"meal_transport_allowance"`
	CommissionRatePercent    float64   `gorm:"type:numeric(5,2);not null;default:0" json:"commission_rate_percent"` // % omzet penjualan POS
	CommissionFixedPerItem   int64     `gorm:"not null;default:0" json:"commission_fixed_per_item"`
	BPJSHealthDeduction      int64     `gorm:"not null;default:0" json:"bpjs_health_deduction"`
	BPJSEmploymentDeduction  int64     `gorm:"not null;default:0" json:"bpjs_employment_deduction"`
	UpdatedAt                time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Employee *Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
}

func (SalaryStructure) TableName() string {
	return "salary_structures"
}

// AttendanceLog tracks clock-in/out, liveness photo verification, late minutes, and overtime.
type AttendanceLog struct {
	ID                uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID              uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_attendance_tenant_uuid,priority:2" json:"uuid"`
	TenantID          uint64     `gorm:"not null;uniqueIndex:idx_attendance_tenant_uuid,priority:1;uniqueIndex:idx_attendance_employee_date,priority:1" json:"tenant_id"`
	EmployeeID        uint64     `gorm:"not null;uniqueIndex:idx_attendance_employee_date,priority:2;index:idx_attendance_period,priority:2" json:"employee_id"`
	OutletID          uint64     `gorm:"not null;index" json:"outlet_id"`
	LogDate           time.Time  `gorm:"type:date;not null;uniqueIndex:idx_attendance_employee_date,priority:3;index:idx_attendance_period,priority:3" json:"log_date"`
	ClockIn           *time.Time `json:"clock_in,omitempty"`
	ClockOut          *time.Time `json:"clock_out,omitempty"`
	ClockInPhotoURL   string     `gorm:"type:varchar(500)" json:"clock_in_photo_url"`
	ClockOutPhotoURL  string     `gorm:"type:varchar(500)" json:"clock_out_photo_url"`
	LateMinutes       int        `gorm:"not null;default:0" json:"late_minutes"`
	OvertimeMinutes   int        `gorm:"not null;default:0" json:"overtime_minutes"`
	Status            string     `gorm:"type:varchar(30);not null;default:'PRESENT'" json:"status"` // 'PRESENT', 'LATE', 'ABSENT', 'LEAVE', 'SICK'
	CreatedAt         time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Employee *Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Outlet   *Outlet   `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
}

func (AttendanceLog) TableName() string {
	return "attendance_logs"
}

// PayrollCommissionPending queues earned sales commissions from completed POS transactions.
type PayrollCommissionPending struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	TenantID         uint64    `gorm:"not null;index:idx_commissions_employee_status,priority:1" json:"tenant_id"`
	EmployeeID       uint64    `gorm:"not null;index:idx_commissions_employee_status,priority:2" json:"employee_id"`
	OrderID          uint64    `gorm:"not null;index" json:"order_id"`
	CommissionAmount int64     `gorm:"not null" json:"commission_amount"`
	Status           string    `gorm:"type:varchar(20);not null;default:'PENDING';index:idx_commissions_employee_status,priority:3" json:"status"` // 'PENDING', 'CALCULATED', 'PAID', 'VOID'
	CreatedAt        time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Relations
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Employee *Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Order    *PosOrder `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

func (PayrollCommissionPending) TableName() string {
	return "payroll_commissions_pending"
}

// PayrollPeriod represents monthly payroll cycles.
type PayrollPeriod struct {
	ID          uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID        uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_payroll_period_tenant_uuid,priority:2" json:"uuid"`
	TenantID    uint64            `gorm:"not null;uniqueIndex:idx_payroll_period_tenant_uuid,priority:1;uniqueIndex:idx_payroll_period_tenant_name,priority:1" json:"tenant_id"`
	PeriodName  string            `gorm:"type:varchar(100);not null;uniqueIndex:idx_payroll_period_tenant_name,priority:2" json:"period_name"` // e.g. 'November 2026'
	StartDate   time.Time         `gorm:"type:date;not null" json:"start_date"`
	EndDate     time.Time         `gorm:"type:date;not null" json:"end_date"`
	PaymentDate time.Time         `gorm:"type:date;not null" json:"payment_date"`
	Status      string            `gorm:"type:varchar(30);not null;default:'DRAFT'" json:"status"` // 'DRAFT', 'LOCKED', 'FINALIZED'
	TotalPayout int64             `gorm:"not null;default:0" json:"total_payout"`
	FinalizedBy *uint64           `gorm:"index" json:"finalized_by,omitempty"`
	FinalizedAt *time.Time        `json:"finalized_at,omitempty"`
	CreatedAt   time.Time         `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant    *Tenant           `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Finalizer *User             `gorm:"foreignKey:FinalizedBy" json:"finalizer,omitempty"`
	Snapshots []PayrollSnapshot `gorm:"foreignKey:PeriodID" json:"snapshots,omitempty"`
}

func (PayrollPeriod) TableName() string {
	return "payroll_periods"
}

// PayrollSnapshot stores immutable historical payslip data once a period is FINALIZED.
type PayrollSnapshot struct {
	ID                uint64                `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID              uuid.UUID             `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_payroll_snapshot_tenant_uuid,priority:2" json:"uuid"`
	TenantID          uint64                `gorm:"not null;uniqueIndex:idx_payroll_snapshot_tenant_uuid,priority:1;index:idx_payroll_snapshots_period,priority:1" json:"tenant_id"`
	PeriodID          uint64                `gorm:"not null;uniqueIndex:idx_payroll_snapshot_period_emp,priority:1;index:idx_payroll_snapshots_period,priority:2" json:"period_id"`
	EmployeeID        uint64                `gorm:"not null;uniqueIndex:idx_payroll_snapshot_period_emp,priority:2" json:"employee_id"`
	EmployeeCode      string                `gorm:"type:varchar(50);not null" json:"employee_code"`
	EmployeeName      string                `gorm:"type:varchar(255);not null" json:"employee_name"`
	JobTitle          string                `gorm:"type:varchar(100);not null" json:"job_title"`
	BankName          string                `gorm:"type:varchar(100)" json:"bank_name"`
	BankAccountNumber string                `gorm:"type:varchar(100)" json:"bank_account_number"`
	BasicSalary       int64                 `gorm:"not null" json:"basic_salary"`
	TotalAllowances   int64                 `gorm:"not null;default:0" json:"total_allowances"`
	TotalOvertimePay  int64                 `gorm:"not null;default:0" json:"total_overtime_pay"`
	TotalCommissions  int64                 `gorm:"not null;default:0" json:"total_commissions"`
	TotalDeductions   int64                 `gorm:"not null;default:0" json:"total_deductions"` // Kasbon, keterlambatan, BPJS, pajak
	NetSalary         int64                 `gorm:"not null" json:"net_salary"`
	SlipPDFURL        string                `gorm:"type:varchar(500)" json:"slip_pdf_url"`
	Status            string                `gorm:"type:varchar(30);not null;default:'FINALIZED'" json:"status"`
	CreatedAt         time.Time             `gorm:"not null;default:now()" json:"created_at"`

	// Relations
	Tenant   *Tenant               `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Period   *PayrollPeriod        `gorm:"foreignKey:PeriodID" json:"period,omitempty"`
	Employee *Employee             `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Items    []PayrollItemSnapshot `gorm:"foreignKey:SnapshotID" json:"items,omitempty"`
}

func (PayrollSnapshot) TableName() string {
	return "payroll_snapshots"
}

// PayrollItemSnapshot breaks down individual earnings and deductions on a payslip.
type PayrollItemSnapshot struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	SnapshotID    uint64    `gorm:"not null;index" json:"snapshot_id"`
	ItemType      string    `gorm:"type:varchar(20);not null" json:"item_type"` // 'EARNING', 'DEDUCTION'
	ComponentName string    `gorm:"type:varchar(100);not null" json:"component_name"` // 'Gaji Pokok', 'Komisi POS', 'Potongan Kasbon'
	Amount        int64     `gorm:"not null" json:"amount"`
	Notes         string    `gorm:"type:varchar(255)" json:"notes"`

	// Relations
	Snapshot *PayrollSnapshot `gorm:"foreignKey:SnapshotID" json:"snapshot,omitempty"`
}

func (PayrollItemSnapshot) TableName() string {
	return "payroll_item_snapshots"
}
