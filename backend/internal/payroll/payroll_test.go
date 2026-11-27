package payroll_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"github.com/handokodenih/umkm_pos/backend/internal/payroll"
	pkghash "github.com/handokodenih/umkm_pos/backend/pkg/hash"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
	"gorm.io/gorm"
)

type mockPayrollRepo struct {
	employees   []model.Employee
	salaries    []model.SalaryStructure
	users       []model.User
	attendances []model.AttendanceLog
	commissions []model.PayrollCommissionPending
	periods     []model.PayrollPeriod
	snapshots   []model.PayrollSnapshot

	empSeq  uint64
	salSeq  uint64
	attSeq  uint64
	commSeq uint64
	perSeq  uint64
	snapSeq uint64
	itemSeq uint64
}

func newMockPayrollRepo() *mockPayrollRepo {
	return &mockPayrollRepo{}
}

func (m *mockPayrollRepo) WithTransaction(ctx context.Context, fn func(txRepo payroll.Repository) error) error {
	return fn(m)
}

func (m *mockPayrollRepo) CreateEmployee(ctx context.Context, emp *model.Employee) error {
	m.empSeq++
	emp.ID = m.empSeq
	emp.UUID = uuid.New()
	emp.CreatedAt = time.Now().UTC()
	emp.UpdatedAt = time.Now().UTC()
	m.employees = append(m.employees, *emp)
	return nil
}

