package promotion

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
)

// Usecase defines the business logic contract for promotions and customer loyalty.
type Usecase interface {
	// Customers
	CreateCustomer(ctx context.Context, tenantID uint64, req CreateCustomerRequest) (*CustomerResponse, error)
	GetCustomer(ctx context.Context, tenantID uint64, u uuid.UUID) (*CustomerResponse, error)
	UpdateCustomer(ctx context.Context, tenantID uint64, u uuid.UUID, req UpdateCustomerRequest) (*CustomerResponse, error)
	ListCustomers(ctx context.Context, tenantID uint64, search string, limit, offset int) ([]CustomerResponse, int64, error)

	// Promotions
	CreatePromotion(ctx context.Context, tenantID uint64, req CreatePromotionRequest) (*PromotionResponse, error)
	GetPromotion(ctx context.Context, tenantID uint64, u uuid.UUID) (*PromotionResponse, error)
	UpdatePromotion(ctx context.Context, tenantID uint64, u uuid.UUID, req UpdatePromotionRequest) (*PromotionResponse, error)
	DeletePromotion(ctx context.Context, tenantID uint64, u uuid.UUID) error
	ListPromotions(ctx context.Context, tenantID uint64, promoType *string, isActive *bool, limit, offset int) ([]PromotionResponse, int64, error)

	// Engine Evaluation
	EvaluateCart(ctx context.Context, tenantID uint64, req EvaluatePromotionRequest) (*EvaluatePromotionResponse, error)
	RecordPromotionUsage(ctx context.Context, tenantID, promoID, outletID, orderID uint64, customerID *uint64, discountAmount int64) error
}

type usecase struct {
	repo Repository
}

// NewUsecase returns an instance of promotion Usecase.
func NewUsecase(repo Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) CreateCustomer(ctx context.Context, tenantID uint64, req CreateCustomerRequest) (*CustomerResponse, error) {
	existing, _ := u.repo.FindCustomerByPhone(ctx, tenantID, req.Phone)
	if existing != nil {
		return nil, fmt.Errorf("customer with phone number '%s' already exists", req.Phone)
	}

	tier := req.MemberTier
	if tier == "" {
		tier = "REGULAR"
	}

	customer := model.Customer{
		TenantID:   tenantID,
		Code:       strings.ToUpper(strings.TrimSpace(req.Code)),
		Name:       strings.TrimSpace(req.Name),
		Phone:      strings.TrimSpace(req.Phone),
		Email:      strings.TrimSpace(req.Email),
		MemberTier: tier,
		TotalSpend: 0,
		IsActive:   true,
	}

	if err := u.repo.CreateCustomer(ctx, &customer); err != nil {
		return nil, fmt.Errorf("failed to register customer: %w", err)
	}

	return &CustomerResponse{
		ID:         customer.ID,
		UUID:       customer.UUID,
		Code:       customer.Code,
		Name:       customer.Name,
		Phone:      customer.Phone,
		Email:      customer.Email,
		MemberTier: customer.MemberTier,
		TotalSpend: customer.TotalSpend,
		IsActive:   customer.IsActive,
		CreatedAt:  customer.CreatedAt,
	}, nil
}

func (u *usecase) GetCustomer(ctx context.Context, tenantID uint64, customerUUID uuid.UUID) (*CustomerResponse, error) {
	customer, err := u.repo.FindCustomerByUUID(ctx, tenantID, customerUUID)
	if err != nil {
		return nil, err
	}

	return &CustomerResponse{
		ID:         customer.ID,
		UUID:       customer.UUID,
		Code:       customer.Code,
		Name:       customer.Name,
		Phone:      customer.Phone,
		Email:      customer.Email,
		MemberTier: customer.MemberTier,
		TotalSpend: customer.TotalSpend,
		IsActive:   customer.IsActive,
		CreatedAt:  customer.CreatedAt,
	}, nil
}

