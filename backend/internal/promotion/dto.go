package promotion

import (
	"time"

	"github.com/google/uuid"
)

// CreateCustomerRequest payload to register a customer.
type CreateCustomerRequest struct {
	Code       string `json:"code" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Email      string `json:"email"`
	MemberTier string `json:"member_tier"` // REGULAR, SILVER, GOLD, PLATINUM
}

// UpdateCustomerRequest payload to edit customer data.
type UpdateCustomerRequest struct {
	Name       *string `json:"name,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Email      *string `json:"email,omitempty"`
	MemberTier *string `json:"member_tier,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
}

// CustomerResponse represents customer record output.
type CustomerResponse struct {
	ID         uint64    `json:"id"`
	UUID       uuid.UUID `json:"uuid"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Phone      string    `json:"phone"`
	Email      string    `json:"email"`
	MemberTier string    `json:"member_tier"`
	TotalSpend int64     `json:"total_spend"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreatePromotionRuleRequest fine-tunes targeting parameters.
type CreatePromotionRuleRequest struct {
	RuleType        string     `json:"rule_type" binding:"required,oneof=INCLUDE_CATEGORY EXCLUDE_CATEGORY INCLUDE_PRODUCT EXCLUDE_PRODUCT BUY_X_GET_Y"`
	CategoryUUID    *uuid.UUID `json:"category_uuid,omitempty"`
	VariantUUID     *uuid.UUID `json:"variant_uuid,omitempty"`
	BuyVariantUUID  *uuid.UUID `json:"buy_variant_uuid,omitempty"`
	BuyQuantity     *int       `json:"buy_quantity,omitempty"`
	RewardVariantUUID *uuid.UUID `json:"reward_variant_uuid,omitempty"`
	RewardQuantity  *int       `json:"reward_quantity,omitempty"`
}

// CreatePromotionRequest payload for creating marketing promotions.
type CreatePromotionRequest struct {
	Code              string                       `json:"code" binding:"required"`
	Name              string                       `json:"name" binding:"required"`
	Description       string                       `json:"description"`
	PromoType         string                       `json:"promo_type" binding:"required,oneof=AUTOMATIC COUPON_CODE"`
	DiscountType      string                       `json:"discount_type" binding:"required,oneof=PERCENTAGE FIXED_AMOUNT SPECIAL_PRICE"`
	DiscountValue     int64                        `json:"discount_value" binding:"required,gt=0"`
	MaxDiscountAmount *int64                       `json:"max_discount_amount,omitempty"`
	MinOrderAmount    int64                        `json:"min_order_amount" binding:"gte=0"`
	MinQuantity       int64                        `json:"min_quantity" binding:"gte=1"`
	TargetScope       string                       `json:"target_scope" binding:"required,oneof=ENTIRE_TRANSACTION SPECIFIC_PRODUCTS SPECIFIC_CATEGORIES"`
	StartDate         time.Time                    `json:"start_date" binding:"required"`
	EndDate           time.Time                    `json:"end_date" binding:"required"`
	StartTime         *string                      `json:"start_time,omitempty"` // '14:00:00'
	EndTime           *string                      `json:"end_time,omitempty"`   // '17:00:00'
	DaysOfWeek        string                       `json:"days_of_week"`         // 'ALL' or '1,2,3,4,5'
	DaysOfMonth       string                       `json:"days_of_month"`        // 'ALL' or '25,26,27,28,29,30'
	ApplicableMonths  string                       `json:"applicable_months"`    // 'ALL' or '3,4'
	TotalQuota        *int                         `json:"total_quota,omitempty"`
	DailyQuota        *int                         `json:"daily_quota,omitempty"`
	PerCustomerQuota  int                          `json:"per_customer_quota"`
	TotalBudget       *int64                       `json:"total_budget,omitempty"`
	IsStackable       bool                         `json:"is_stackable"`
	OutletIDs         []uint64                     `json:"outlet_ids,omitempty"`
	Rules             []CreatePromotionRuleRequest `json:"rules,omitempty"`
}

// UpdatePromotionRequest modifies existing promotion parameters.
type UpdatePromotionRequest struct {
	Name              *string    `json:"name,omitempty"`
	Description       *string    `json:"description,omitempty"`
	DiscountValue     *int64     `json:"discount_value,omitempty"`
	MaxDiscountAmount *int64     `json:"max_discount_amount,omitempty"`
	MinOrderAmount    *int64     `json:"min_order_amount,omitempty"`
	MinQuantity       *int64     `json:"min_quantity,omitempty"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	StartTime         *string    `json:"start_time,omitempty"`
	EndTime           *string    `json:"end_time,omitempty"`
	DaysOfWeek        *string    `json:"days_of_week,omitempty"`
	DaysOfMonth       *string    `json:"days_of_month,omitempty"`
	ApplicableMonths  *string    `json:"applicable_months,omitempty"`
	TotalQuota        *int       `json:"total_quota,omitempty"`
	DailyQuota        *int       `json:"daily_quota,omitempty"`
	PerCustomerQuota  *int       `json:"per_customer_quota,omitempty"`
	TotalBudget       *int64     `json:"total_budget,omitempty"`
	IsActive          *bool      `json:"is_active,omitempty"`
	IsStackable       *bool      `json:"is_stackable,omitempty"`
}

