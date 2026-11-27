package pos_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"github.com/handokodenih/umkm_pos/backend/internal/pos"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
	"gorm.io/gorm"
)

type mockPosRepo struct {
	shifts         []model.PosShift
	cashLogs       []model.PosShiftCashLog
	paymentMethods []model.PosPaymentMethod
	orders         []model.PosOrder
	orderItems     []model.PosOrderItem
	orderPayments  []model.PosOrderPayment
	stocks         []model.InventoryStock
	movements      []model.StockMovement
	locations      []model.InventoryLocation
	variants       []model.ProductVariant
	uoms           []model.ProductUOM
	outlets        []model.Outlet
	tenants        []model.Tenant
	users          []model.User
	employees      []model.Employee
	commissions    []model.PayrollCommissionPending

	shiftSeq uint64
	logSeq   uint64
	pmSeq    uint64
	ordSeq   uint64
	itmSeq   uint64
	paySeq   uint64
	stkSeq   uint64
	movSeq   uint64
}

func newMockPosRepo() *mockPosRepo {
	r := &mockPosRepo{}
	r.paymentMethods = []model.PosPaymentMethod{
		{ID: 1, UUID: uuid.New(), TenantID: 1, Code: "CASH", Name: "Uang Tunai", RequiresReference: false, IsActive: true},
		{ID: 2, UUID: uuid.New(), TenantID: 1, Code: "QRIS", Name: "QRIS", RequiresReference: true, IsActive: true},
		{ID: 3, UUID: uuid.New(), TenantID: 1, Code: "DEBIT_CARD", Name: "Debit Card EDC", RequiresReference: true, IsActive: true},
	}
	r.pmSeq = 3
	return r
}

func (m *mockPosRepo) WithTransaction(ctx context.Context, fn func(txRepo pos.Repository) error) error {
	return fn(m)
}

func (m *mockPosRepo) CreateShift(ctx context.Context, shift *model.PosShift) error {
	m.shiftSeq++
	shift.ID = m.shiftSeq
	shift.UUID = uuid.New()
	shift.CreatedAt = time.Now()
	shift.UpdatedAt = time.Now()
	m.shifts = append(m.shifts, *shift)
	return nil
}

