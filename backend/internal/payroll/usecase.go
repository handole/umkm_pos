package payroll

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	pkghash "github.com/handokodenih/umkm_pos/backend/pkg/hash"
)

// Usecase defines business logic operations for attendance, commissions, and payroll.
type Usecase interface {
	// Employees & Salary Structure
	CreateEmployee(ctx context.Context, tenantID uint64, req CreateEmployeeRequest) (*model.Employee, error)
	GetEmployee(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Employee, error)
	ListEmployees(ctx context.Context, tenantID uint64, query ListEmployeesQuery) (*PagedResponse[model.Employee], error)
	UpdateEmployee(ctx context.Context, tenantID uint64, u uuid.UUID, req UpdateEmployeeRequest) (*model.Employee, error)
	UpsertSalaryStructure(ctx context.Context, tenantID uint64, employeeUUID uuid.UUID, req SalaryStructureRequest) (*model.SalaryStructure, error)

	// Attendance
	ClockIn(ctx context.Context, tenantID uint64, req ClockInRequest) (*model.AttendanceLog, error)
	ClockOut(ctx context.Context, tenantID uint64, req ClockOutRequest) (*model.AttendanceLog, error)
	RecordAttendance(ctx context.Context, tenantID uint64, req RecordAttendanceRequest) (*model.AttendanceLog, error)
	ListAttendance(ctx context.Context, tenantID uint64, query ListAttendanceQuery) (*PagedResponse[model.AttendanceLog], error)

	// Commissions
	CalculateOrderCommission(ctx context.Context, tenantID uint64, req CalculateCommissionRequest) (*model.PayrollCommissionPending, error)
	ListCommissions(ctx context.Context, tenantID uint64, query ListCommissionsQuery) (*PagedResponse[model.PayrollCommissionPending], error)
	VoidCommission(ctx context.Context, tenantID uint64, commissionUUID uuid.UUID) (*model.PayrollCommissionPending, error)

	// Payroll Periods & Immutable Finalization
	CreatePeriod(ctx context.Context, tenantID uint64, req CreatePeriodRequest) (*model.PayrollPeriod, error)
	ListPeriods(ctx context.Context, tenantID uint64, query ListPeriodsQuery) (*PagedResponse[model.PayrollPeriod], error)
	GetPeriod(ctx context.Context, tenantID uint64, periodUUID uuid.UUID) (*PeriodDetailResponse, error)
	PreviewPeriodPayroll(ctx context.Context, tenantID uint64, periodUUID uuid.UUID, query PreviewPeriodQuery) (*PeriodPayrollPreviewResponse, error)
	FinalizePeriod(ctx context.Context, tenantID uint64, finalizedBy uint64, periodUUID uuid.UUID, req FinalizePeriodRequest) (*PeriodDetailResponse, error)
	ListSnapshots(ctx context.Context, tenantID uint64, query SnapshotListQuery) (*PagedResponse[model.PayrollSnapshot], error)
	GetSnapshot(ctx context.Context, tenantID uint64, snapshotUUID uuid.UUID) (*model.PayrollSnapshot, error)
}

type usecase struct {
	repo Repository
}

// NewUsecase creates a new payroll Usecase instance.
func NewUsecase(repo Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) CreateEmployee(ctx context.Context, tenantID uint64, req CreateEmployeeRequest) (*model.Employee, error) {
	existing, err := u.repo.FindEmployeeByCode(ctx, tenantID, req.EmployeeCode)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("employee code already exists")
	}

	joinedDate, err := parseDate(req.JoinedDate)
	if err != nil {
		return nil, errors.New("invalid joined_date format, expected YYYY-MM-DD")
	}

	status := req.EmploymentStatus
	if status == "" {
		status = "PERMANENT"
	}

	emp := &model.Employee{
		TenantID:          tenantID,
		UserID:            req.UserID,
		OutletID:          req.OutletID,
		EmployeeCode:      req.EmployeeCode,
		FullName:          req.FullName,
		JobTitle:          req.JobTitle,
		Phone:             req.Phone,
		BankName:          req.BankName,
		BankAccountNumber: req.BankAccountNumber,
		BankAccountHolder: req.BankAccountHolder,
		JoinedDate:        joinedDate,
		EmploymentStatus:  status,
		IsActive:          true,
	}

	err = u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		if err := txRepo.CreateEmployee(ctx, emp); err != nil {
			return err
		}
		if req.SalaryStructure != nil {
			ss := buildSalaryStructure(tenantID, emp.ID, *req.SalaryStructure)
			if err := txRepo.UpsertSalaryStructure(ctx, ss); err != nil {
				return err
			}
			emp.SalaryStructure = ss
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return emp, nil
}

