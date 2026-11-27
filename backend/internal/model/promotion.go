package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Customer represents a retail buyer or registered store member.
type Customer struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID       uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_customers_tenant_uuid,priority:2" json:"uuid"`
	TenantID   uint64         `gorm:"not null;uniqueIndex:idx_customers_tenant_uuid,priority:1;uniqueIndex:idx_customers_tenant_code,priority:1;uniqueIndex:idx_customers_tenant_phone,priority:1" json:"tenant_id"`
	Code       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_customers_tenant_code,priority:2" json:"code"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	Phone      string         `gorm:"type:varchar(30);uniqueIndex:idx_customers_tenant_phone,priority:2" json:"phone"`
	Email      string         `gorm:"type:varchar(255)" json:"email"`
	MemberTier string         `gorm:"type:varchar(50);not null;default:'REGULAR'" json:"member_tier"` // 'REGULAR', 'SILVER', 'GOLD', 'PLATINUM'
	TotalSpend int64          `gorm:"not null;default:0" json:"total_spend"`                          // Akumulasi belanja (BIGINT)
	IsActive   bool           `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt  time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (Customer) TableName() string {
	return "customers"
}

// Promotion defines promotional rules, discounts, vouchers, and happy hour schedules.
type Promotion struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID              uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_promotions_tenant_uuid,priority:2" json:"uuid"`
	TenantID          uint64         `gorm:"not null;uniqueIndex:idx_promotions_tenant_uuid,priority:1;uniqueIndex:idx_promotions_tenant_code,priority:1" json:"tenant_id"`
	Code              string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_promotions_tenant_code,priority:2" json:"code"`
	Name              string         `gorm:"type:varchar(255);not null" json:"name"`
	Description       string         `gorm:"type:text" json:"description"`
	PromoType         string         `gorm:"type:varchar(30);not null;default:'AUTOMATIC'" json:"promo_type"`        // 'AUTOMATIC', 'COUPON_CODE'
	DiscountType      string         `gorm:"type:varchar(30);not null" json:"discount_type"`                         // 'PERCENTAGE', 'FIXED_AMOUNT', 'BUY_X_GET_Y', 'SPECIAL_PRICE'
	DiscountValue     int64          `gorm:"not null" json:"discount_value"`                                         // Persen (misal 15) atau nominal (misal 10000)
	MaxDiscountAmount *int64         `json:"max_discount_amount,omitempty"`                                          // Plafon nominal untuk diskon persentase
	MinOrderAmount    int64          `gorm:"not null;default:0" json:"min_order_amount"`                             // Syarat minimal belanja
	MinQuantity       int64          `gorm:"not null;default:1" json:"min_quantity"`                                 // Syarat minimal item
	TargetScope       string         `gorm:"type:varchar(30);not null;default:'ENTIRE_TRANSACTION'" json:"target_scope"` // 'ENTIRE_TRANSACTION', 'SPECIFIC_PRODUCTS', 'SPECIFIC_CATEGORIES'

	// Parameter Waktu & Kalender
	StartDate        time.Time      `gorm:"type:date;not null;index:idx_promotions_schedule,priority:2" json:"start_date"`
	EndDate          time.Time      `gorm:"type:date;not null;index:idx_promotions_schedule,priority:3" json:"end_date"`
	StartTime        *string        `gorm:"type:time" json:"start_time,omitempty"`                                  // '14:00:00'
	EndTime          *string        `gorm:"type:time" json:"end_time,omitempty"`                                    // '17:00:00'
	DaysOfWeek       string         `gorm:"type:varchar(50);not null;default:'ALL'" json:"days_of_week"`            // 'ALL', '1,2,3,4,5', '6,7'
	DaysOfMonth      string         `gorm:"type:varchar(100);not null;default:'ALL'" json:"days_of_month"`          // 'ALL', '25,26,27,28,29,30'
	ApplicableMonths string         `gorm:"type:varchar(50);not null;default:'ALL'" json:"applicable_months"`        // 'ALL', '3,4', '12'

	// Kuota & Budget Subsidi
	TotalQuota         *int           `json:"total_quota,omitempty"`
	DailyQuota         *int           `json:"daily_quota,omitempty"`
	PerCustomerQuota   int            `gorm:"not null;default:1" json:"per_customer_quota"`
	CurrentUsageCount  int            `gorm:"not null;default:0" json:"current_usage_count"`
	TotalBudget        *int64         `json:"total_budget,omitempty"`
	CurrentSpendAmount int64          `gorm:"not null;default:0" json:"current_spend_amount"`

	IsStackable bool           `gorm:"not null;default:false" json:"is_stackable"`
	IsActive    bool           `gorm:"not null;default:true;index:idx_promotions_schedule,priority:4" json:"is_active"`
	CreatedAt   time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant    *Tenant              `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Outlets   []Outlet             `gorm:"many2many:promotion_outlets;" json:"outlets,omitempty"`
	Rules     []PromotionRule      `gorm:"foreignKey:PromotionID" json:"rules,omitempty"`
	UsageLogs []PromotionUsageLog  `gorm:"foreignKey:PromotionID" json:"usage_logs,omitempty"`
}

func (Promotion) TableName() string {
	return "promotions"
}

// PromotionOutlet restricts promotion eligibility to specified outlets.
type PromotionOutlet struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	PromotionID uint64    `gorm:"not null;uniqueIndex:idx_promo_outlets,priority:1" json:"promotion_id"`
	OutletID    uint64    `gorm:"not null;uniqueIndex:idx_promo_outlets,priority:2" json:"outlet_id"`

	// Relations
	Promotion *Promotion `gorm:"foreignKey:PromotionID" json:"promotion,omitempty"`
	Outlet    *Outlet    `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
}

