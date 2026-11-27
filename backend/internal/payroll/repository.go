package payroll

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/gorm"
)

// Repository defines persistence operations for the payroll & attendance domain.
type Repository interface {
	WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error

	// Employees & Salary Structures
	CreateEmployee(ctx context.Context, emp *model.Employee) error
	FindEmployeeByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Employee, error)
	FindEmployeeByCode(ctx context.Context, tenantID uint64, code string) (*model.Employee, error)
	ListEmployees(ctx context.Context, tenantID uint64, query ListEmployeesQuery) ([]model.Employee, int64, error)
	ListActiveEmployeesWithSalary(ctx context.Context, tenantID uint64) ([]model.Employee, error)
	UpdateEmployee(ctx context.Context, emp *model.Employee) error
	UpsertSalaryStructure(ctx context.Context, ss *model.SalaryStructure) error
	FindUserByID(ctx context.Context, tenantID uint64, userID uint64) (*model.User, error)

	// Attendance
	FindAttendanceByEmployeeDate(ctx context.Context, tenantID uint64, employeeID uint64, logDate time.Time) (*model.AttendanceLog, error)
	CreateAttendance(ctx context.Context, log *model.AttendanceLog) error
	UpdateAttendance(ctx context.Context, log *model.AttendanceLog) error
	ListAttendance(ctx context.Context, tenantID uint64, query ListAttendanceQuery) ([]model.AttendanceLog, int64, error)
	ListAttendanceByEmployeePeriod(ctx context.Context, tenantID uint64, employeeID uint64, startDate, endDate time.Time) ([]model.AttendanceLog, error)

	// Commissions
	FindCommissionByOrder(ctx context.Context, tenantID uint64, orderID uint64, employeeID uint64) (*model.PayrollCommissionPending, error)
	FindCommissionByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollCommissionPending, error)
	CreateCommission(ctx context.Context, comm *model.PayrollCommissionPending) error
	UpdateCommission(ctx context.Context, comm *model.PayrollCommissionPending) error
	ListCommissions(ctx context.Context, tenantID uint64, query ListCommissionsQuery) ([]model.PayrollCommissionPending, int64, error)
	ListPendingCommissionsByEmployee(ctx context.Context, tenantID uint64, employeeID uint64) ([]model.PayrollCommissionPending, error)
	MarkCommissionsPaid(ctx context.Context, tenantID uint64, ids []uint64) error

	// Payroll Periods & Snapshots
	CreatePeriod(ctx context.Context, period *model.PayrollPeriod) error
	FindPeriodByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollPeriod, error)
	FindPeriodByName(ctx context.Context, tenantID uint64, name string) (*model.PayrollPeriod, error)
	ListPeriods(ctx context.Context, tenantID uint64, query ListPeriodsQuery) ([]model.PayrollPeriod, int64, error)
	UpdatePeriod(ctx context.Context, period *model.PayrollPeriod) error
	CreateSnapshot(ctx context.Context, snapshot *model.PayrollSnapshot) error
	ListSnapshots(ctx context.Context, tenantID uint64, query SnapshotListQuery) ([]model.PayrollSnapshot, int64, error)
	FindSnapshotByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollSnapshot, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates a new GORM-backed payroll Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormRepository{db: tx})
	})
}

func (r *gormRepository) CreateEmployee(ctx context.Context, emp *model.Employee) error {
	return r.db.WithContext(ctx).Create(emp).Error
}

func (r *gormRepository) FindEmployeeByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Employee, error) {
	var emp model.Employee
	err := r.db.WithContext(ctx).
		Preload("SalaryStructure").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&emp).Error
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