func (u *usecase) GetEmployee(ctx context.Context, tenantID uint64, id uuid.UUID) (*model.Employee, error) {
	return u.repo.FindEmployeeByUUID(ctx, tenantID, id)
}

func (u *usecase) ListEmployees(ctx context.Context, tenantID uint64, query ListEmployeesQuery) (*PagedResponse[model.Employee], error) {
	items, total, err := u.repo.ListEmployees(ctx, tenantID, query)
	if err != nil {
		return nil, err
	}
	return toPagedResponse(items, total, query.Page, query.Limit), nil
}

func (u *usecase) UpdateEmployee(ctx context.Context, tenantID uint64, id uuid.UUID, req UpdateEmployeeRequest) (*model.Employee, error) {
	emp, err := u.repo.FindEmployeeByUUID(ctx, tenantID, id)
	if err != nil {
		return nil, errors.New("employee not found")
	}

	if req.UserID != nil {
		emp.UserID = req.UserID
	}
	if req.OutletID != nil {
		emp.OutletID = *req.OutletID
	}
	if req.FullName != nil {
		emp.FullName = *req.FullName
	}
	if req.JobTitle != nil {
		emp.JobTitle = *req.JobTitle
	}
	if req.Phone != nil {
		emp.Phone = *req.Phone
	}
	if req.BankName != nil {
		emp.BankName = *req.BankName
	}
	if req.BankAccountNumber != nil {
		emp.BankAccountNumber = *req.BankAccountNumber
	}
	if req.BankAccountHolder != nil {
		emp.BankAccountHolder = *req.BankAccountHolder
	}
	if req.EmploymentStatus != nil {
		emp.EmploymentStatus = *req.EmploymentStatus
	}
	if req.IsActive != nil {
		emp.IsActive = *req.IsActive
	}
	emp.UpdatedAt = time.Now().UTC()

	if err := u.repo.UpdateEmployee(ctx, emp); err != nil {
		return nil, err
	}
	return emp, nil
}

func (u *usecase) UpsertSalaryStructure(ctx context.Context, tenantID uint64, employeeUUID uuid.UUID, req SalaryStructureRequest) (*model.SalaryStructure, error) {
	emp, err := u.repo.FindEmployeeByUUID(ctx, tenantID, employeeUUID)
	if err != nil {
		return nil, errors.New("employee not found")
	}

	ss := buildSalaryStructure(tenantID, emp.ID, req)
	if err := u.repo.UpsertSalaryStructure(ctx, ss); err != nil {
		return nil, err
	}
	return ss, nil
}