func (m *mockPayrollRepo) FindEmployeeByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Employee, error) {
	for i := range m.employees {
		if m.employees[i].TenantID == tenantID && m.employees[i].UUID == u {
			emp := m.employees[i]
			for _, s := range m.salaries {
				if s.EmployeeID == emp.ID {
					sc := s
					emp.SalaryStructure = &sc
					break
				}
			}
			return &emp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) FindEmployeeByCode(ctx context.Context, tenantID uint64, code string) (*model.Employee, error) {
	for _, e := range m.employees {
		if e.TenantID == tenantID && e.EmployeeCode == code {
			ec := e
			return &ec, nil
		}
	}
	return nil, nil
}

func (m *mockPayrollRepo) ListEmployees(ctx context.Context, tenantID uint64, query payroll.ListEmployeesQuery) ([]model.Employee, int64, error) {
	var res []model.Employee
	for _, e := range m.employees {
		if e.TenantID != tenantID {
			continue
		}
		if query.OutletID > 0 && e.OutletID != query.OutletID {
			continue
		}
		if query.IsActive != nil && e.IsActive != *query.IsActive {
			continue
		}
		res = append(res, e)
	}
	return res, int64(len(res)), nil
}

func (m *mockPayrollRepo) ListActiveEmployeesWithSalary(ctx context.Context, tenantID uint64) ([]model.Employee, error) {
	var res []model.Employee
	for _, e := range m.employees {
		if e.TenantID == tenantID && e.IsActive {
			ec := e
			for _, s := range m.salaries {
				if s.EmployeeID == ec.ID {
					sc := s
					ec.SalaryStructure = &sc
					break
				}
			}
			res = append(res, ec)
		}
	}
	return res, nil
}

func (m *mockPayrollRepo) UpdateEmployee(ctx context.Context, emp *model.Employee) error {
	for i := range m.employees {
		if m.employees[i].ID == emp.ID {
			m.employees[i] = *emp
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) UpsertSalaryStructure(ctx context.Context, ss *model.SalaryStructure) error {
	for i := range m.salaries {
		if m.salaries[i].TenantID == ss.TenantID && m.salaries[i].EmployeeID == ss.EmployeeID {
			ss.ID = m.salaries[i].ID
			ss.UUID = m.salaries[i].UUID
			m.salaries[i] = *ss
			return nil
		}
	}
	m.salSeq++
	ss.ID = m.salSeq
	ss.UUID = uuid.New()
	m.salaries = append(m.salaries, *ss)
	return nil
}

func (m *mockPayrollRepo) FindUserByID(ctx context.Context, tenantID uint64, userID uint64) (*model.User, error) {
	for _, u := range m.users {
		if u.TenantID == tenantID && u.ID == userID {
			uc := u
			return &uc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) FindAttendanceByEmployeeDate(ctx context.Context, tenantID uint64, employeeID uint64, logDate time.Time) (*model.AttendanceLog, error) {
	target := logDate.Format("2006-01-02")
	for i := range m.attendances {
		if m.attendances[i].TenantID == tenantID &&
			m.attendances[i].EmployeeID == employeeID &&
			m.attendances[i].LogDate.Format("2006-01-02") == target {
			ac := m.attendances[i]
			return &ac, nil
		}
	}
	return nil, nil
}

func (m *mockPayrollRepo) CreateAttendance(ctx context.Context, log *model.AttendanceLog) error {
	m.attSeq++
	log.ID = m.attSeq
	log.UUID = uuid.New()
	log.CreatedAt = time.Now().UTC()
	log.UpdatedAt = time.Now().UTC()
	m.attendances = append(m.attendances, *log)
	return nil
}

func (m *mockPayrollRepo) UpdateAttendance(ctx context.Context, log *model.AttendanceLog) error {
	for i := range m.attendances {
		if m.attendances[i].ID == log.ID {
			m.attendances[i] = *log
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) ListAttendance(ctx context.Context, tenantID uint64, query payroll.ListAttendanceQuery) ([]model.AttendanceLog, int64, error) {
	var res []model.AttendanceLog
	for _, a := range m.attendances {
		if a.TenantID == tenantID {
			res = append(res, a)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPayrollRepo) ListAttendanceByEmployeePeriod(ctx context.Context, tenantID uint64, employeeID uint64, startDate, endDate time.Time) ([]model.AttendanceLog, error) {
	var res []model.AttendanceLog
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")
	for _, a := range m.attendances {
		d := a.LogDate.Format("2006-01-02")
		if a.TenantID == tenantID && a.EmployeeID == employeeID && d >= startStr && d <= endStr {
			res = append(res, a)
		}
	}
	return res, nil
}

func (m *mockPayrollRepo) FindCommissionByOrder(ctx context.Context, tenantID uint64, orderID uint64, employeeID uint64) (*model.PayrollCommissionPending, error) {
	for _, c := range m.commissions {
		if c.TenantID == tenantID && c.OrderID == orderID && c.EmployeeID == employeeID {
			cc := c
			return &cc, nil
		}
	}
	return nil, nil
}

func (m *mockPayrollRepo) FindCommissionByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollCommissionPending, error) {
	for _, c := range m.commissions {
		if c.TenantID == tenantID && c.UUID == u {
			cc := c
			return &cc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) CreateCommission(ctx context.Context, comm *model.PayrollCommissionPending) error {
	m.commSeq++
	comm.ID = m.commSeq
	comm.UUID = uuid.New()
	comm.CreatedAt = time.Now().UTC()
	m.commissions = append(m.commissions, *comm)
	return nil
}

func (m *mockPayrollRepo) UpdateCommission(ctx context.Context, comm *model.PayrollCommissionPending) error {
	for i := range m.commissions {
		if m.commissions[i].ID == comm.ID {
			m.commissions[i] = *comm
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) ListCommissions(ctx context.Context, tenantID uint64, query payroll.ListCommissionsQuery) ([]model.PayrollCommissionPending, int64, error) {
	var res []model.PayrollCommissionPending
	for _, c := range m.commissions {
		if c.TenantID == tenantID {
			res = append(res, c)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPayrollRepo) ListPendingCommissionsByEmployee(ctx context.Context, tenantID uint64, employeeID uint64) ([]model.PayrollCommissionPending, error) {
	var res []model.PayrollCommissionPending
	for _, c := range m.commissions {
		if c.TenantID == tenantID && c.EmployeeID == employeeID && c.Status == "PENDING" {
			res = append(res, c)
		}
	}
	return res, nil
}

func (m *mockPayrollRepo) MarkCommissionsPaid(ctx context.Context, tenantID uint64, ids []uint64) error {
	idSet := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	for i := range m.commissions {
		if m.commissions[i].TenantID == tenantID && idSet[m.commissions[i].ID] {
			m.commissions[i].Status = "PAID"
		}
	}
	return nil
}

func (m *mockPayrollRepo) CreatePeriod(ctx context.Context, period *model.PayrollPeriod) error {
	m.perSeq++
	period.ID = m.perSeq
	period.UUID = uuid.New()
	period.CreatedAt = time.Now().UTC()
	period.UpdatedAt = time.Now().UTC()
	m.periods = append(m.periods, *period)
	return nil
}

func (m *mockPayrollRepo) FindPeriodByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollPeriod, error) {
	for _, p := range m.periods {
		if p.TenantID == tenantID && p.UUID == u {
			pc := p
			return &pc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) FindPeriodByName(ctx context.Context, tenantID uint64, name string) (*model.PayrollPeriod, error) {
	for _, p := range m.periods {
		if p.TenantID == tenantID && p.PeriodName == name {
			pc := p
			return &pc, nil
		}
	}
	return nil, nil
}

func (m *mockPayrollRepo) ListPeriods(ctx context.Context, tenantID uint64, query payroll.ListPeriodsQuery) ([]model.PayrollPeriod, int64, error) {
	var res []model.PayrollPeriod
	for _, p := range m.periods {
		if p.TenantID == tenantID {
			res = append(res, p)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPayrollRepo) UpdatePeriod(ctx context.Context, period *model.PayrollPeriod) error {
	for i := range m.periods {
		if m.periods[i].ID == period.ID {
			m.periods[i] = *period
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPayrollRepo) CreateSnapshot(ctx context.Context, snapshot *model.PayrollSnapshot) error {
	m.snapSeq++
	snapshot.ID = m.snapSeq
	snapshot.UUID = uuid.New()
	snapshot.CreatedAt = time.Now().UTC()
	for i := range snapshot.Items {
		m.itemSeq++
		snapshot.Items[i].ID = m.itemSeq
		snapshot.Items[i].UUID = uuid.New()
		snapshot.Items[i].SnapshotID = snapshot.ID
	}
	m.snapshots = append(m.snapshots, *snapshot)
	return nil
}

func (m *mockPayrollRepo) ListSnapshots(ctx context.Context, tenantID uint64, query payroll.SnapshotListQuery) ([]model.PayrollSnapshot, int64, error) {
	var res []model.PayrollSnapshot
	for _, s := range m.snapshots {
		if s.TenantID == tenantID {
			res = append(res, s)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPayrollRepo) FindSnapshotByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PayrollSnapshot, error) {
	for _, s := range m.snapshots {
		if s.TenantID == tenantID && s.UUID == u {
			sc := s
			return &sc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func TestAttendanceAndPayrollLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newMockPayrollRepo()
	uc := payroll.NewUsecase(repo)
	tenantID := uint64(1)
	userID := uint64(10)

	repo.users = append(repo.users, model.User{
		ID:       userID,
		TenantID: tenantID,
		Username: "budi.kasir",
		PinHash:  pkghash.HashPIN("123456", "budi.kasir"),
		IsActive: true,
	})

	// 1. Register Employee + Salary Structure
	emp, err := uc.CreateEmployee(ctx, tenantID, payroll.CreateEmployeeRequest{
		UserID:       &userID,
		OutletID:     1,
		EmployeeCode: "EMP-001",
		FullName:     "Budi Santoso",
		JobTitle:     "Kasir Senior",
		JoinedDate:   "2026-01-10",
		SalaryStructure: &payroll.SalaryStructureRequest{
			BasicSalary:             3000000,
			FixedAllowance:          500000,
			MealTransportAllowance:  300000,
			CommissionRatePercent:   2.5,
			CommissionFixedPerItem:  1000,
			BPJSHealthDeduction:     50000,
			BPJSEmploymentDeduction: 100000,
		},
	})
	if err != nil {
		t.Fatalf("CreateEmployee failed: %v", err)
	}

	// Duplicate employee code should fail
	if _, err := uc.CreateEmployee(ctx, tenantID, payroll.CreateEmployeeRequest{
		OutletID:     1,
		EmployeeCode: "EMP-001",
		FullName:     "Duplikat",
		JobTitle:     "Kasir",
		JoinedDate:   "2026-02-01",
	}); err == nil {
		t.Fatal("expected duplicate employee code error")
	}

	// 2. Kiosk Attendance Clock-in with wrong PIN should fail
	if _, err := uc.ClockIn(ctx, tenantID, payroll.ClockInRequest{
		EmployeeUUID: emp.UUID,
		OutletID:     1,
		Pin:          "999999",
		LogDate:      "2026-11-10",
	}); err == nil {
		t.Fatal("expected invalid PIN error")
	}

	// Valid Clock-in (late 15 mins)
	att, err := uc.ClockIn(ctx, tenantID, payroll.ClockInRequest{
		EmployeeUUID: emp.UUID,
		OutletID:     1,
		Pin:          "123456",
		PhotoURL:     "https://cdn.example.com/selfie-in.jpg",
		LogDate:      "2026-11-10",
		LateMinutes:  15,
	})
	if err != nil {
		t.Fatalf("ClockIn failed: %v", err)
	}
	if att.Status != "LATE" {
		t.Fatalf("expected status LATE, got %s", att.Status)
	}

	// Valid Clock-out (overtime 120 mins = 2 hours)
	attOut, err := uc.ClockOut(ctx, tenantID, payroll.ClockOutRequest{
		EmployeeUUID:    emp.UUID,
		Pin:             "123456",
		PhotoURL:        "https://cdn.example.com/selfie-out.jpg",
		LogDate:         "2026-11-10",
		OvertimeMinutes: 120,
	})
	if err != nil {
		t.Fatalf("ClockOut failed: %v", err)
	}
	if attOut.OvertimeMinutes != 120 {
		t.Fatalf("expected 120 overtime minutes, got %d", attOut.OvertimeMinutes)
	}

	// Double Clock-out should fail
	if _, err := uc.ClockOut(ctx, tenantID, payroll.ClockOutRequest{
		EmployeeUUID: emp.UUID,
		Pin:          "123456",
		LogDate:      "2026-11-10",
	}); err == nil {
		t.Fatal("expected double clock-out error")
	}

	// 3. Calculate POS Commission (2.5% of 1,000,000 = 25,000 + 5 items * 1,000 = 30,000)
	comm, err := uc.CalculateOrderCommission(ctx, tenantID, payroll.CalculateCommissionRequest{
		EmployeeUUID: emp.UUID,
		OrderID:      501,
		OrderTotal:   1000000,
		TotalItems:   5,
	})
	if err != nil {
		t.Fatalf("CalculateOrderCommission failed: %v", err)
	}
	if comm.CommissionAmount != 30000 {
		t.Fatalf("expected 30000 commission, got %d", comm.CommissionAmount)
	}

	// Idempotent commission check
	commDup, err := uc.CalculateOrderCommission(ctx, tenantID, payroll.CalculateCommissionRequest{
		EmployeeUUID: emp.UUID,
		OrderID:      501,
		OrderTotal:   1000000,
		TotalItems:   5,
	})
	if err != nil || commDup.UUID != comm.UUID {
		t.Fatalf("expected idempotent commission return")
	}

	// 4. Create Period & Preview
	period, err := uc.CreatePeriod(ctx, tenantID, payroll.CreatePeriodRequest{
		PeriodName:  "November 2026",
		StartDate:   "2026-11-01",
		EndDate:     "2026-11-30",
		PaymentDate: "2026-12-01",
	})
	if err != nil {
		t.Fatalf("CreatePeriod failed: %v", err)
	}

	preview, err := uc.PreviewPeriodPayroll(ctx, tenantID, period.UUID, payroll.PreviewPeriodQuery{
		OvertimeRatePerHour: 25000,
		LatePenaltyPerMin:   1000,
	})
	if err != nil {
		t.Fatalf("PreviewPeriodPayroll failed: %v", err)
	}
	// Gross = 3,000,000 + 800,000 (allowances) + 50,000 (2h overtime) + 30,000 (commission) = 3,880,000
	// Deductions = 50,000 + 100,000 + 15,000 (15m late) = 165,000
	// Net = 3,715,000
	if preview.TotalPayout != 3715000 {
		t.Fatalf("expected preview total payout 3715000, got %d", preview.TotalPayout)
	}

	// 5. Finalize Period with a Kasbon deduction of 100,000
	finalized, err := uc.FinalizePeriod(ctx, tenantID, userID, period.UUID, payroll.FinalizePeriodRequest{
		OvertimeRatePerHour: 25000,
		LatePenaltyPerMin:   1000,
		Adjustments: []payroll.PayrollAdjustmentInput{
			{
				EmployeeUUID:  emp.UUID,
				ItemType:      "DEDUCTION",
				ComponentName: "Potongan Kasbon",
				Amount:        100000,
				Notes:         "Cicilan kasbon ke-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("FinalizePeriod failed: %v", err)
	}
	if finalized.Period.Status != "FINALIZED" {
		t.Fatalf("expected FINALIZED status, got %s", finalized.Period.Status)
	}
	if finalized.Period.TotalPayout != 3615000 {
		t.Fatalf("expected finalized payout 3615000, got %d", finalized.Period.TotalPayout)
	}

	// Pending commission must now be PAID
	if repo.commissions[0].Status != "PAID" {
		t.Fatalf("expected commission status PAID, got %s", repo.commissions[0].Status)
	}

	// Double finalization must fail
	if _, err := uc.FinalizePeriod(ctx, tenantID, userID, period.UUID, payroll.FinalizePeriodRequest{}); err == nil {
		t.Fatal("expected error when finalizing an already finalized period")
	}

	// 6. Immutability Check: Update salary structure next month, snapshot must remain unchanged
	_, err = uc.UpsertSalaryStructure(ctx, tenantID, emp.UUID, payroll.SalaryStructureRequest{
		BasicSalary: 5000000,
	})
	if err != nil {
		t.Fatalf("UpsertSalaryStructure failed: %v", err)
	}
	snap, err := uc.GetSnapshot(ctx, tenantID, finalized.Snapshots[0].UUID)
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if snap.BasicSalary != 3000000 || snap.NetSalary != 3615000 {
		t.Fatalf("snapshot mutated after salary update: basic=%d net=%d", snap.BasicSalary, snap.NetSalary)
	}

	// 7. Payslip PDF generation check
	pdfBytes := payroll.GeneratePayslipPDF(snap)
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-1.4")) || !bytes.Contains(pdfBytes, []byte("Budi Santoso")) {
		t.Fatalf("generated PDF invalid or missing employee name")
	}
}

func TestPayrollHTTPRoutesAndRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockPayrollRepo()
	uc := payroll.NewUsecase(repo)
	handler := payroll.NewHandler(uc)

	jwtSvc := pkgjwt.NewService("test-secret", time.Hour, time.Hour)
	r := gin.New()
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api, middleware.Authenticate(jwtSvc))

	// Cashier without payroll:finalize permission
	cashierPair, _ := jwtSvc.GenerateTokenPair(
		2, uuid.New(), 1, uuid.New(), "kasir", "CASHIER", []string{"pos:checkout"}, []uint64{1},
	)
	cashierToken := cashierPair.AccessToken

	// HR/Admin with full payroll permissions
	adminPair, _ := jwtSvc.GenerateTokenPair(
		1, uuid.New(), 1, uuid.New(), "admin", "SUPERADMIN", []string{"payroll:read", "payroll:finalize"}, []uint64{1},
	)
	adminToken := adminPair.AccessToken

	body, _ := json.Marshal(payroll.CreateEmployeeRequest{
		OutletID:     1,
		EmployeeCode: "EMP-100",
		FullName:     "Siti Aminah",
		JobTitle:     "Staff Gudang",
		JoinedDate:   "2026-03-01",
	})

	// Cashier should get 403 Forbidden on POST /payroll/employees
	reqForbidden := httptest.NewRequest(http.MethodPost, "/api/v1/payroll/employees", bytes.NewReader(body))
	reqForbidden.Header.Set("Authorization", "Bearer "+cashierToken)
	reqForbidden.Header.Set("Content-Type", "application/json")
	wForbidden := httptest.NewRecorder()
	r.ServeHTTP(wForbidden, reqForbidden)
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cashier, got %d", wForbidden.Code)
	}

	// Admin should get 201 Created
	reqOK := httptest.NewRequest(http.MethodPost, "/api/v1/payroll/employees", bytes.NewReader(body))
	reqOK.Header.Set("Authorization", "Bearer "+adminToken)
	reqOK.Header.Set("Content-Type", "application/json")
	wOK := httptest.NewRecorder()
	r.ServeHTTP(wOK, reqOK)
	if wOK.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for admin, got %d: %s", wOK.Code, wOK.Body.String())
	}
}