func (m *mockPosRepo) FindOpenShiftByCashier(ctx context.Context, tenantID, cashierID uint64) (*model.PosShift, error) {
	for i := range m.shifts {
		s := &m.shifts[i]
		if s.TenantID == tenantID && s.CashierUserID == cashierID && s.Status == "OPEN" {
			cp := *s
			var logs []model.PosShiftCashLog
			for _, cl := range m.cashLogs {
				if cl.ShiftID == cp.ID {
					logs = append(logs, cl)
				}
			}
			cp.CashLogs = logs
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockPosRepo) FindShiftByID(ctx context.Context, tenantID, shiftID uint64) (*model.PosShift, error) {
	for i := range m.shifts {
		s := &m.shifts[i]
		if s.TenantID == tenantID && s.ID == shiftID {
			cp := *s
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) UpdateShift(ctx context.Context, shift *model.PosShift) error {
	for i, s := range m.shifts {
		if s.ID == shift.ID {
			m.shifts[i] = *shift
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPosRepo) ListShifts(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, limit, offset int) ([]model.PosShift, int64, error) {
	var res []model.PosShift
	for _, s := range m.shifts {
		if s.TenantID == tenantID {
			if outletID != nil && s.OutletID != *outletID {
				continue
			}
			if cashierID != nil && s.CashierUserID != *cashierID {
				continue
			}
			res = append(res, s)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPosRepo) CreateCashLog(ctx context.Context, log *model.PosShiftCashLog) error {
	m.logSeq++
	log.ID = m.logSeq
	log.UUID = uuid.New()
	log.CreatedAt = time.Now()
	m.cashLogs = append(m.cashLogs, *log)
	return nil
}

func (m *mockPosRepo) ListPaymentMethods(ctx context.Context, tenantID uint64) ([]model.PosPaymentMethod, error) {
	var res []model.PosPaymentMethod
	for _, pm := range m.paymentMethods {
		if pm.TenantID == tenantID && pm.IsActive {
			res = append(res, pm)
		}
	}
	return res, nil
}

func (m *mockPosRepo) FindPaymentMethodByCode(ctx context.Context, tenantID uint64, code string) (*model.PosPaymentMethod, error) {
	for _, pm := range m.paymentMethods {
		if pm.TenantID == tenantID && pm.Code == code && pm.IsActive {
			return &pm, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) SeedDefaultPaymentMethods(ctx context.Context, tenantID uint64) error {
	return nil
}

func (m *mockPosRepo) FindOrderByTenantAndIdempotency(ctx context.Context, tenantID uint64, key uuid.UUID) (*model.PosOrder, error) {
	for _, o := range m.orders {
		if o.TenantID == tenantID && o.IdempotencyKey == key {
			cp := o
			m.attachOrderRelations(&cp)
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockPosRepo) FindOrderByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PosOrder, error) {
	for _, o := range m.orders {
		if o.TenantID == tenantID && o.UUID == u {
			cp := o
			m.attachOrderRelations(&cp)
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) attachOrderRelations(o *model.PosOrder) {
	for _, out := range m.outlets {
		if out.ID == o.OutletID {
			outCopy := out
			o.Outlet = &outCopy
			break
		}
	}
	for _, u := range m.users {
		if u.ID == o.CashierUserID {
			uCopy := u
			o.Cashier = &uCopy
			break
		}
	}
	var items []model.PosOrderItem
	for _, it := range m.orderItems {
		if it.OrderID == o.ID {
			itCopy := it
			for _, v := range m.variants {
				if v.ID == it.VariantID {
					vCopy := v
					itCopy.Variant = &vCopy
					break
				}
			}
			for _, u := range m.uoms {
				if it.UOMID != nil && u.ID == *it.UOMID {
					uCopy := u
					itCopy.UOM = &uCopy
					break
				}
			}
			items = append(items, itCopy)
		}
	}
	o.Items = items

	var payments []model.PosOrderPayment
	for _, p := range m.orderPayments {
		if p.OrderID == o.ID {
			pCopy := p
			for _, pm := range m.paymentMethods {
				if pm.ID == p.PaymentMethodID {
					pmCopy := pm
					pCopy.PaymentMethod = &pmCopy
					break
				}
			}
			payments = append(payments, pCopy)
		}
	}
	o.Payments = payments
}

func (m *mockPosRepo) CreateOrder(ctx context.Context, order *model.PosOrder) error {
	m.ordSeq++
	order.ID = m.ordSeq
	order.UUID = uuid.New()
	order.CreatedAt = time.Now()
	m.orders = append(m.orders, *order)
	return nil
}

func (m *mockPosRepo) UpdateOrder(ctx context.Context, order *model.PosOrder) error {
	for i, o := range m.orders {
		if o.ID == order.ID {
			m.orders[i] = *order
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPosRepo) CreateOrderItem(ctx context.Context, item *model.PosOrderItem) error {
	m.itmSeq++
	item.ID = m.itmSeq
	item.UUID = uuid.New()
	m.orderItems = append(m.orderItems, *item)
	return nil
}

func (m *mockPosRepo) CreateOrderPayment(ctx context.Context, payment *model.PosOrderPayment) error {
	m.paySeq++
	payment.ID = m.paySeq
	payment.UUID = uuid.New()
	payment.PaidAt = time.Now()
	m.orderPayments = append(m.orderPayments, *payment)
	return nil
}

func (m *mockPosRepo) ListOrders(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, status *string, limit, offset int) ([]model.PosOrder, int64, error) {
	var res []model.PosOrder
	for _, o := range m.orders {
		if o.TenantID == tenantID {
			if outletID != nil && o.OutletID != *outletID {
				continue
			}
			if cashierID != nil && o.CashierUserID != *cashierID {
				continue
			}
			if status != nil && *status != "" && o.Status != *status {
				continue
			}
			cp := o
			m.attachOrderRelations(&cp)
			res = append(res, cp)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPosRepo) GetStockWithLock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error) {
	for i, s := range m.stocks {
		if s.TenantID == tenantID && s.LocationID == locationID && s.VariantID == variantID {
			return &m.stocks[i], nil
		}
	}
	m.stkSeq++
	newStock := model.InventoryStock{
		ID:         m.stkSeq,
		UUID:       uuid.New(),
		TenantID:   tenantID,
		LocationID: locationID,
		VariantID:  variantID,
		Quantity:   0,
		UpdatedAt:  time.Now(),
	}
	m.stocks = append(m.stocks, newStock)
	return &m.stocks[len(m.stocks)-1], nil
}

func (m *mockPosRepo) SaveStock(ctx context.Context, stock *model.InventoryStock) error {
	for i, s := range m.stocks {
		if s.ID == stock.ID || (s.TenantID == stock.TenantID && s.LocationID == stock.LocationID && s.VariantID == stock.VariantID) {
			m.stocks[i].Quantity = stock.Quantity
			m.stocks[i].UpdatedAt = time.Now()
			return nil
		}
	}
	m.stkSeq++
	stock.ID = m.stkSeq
	if stock.UUID == uuid.Nil {
		stock.UUID = uuid.New()
	}
	stock.UpdatedAt = time.Now()
	m.stocks = append(m.stocks, *stock)
	return nil
}

func (m *mockPosRepo) RecordMovement(ctx context.Context, movement *model.StockMovement) error {
	m.movSeq++
	movement.ID = m.movSeq
	movement.UUID = uuid.New()
	movement.CreatedAt = time.Now()
	m.movements = append(m.movements, *movement)
	return nil
}

func (m *mockPosRepo) FindLocationByID(ctx context.Context, tenantID, locationID uint64) (*model.InventoryLocation, error) {
	for _, l := range m.locations {
		if l.TenantID == tenantID && l.ID == locationID {
			return &l, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindLocationByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.InventoryLocation, error) {
	for _, l := range m.locations {
		if l.TenantID == tenantID && l.UUID == u {
			return &l, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindDefaultStoreFrontLocation(ctx context.Context, tenantID, outletID uint64) (*model.InventoryLocation, error) {
	for _, l := range m.locations {
		if l.TenantID == tenantID && l.OutletID == outletID && l.Type == "STORE_FRONT" {
			return &l, nil
		}
	}
	for _, l := range m.locations {
		if l.TenantID == tenantID && l.OutletID == outletID {
			return &l, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error) {
	for _, v := range m.variants {
		if v.TenantID == tenantID && v.UUID == u {
			return &v, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindUOMByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductUOM, error) {
	for _, uom := range m.uoms {
		if uom.TenantID == tenantID && uom.UUID == u {
			return &uom, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindOutletByID(ctx context.Context, tenantID, outletID uint64) (*model.Outlet, error) {
	for _, o := range m.outlets {
		if o.TenantID == tenantID && o.ID == outletID {
			return &o, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindTenantByID(ctx context.Context, tenantID uint64) (*model.Tenant, error) {
	for _, t := range m.tenants {
		if t.ID == tenantID {
			return &t, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindUserByID(ctx context.Context, userID uint64) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == userID {
			return &u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPosRepo) FindEmployeeByUserID(ctx context.Context, tenantID, userID uint64) (*model.Employee, error) {
	for _, e := range m.employees {
		if e.TenantID == tenantID && e.UserID != nil && *e.UserID == userID && e.IsActive {
			ec := e
			return &ec, nil
		}
	}
	return nil, nil
}

func (m *mockPosRepo) CreatePendingCommission(ctx context.Context, comm *model.PayrollCommissionPending) error {
	comm.ID = uint64(len(m.commissions) + 1)
	comm.UUID = uuid.New()
	comm.CreatedAt = time.Now()
	m.commissions = append(m.commissions, *comm)
	return nil
}

func (m *mockPosRepo) VoidPendingCommissionsByOrder(ctx context.Context, tenantID, orderID uint64) error {
	for i := range m.commissions {
		if m.commissions[i].TenantID == tenantID && m.commissions[i].OrderID == orderID && m.commissions[i].Status == "PENDING" {
			m.commissions[i].Status = "VOID"
		}
	}
	return nil
}

func (m *mockPosRepo) ListOrdersForReport(ctx context.Context, tenantID uint64, outletID *uint64, startDate, endDate *time.Time) ([]model.PosOrder, error) {
	var res []model.PosOrder
	for _, o := range m.orders {
		if o.TenantID != tenantID {
			continue
		}
		if outletID != nil && o.OutletID != *outletID {
			continue
		}
		if startDate != nil && o.OrderedAt.Before(*startDate) {
			continue
		}
		if endDate != nil && o.OrderedAt.After(*endDate) {
			continue
		}
		cp := o
		m.attachOrderRelations(&cp)
		res = append(res, cp)
	}
	return res, nil
}

func setupPosTestApp(repo pos.Repository) (*gin.Engine, string, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtService := pkgjwt.NewService("test-secret-pos-key", time.Hour, 24*time.Hour)

	cashierTokens, _ := jwtService.GenerateTokenPair(
		10, uuid.New(), 1, uuid.New(), "kasir_budi", "CASHIER",
		[]string{"pos:checkout", "pos:shift"}, []uint64{1},
	)

	managerTokens, _ := jwtService.GenerateTokenPair(
		1, uuid.New(), 1, uuid.New(), "manager_budi", "STORE_MANAGER",
		[]string{"pos:checkout", "pos:shift", "pos:void"}, []uint64{1},
	)

	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyTenantID, uint64(1))
		c.Next()
	})

	apiV1 := r.Group("/api/v1")
	uc := pos.NewUsecase(repo)
	handler := pos.NewHandler(uc)
	handler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

	return r, cashierTokens.AccessToken, managerTokens.AccessToken
}

func TestPosShiftLifecycleAndCashLogs(t *testing.T) {
	repo := newMockPosRepo()
	r, cashierToken, _ := setupPosTestApp(repo)

	// 1. Open shift with initial cash 100000
	openPayload := pos.OpenShiftRequest{
		OutletID:     1,
		StartingCash: 100000,
		Notes:        "Shift pagi kasir 1",
	}
	bodyOpen, _ := json.Marshal(openPayload)
	reqOpen := httptest.NewRequest(http.MethodPost, "/api/v1/pos/shifts/open", bytes.NewBuffer(bodyOpen))
	reqOpen.Header.Set("Authorization", "Bearer "+cashierToken)
	reqOpen.Header.Set("Content-Type", "application/json")
	wOpen := httptest.NewRecorder()
	r.ServeHTTP(wOpen, reqOpen)

	if wOpen.Code != http.StatusCreated {
		t.Fatalf("expected open shift 201, got %d: %s", wOpen.Code, wOpen.Body.String())
	}

	var shiftRes struct {
		Data pos.ShiftResponse `json:"data"`
	}
	json.Unmarshal(wOpen.Body.Bytes(), &shiftRes)
	if shiftRes.Data.ExpectedCash != 100000 || shiftRes.Data.Status != "OPEN" {
		t.Fatalf("unexpected shift status: %+v", shiftRes.Data)
	}

	// 2. Add petty cash in (+50000)
	cashInPayload := pos.CashLogRequest{
		Type:   "CASH_IN",
		Amount: 50000,
		Reason: "Tambahan uang kembalian pecahan 2000",
	}
	bodyIn, _ := json.Marshal(cashInPayload)
	reqIn := httptest.NewRequest(http.MethodPost, "/api/v1/pos/shifts/cash-log", bytes.NewBuffer(bodyIn))
	reqIn.Header.Set("Authorization", "Bearer "+cashierToken)
	reqIn.Header.Set("Content-Type", "application/json")
	wIn := httptest.NewRecorder()
	r.ServeHTTP(wIn, reqIn)

	if wIn.Code != http.StatusCreated {
		t.Fatalf("expected cash in 201, got %d: %s", wIn.Code, wIn.Body.String())
	}

	// 3. Add cash out (-20000)
	cashOutPayload := pos.CashLogRequest{
		Type:   "CASH_OUT",
		Amount: 20000,
		Reason: "Beli isolasi dan kantong plastik",
	}
	bodyOut, _ := json.Marshal(cashOutPayload)
	reqOut := httptest.NewRequest(http.MethodPost, "/api/v1/pos/shifts/cash-log", bytes.NewBuffer(bodyOut))
	reqOut.Header.Set("Authorization", "Bearer "+cashierToken)
	reqOut.Header.Set("Content-Type", "application/json")
	wOut := httptest.NewRecorder()
	r.ServeHTTP(wOut, reqOut)

	if wOut.Code != http.StatusCreated {
		t.Fatalf("expected cash out 201, got %d: %s", wOut.Code, wOut.Body.String())
	}

	// 4. Verify current shift expected cash: 100000 + 50000 - 20000 = 130000
	reqCur := httptest.NewRequest(http.MethodGet, "/api/v1/pos/shifts/current", nil)
	reqCur.Header.Set("Authorization", "Bearer "+cashierToken)
	wCur := httptest.NewRecorder()
	r.ServeHTTP(wCur, reqCur)

	if wCur.Code != http.StatusOK {
		t.Fatalf("expected current shift 200, got %d: %s", wCur.Code, wCur.Body.String())
	}

	var curRes struct {
		Data pos.ShiftResponse `json:"data"`
	}
	json.Unmarshal(wCur.Body.Bytes(), &curRes)
	if curRes.Data.ExpectedCash != 130000 {
		t.Fatalf("expected expected_cash 130000, got %d", curRes.Data.ExpectedCash)
	}

	// 5. Close shift with physical cash 130000 -> Discrepancy 0
	closePayload := pos.CloseShiftRequest{
		ActualCash: 130000,
		Notes:      "Hitungan fisik laci kas pas",
	}
	bodyClose, _ := json.Marshal(closePayload)
	reqClose := httptest.NewRequest(http.MethodPost, "/api/v1/pos/shifts/close", bytes.NewBuffer(bodyClose))
	reqClose.Header.Set("Authorization", "Bearer "+cashierToken)
	reqClose.Header.Set("Content-Type", "application/json")
	wClose := httptest.NewRecorder()
	r.ServeHTTP(wClose, reqClose)

	if wClose.Code != http.StatusOK {
		t.Fatalf("expected close shift 200, got %d: %s", wClose.Code, wClose.Body.String())
	}

	var closedRes struct {
		Data pos.ShiftResponse `json:"data"`
	}
	json.Unmarshal(wClose.Body.Bytes(), &closedRes)
	if closedRes.Data.Status != "CLOSED" || *closedRes.Data.CashDiscrepancy != 0 {
		t.Fatalf("expected closed shift with 0 discrepancy, got %+v", closedRes.Data)
	}
}

func TestPosCheckoutIdempotencyAndPessimisticStock(t *testing.T) {
	repo := newMockPosRepo()
	r, cashierToken, _ := setupPosTestApp(repo)

	locUUID := uuid.New()
	loc := model.InventoryLocation{
		ID: 1, UUID: locUUID, TenantID: 1, OutletID: 1, Code: "STORE-01", Name: "Rak Depan Kasir", Type: "STORE_FRONT", IsActive: true,
	}
	repo.locations = append(repo.locations, loc)

	variantUUID := uuid.New()
	variant := model.ProductVariant{
		ID: 101, UUID: variantUUID, TenantID: 1, SKU: "IND-01", VariantName: "Indomie Goreng", BaseSellPrice: 3500, BaseCostPrice: 2800,
	}
	repo.variants = append(repo.variants, variant)

	// Initial stock = 20
	repo.stocks = append(repo.stocks, model.InventoryStock{
		ID: 1, UUID: uuid.New(), TenantID: 1, LocationID: loc.ID, VariantID: variant.ID, Quantity: 20,
	})

	idempotencyKey := uuid.New()
	checkoutPayload := pos.CheckoutOrderRequest{
		IdempotencyKey: idempotencyKey,
		OutletID:       1,
		LocationUUID:   &locUUID,
		Items: []pos.CheckoutItemRequest{
			{
				VariantUUID: variantUUID,
				Quantity:    3, // Buy 3 -> Total 10500
			},
		},
		Payments: []pos.CheckoutPaymentRequest{
			{
				PaymentMethodCode: "CASH",
				Amount:            15000, // Bayar 15000 -> Kembalian 4500
			},
		},
	}
	bodyCheck, _ := json.Marshal(checkoutPayload)

	// 1. Initial checkout request
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewBuffer(bodyCheck))
	req1.Header.Set("Authorization", "Bearer "+cashierToken)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("expected checkout 201, got %d: %s", w1.Code, w1.Body.String())
	}

	var order1 struct {
		Data pos.OrderResponse `json:"data"`
	}
	json.Unmarshal(w1.Body.Bytes(), &order1)
	if order1.Data.TotalAmount != 10500 || order1.Data.ChangeAmount != 4500 {
		t.Fatalf("expected total 10500 change 4500, got total %d change %d", order1.Data.TotalAmount, order1.Data.ChangeAmount)
	}

	// Verify stock is now 20 - 3 = 17
	stock, _ := repo.GetStockWithLock(context.Background(), 1, loc.ID, variant.ID)
	if stock.Quantity != 17 {
		t.Fatalf("expected remaining stock 17, got %d", stock.Quantity)
	}

	// 2. Retry with same Idempotency Key -> Must return the same order without double decrement
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewBuffer(bodyCheck))
	req2.Header.Set("Authorization", "Bearer "+cashierToken)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var order2 struct {
		Data pos.OrderResponse `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &order2)
	if order2.Data.OrderNumber != order1.Data.OrderNumber {
		t.Fatalf("expected identical order from idempotency key retry, got %s vs %s", order2.Data.OrderNumber, order1.Data.OrderNumber)
	}

	// Stock must remain 17 (no double charge)
	stockAfterRetry, _ := repo.GetStockWithLock(context.Background(), 1, loc.ID, variant.ID)
	if stockAfterRetry.Quantity != 17 {
		t.Fatalf("stock was deducted multiple times! expected 17, got %d", stockAfterRetry.Quantity)
	}

	// 3. Stock Exceeded Test: Try to buy 50 when only 17 available
	excessPayload := pos.CheckoutOrderRequest{
		IdempotencyKey: uuid.New(),
		OutletID:       1,
		LocationUUID:   &locUUID,
		Items: []pos.CheckoutItemRequest{
			{
				VariantUUID: variantUUID,
				Quantity:    50, // Available 17
			},
		},
		Payments: []pos.CheckoutPaymentRequest{
			{PaymentMethodCode: "CASH", Amount: 200000},
		},
	}
	bodyExcess, _ := json.Marshal(excessPayload)
	reqExcess := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewBuffer(bodyExcess))
	reqExcess.Header.Set("Authorization", "Bearer "+cashierToken)
	reqExcess.Header.Set("Content-Type", "application/json")
	wExcess := httptest.NewRecorder()
	r.ServeHTTP(wExcess, reqExcess)

	if wExcess.Code != http.StatusBadRequest {
		t.Fatalf("expected stock overflow to return 400 Bad Request, got %d", wExcess.Code)
	}
}

func TestPosSplitPaymentThermalReceiptAndVoid(t *testing.T) {
	repo := newMockPosRepo()
	r, cashierToken, managerToken := setupPosTestApp(repo)

	locUUID := uuid.New()
	loc := model.InventoryLocation{
		ID: 1, UUID: locUUID, TenantID: 1, OutletID: 1, Code: "STORE-01", Name: "Toko Utama", Type: "STORE_FRONT", IsActive: true,
	}
	repo.locations = append(repo.locations, loc)

	variantUUID := uuid.New()
	variant := model.ProductVariant{
		ID: 201, UUID: variantUUID, TenantID: 1, SKU: "KOPI-01", VariantName: "Kopi Hitam", BaseSellPrice: 3000, BaseCostPrice: 2000,
	}
	repo.variants = append(repo.variants, variant)

	// UOM Karton (1 Dus = 40 Pcs, harga 100.000)
	uomUUID := uuid.New()
	uom := model.ProductUOM{
		ID: 1, UUID: uomUUID, TenantID: 1, VariantID: variant.ID, UOMName: "Dus", ConversionRate: 40, SellPrice: 100000,
	}
	repo.uoms = append(repo.uoms, uom)

	// Stock = 100 pcs
	repo.stocks = append(repo.stocks, model.InventoryStock{
		ID: 1, UUID: uuid.New(), TenantID: 1, LocationID: loc.ID, VariantID: variant.ID, Quantity: 100,
	})

	// 1. Checkout with Split Payment: 1 Dus Kopi (100.000)
	// Bayar: Tunai 40.000 + QRIS 60.000 = 100.000
	checkoutPayload := pos.CheckoutOrderRequest{
		IdempotencyKey: uuid.New(),
		OutletID:       1,
		LocationUUID:   &locUUID,
		Items: []pos.CheckoutItemRequest{
			{
				VariantUUID: variantUUID,
				UOMUUID:     &uomUUID,
				Quantity:    1, // 1 Dus = 40 Pcs
			},
		},
		Payments: []pos.CheckoutPaymentRequest{
			{
				PaymentMethodCode: "CASH",
				Amount:            40000,
			},
			{
				PaymentMethodCode: "QRIS",
				Amount:            60000,
				ReferenceNumber:   "QRIS-REF-998811",
			},
		},
	}
	bodyCheck, _ := json.Marshal(checkoutPayload)
	reqCheck := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewBuffer(bodyCheck))
	reqCheck.Header.Set("Authorization", "Bearer "+cashierToken)
	reqCheck.Header.Set("Content-Type", "application/json")
	wCheck := httptest.NewRecorder()
	r.ServeHTTP(wCheck, reqCheck)

	if wCheck.Code != http.StatusCreated {
		t.Fatalf("expected split payment checkout 201, got %d: %s", wCheck.Code, wCheck.Body.String())
	}

	var orderRes struct {
		Data pos.OrderResponse `json:"data"`
	}
	json.Unmarshal(wCheck.Body.Bytes(), &orderRes)
	orderUUID := orderRes.Data.UUID

	// Check stock deducted by base quantity: 100 - 40 = 60
	stock, _ := repo.GetStockWithLock(context.Background(), 1, loc.ID, variant.ID)
	if stock.Quantity != 60 {
		t.Fatalf("expected remaining stock 60 after carton purchase, got %d", stock.Quantity)
	}

	// 2. Fetch formatted thermal receipt
	receiptUrl := fmt.Sprintf("/api/v1/pos/orders/%s/receipt", orderUUID)
	reqReceipt := httptest.NewRequest(http.MethodGet, receiptUrl, nil)
	reqReceipt.Header.Set("Authorization", "Bearer "+cashierToken)
	wReceipt := httptest.NewRecorder()
	r.ServeHTTP(wReceipt, reqReceipt)

	if wReceipt.Code != http.StatusOK {
		t.Fatalf("expected receipt status 200, got %d: %s", wReceipt.Code, wReceipt.Body.String())
	}

	var receiptRes struct {
		Data pos.OrderReceiptResponse `json:"data"`
	}
	json.Unmarshal(wReceipt.Body.Bytes(), &receiptRes)
	if receiptRes.Data.Summary.Total != 100000 || len(receiptRes.Data.Payments) != 2 {
		t.Fatalf("unexpected receipt structure: %+v", receiptRes.Data)
	}

	// 3. Manager Voids the order
	voidUrl := fmt.Sprintf("/api/v1/pos/orders/%s/void", orderUUID)
	voidPayload := pos.VoidOrderRequest{
		Reason: "Pembeli salah pesan varian rasa",
	}
	bodyVoid, _ := json.Marshal(voidPayload)
	reqVoid := httptest.NewRequest(http.MethodPost, voidUrl, bytes.NewBuffer(bodyVoid))
	reqVoid.Header.Set("Authorization", "Bearer "+managerToken)
	reqVoid.Header.Set("Content-Type", "application/json")
	wVoid := httptest.NewRecorder()
	r.ServeHTTP(wVoid, reqVoid)

	if wVoid.Code != http.StatusOK {
		t.Fatalf("expected void status 200, got %d: %s", wVoid.Code, wVoid.Body.String())
	}

	// Verify order status is VOID
	var voidRes struct {
		Data pos.OrderResponse `json:"data"`
	}
	json.Unmarshal(wVoid.Body.Bytes(), &voidRes)
	if voidRes.Data.Status != "VOID" {
		t.Fatalf("expected order status VOID, got %s", voidRes.Data.Status)
	}

	// Verify stock is restored back: 60 + 40 = 100
	stockRestored, _ := repo.GetStockWithLock(context.Background(), 1, loc.ID, variant.ID)
	if stockRestored.Quantity != 100 {
		t.Fatalf("expected stock restored to 100, got %d", stockRestored.Quantity)
	}
}

func TestAutoCommissionAndSalesReport(t *testing.T) {
	ctx := context.Background()
	repo := newMockPosRepo()
	uc := pos.NewUsecase(repo)

	loc := model.InventoryLocation{ID: 1, UUID: uuid.New(), TenantID: 1, OutletID: 1, Code: "OUTLET-1", Name: "Toko", Type: "STORE_FRONT", IsActive: true}
	variant := model.ProductVariant{ID: 1, UUID: uuid.New(), TenantID: 1, SKU: "SKU-1", VariantName: "Kopi", BaseCostPrice: 10000, BaseSellPrice: 25000}
	uom := model.ProductUOM{ID: 1, UUID: uuid.New(), TenantID: 1, VariantID: 1, UOMName: "Cup", ConversionRate: 1, SellPrice: 25000}
	repo.locations = append(repo.locations, loc)
	repo.variants = append(repo.variants, variant)
	repo.uoms = append(repo.uoms, uom)
	repo.stocks = append(repo.stocks, model.InventoryStock{ID: 1, TenantID: 1, LocationID: 1, VariantID: 1, Quantity: 50})

	salesUserID := uint64(77)
	repo.employees = append(repo.employees, model.Employee{
		ID:       10,
		TenantID: 1,
		UserID:   &salesUserID,
		IsActive: true,
		SalaryStructure: &model.SalaryStructure{
			CommissionRatePercent:  5.0,
			CommissionFixedPerItem: 500,
		},
	})

	// Checkout 2 items = 50,000 => 5% (2,500) + 2*500 (1,000) = 3,500 commission
	orderRes, err := uc.CheckoutOrder(ctx, 1, 1, pos.CheckoutOrderRequest{
		IdempotencyKey:    uuid.New(),
		OutletID:          1,
		SalespersonUserID: &salesUserID,
		Items: []pos.CheckoutItemRequest{
			{VariantUUID: variant.UUID, UOMUUID: &uom.UUID, Quantity: 2},
		},
		Payments: []pos.CheckoutPaymentRequest{
			{PaymentMethodCode: "QRIS", Amount: 50000, ReferenceNumber: "QR-001"},
		},
	})
	if err != nil {
		t.Fatalf("CheckoutOrder failed: %v", err)
	}
	if len(repo.commissions) != 1 || repo.commissions[0].CommissionAmount != 3500 || repo.commissions[0].Status != "PENDING" {
		t.Fatalf("unexpected auto commission: %+v", repo.commissions)
	}

	// Sales Report Summary
	report, err := uc.GetSalesReport(ctx, 1, pos.SalesReportQuery{})
	if err != nil {
		t.Fatalf("GetSalesReport failed: %v", err)
	}
	if report.CompletedOrders != 1 || report.NetSales != 50000 || report.TotalCOGS != 20000 || report.GrossProfit != 30000 {
		t.Fatalf("unexpected sales report: %+v", report)
	}

	// CSV Export
	csvData, err := uc.ExportSalesReportCSV(ctx, 1, pos.SalesReportQuery{})
	if err != nil || !bytes.Contains(csvData, []byte(orderRes.OrderNumber)) {
		t.Fatalf("unexpected CSV export: err=%v", err)
	}

	// Voiding order should also void pending commission
	if _, err := uc.VoidOrder(ctx, 1, 99, orderRes.UUID, pos.VoidOrderRequest{Reason: "Test void"}); err != nil {
		t.Fatalf("VoidOrder failed: %v", err)
	}
	if repo.commissions[0].Status != "VOID" {
		t.Fatalf("expected commission status VOID after order void, got %s", repo.commissions[0].Status)
	}
}