// PromotionRuleResponse details an attached promotion targeting rule.
type PromotionRuleResponse struct {
	ID              uint64    `json:"id"`
	UUID            uuid.UUID `json:"uuid"`
	RuleType        string    `json:"rule_type"`
	CategoryID      *uint64   `json:"category_id,omitempty"`
	CategoryName    string    `json:"category_name,omitempty"`
	VariantID       *uint64   `json:"variant_id,omitempty"`
	VariantSKU      string    `json:"variant_sku,omitempty"`
	VariantName     string    `json:"variant_name,omitempty"`
	BuyQuantity     *int      `json:"buy_quantity,omitempty"`
	RewardQuantity  *int      `json:"reward_quantity,omitempty"`
}

// PromotionResponse output schema for campaigns.
type PromotionResponse struct {
	ID                 uint64                  `json:"id"`
	UUID               uuid.UUID               `json:"uuid"`
	Code               string                  `json:"code"`
	Name               string                  `json:"name"`
	Description        string                  `json:"description"`
	PromoType          string                  `json:"promo_type"`
	DiscountType       string                  `json:"discount_type"`
	DiscountValue      int64                   `json:"discount_value"`
	MaxDiscountAmount  *int64                  `json:"max_discount_amount,omitempty"`
	MinOrderAmount     int64                   `json:"min_order_amount"`
	MinQuantity        int64                   `json:"min_quantity"`
	TargetScope        string                  `json:"target_scope"`
	StartDate          time.Time               `json:"start_date"`
	EndDate            time.Time               `json:"end_date"`
	StartTime          *string                 `json:"start_time,omitempty"`
	EndTime            *string                 `json:"end_time,omitempty"`
	DaysOfWeek         string                  `json:"days_of_week"`
	DaysOfMonth        string                  `json:"days_of_month"`
	ApplicableMonths   string                  `json:"applicable_months"`
	TotalQuota         *int                    `json:"total_quota,omitempty"`
	DailyQuota         *int                    `json:"daily_quota,omitempty"`
	PerCustomerQuota   int                     `json:"per_customer_quota"`
	CurrentUsageCount  int                     `json:"current_usage_count"`
	TotalBudget        *int64                  `json:"total_budget,omitempty"`
	CurrentSpendAmount int64                   `json:"current_spend_amount"`
	IsStackable        bool                    `json:"is_stackable"`
	IsActive           bool                    `json:"is_active"`
	CreatedAt          time.Time               `json:"created_at"`
	Rules              []PromotionRuleResponse `json:"rules,omitempty"`
}

// CartItemDTO represents an item in shopping cart for promo evaluation.
type CartItemDTO struct {
	VariantUUID uuid.UUID `json:"variant_uuid" binding:"required"`
	CategoryID  *uint64   `json:"category_id,omitempty"`
	Quantity    int64     `json:"quantity" binding:"required,gt=0"`
	UnitPrice   int64     `json:"unit_price" binding:"required,gte=0"`
}

// EvaluatePromotionRequest evaluates applicable discounts on shopping cart.
type EvaluatePromotionRequest struct {
	OutletID        uint64        `json:"outlet_id" binding:"required"`
	CustomerUUID    *uuid.UUID    `json:"customer_uuid,omitempty"`
	CouponCode      string        `json:"coupon_code"`
	TransactionTime *time.Time    `json:"transaction_time,omitempty"`
	Items           []CartItemDTO `json:"items" binding:"required,min=1,dive"`
}

// EvaluatePromotionResponse returns best eligible promotion calculation.
type EvaluatePromotionResponse struct {
	IsEligible        bool               `json:"is_eligible"`
	Message           string             `json:"message"`
	AppliedPromotion  *PromotionResponse `json:"applied_promotion,omitempty"`
	OriginalSubtotal  int64              `json:"original_subtotal"`
	DiscountAmount    int64              `json:"discount_amount"`
	FinalTotal        int64              `json:"final_total"`
}
