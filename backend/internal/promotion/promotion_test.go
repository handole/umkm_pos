package promotion_test

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
	"github.com/handokodenih/umkm_pos/backend/internal/promotion"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
	"gorm.io/gorm"
)

type mockPromotionRepo struct {
	customers []model.Customer
	promos    []model.Promotion
	rules     []model.PromotionRule
	outlets   []model.PromotionOutlet
	logs      []model.PromotionUsageLog
	variants  []model.ProductVariant
	categories []model.Category

	custSeq uint64
	prmSeq  uint64
	rulSeq  uint64
	logSeq  uint64
}

func newMockPromotionRepo() *mockPromotionRepo {
	return &mockPromotionRepo{}
}

func (m *mockPromotionRepo) WithTransaction(ctx context.Context, fn func(txRepo promotion.Repository) error) error {
	return fn(m)
}

func (m *mockPromotionRepo) CreateCustomer(ctx context.Context, customer *model.Customer) error {
	m.custSeq++
	customer.ID = m.custSeq
	customer.UUID = uuid.New()
	customer.CreatedAt = time.Now()
	customer.UpdatedAt = time.Now()
	m.customers = append(m.customers, *customer)
	return nil
}

func (m *mockPromotionRepo) FindCustomerByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Customer, error) {
	for _, c := range m.customers {
		if c.TenantID == tenantID && c.UUID == u {
			return &c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPromotionRepo) FindCustomerByID(ctx context.Context, tenantID uint64, id uint64) (*model.Customer, error) {
	for _, c := range m.customers {
		if c.TenantID == tenantID && c.ID == id {
			return &c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPromotionRepo) FindCustomerByPhone(ctx context.Context, tenantID uint64, phone string) (*model.Customer, error) {
	for _, c := range m.customers {
		if c.TenantID == tenantID && c.Phone == phone {
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockPromotionRepo) UpdateCustomer(ctx context.Context, customer *model.Customer) error {
	for i, c := range m.customers {
		if c.ID == customer.ID {
			m.customers[i] = *customer
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPromotionRepo) ListCustomers(ctx context.Context, tenantID uint64, search string, limit, offset int) ([]model.Customer, int64, error) {
	var res []model.Customer
	for _, c := range m.customers {
		if c.TenantID == tenantID {
			res = append(res, c)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPromotionRepo) CreatePromotion(ctx context.Context, promo *model.Promotion) error {
	m.prmSeq++
	promo.ID = m.prmSeq
	promo.UUID = uuid.New()
	promo.CreatedAt = time.Now()
	promo.UpdatedAt = time.Now()
	m.promos = append(m.promos, *promo)
	return nil
}

func (m *mockPromotionRepo) FindPromotionByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Promotion, error) {
	for _, p := range m.promos {
		if p.TenantID == tenantID && p.UUID == u {
			cp := p
			m.attachPromotionRelations(&cp)
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPromotionRepo) FindPromotionByCode(ctx context.Context, tenantID uint64, code string) (*model.Promotion, error) {
	for _, p := range m.promos {
		if p.TenantID == tenantID && p.Code == code && p.IsActive {
			cp := p
			m.attachPromotionRelations(&cp)
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockPromotionRepo) attachPromotionRelations(p *model.Promotion) {
	var rules []model.PromotionRule
	for _, r := range m.rules {
		if r.PromotionID == p.ID {
			rCopy := r
			for _, v := range m.variants {
				if r.VariantID != nil && v.ID == *r.VariantID {
					vCopy := v
					rCopy.Variant = &vCopy
					break
				}
			}
			for _, c := range m.categories {
				if r.CategoryID != nil && c.ID == *r.CategoryID {
					cCopy := c
					rCopy.Category = &cCopy
					break
				}
			}
			rules = append(rules, rCopy)
		}
	}
	p.Rules = rules
}

func (m *mockPromotionRepo) UpdatePromotion(ctx context.Context, promo *model.Promotion) error {
	for i, p := range m.promos {
		if p.ID == promo.ID {
			m.promos[i] = *promo
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockPromotionRepo) DeletePromotion(ctx context.Context, tenantID uint64, u uuid.UUID) error {
	for i, p := range m.promos {
		if p.TenantID == tenantID && p.UUID == u {
			m.promos = append(m.promos[:i], m.promos[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockPromotionRepo) ListPromotions(ctx context.Context, tenantID uint64, promoType *string, isActive *bool, limit, offset int) ([]model.Promotion, int64, error) {
	var res []model.Promotion
	for _, p := range m.promos {
		if p.TenantID == tenantID {
			if promoType != nil && *promoType != "" && p.PromoType != *promoType {
				continue
			}
			if isActive != nil && p.IsActive != *isActive {
				continue
			}
			cp := p
			m.attachPromotionRelations(&cp)
			res = append(res, cp)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPromotionRepo) ListActivePromotionsForOutlet(ctx context.Context, tenantID, outletID uint64, checkTime time.Time) ([]model.Promotion, error) {
	dateStr := checkTime.Format("2006-01-02")
	var res []model.Promotion
	for _, p := range m.promos {
		if p.TenantID == tenantID && p.IsActive {
			startStr := p.StartDate.Format("2006-01-02")
			endStr := p.EndDate.Format("2006-01-02")
			if dateStr >= startStr && dateStr <= endStr {
				cp := p
				m.attachPromotionRelations(&cp)
				res = append(res, cp)
			}
		}
	}
	return res, nil
}

func (m *mockPromotionRepo) CreatePromotionRule(ctx context.Context, rule *model.PromotionRule) error {
	m.rulSeq++
	rule.ID = m.rulSeq
	rule.UUID = uuid.New()
	rule.CreatedAt = time.Now()
	m.rules = append(m.rules, *rule)
	return nil
}

func (m *mockPromotionRepo) DeletePromotionRules(ctx context.Context, promoID uint64) error {
	var remaining []model.PromotionRule
	for _, r := range m.rules {
		if r.PromotionID != promoID {
			remaining = append(remaining, r)
		}
	}
	m.rules = remaining
	return nil
}

func (m *mockPromotionRepo) AssignOutlets(ctx context.Context, promoID uint64, outletIDs []uint64) error {
	for _, oid := range outletIDs {
		m.outlets = append(m.outlets, model.PromotionOutlet{
			PromotionID: promoID,
			OutletID:    oid,
		})
	}
	return nil
}

func (m *mockPromotionRepo) RecordUsageLog(ctx context.Context, log *model.PromotionUsageLog) error {
	m.logSeq++
	log.ID = m.logSeq
	log.UUID = uuid.New()
	log.UsedAt = time.Now()
	m.logs = append(m.logs, *log)
	return nil
}

func (m *mockPromotionRepo) CountCustomerUsage(ctx context.Context, promoID, customerID uint64) (int, error) {
	count := 0
	for _, l := range m.logs {
		if l.PromotionID == promoID && l.CustomerID != nil && *l.CustomerID == customerID {
			count++
		}
	}
	return count, nil
}

func (m *mockPromotionRepo) CountDailyUsage(ctx context.Context, promoID uint64, date time.Time) (int, error) {
	count := 0
	dateStr := date.Format("2006-01-02")
	for _, l := range m.logs {
		if l.PromotionID == promoID && l.UsedAt.Format("2006-01-02") == dateStr {
			count++
		}
	}
	return count, nil
}

func (m *mockPromotionRepo) FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error) {
	for _, v := range m.variants {
		if v.TenantID == tenantID && v.UUID == u {
			return &v, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockPromotionRepo) FindCategoryByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Category, error) {
	for _, c := range m.categories {
		if c.TenantID == tenantID && c.UUID == u {
			return &c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func setupPromotionTestApp(repo promotion.Repository) (*gin.Engine, string, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtService := pkgjwt.NewService("test-secret-promotion-key", time.Hour, 24*time.Hour)

	managerTokens, _ := jwtService.GenerateTokenPair(
		1, uuid.New(), 1, uuid.New(), "manager_budi", "STORE_MANAGER",
		[]string{"promotion:manage", "pos:checkout"}, []uint64{1},
	)

	cashierTokens, _ := jwtService.GenerateTokenPair(
		2, uuid.New(), 1, uuid.New(), "kasir_siti", "CASHIER",
		[]string{"pos:checkout"}, []uint64{1},
	)

	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyTenantID, uint64(1))
		c.Next()
	})

	apiV1 := r.Group("/api/v1")
	uc := promotion.NewUsecase(repo)
	handler := promotion.NewHandler(uc)
	handler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

	return r, managerTokens.AccessToken, cashierTokens.AccessToken
}

func TestCustomerRegistrationAndListing(t *testing.T) {
	repo := newMockPromotionRepo()
	r, managerToken, _ := setupPromotionTestApp(repo)

	// 1. Create customer
	createCust := promotion.CreateCustomerRequest{
		Code:       "MBR-001",
		Name:       "Andi Pratama",
		Phone:      "08123456789",
		Email:      "andi@example.com",
		MemberTier: "GOLD",
	}
	body, _ := json.Marshal(createCust)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+managerToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected create customer 201, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data promotion.CustomerResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.Name != "Andi Pratama" || res.Data.MemberTier != "GOLD" {
		t.Fatalf("unexpected customer response: %+v", res.Data)
	}

	// 2. Duplicate phone check -> Should fail with 400 Bad Request
	reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewBuffer(body))
	reqDup.Header.Set("Authorization", "Bearer "+managerToken)
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate customer to fail with 400, got %d", wDup.Code)
	}
}

func TestPromotionDateAndTimeSchedulingHappyHour(t *testing.T) {
	repo := newMockPromotionRepo()
	r, _, cashierToken := setupPromotionTestApp(repo)

	variantUUID := uuid.New()
	repo.variants = append(repo.variants, model.ProductVariant{
		ID: 1, UUID: variantUUID, TenantID: 1, SKU: "KOPI-01", VariantName: "Kopi Arabika", BaseSellPrice: 30000,
	})

	startTime := "14:00:00"
	endTime := "17:00:00"
	maxDisc := int64(25000)

	// Happy hour promotion: 20% discount between 14:00 and 17:00
	promo := model.Promotion{
		ID:                1,
		UUID:              uuid.New(),
		TenantID:          1,
		Code:              "HAPPYHOUR",
		Name:              "Happy Hour Sore 20%",
		PromoType:         "AUTOMATIC",
		DiscountType:      "PERCENTAGE",
		DiscountValue:     20,
		MaxDiscountAmount: &maxDisc,
		MinOrderAmount:    50000,
		MinQuantity:       1,
		TargetScope:       "ENTIRE_TRANSACTION",
		StartDate:         time.Now().Add(-24 * time.Hour),
		EndDate:           time.Now().Add(24 * time.Hour),
		StartTime:         &startTime,
		EndTime:           &endTime,
		DaysOfWeek:        "ALL",
		DaysOfMonth:       "ALL",
		ApplicableMonths:  "ALL",
		IsActive:          true,
	}
	repo.promos = append(repo.promos, promo)

	// 1. Evaluate at 15:30 (INSIDE happy hour) with subtotal 100000 (buy 4 x 30000 = 120000)
	timeInside := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 15, 30, 0, 0, time.Local)
	cartInside := promotion.EvaluatePromotionRequest{
		OutletID:        1,
		TransactionTime: &timeInside,
		Items: []promotion.CartItemDTO{
			{
				VariantUUID: variantUUID,
				Quantity:    4,
				UnitPrice:   30000,
			},
		},
	}
	bodyInside, _ := json.Marshal(cartInside)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/promotions/evaluate", bytes.NewBuffer(bodyInside))
	req1.Header.Set("Authorization", "Bearer "+cashierToken)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected evaluate status 200, got %d: %s", w1.Code, w1.Body.String())
	}

	var res1 struct {
		Data promotion.EvaluatePromotionResponse `json:"data"`
	}
	json.Unmarshal(w1.Body.Bytes(), &res1)
	// 20% of 120000 is 24000 (under 25000 max cap)
	if !res1.Data.IsEligible || res1.Data.DiscountAmount != 24000 {
		t.Fatalf("expected discount 24000 inside happy hour, got %+v", res1.Data)
	}

	// 2. Evaluate at 10:00 (OUTSIDE happy hour)
	timeOutside := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 10, 0, 0, 0, time.Local)
	cartOutside := promotion.EvaluatePromotionRequest{
		OutletID:        1,
		TransactionTime: &timeOutside,
		Items: []promotion.CartItemDTO{
			{
				VariantUUID: variantUUID,
				Quantity:    4,
				UnitPrice:   30000,
			},
		},
	}
	bodyOutside, _ := json.Marshal(cartOutside)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/promotions/evaluate", bytes.NewBuffer(bodyOutside))
	req2.Header.Set("Authorization", "Bearer "+cashierToken)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var res2 struct {
		Data promotion.EvaluatePromotionResponse `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &res2)
	if res2.Data.IsEligible || res2.Data.DiscountAmount != 0 {
		t.Fatalf("expected no discount outside happy hour, got %+v", res2.Data)
	}
}

func TestCouponCodeValidationAndTargetedProductScope(t *testing.T) {
	repo := newMockPromotionRepo()
	r, _, cashierToken := setupPromotionTestApp(repo)

	varPromoUUID := uuid.New()
	varNormalUUID := uuid.New()
	repo.variants = append(repo.variants,
		model.ProductVariant{ID: 10, UUID: varPromoUUID, TenantID: 1, SKU: "SUSU-01", VariantName: "Susu UHT"},
		model.ProductVariant{ID: 20, UUID: varNormalUUID, TenantID: 1, SKU: "ROTI-01", VariantName: "Roti Tawar"},
	)

	// Coupon code promo: DISKON20K, target only SUSU-01
	promo := model.Promotion{
		ID:            2,
		UUID:          uuid.New(),
		TenantID:      1,
		Code:          "DISKON20K",
		Name:          "Kupon Diskon Susu 20 Ribu",
		PromoType:     "COUPON_CODE",
		DiscountType:  "FIXED_AMOUNT",
		DiscountValue: 20000,
		TargetScope:   "SPECIFIC_PRODUCTS",
		StartDate:     time.Now().Add(-24 * time.Hour),
		EndDate:       time.Now().Add(24 * time.Hour),
		DaysOfWeek:    "ALL",
		DaysOfMonth:   "ALL",
		IsActive:      true,
	}
	repo.promos = append(repo.promos, promo)

	varID := uint64(10)
	rule := model.PromotionRule{
		PromotionID: promo.ID,
		RuleType:    "INCLUDE_PRODUCT",
		VariantID:   &varID,
	}
	repo.rules = append(repo.rules, rule)

	// 1. Test Valid Coupon Code on Targeted Product
	cart := promotion.EvaluatePromotionRequest{
		OutletID:   1,
		CouponCode: "DISKON20K",
		Items: []promotion.CartItemDTO{
			{
				VariantUUID: varPromoUUID,
				Quantity:    2,
				UnitPrice:   15000, // Subtotal Susu = 30000
			},
			{
				VariantUUID: varNormalUUID,
				Quantity:    1,
				UnitPrice:   25000, // Subtotal Roti = 25000
			},
		},
	}
	body, _ := json.Marshal(cart)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/promotions/evaluate", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected evaluate status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data promotion.EvaluatePromotionResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	// Original: 30000 + 25000 = 55000. Discount = 20000. Final = 35000.
	if !res.Data.IsEligible || res.Data.DiscountAmount != 20000 || res.Data.FinalTotal != 35000 {
		t.Fatalf("expected discount 20000 on coupon, got %+v", res.Data)
	}

	// 2. Test Invalid Coupon Code
	cartInvalid := promotion.EvaluatePromotionRequest{
		OutletID:   1,
		CouponCode: "KODE_PALSU",
		Items: []promotion.CartItemDTO{
			{VariantUUID: varPromoUUID, Quantity: 1, UnitPrice: 15000},
		},
	}
	bodyInv, _ := json.Marshal(cartInvalid)
	reqInv := httptest.NewRequest(http.MethodPost, "/api/v1/promotions/evaluate", bytes.NewBuffer(bodyInv))
	reqInv.Header.Set("Authorization", "Bearer "+cashierToken)
	reqInv.Header.Set("Content-Type", "application/json")
	wInv := httptest.NewRecorder()
	r.ServeHTTP(wInv, reqInv)

	var resInv struct {
		Data promotion.EvaluatePromotionResponse `json:"data"`
	}
	json.Unmarshal(wInv.Body.Bytes(), &resInv)
	if resInv.Data.IsEligible {
		t.Fatalf("expected invalid coupon to not be eligible, got %+v", resInv.Data)
	}
}