func (r *gormRepository) FindEmployeeByCode(ctx context.Context, tenantID uint64, code string) (*model.Employee, error) {
	var emp model.Employee
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND employee_code = ?", tenantID, code).
		First(&emp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

func (r *gormRepository) ListEmployees(ctx context.Context, tenantID uint64, query ListEmployeesQuery) ([]model.Employee, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.Employee{}).Where("tenant_id = ?", tenantID)

	if query.OutletID > 0 {
		db = db.Where("outlet_id = ?", query.OutletID)
	}
	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}
	if query.EmploymentStatus != "" {
		db = db.Where("employment_status = ?", query.EmploymentStatus)
	}
	if query.Search != "" {
		like := "%" + query.Search + "%"
		db = db.Where("full_name ILIKE ? OR employee_code ILIKE ?", like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := normalizePagination(query.Page, query.Limit)
	var items []model.Employee
	err := db.Preload("SalaryStructure").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&items).Error
	return items, total, err
}

func (r *gormRepository) ListActiveEmployeesWithSalary(ctx context.Context, tenantID uint64) ([]model.Employee, error) {
	var items []model.Employee
	err := r.db.WithContext(ctx).
		Preload("SalaryStructure").
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

func (r *gormRepository) UpdateEmployee(ctx context.Context, emp *model.Employee) error {
	return r.db.WithContext(ctx).Save(emp).Error
}

func (r *gormRepository) UpsertSalaryStructure(ctx context.Context, ss *model.SalaryStructure) error {
	var existing model.SalaryStructure
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND employee_id = ?", ss.TenantID, ss.EmployeeID).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(ss).Error
	}
	if err != nil {
		return err
	}
	ss.ID = existing.ID
	ss.UUID = existing.UUID
	return r.db.WithContext(ctx).Save(ss).Error
}

func (r *gormRepository) FindUserByID(ctx context.Context, tenantID uint64, userID uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) FindAttendanceByEmployeeDate(ctx context.Context, tenantID uint64, employeeID uint64, logDate time.Time) (*model.AttendanceLog, error) {
	var log model.AttendanceLog
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND employee_id = ? AND log_date = ?", tenantID, employeeID, logDate.Format("2006-01-02")).
		First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *gormRepository) CreateAttendance(ctx context.Context, log *model.AttendanceLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *gormRepository) UpdateAttendance(ctx context.Context, log *model.AttendanceLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *gormRepository) ListAttendance(ctx context.Context, tenantID uint64, query ListAttendanceQuery) ([]model.AttendanceLog, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.AttendanceLog{}).Where("tenant_id = ?", tenantID)

	if query.EmployeeUUID != "" {
		u, err := uuid.Parse(query.EmployeeUUID)
		if err == nil {
			var emp model.Employee
			if err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&emp).Error; err == nil {
				db = db.Where("employee_id = ?", emp.ID)
			} else {
				return []model.AttendanceLog{}, 0, nil
			}
		}
	}
	if query.OutletID > 0 {
		db = db.Where("outlet_id = ?", query.OutletID)
	}
	if query.StartDate != "" {
		db = db.Where("log_date >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		db = db.Where("log_date <= ?", query.EndDate)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := normalizePagination(query.Page, query.Limit)
	var items []model.AttendanceLog
	err := db.Preload("Employee").
		Order("log_date DESC, id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&items).Error
	return items, total, err
}

func (r *gormRepository) ListAttendanceByEmployeePeriod(ctx context.Context, tenantID uint64, employeeID uint64, startDate, endDate time.Time) ([]model.AttendanceLog, error) {
	var logs []model.AttendanceLog
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND employee_id = ? AND log_date >= ? AND log_date <= ?",
			tenantID, employeeID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Find(&logs).Error
	return logs, err
}

func (r *gormRepository) FindCommissionByOrder(ctx context.Context, tenantID uint64, orderID uint64, employeeID uint64) (*model.PayrollCommissionPending, error) {
	var comm model.PayrollCommissionPending
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND order_id = ? AND employee_id = ?", tenantID, orderID, employeeID).
		First(&comm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &comm, nil
}

func (r *gormRepository) FindCommissionByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollCommissionPending, error) {
	var comm model.PayrollCommissionPending
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&comm).Error
	if err != nil {
		return nil, err
	}
	return &comm, nil
}

func (r *gormRepository) CreateCommission(ctx context.Context, comm *model.PayrollCommissionPending) error {
	return r.db.WithContext(ctx).Create(comm).Error
}

func (r *gormRepository) UpdateCommission(ctx context.Context, comm *model.PayrollCommissionPending) error {
	return r.db.WithContext(ctx).Save(comm).Error
}

func (r *gormRepository) ListCommissions(ctx context.Context, tenantID uint64, query ListCommissionsQuery) ([]model.PayrollCommissionPending, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.PayrollCommissionPending{}).Where("tenant_id = ?", tenantID)

	if query.EmployeeUUID != "" {
		u, err := uuid.Parse(query.EmployeeUUID)
		if err == nil {
			var emp model.Employee
			if err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&emp).Error; err == nil {
				db = db.Where("employee_id = ?", emp.ID)
			} else {
				return []model.PayrollCommissionPending{}, 0, nil
			}
		}
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := normalizePagination(query.Page, query.Limit)
	var items []model.PayrollCommissionPending
	err := db.Preload("Employee").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&items).Error
	return items, total, err
}