func (PromotionOutlet) TableName() string {
	return "promotion_outlets"
}

// PromotionRule sets fine-grained targeting (Category/Product inclusions, exclusions, Buy X Get Y, Tiered Qty).
type PromotionRule struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	PromotionID       uint64    `gorm:"not null;index:idx_promo_rules,priority:1" json:"promotion_id"`
	RuleType          string    `gorm:"type:varchar(30);not null;index:idx_promo_rules,priority:2" json:"rule_type"` // 'INCLUDE_CATEGORY', 'EXCLUDE_CATEGORY', 'INCLUDE_PRODUCT', 'EXCLUDE_PRODUCT', 'BUY_X_GET_Y', 'TIERED_QTY'
	CategoryID        *uint64   `gorm:"index" json:"category_id,omitempty"`
	VariantID         *uint64   `gorm:"index" json:"variant_id,omitempty"`
	BuyVariantID      *uint64   `gorm:"index" json:"buy_variant_id,omitempty"`
	BuyQuantity       *int      `json:"buy_quantity,omitempty"`
	RewardVariantID   *uint64   `gorm:"index" json:"reward_variant_id,omitempty"`
	RewardQuantity    *int      `json:"reward_quantity,omitempty"`
	TierMinQty        *int      `json:"tier_min_qty,omitempty"`
	TierMaxQty        *int      `json:"tier_max_qty,omitempty"`
	TierDiscountValue *int64    `json:"tier_discount_value,omitempty"`
	CreatedAt         time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Relations
	Promotion     *Promotion      `gorm:"foreignKey:PromotionID" json:"promotion,omitempty"`
	Category      *Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Variant       *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
	BuyVariant    *ProductVariant `gorm:"foreignKey:BuyVariantID" json:"buy_variant,omitempty"`
	RewardVariant *ProductVariant `gorm:"foreignKey:RewardVariantID" json:"reward_variant,omitempty"`
}

func (PromotionRule) TableName() string {
	return "promotion_rules"
}

// PromotionUsageLog records immutable audit entries for each promotion utilization.
type PromotionUsageLog struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	TenantID       uint64    `gorm:"not null;index:idx_promo_usage_audit,priority:1" json:"tenant_id"`
	PromotionID    uint64    `gorm:"not null;index:idx_promo_usage_audit,priority:2" json:"promotion_id"`
	OutletID       uint64    `gorm:"not null;index" json:"outlet_id"`
	OrderID        uint64    `gorm:"not null;index" json:"order_id"`
	CustomerID     *uint64   `gorm:"index:idx_promo_usage_audit,priority:3" json:"customer_id,omitempty"`
	DiscountAmount int64     `gorm:"not null" json:"discount_amount"` // Nominal subsidi diskon yang diterima
	UsedAt         time.Time `gorm:"not null;default:now();index:idx_promo_usage_audit,priority:4" json:"used_at"`

	// Relations
	Tenant    *Tenant    `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Promotion *Promotion `gorm:"foreignKey:PromotionID" json:"promotion,omitempty"`
	Outlet    *Outlet    `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
	Customer  *Customer  `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}

func (PromotionUsageLog) TableName() string {
	return "promotion_usage_logs"
}