func (u *usecase) ClockIn(ctx context.Context, tenantID uint64, req ClockInRequest) (*model.AttendanceLog, error) {
	emp, err := u.repo.FindEmployeeByUUID(ctx, tenantID, req.EmployeeUUID)
	if err != nil {
		return nil, errors.New("employee not found")
	}
	if !emp.IsActive {
		return nil, errors.New("employee is inactive")
	}

	if err := u.verifyEmployeePIN(ctx, tenantID, emp, req.Pin); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	clockInAt := now
	if req.ClockInTime != "" {
		parsed, err := time.Parse(time.RFC3339, req.ClockInTime)
		if err != nil {
			return nil, errors.New("invalid clock_in_at format, expected RFC3339")
		}
		clockInAt = parsed.UTC()
	}

	logDate := time.Date(clockInAt.Year(), clockInAt.Month(), clockInAt.Day(), 0, 0, 0, 0, time.UTC)
	if req.LogDate != "" {
		parsed, err := parseDate(req.LogDate)
		if err != nil {
			return nil, errors.New("invalid log_date format, expected YYYY-MM-DD")
		}
		logDate = parsed
	}

	existing, err := u.repo.FindAttendanceByEmployeeDate(ctx, tenantID, emp.ID, logDate)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ClockIn != nil {
		return nil, errors.New("employee already clocked in for this date")
	}

	status := "PRESENT"
	if req.LateMinutes > 0 {
		status = "LATE"
	}

	if existing != nil {
		existing.OutletID = req.OutletID
		existing.ClockIn = &clockInAt
		existing.ClockInPhotoURL = req.PhotoURL
		existing.LateMinutes = req.LateMinutes
		existing.Status = status
		existing.UpdatedAt = now
		if err := u.repo.UpdateAttendance(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	entry := &model.AttendanceLog{
		TenantID:        tenantID,
		EmployeeID:      emp.ID,
		OutletID:        req.OutletID,
		LogDate:         logDate,
		ClockIn:         &clockInAt,
		ClockInPhotoURL: req.PhotoURL,
		LateMinutes:     req.LateMinutes,
		Status:          status,
	}
	if err := u.repo.CreateAttendance(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (u *usecase) ClockOut(ctx context.Context, tenantID uint64, req ClockOutRequest) (*model.AttendanceLog, error) {
	emp, err := u.repo.FindEmployeeByUUID(ctx, tenantID, req.EmployeeUUID)
	if err != nil {
		return nil, errors.New("employee not found")
	}

	if err := u.verifyEmployeePIN(ctx, tenantID, emp, req.Pin); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	clockOutAt := now
	if req.ClockOutTime != "" {
		parsed, err := time.Parse(time.RFC3339, req.ClockOutTime)
		if err != nil {
			return nil, errors.New("invalid clock_out_at format, expected RFC3339")
		}
		clockOutAt = parsed.UTC()
	}

	logDate := time.Date(clockOutAt.Year(), clockOutAt.Month(), clockOutAt.Day(), 0, 0, 0, 0, time.UTC)
	if req.LogDate != "" {
		parsed, err := parseDate(req.LogDate)
		if err != nil {
			return nil, errors.New("invalid log_date format, expected YYYY-MM-DD")
		}
		logDate = parsed
	}

	existing, err := u.repo.FindAttendanceByEmployeeDate(ctx, tenantID, emp.ID, logDate)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.ClockIn == nil {
		return nil, errors.New("cannot clock out before clocking in")
	}
	if existing.ClockOut != nil {
		return nil, errors.New("employee already clocked out for this date")
	}

	existing.ClockOut = &clockOutAt
	existing.ClockOutPhotoURL = req.PhotoURL
	existing.OvertimeMinutes = req.OvertimeMinutes
	existing.UpdatedAt = now

	if err := u.repo.UpdateAttendance(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (u *usecase) RecordAttendance(ctx context.Context, tenantID uint64, req RecordAttendanceRequest) (*model.AttendanceLog, error) {
	emp, err := u.repo.FindEmployeeByUUID(ctx, tenantID, req.EmployeeUUID)
	if err != nil {
		return nil, errors.New("employee not found")
	}

	logDate, err := parseDate(req.LogDate)
	if err != nil {
		return nil, errors.New("invalid log_date format, expected YYYY-MM-DD")
	}

	existing, err := u.repo.FindAttendanceByEmployeeDate(ctx, tenantID, emp.ID, logDate)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		existing.OutletID = req.OutletID
		existing.Status = req.Status
		existing.LateMinutes = req.LateMinutes
		existing.OvertimeMinutes = req.OvertimeMinutes
		existing.UpdatedAt = time.Now().UTC()
		if err := u.repo.UpdateAttendance(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	entry := &model.AttendanceLog{
		TenantID:        tenantID,
		EmployeeID:      emp.ID,
		OutletID:        req.OutletID,
		LogDate:         logDate,
		LateMinutes:     req.LateMinutes,
		OvertimeMinutes: req.OvertimeMinutes,
		Status:          req.Status,
	}
	if err := u.repo.CreateAttendance(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (u *usecase) ListAttendance(ctx context.Context, tenantID uint64, query ListAttendanceQuery) (*PagedResponse[model.AttendanceLog], error) {
	items, total, err := u.repo.ListAttendance(ctx, tenantID, query)
	if err != nil {
		return nil, err
	}
	return toPagedResponse(items, total, query.Page, query.Limit), nil
}

func (u *usecase) CalculateOrderCommission(ctx context.Context, tenantID uint64, req CalculateCommissionRequest) (*model.PayrollCommissionPending, error) {
	emp, err := u.repo.FindEmployeeByUUID(ctx, tenantID, req.EmployeeUUID)
	if err != nil {
		return nil, errors.New("employee not found")
	}
	if emp.SalaryStructure == nil {
		return nil, errors.New("employee does not have a salary structure configured")
	}

	// Idempotent per (tenant, order, employee)
	existing, err := u.repo.FindCommissionByOrder(ctx, tenantID, req.OrderID, emp.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	rateAmount := int64(math.Round(float64(req.OrderTotal) * (emp.SalaryStructure.CommissionRatePercent / 100.0)))
	perItemAmount := int64(req.TotalItems) * emp.SalaryStructure.CommissionFixedPerItem
	totalCommission := rateAmount + perItemAmount

	comm := &model.PayrollCommissionPending{
		TenantID:         tenantID,
		EmployeeID:       emp.ID,
		OrderID:          req.OrderID,
		CommissionAmount: totalCommission,
		Status:           "PENDING",
	}
	if err := u.repo.CreateCommission(ctx, comm); err != nil {
		return nil, err
	}
	return comm, nil
}

func (u *usecase) ListCommissions(ctx context.Context, tenantID uint64, query ListCommissionsQuery) (*PagedResponse[model.PayrollCommissionPending], error) {
	items, total, err := u.repo.ListCommissions(ctx, tenantID, query)
	if err != nil {
		return nil, err
	}
	return toPagedResponse(items, total, query.Page, query.Limit), nil
}

func (u *usecase) VoidCommission(ctx context.Context, tenantID uint64, commissionUUID uuid.UUID) (*model.PayrollCommissionPending, error) {
	comm, err := u.repo.FindCommissionByUUID(ctx, tenantID, commissionUUID)
	if err != nil {
		return nil, errors.New("commission record not found")
	}
	if comm.Status == "PAID" {
		return nil, errors.New("cannot void a commission that has already been paid")
	}
	comm.Status = "VOID"
	if err := u.repo.UpdateCommission(ctx, comm); err != nil {
		return nil, err
	}
	return comm, nil
}

func (u *usecase) CreatePeriod(ctx context.Context, tenantID uint64, req CreatePeriodRequest) (*model.PayrollPeriod, error) {
	existing, err := u.repo.FindPeriodByName(ctx, tenantID, req.PeriodName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("payroll period with this name already exists")
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, errors.New("invalid start_date format, expected YYYY-MM-DD")
	}
	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return nil, errors.New("invalid end_date format, expected YYYY-MM-DD")
	}
	paymentDate, err := parseDate(req.PaymentDate)
	if err != nil {
		return nil, errors.New("invalid payment_date format, expected YYYY-MM-DD")
	}
	if endDate.Before(startDate) {
		return nil, errors.New("end_date cannot be before start_date")
	}

	period := &model.PayrollPeriod{
		TenantID:    tenantID,
		PeriodName:  req.PeriodName,
		StartDate:   startDate,
		EndDate:     endDate,
		PaymentDate: paymentDate,
		Status:      "DRAFT",
		TotalPayout: 0,
	}
	if err := u.repo.CreatePeriod(ctx, period); err != nil {
		return nil, err
	}
	return period, nil
}

func (u *usecase) ListPeriods(ctx context.Context, tenantID uint64, query ListPeriodsQuery) (*PagedResponse[model.PayrollPeriod], error) {
	items, total, err := u.repo.ListPeriods(ctx, tenantID, query)
	if err != nil {
		return nil, err
	}
	return toPagedResponse(items, total, query.Page, query.Limit), nil
}

func (u *usecase) GetPeriod(ctx context.Context, tenantID uint64, periodUUID uuid.UUID) (*PeriodDetailResponse, error) {
	period, err := u.repo.FindPeriodByUUID(ctx, tenantID, periodUUID)
	if err != nil {
		return nil, errors.New("payroll period not found")
	}

	res := &PeriodDetailResponse{Period: *period}
	if period.Status == "FINALIZED" {
		snaps, _, err := u.repo.ListSnapshots(ctx, tenantID, SnapshotListQuery{
			PeriodUUID: period.UUID.String(),
			Page:       1,
			Limit:      100,
		})
		if err != nil {
			return nil, err
		}
		res.Snapshots = snaps
	}
	return res, nil
}

func (u *usecase) PreviewPeriodPayroll(ctx context.Context, tenantID uint64, periodUUID uuid.UUID, query PreviewPeriodQuery) (*PeriodPayrollPreviewResponse, error) {
	period, err := u.repo.FindPeriodByUUID(ctx, tenantID, periodUUID)
	if err != nil {
		return nil, errors.New("payroll period not found")
	}

	employees, err := u.repo.ListActiveEmployeesWithSalary(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	previews := make([]EmployeePayrollPreview, 0, len(employees))
	var totalPayout int64

	for _, emp := range employees {
		calc, _, err := u.calculateEmployeePayroll(ctx, u.repo, tenantID, emp, period, query.OvertimeRatePerHour, query.LatePenaltyPerMin, nil)
		if err != nil {
			return nil, err
		}
		totalPayout += calc.NetSalary
		previews = append(previews, calc)
	}

	return &PeriodPayrollPreviewResponse{
		Period:      *period,
		TotalPayout: totalPayout,
		Employees:   previews,
	}, nil
}

func (u *usecase) FinalizePeriod(ctx context.Context, tenantID uint64, finalizedBy uint64, periodUUID uuid.UUID, req FinalizePeriodRequest) (*PeriodDetailResponse, error) {
	// Index manual adjustments by employee UUID
	adjByEmp := make(map[uuid.UUID][]PayrollAdjustmentInput)
	for _, adj := range req.Adjustments {
		adjByEmp[adj.EmployeeUUID] = append(adjByEmp[adj.EmployeeUUID], adj)
	}

	var result *PeriodDetailResponse

	err := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		period, err := txRepo.FindPeriodByUUID(ctx, tenantID, periodUUID)
		if err != nil {
			return errors.New("payroll period not found")
		}
		if period.Status == "FINALIZED" {
			return errors.New("payroll period is already finalized and immutable")
		}

		employees, err := txRepo.ListActiveEmployeesWithSalary(ctx, tenantID)
		if err != nil {
			return err
		}

		var totalPayout int64
		snapshots := make([]model.PayrollSnapshot, 0, len(employees))

		for _, emp := range employees {
			calc, commissionIDs, err := u.calculateEmployeePayroll(
				ctx, txRepo, tenantID, emp, period,
				req.OvertimeRatePerHour, req.LatePenaltyPerMin, adjByEmp[emp.UUID],
			)
			if err != nil {
				return err
			}

			snap := model.PayrollSnapshot{
				TenantID:          tenantID,
				PeriodID:          period.ID,
				EmployeeID:        emp.ID,
				EmployeeCode:      emp.EmployeeCode,
				EmployeeName:      emp.FullName,
				JobTitle:          emp.JobTitle,
				BankName:          emp.BankName,
				BankAccountNumber: emp.BankAccountNumber,
				BasicSalary:       calc.BasicSalary,
				TotalAllowances:   calc.TotalAllowances,
				TotalOvertimePay:  calc.TotalOvertimePay,
				TotalCommissions:  calc.TotalCommissions,
				TotalDeductions:   calc.TotalDeductions,
				NetSalary:         calc.NetSalary,
				Status:            "FINALIZED",
				Items:             calc.Items,
			}

			if err := txRepo.CreateSnapshot(ctx, &snap); err != nil {
				return err
			}
			if err := txRepo.MarkCommissionsPaid(ctx, tenantID, commissionIDs); err != nil {
				return err
			}

			totalPayout += snap.NetSalary
			snapshots = append(snapshots, snap)
		}

		now := time.Now().UTC()
		period.Status = "FINALIZED"
		period.TotalPayout = totalPayout
		period.FinalizedBy = &finalizedBy
		period.FinalizedAt = &now
		period.UpdatedAt = now

		if err := txRepo.UpdatePeriod(ctx, period); err != nil {
			return err
		}

		result = &PeriodDetailResponse{
			Period:    *period,
			Snapshots: snapshots,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (u *usecase) ListSnapshots(ctx context.Context, tenantID uint64, query SnapshotListQuery) (*PagedResponse[model.PayrollSnapshot], error) {
	items, total, err := u.repo.ListSnapshots(ctx, tenantID, query)
	if err != nil {
		return nil, err
	}
	return toPagedResponse(items, total, query.Page, query.Limit), nil
}

func (u *usecase) GetSnapshot(ctx context.Context, tenantID uint64, snapshotUUID uuid.UUID) (*model.PayrollSnapshot, error) {
	return u.repo.FindSnapshotByUUID(ctx, tenantID, snapshotUUID)
}

func (u *usecase) verifyEmployeePIN(ctx context.Context, tenantID uint64, emp *model.Employee, pin string) error {
	if emp.UserID == nil {
		return nil
	}
	user, err := u.repo.FindUserByID(ctx, tenantID, *emp.UserID)
	if err != nil || user.PinHash == "" {
		return nil
	}
	if pin == "" {
		return errors.New("PIN is required for kiosk attendance")
	}
	if !pkghash.CheckPINHash(pin, user.Username, user.PinHash) {
		return errors.New("invalid PIN")
	}
	return nil
}

func (u *usecase) calculateEmployeePayroll(
	ctx context.Context,
	repo Repository,
	tenantID uint64,
	emp model.Employee,
	period *model.PayrollPeriod,
	overtimeRatePerHour int64,
	latePenaltyPerMin int64,
	adjustments []PayrollAdjustmentInput,
) (EmployeePayrollPreview, []uint64, error) {
	logs, err := repo.ListAttendanceByEmployeePeriod(ctx, tenantID, emp.ID, period.StartDate, period.EndDate)
	if err != nil {
		return EmployeePayrollPreview{}, nil, err
	}
	att := summarizeAttendance(logs)

	pendingComms, err := repo.ListPendingCommissionsByEmployee(ctx, tenantID, emp.ID)
	if err != nil {
		return EmployeePayrollPreview{}, nil, err
	}

	var totalCommissions int64
	commIDs := make([]uint64, 0, len(pendingComms))
	for _, c := range pendingComms {
		totalCommissions += c.CommissionAmount
		commIDs = append(commIDs, c.ID)
	}

	var basicSalary, fixedAllowance, mealAllowance, bpjsHealth, bpjsEmp int64
	if emp.SalaryStructure != nil {
		basicSalary = emp.SalaryStructure.BasicSalary
		fixedAllowance = emp.SalaryStructure.FixedAllowance
		mealAllowance = emp.SalaryStructure.MealTransportAllowance
		bpjsHealth = emp.SalaryStructure.BPJSHealthDeduction
		bpjsEmp = emp.SalaryStructure.BPJSEmploymentDeduction
	}

	totalAllowances := fixedAllowance + mealAllowance
	overtimePay := (int64(att.TotalOvertimeMins) * overtimeRatePerHour) / 60
	latePenalty := int64(att.TotalLateMinutes) * latePenaltyPerMin

	items := make([]model.PayrollItemSnapshot, 0)
	if basicSalary > 0 {
		items = append(items, model.PayrollItemSnapshot{ItemType: "EARNING", ComponentName: "Gaji Pokok", Amount: basicSalary})
	}
	if fixedAllowance > 0 {
		items = append(items, model.PayrollItemSnapshot{ItemType: "EARNING", ComponentName: "Tunjangan Tetap", Amount: fixedAllowance})
	}
	if mealAllowance > 0 {
		items = append(items, model.PayrollItemSnapshot{ItemType: "EARNING", ComponentName: "Uang Makan & Transport", Amount: mealAllowance})
	}
	if overtimePay > 0 {
		items = append(items, model.PayrollItemSnapshot{
			ItemType:      "EARNING",
			ComponentName: "Upah Lembur",
			Amount:        overtimePay,
			Notes:         fmt.Sprintf("%d menit lembur", att.TotalOvertimeMins),
		})
	}
	if totalCommissions > 0 {
		items = append(items, model.PayrollItemSnapshot{
			ItemType:      "EARNING",
			ComponentName: "Komisi Penjualan POS",
			Amount:        totalCommissions,
			Notes:         fmt.Sprintf("%d transaksi", len(pendingComms)),
		})
	}
	if bpjsHealth > 0 {
		items = append(items, model.PayrollItemSnapshot{ItemType: "DEDUCTION", ComponentName: "Potongan BPJS Kesehatan", Amount: bpjsHealth})
	}
	if bpjsEmp > 0 {
		items = append(items, model.PayrollItemSnapshot{ItemType: "DEDUCTION", ComponentName: "Potongan BPJS Ketenagakerjaan", Amount: bpjsEmp})
	}
	if latePenalty > 0 {
		items = append(items, model.PayrollItemSnapshot{
			ItemType:      "DEDUCTION",
			ComponentName: "Potongan Keterlambatan",
			Amount:        latePenalty,
			Notes:         fmt.Sprintf("%d menit terlambat", att.TotalLateMinutes),
		})
	}

	var extraEarnings, extraDeductions int64
	for _, adj := range adjustments {
		items = append(items, model.PayrollItemSnapshot{
			ItemType:      adj.ItemType,
			ComponentName: adj.ComponentName,
			Amount:        adj.Amount,
			Notes:         adj.Notes,
		})
		if adj.ItemType == "EARNING" {
			extraEarnings += adj.Amount
		} else {
			extraDeductions += adj.Amount
		}
	}

	totalDeductions := bpjsHealth + bpjsEmp + latePenalty + extraDeductions
	grossSalary := basicSalary + totalAllowances + overtimePay + totalCommissions + extraEarnings
	netSalary := grossSalary - totalDeductions
	if netSalary < 0 {
		netSalary = 0
	}

	return EmployeePayrollPreview{
		EmployeeUUID:      emp.UUID,
		EmployeeCode:      emp.EmployeeCode,
		EmployeeName:      emp.FullName,
		JobTitle:          emp.JobTitle,
		PresentDays:       att.PresentDays,
		LateDays:          att.LateDays,
		AbsentDays:        att.AbsentDays,
		TotalLateMinutes:  att.TotalLateMinutes,
		TotalOvertimeMins: att.TotalOvertimeMins,
		BasicSalary:       basicSalary,
		TotalAllowances:   totalAllowances + extraEarnings,
		TotalOvertimePay:  overtimePay,
		TotalCommissions:  totalCommissions,
		TotalDeductions:   totalDeductions,
		NetSalary:         netSalary,
		Items:             items,
	}, commIDs, nil
}

func summarizeAttendance(logs []model.AttendanceLog) AttendanceSummary {
	var s AttendanceSummary
	for _, l := range logs {
		switch l.Status {
		case "PRESENT":
			s.PresentDays++
		case "LATE":
			s.LateDays++
		case "ABSENT":
			s.AbsentDays++
		}
		s.TotalLateMinutes += l.LateMinutes
		s.TotalOvertimeMins += l.OvertimeMinutes
	}
	return s
}

func buildSalaryStructure(tenantID, employeeID uint64, req SalaryStructureRequest) *model.SalaryStructure {
	return &model.SalaryStructure{
		TenantID:                tenantID,
		EmployeeID:              employeeID,
		BasicSalary:             req.BasicSalary,
		FixedAllowance:          req.FixedAllowance,
		MealTransportAllowance:  req.MealTransportAllowance,
		CommissionRatePercent:   req.CommissionRatePercent,
		CommissionFixedPerItem:  req.CommissionFixedPerItem,
		BPJSHealthDeduction:     req.BPJSHealthDeduction,
		BPJSEmploymentDeduction: req.BPJSEmploymentDeduction,
		UpdatedAt:               time.Now().UTC(),
	}
}

func toPagedResponse[T any](items []T, total int64, page, limit int) *PagedResponse[T] {
	p, l := normalizePagination(page, limit)
	totalPages := (total + int64(l) - 1) / int64(l)
	if totalPages == 0 {
		totalPages = 1
	}
	return &PagedResponse[T]{
		Items:      items,
		Total:      total,
		Page:       p,
		Limit:      l,
		TotalPages: totalPages,
	}
}
