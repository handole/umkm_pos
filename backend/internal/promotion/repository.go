package promotion

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/gorm"
)

// Repository defines data access operations for marketing promotions and customers.
type Repository interface {
	WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error

	// Customers
	CreateCustomer(ctx context.Context, customer *model.Customer) error
	FindCustomerByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Customer, error)
	FindCustomerByID(ctx context.Context, tenantID uint64, id uint64) (*model.Customer, error)
	FindCustomerByPhone(ctx context.Context, tenantID uint64, phone string) (*model.Customer, error)
	UpdateCustomer(ctx context.Context, customer *model.Customer) error
	ListCustomers(ctx context.Context, tenantID uint64, search string, limit, offset int) ([]model.Customer, int64, error)

	// Promotions
	CreatePromotion(ctx context.Context, promo *model.Promotion) error
	FindPromotionByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Promotion, error)
	FindPromotionByCode(ctx context.Context, tenantID uint64, code string) (*model.Promotion, error)
	UpdatePromotion(ctx context.Context, promo *model.Promotion) error
	DeletePromotion(ctx context.Context, tenantID uint64, u uuid.UUID) error
	ListPromotions(ctx context.Context, tenantID uint64, promoType *string, isActive *bool, limit, offset int) ([]model.Promotion, int64, error)
	ListActivePromotionsForOutlet(ctx context.Context, tenantID, outletID uint64, checkTime time.Time) ([]model.Promotion, error)

	// Promotion Rules and Outlets
	CreatePromotionRule(ctx context.Context, rule *model.PromotionRule) error
	DeletePromotionRules(ctx context.Context, promoID uint64) error
	AssignOutlets(ctx context.Context, promoID uint64, outletIDs []uint64) error

	// Usage Logging and Quota checks
	RecordUsageLog(ctx context.Context, log *model.PromotionUsageLog) error
	CountCustomerUsage(ctx context.Context, promoID, customerID uint64) (int, error)
	CountDailyUsage(ctx context.Context, promoID uint64, date time.Time) (int, error)

	// Lookups
	FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error)
	FindCategoryByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Category, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new instance of promotion Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &repository{db: tx}
		return fn(txRepo)
	})
}

func (r *repository) CreateCustomer(ctx context.Context, customer *model.Customer) error {
	return r.db.WithContext(ctx).Create(customer).Error
}

func (r *repository) FindCustomerByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Customer, error) {
	var customer model.Customer
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *repository) FindCustomerByID(ctx context.Context, tenantID uint64, id uint64) (*model.Customer, error) {
	var customer model.Customer
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *repository) FindCustomerByPhone(ctx context.Context, tenantID uint64, phone string) (*model.Customer, error) {
	var customer model.Customer
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND phone = ?", tenantID, phone).
		First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *repository) UpdateCustomer(ctx context.Context, customer *model.Customer) error {
	return r.db.WithContext(ctx).Save(customer).Error
}

func (r *repository) ListCustomers(ctx context.Context, tenantID uint64, search string, limit, offset int) ([]model.Customer, int64, error) {
	var customers []model.Customer
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Customer{}).Where("tenant_id = ?", tenantID)
	if search != "" {
		s := "%" + search + "%"
		query = query.Where("name ILIKE ? OR phone ILIKE ? OR code ILIKE ?", s, s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	err := query.Order("id DESC").Limit(limit).Offset(offset).Find(&customers).Error
	return customers, total, err
}

func (r *repository) CreatePromotion(ctx context.Context, promo *model.Promotion) error {
	return r.db.WithContext(ctx).Create(promo).Error
}

func (r *repository) FindPromotionByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Promotion, error) {
	var promo model.Promotion
	err := r.db.WithContext(ctx).
		Preload("Outlets").
		Preload("Rules.Category").
		Preload("Rules.Variant").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&promo).Error
	if err != nil {
		return nil, err
	}
	return &promo, nil
}