func (r *gormRepository) ListPendingCommissionsByEmployee(ctx context.Context, tenantID uint64, employeeID uint64) ([]model.PayrollCommissionPending, error) {
	var comms []model.PayrollCommissionPending
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND employee_id = ? AND status = ?", tenantID, employeeID, "PENDING").
		Find(&comms).Error
	return comms, err
}

func (r *gormRepository) MarkCommissionsPaid(ctx context.Context, tenantID uint64, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.PayrollCommissionPending{}).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Update("status", "PAID").Error
}

func (r *gormRepository) CreatePeriod(ctx context.Context, period *model.PayrollPeriod) error {
	return r.db.WithContext(ctx).Create(period).Error
}

func (r *gormRepository) FindPeriodByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollPeriod, error) {
	var period model.PayrollPeriod
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&period).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func (r *gormRepository) FindPeriodByName(ctx context.Context, tenantID uint64, name string) (*model.PayrollPeriod, error) {
	var period model.PayrollPeriod
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND period_name = ?", tenantID, name).
		First(&period).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func (r *gormRepository) ListPeriods(ctx context.Context, tenantID uint64, query ListPeriodsQuery) ([]model.PayrollPeriod, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.PayrollPeriod{}).Where("tenant_id = ?", tenantID)
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := normalizePagination(query.Page, query.Limit)
	var items []model.PayrollPeriod
	err := db.Order("start_date DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&items).Error
	return items, total, err
}

func (r *gormRepository) UpdatePeriod(ctx context.Context, period *model.PayrollPeriod) error {
	return r.db.WithContext(ctx).Save(period).Error
}

func (r *gormRepository) CreateSnapshot(ctx context.Context, snapshot *model.PayrollSnapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}

func (r *gormRepository) ListSnapshots(ctx context.Context, tenantID uint64, query SnapshotListQuery) ([]model.PayrollSnapshot, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.PayrollSnapshot{}).Where("tenant_id = ?", tenantID)

	if query.PeriodUUID != "" {
		u, err := uuid.Parse(query.PeriodUUID)
		if err == nil {
			var p model.PayrollPeriod
			if err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&p).Error; err == nil {
				db = db.Where("period_id = ?", p.ID)
			} else {
				return []model.PayrollSnapshot{}, 0, nil
			}
		}
	}
	if query.EmployeeUUID != "" {
		u, err := uuid.Parse(query.EmployeeUUID)
		if err == nil {
			var emp model.Employee
			if err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&emp).Error; err == nil {
				db = db.Where("employee_id = ?", emp.ID)
			} else {
				return []model.PayrollSnapshot{}, 0, nil
			}
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := normalizePagination(query.Page, query.Limit)
	var items []model.PayrollSnapshot
	err := db.Preload("Items").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&items).Error
	return items, total, err
}

func (r *gormRepository) FindSnapshotByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollSnapshot, error) {
	var snap model.PayrollSnapshot
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Period").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&snap).Error
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func normalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}