func (u *usecase) UpdateCustomer(ctx context.Context, tenantID uint64, customerUUID uuid.UUID, req UpdateCustomerRequest) (*CustomerResponse, error) {
	customer, err := u.repo.FindCustomerByUUID(ctx, tenantID, customerUUID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		customer.Name = strings.TrimSpace(*req.Name)
	}
	if req.Phone != nil {
		customer.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Email != nil {
		customer.Email = strings.TrimSpace(*req.Email)
	}
	if req.MemberTier != nil {
		customer.MemberTier = strings.ToUpper(strings.TrimSpace(*req.MemberTier))
	}
	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}

	if err := u.repo.UpdateCustomer(ctx, customer); err != nil {
		return nil, err
	}

	return u.GetCustomer(ctx, tenantID, customerUUID)
}

func (u *usecase) ListCustomers(ctx context.Context, tenantID uint64, search string, limit, offset int) ([]CustomerResponse, int64, error) {
	customers, total, err := u.repo.ListCustomers(ctx, tenantID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	res := make([]CustomerResponse, len(customers))
	for i, c := range customers {
		res[i] = CustomerResponse{
			ID:         c.ID,
			UUID:       c.UUID,
			Code:       c.Code,
			Name:       c.Name,
			Phone:      c.Phone,
			Email:      c.Email,
			MemberTier: c.MemberTier,
			TotalSpend: c.TotalSpend,
			IsActive:   c.IsActive,
			CreatedAt:  c.CreatedAt,
		}
	}
	return res, total, nil
}

func (u *usecase) CreatePromotion(ctx context.Context, tenantID uint64, req CreatePromotionRequest) (*PromotionResponse, error) {
	existing, _ := u.repo.FindPromotionByCode(ctx, tenantID, req.Code)
	if existing != nil {
		return nil, fmt.Errorf("promotion code '%s' already exists", req.Code)
	}

	daysOfWeek := req.DaysOfWeek
	if daysOfWeek == "" {
		daysOfWeek = "ALL"
	}
	daysOfMonth := req.DaysOfMonth
	if daysOfMonth == "" {
		daysOfMonth = "ALL"
	}
	applicableMonths := req.ApplicableMonths
	if applicableMonths == "" {
		applicableMonths = "ALL"
	}
	perCustQuota := req.PerCustomerQuota
	if perCustQuota <= 0 {
		perCustQuota = 1
	}

	promo := model.Promotion{
		TenantID:          tenantID,
		Code:              strings.ToUpper(strings.TrimSpace(req.Code)),
		Name:              strings.TrimSpace(req.Name),
		Description:       strings.TrimSpace(req.Description),
		PromoType:         req.PromoType,
		DiscountType:      req.DiscountType,
		DiscountValue:     req.DiscountValue,
		MaxDiscountAmount: req.MaxDiscountAmount,
		MinOrderAmount:    req.MinOrderAmount,
		MinQuantity:       req.MinQuantity,
		TargetScope:       req.TargetScope,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		StartTime:         req.StartTime,
		EndTime:           req.EndTime,
		DaysOfWeek:        daysOfWeek,
		DaysOfMonth:       daysOfMonth,
		ApplicableMonths:  applicableMonths,
		TotalQuota:        req.TotalQuota,
		DailyQuota:        req.DailyQuota,
		PerCustomerQuota:  perCustQuota,
		TotalBudget:       req.TotalBudget,
		IsStackable:       req.IsStackable,
		IsActive:          true,
	}

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		if err := txRepo.CreatePromotion(ctx, &promo); err != nil {
			return fmt.Errorf("failed to save promotion: %w", err)
		}

		if len(req.OutletIDs) > 0 {
			if err := txRepo.AssignOutlets(ctx, promo.ID, req.OutletIDs); err != nil {
				return fmt.Errorf("failed to bind outlet restrictions: %w", err)
			}
		}

		for _, r := range req.Rules {
			var catID *uint64
			var varID *uint64
			var buyVarID *uint64
			var rewardVarID *uint64

			if r.CategoryUUID != nil {
				cat, errCat := txRepo.FindCategoryByUUID(ctx, tenantID, *r.CategoryUUID)
				if errCat != nil {
					return fmt.Errorf("category %s not found: %w", *r.CategoryUUID, errCat)
				}
				catID = &cat.ID
			}

			if r.VariantUUID != nil {
				v, errVar := txRepo.FindVariantByUUID(ctx, tenantID, *r.VariantUUID)
				if errVar != nil {
					return fmt.Errorf("variant %s not found: %w", *r.VariantUUID, errVar)
				}
				varID = &v.ID
			}

			if r.BuyVariantUUID != nil {
				bv, errBv := txRepo.FindVariantByUUID(ctx, tenantID, *r.BuyVariantUUID)
				if errBv != nil {
					return fmt.Errorf("buy variant not found: %w", errBv)
				}
				buyVarID = &bv.ID
			}

			if r.RewardVariantUUID != nil {
				rv, errRv := txRepo.FindVariantByUUID(ctx, tenantID, *r.RewardVariantUUID)
				if errRv != nil {
					return fmt.Errorf("reward variant not found: %w", errRv)
				}
				rewardVarID = &rv.ID
			}

			ruleModel := model.PromotionRule{
				PromotionID:     promo.ID,
				RuleType:        r.RuleType,
				CategoryID:      catID,
				VariantID:       varID,
				BuyVariantID:    buyVarID,
				BuyQuantity:     r.BuyQuantity,
				RewardVariantID: rewardVarID,
				RewardQuantity:  r.RewardQuantity,
			}

			if err := txRepo.CreatePromotionRule(ctx, &ruleModel); err != nil {
				return fmt.Errorf("failed to save rule: %w", err)
			}
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return u.GetPromotion(ctx, tenantID, promo.UUID)
}

func (u *usecase) GetPromotion(ctx context.Context, tenantID uint64, promoUUID uuid.UUID) (*PromotionResponse, error) {
	promo, err := u.repo.FindPromotionByUUID(ctx, tenantID, promoUUID)
	if err != nil {
		return nil, err
	}
	return u.mapPromoToResponse(promo), nil
}

func (u *usecase) UpdatePromotion(ctx context.Context, tenantID uint64, promoUUID uuid.UUID, req UpdatePromotionRequest) (*PromotionResponse, error) {
	promo, err := u.repo.FindPromotionByUUID(ctx, tenantID, promoUUID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		promo.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		promo.Description = strings.TrimSpace(*req.Description)
	}
	if req.DiscountValue != nil {
		promo.DiscountValue = *req.DiscountValue
	}
	if req.MaxDiscountAmount != nil {
		promo.MaxDiscountAmount = req.MaxDiscountAmount
	}
	if req.MinOrderAmount != nil {
		promo.MinOrderAmount = *req.MinOrderAmount
	}
	if req.MinQuantity != nil {
		promo.MinQuantity = *req.MinQuantity
	}
	if req.StartDate != nil {
		promo.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		promo.EndDate = *req.EndDate
	}
	if req.StartTime != nil {
		promo.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		promo.EndTime = req.EndTime
	}
	if req.DaysOfWeek != nil {
		promo.DaysOfWeek = *req.DaysOfWeek
	}
	if req.DaysOfMonth != nil {
		promo.DaysOfMonth = *req.DaysOfMonth
	}
	if req.ApplicableMonths != nil {
		promo.ApplicableMonths = *req.ApplicableMonths
	}
	if req.TotalQuota != nil {
		promo.TotalQuota = req.TotalQuota
	}
	if req.DailyQuota != nil {
		promo.DailyQuota = req.DailyQuota
	}
	if req.PerCustomerQuota != nil {
		promo.PerCustomerQuota = *req.PerCustomerQuota
	}
	if req.TotalBudget != nil {
		promo.TotalBudget = req.TotalBudget
	}
	if req.IsActive != nil {
		promo.IsActive = *req.IsActive
	}
	if req.IsStackable != nil {
		promo.IsStackable = *req.IsStackable
	}

	if err := u.repo.UpdatePromotion(ctx, promo); err != nil {
		return nil, err
	}

	return u.GetPromotion(ctx, tenantID, promoUUID)
}

func (u *usecase) DeletePromotion(ctx context.Context, tenantID uint64, promoUUID uuid.UUID) error {
	return u.repo.DeletePromotion(ctx, tenantID, promoUUID)
}

func (u *usecase) ListPromotions(ctx context.Context, tenantID uint64, promoType *string, isActive *bool, limit, offset int) ([]PromotionResponse, int64, error) {
	promos, total, err := u.repo.ListPromotions(ctx, tenantID, promoType, isActive, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	res := make([]PromotionResponse, len(promos))
	for i := range promos {
		res[i] = *u.mapPromoToResponse(&promos[i])
	}
	return res, total, nil
}

func (u *usecase) EvaluateCart(ctx context.Context, tenantID uint64, req EvaluatePromotionRequest) (*EvaluatePromotionResponse, error) {
	evalTime := time.Now()
	if req.TransactionTime != nil {
		evalTime = *req.TransactionTime
	}

	var customer *model.Customer
	if req.CustomerUUID != nil {
		cust, err := u.repo.FindCustomerByUUID(ctx, tenantID, *req.CustomerUUID)
		if err == nil {
			customer = cust
		}
	}

	var cartSubtotal int64
	var cartTotalQuantity int64
	for _, it := range req.Items {
		lineTotal := it.UnitPrice * it.Quantity
		cartSubtotal += lineTotal
		cartTotalQuantity += it.Quantity
	}

	emptyResponse := &EvaluatePromotionResponse{
		IsEligible:       false,
		Message:          "Tidak ada promosi yang memenuhi syarat",
		OriginalSubtotal: cartSubtotal,
		DiscountAmount:   0,
		FinalTotal:       cartSubtotal,
	}

	var candidates []model.Promotion

	if req.CouponCode != "" {
		code := strings.ToUpper(strings.TrimSpace(req.CouponCode))
		promo, err := u.repo.FindPromotionByCode(ctx, tenantID, code)
		if err != nil || promo == nil {
			return &EvaluatePromotionResponse{
				IsEligible:       false,
				Message:          fmt.Sprintf("Kode kupon '%s' tidak valid atau telah berakhir", code),
				OriginalSubtotal: cartSubtotal,
				DiscountAmount:   0,
				FinalTotal:       cartSubtotal,
			}, nil
		}
		candidates = append(candidates, *promo)
	} else {
		promos, err := u.repo.ListActivePromotionsForOutlet(ctx, tenantID, req.OutletID, evalTime)
		if err != nil {
			return emptyResponse, nil
		}
		for _, p := range promos {
			if p.PromoType == "AUTOMATIC" {
				candidates = append(candidates, p)
			}
		}
	}

	if len(candidates) == 0 {
		return emptyResponse, nil
	}

	var bestPromo *model.Promotion
	var maxDiscount int64

	for i := range candidates {
		p := &candidates[i]
		if !u.isTimeEligible(p, evalTime) {
			continue
		}
		if !u.isQuotaEligible(ctx, p, customer, evalTime) {
			continue
		}

		if cartSubtotal < p.MinOrderAmount || cartTotalQuantity < p.MinQuantity {
			continue
		}

		eligibleSubtotal := u.calculateEligibleSubtotal(ctx, tenantID, p, req.Items)
		if eligibleSubtotal <= 0 {
			continue
		}

		discount := u.calculateDiscount(p, eligibleSubtotal)
		if discount > maxDiscount {
			maxDiscount = discount
			bestPromo = p
		}
	}

	if bestPromo == nil || maxDiscount <= 0 {
		return emptyResponse, nil
	}

	finalTotal := cartSubtotal - maxDiscount
	if finalTotal < 0 {
		finalTotal = 0
	}

	return &EvaluatePromotionResponse{
		IsEligible:       true,
		Message:          fmt.Sprintf("Promosi '%s' berhasil diterapkan", bestPromo.Name),
		AppliedPromotion: u.mapPromoToResponse(bestPromo),
		OriginalSubtotal: cartSubtotal,
		DiscountAmount:   maxDiscount,
		FinalTotal:       finalTotal,
	}, nil
}

func (u *usecase) isTimeEligible(p *model.Promotion, t time.Time) bool {
	// 1. Day of Week check (1: Monday .. 7: Sunday)
	if p.DaysOfWeek != "ALL" && p.DaysOfWeek != "" {
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday as 7
		}
		dayStr := strconv.Itoa(weekday)
		matched := false
		for _, d := range strings.Split(p.DaysOfWeek, ",") {
			if strings.TrimSpace(d) == dayStr {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 2. Day of Month check (1..31)
	if p.DaysOfMonth != "ALL" && p.DaysOfMonth != "" {
		dayOfMonthStr := strconv.Itoa(t.Day())
		matched := false
		for _, d := range strings.Split(p.DaysOfMonth, ",") {
			if strings.TrimSpace(d) == dayOfMonthStr {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 3. Month check (1..12)
	if p.ApplicableMonths != "ALL" && p.ApplicableMonths != "" {
		monthStr := strconv.Itoa(int(t.Month()))
		matched := false
		for _, m := range strings.Split(p.ApplicableMonths, ",") {
			if strings.TrimSpace(m) == monthStr {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 4. Time of day check (Happy Hour)
	if p.StartTime != nil && p.EndTime != nil && *p.StartTime != "" && *p.EndTime != "" {
		currentTimeStr := t.Format("15:04:05")
		if currentTimeStr < *p.StartTime || currentTimeStr > *p.EndTime {
			return false
		}
	}

	return true
}

func (u *usecase) isQuotaEligible(ctx context.Context, p *model.Promotion, customer *model.Customer, t time.Time) bool {
	if p.TotalQuota != nil && p.CurrentUsageCount >= *p.TotalQuota {
		return false
	}

	if p.TotalBudget != nil && p.CurrentSpendAmount >= *p.TotalBudget {
		return false
	}

	if p.DailyQuota != nil {
		dailyUsed, err := u.repo.CountDailyUsage(ctx, p.ID, t)
		if err == nil && dailyUsed >= *p.DailyQuota {
			return false
		}
	}

	if customer != nil && p.PerCustomerQuota > 0 {
		custUsed, err := u.repo.CountCustomerUsage(ctx, p.ID, customer.ID)
		if err == nil && custUsed >= p.PerCustomerQuota {
			return false
		}
	}

	return true
}

func (u *usecase) calculateEligibleSubtotal(ctx context.Context, tenantID uint64, p *model.Promotion, items []CartItemDTO) int64 {
	if p.TargetScope == "ENTIRE_TRANSACTION" {
		var total int64
		for _, it := range items {
			total += it.UnitPrice * it.Quantity
		}
		return total
	}

	// Map rules for included products or categories
	includedVariants := make(map[uint64]bool)
	includedCategories := make(map[uint64]bool)

	for _, r := range p.Rules {
		if r.RuleType == "INCLUDE_PRODUCT" && r.VariantID != nil {
			includedVariants[*r.VariantID] = true
		}
		if r.RuleType == "INCLUDE_CATEGORY" && r.CategoryID != nil {
			includedCategories[*r.CategoryID] = true
		}
	}

	var eligibleTotal int64
	for _, it := range items {
		v, err := u.repo.FindVariantByUUID(ctx, tenantID, it.VariantUUID)
		if err != nil {
			continue
		}

		matched := false
		if p.TargetScope == "SPECIFIC_PRODUCTS" && includedVariants[v.ID] {
			matched = true
		}
		if p.TargetScope == "SPECIFIC_CATEGORIES" && it.CategoryID != nil && includedCategories[*it.CategoryID] {
			matched = true
		}

		if matched {
			eligibleTotal += it.UnitPrice * it.Quantity
		}
	}

	return eligibleTotal
}

func (u *usecase) calculateDiscount(p *model.Promotion, eligibleSubtotal int64) int64 {
	var discount int64

	switch p.DiscountType {
	case "PERCENTAGE":
		discount = (eligibleSubtotal * p.DiscountValue) / 100
		if p.MaxDiscountAmount != nil && discount > *p.MaxDiscountAmount {
			discount = *p.MaxDiscountAmount
		}
	case "FIXED_AMOUNT":
		discount = p.DiscountValue
		if discount > eligibleSubtotal {
			discount = eligibleSubtotal
		}
	case "SPECIAL_PRICE":
		discount = p.DiscountValue
	}

	return discount
}

func (u *usecase) RecordPromotionUsage(ctx context.Context, tenantID, promoID, outletID, orderID uint64, customerID *uint64, discountAmount int64) error {
	log := model.PromotionUsageLog{
		TenantID:       tenantID,
		PromotionID:    promoID,
		OutletID:       outletID,
		OrderID:        orderID,
		CustomerID:     customerID,
		DiscountAmount: discountAmount,
		UsedAt:         time.Now(),
	}

	return u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		return txRepo.RecordUsageLog(ctx, &log)
	})
}

func (u *usecase) mapPromoToResponse(p *model.Promotion) *PromotionResponse {
	rules := make([]PromotionRuleResponse, len(p.Rules))
	for i, r := range p.Rules {
		catName := ""
		if r.Category != nil {
			catName = r.Category.Name
		}
		varSKU := ""
		varName := ""
		if r.Variant != nil {
			varSKU = r.Variant.SKU
			varName = r.Variant.VariantName
		}
		rules[i] = PromotionRuleResponse{
			ID:             r.ID,
			UUID:           r.UUID,
			RuleType:       r.RuleType,
			CategoryID:     r.CategoryID,
			CategoryName:   catName,
			VariantID:      r.VariantID,
			VariantSKU:     varSKU,
			VariantName:    varName,
			BuyQuantity:    r.BuyQuantity,
			RewardQuantity: r.RewardQuantity,
		}
	}

	return &PromotionResponse{
		ID:                 p.ID,
		UUID:               p.UUID,
		Code:               p.Code,
		Name:               p.Name,
		Description:        p.Description,
		PromoType:          p.PromoType,
		DiscountType:       p.DiscountType,
		DiscountValue:      p.DiscountValue,
		MaxDiscountAmount:  p.MaxDiscountAmount,
		MinOrderAmount:     p.MinOrderAmount,
		MinQuantity:        p.MinQuantity,
		TargetScope:        p.TargetScope,
		StartDate:          p.StartDate,
		EndDate:            p.EndDate,
		StartTime:          p.StartTime,
		EndTime:            p.EndTime,
		DaysOfWeek:         p.DaysOfWeek,
		DaysOfMonth:        p.DaysOfMonth,
		ApplicableMonths:   p.ApplicableMonths,
		TotalQuota:         p.TotalQuota,
		DailyQuota:         p.DailyQuota,
		PerCustomerQuota:   p.PerCustomerQuota,
		CurrentUsageCount:  p.CurrentUsageCount,
		TotalBudget:        p.TotalBudget,
		CurrentSpendAmount: p.CurrentSpendAmount,
		IsStackable:        p.IsStackable,
		IsActive:           p.IsActive,
		CreatedAt:          p.CreatedAt,
		Rules:              rules,
	}
}