func (r *repository) FindPromotionByCode(ctx context.Context, tenantID uint64, code string) (*model.Promotion, error) {
	var promo model.Promotion
	err := r.db.WithContext(ctx).
		Preload("Outlets").
		Preload("Rules.Category").
		Preload("Rules.Variant").
		Where("tenant_id = ? AND code = ? AND is_active = true", tenantID, code).
		First(&promo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &promo, nil
}

func (r *repository) UpdatePromotion(ctx context.Context, promo *model.Promotion) error {
	return r.db.WithContext(ctx).Save(promo).Error
}

func (r *repository) DeletePromotion(ctx context.Context, tenantID uint64, u uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		Delete(&model.Promotion{}).Error
}

func (r *repository) ListPromotions(ctx context.Context, tenantID uint64, promoType *string, isActive *bool, limit, offset int) ([]model.Promotion, int64, error) {
	var promos []model.Promotion
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Promotion{}).Where("tenant_id = ?", tenantID)
	if promoType != nil && *promoType != "" {
		query = query.Where("promo_type = ?", *promoType)
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	err := query.Preload("Outlets").
		Preload("Rules").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&promos).Error

	return promos, total, err
}

func (r *repository) ListActivePromotionsForOutlet(ctx context.Context, tenantID, outletID uint64, checkTime time.Time) ([]model.Promotion, error) {
	dateStr := checkTime.Format("2006-01-02")
	var promos []model.Promotion

	err := r.db.WithContext(ctx).
		Preload("Outlets").
		Preload("Rules.Category").
		Preload("Rules.Variant").
		Where("tenant_id = ? AND is_active = true AND start_date <= ? AND end_date >= ?", tenantID, dateStr, dateStr).
		Find(&promos).Error
	if err != nil {
		return nil, err
	}

	// Filter promotions assigned to this outlet (or unassigned = all outlets)
	var filtered []model.Promotion
	for _, p := range promos {
		if len(p.Outlets) == 0 {
			filtered = append(filtered, p)
			continue
		}
		for _, o := range p.Outlets {
			if o.ID == outletID {
				filtered = append(filtered, p)
				break
			}
		}
	}

	return filtered, nil
}

func (r *repository) CreatePromotionRule(ctx context.Context, rule *model.PromotionRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *repository) DeletePromotionRules(ctx context.Context, promoID uint64) error {
	return r.db.WithContext(ctx).Where("promotion_id = ?", promoID).Delete(&model.PromotionRule{}).Error
}

func (r *repository) AssignOutlets(ctx context.Context, promoID uint64, outletIDs []uint64) error {
	_ = r.db.WithContext(ctx).Where("promotion_id = ?", promoID).Delete(&model.PromotionOutlet{}).Error
	for _, oid := range outletIDs {
		po := model.PromotionOutlet{
			PromotionID: promoID,
			OutletID:    oid,
		}
		if err := r.db.WithContext(ctx).Create(&po).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *repository) RecordUsageLog(ctx context.Context, log *model.PromotionUsageLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *repository) CountCustomerUsage(ctx context.Context, promoID, customerID uint64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PromotionUsageLog{}).
		Where("promotion_id = ? AND customer_id = ?", promoID, customerID).
		Count(&count).Error
	return int(count), err
}

func (r *repository) CountDailyUsage(ctx context.Context, promoID uint64, date time.Time) (int, error) {
	var count int64
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := r.db.WithContext(ctx).
		Model(&model.PromotionUsageLog{}).
		Where("promotion_id = ? AND used_at >= ? AND used_at < ?", promoID, startOfDay, endOfDay).
		Count(&count).Error
	return int(count), err
}

func (r *repository) FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error) {
	var variant model.ProductVariant
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&variant).Error
	if err != nil {
		return nil, err
	}
	return &variant, nil
}

func (r *repository) FindCategoryByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Category, error) {
	var cat model.Category
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&cat).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}
