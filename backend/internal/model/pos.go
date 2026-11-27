package model

import (
	"time"

	"github.com/google/uuid"
)

// PosShift tracks cash drawer balances from shift opening to closing.
type PosShift struct {
	ID              uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID            uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_pos_shifts_tenant_uuid,priority:2" json:"uuid"`
	TenantID        uint64            `gorm:"not null;uniqueIndex:idx_pos_shifts_tenant_uuid,priority:1;uniqueIndex:idx_pos_shifts_tenant_number,priority:1" json:"tenant_id"`
	OutletID        uint64            `gorm:"not null;index" json:"outlet_id"`
	CashierUserID   uint64            `gorm:"not null;index:idx_shifts_cashier,priority:2" json:"cashier_user_id"`
	ShiftNumber     string            `gorm:"type:varchar(100);not null;uniqueIndex:idx_pos_shifts_tenant_number,priority:2" json:"shift_number"`
	StartTime       time.Time         `gorm:"not null;default:now()" json:"start_time"`
	EndTime         *time.Time        `json:"end_time,omitempty"`
	StartingCash    int64             `gorm:"not null;default:0" json:"starting_cash"` // Modal uang laci awal (BIGINT)
	ExpectedCash    int64             `gorm:"not null;default:0" json:"expected_cash"` // Saldo hitung sistem
	ActualCash      *int64            `json:"actual_cash,omitempty"`                  // Hitungan fisik saat tutup
	CashDiscrepancy *int64            `json:"cash_discrepancy,omitempty"`             // Selisih fisik vs sistem
	Status          string            `gorm:"type:varchar(20);not null;default:'OPEN';index:idx_shifts_cashier,priority:3" json:"status"` // 'OPEN', 'CLOSED'
	Notes           string            `gorm:"type:text" json:"notes"`
	CreatedAt       time.Time         `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant   *Tenant            `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Outlet   *Outlet            `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
	Cashier  *User              `gorm:"foreignKey:CashierUserID" json:"cashier,omitempty"`
	CashLogs []PosShiftCashLog  `gorm:"foreignKey:ShiftID" json:"cash_logs,omitempty"`
	Orders   []PosOrder         `gorm:"foreignKey:ShiftID" json:"orders,omitempty"`
}

func (PosShift) TableName() string {
	return "pos_shifts"
}

// PosShiftCashLog records petty cash in / cash out during an open shift.
type PosShiftCashLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID      uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	ShiftID   uint64    `gorm:"not null;index" json:"shift_id"`
	Type      string    `gorm:"type:varchar(20);not null" json:"type"` // 'CASH_IN', 'CASH_OUT'
	Amount    int64     `gorm:"not null" json:"amount"`
	Reason    string    `gorm:"type:varchar(255);not null" json:"reason"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Relations
	Shift *PosShift `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
}

func (PosShiftCashLog) TableName() string {
	return "pos_shift_cash_logs"
}

// PosPaymentMethod defines accepted payment channels (Cash, QRIS, EDC Debit/Credit).
type PosPaymentMethod struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_pm_tenant_uuid,priority:2" json:"uuid"`
	TenantID          uint64    `gorm:"not null;uniqueIndex:idx_pm_tenant_uuid,priority:1;uniqueIndex:idx_pm_tenant_code,priority:1" json:"tenant_id"`
	Code              string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_pm_tenant_code,priority:2" json:"code"` // 'CASH', 'QRIS', 'DEBIT_CARD', 'CREDIT_CARD', 'BANK_TRANSFER'
	Name              string    `gorm:"type:varchar(100);not null" json:"name"`
	RequiresReference bool      `gorm:"not null;default:false" json:"requires_reference"`
	IsActive          bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt         time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (PosPaymentMethod) TableName() string {
	return "pos_payment_methods"
}

// PosOrder represents a completed checkout transaction.
type PosOrder struct {
	ID                      uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                    uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_pos_orders_tenant_uuid,priority:2" json:"uuid"`
	TenantID                uint64            `gorm:"not null;uniqueIndex:idx_pos_orders_tenant_uuid,priority:1;uniqueIndex:idx_pos_orders_tenant_number,priority:1;uniqueIndex:idx_pos_orders_tenant_idempotency,priority:1" json:"tenant_id"`
	OutletID                uint64            `gorm:"not null;index:idx_pos_orders_lookup,priority:1" json:"outlet_id"`
	ShiftID                 *uint64           `gorm:"index" json:"shift_id,omitempty"`
	CashierUserID           uint64            `gorm:"not null;index" json:"cashier_user_id"`
	SalespersonUserID       *uint64           `gorm:"index:idx_pos_orders_salesperson,priority:2" json:"salesperson_user_id,omitempty"` // For payroll sales commissions
	CustomerID              *uint64           `gorm:"index:idx_pos_orders_customer,priority:2" json:"customer_id,omitempty"`
	PromotionID             *uint64           `gorm:"index:idx_pos_orders_promotion,priority:2" json:"promotion_id,omitempty"`
	CouponCode              string            `gorm:"type:varchar(50)" json:"coupon_code"`
	LocationID              uint64            `gorm:"not null;index" json:"location_id"` // Physical location where stock is deducted
	OrderNumber             string            `gorm:"type:varchar(100);not null;uniqueIndex:idx_pos_orders_tenant_number,priority:2" json:"order_number"`
	IdempotencyKey          uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex:idx_pos_orders_tenant_idempotency,priority:2" json:"idempotency_key"`
	Subtotal                int64             `gorm:"not null" json:"subtotal"`
	DiscountAmount          int64             `gorm:"not null;default:0" json:"discount_amount"`
	PromotionDiscountAmount int64             `gorm:"not null;default:0" json:"promotion_discount_amount"`
	TaxAmount               int64             `gorm:"not null;default:0" json:"tax_amount"`
	TotalAmount             int64             `gorm:"not null" json:"total_amount"`
	PaidAmount              int64             `gorm:"not null" json:"paid_amount"`
	ChangeAmount            int64             `gorm:"not null;default:0" json:"change_amount"`
	Status                  string            `gorm:"type:varchar(30);not null;default:'COMPLETED'" json:"status"` // 'COMPLETED', 'VOID', 'REFUNDED'
	OrderedAt               time.Time         `gorm:"not null;default:now();index:idx_pos_orders_lookup,priority:2;index:idx_pos_orders_salesperson,priority:3" json:"ordered_at"`
	CreatedAt               time.Time         `gorm:"not null;default:now()" json:"created_at"`

	// Relations
	Tenant      *Tenant           `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Outlet      *Outlet           `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
	Shift       *PosShift         `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
	Cashier     *User             `gorm:"foreignKey:CashierUserID" json:"cashier,omitempty"`
	Salesperson *User             `gorm:"foreignKey:SalespersonUserID" json:"salesperson,omitempty"`
	Customer    *Customer         `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Promotion   *Promotion        `gorm:"foreignKey:PromotionID" json:"promotion,omitempty"`
	Location    *InventoryLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Items       []PosOrderItem    `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Payments    []PosOrderPayment `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
}

func (PosOrder) TableName() string {
	return "pos_orders"
}

// PosOrderItem represents individual items inside a POS transaction.
type PosOrderItem struct {
	ID                      uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                    uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	OrderID                 uint64          `gorm:"not null;index" json:"order_id"`
	VariantID               uint64          `gorm:"not null;index" json:"variant_id"`
	UOMID                   *uint64         `gorm:"index" json:"uom_id,omitempty"`
	PromotionID             *uint64         `gorm:"index" json:"promotion_id,omitempty"`
	Quantity                int64           `gorm:"not null" json:"quantity"`
	BaseQuantity            int64           `gorm:"not null" json:"base_quantity"`      // Konversi ke base unit untuk kartu stok
	UnitCostPrice           int64           `gorm:"not null" json:"unit_cost_price"`    // HPP saat transaksi
	UnitSellPrice           int64           `gorm:"not null" json:"unit_sell_price"`
	DiscountAmount          int64           `gorm:"not null;default:0" json:"discount_amount"`
	PromotionDiscountAmount int64           `gorm:"not null;default:0" json:"promotion_discount_amount"`
	Subtotal                int64           `gorm:"not null" json:"subtotal"`
	Notes                   string          `gorm:"type:text" json:"notes"`

	// Relations
	Order     *PosOrder       `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Variant   *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
	UOM       *ProductUOM     `gorm:"foreignKey:UOMID" json:"uom,omitempty"`
	Promotion *Promotion      `gorm:"foreignKey:PromotionID" json:"promotion,omitempty"`
}

func (PosOrderItem) TableName() string {
	return "pos_order_items"
}

// PosOrderPayment supports split payment configurations.
type PosOrderPayment struct {
	ID              uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID            uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	OrderID         uint64            `gorm:"not null;index" json:"order_id"`
	PaymentMethodID uint64            `gorm:"not null;index" json:"payment_method_id"`
	Amount          int64             `gorm:"not null" json:"amount"`
	ReferenceNumber string            `gorm:"type:varchar(100)" json:"reference_number"` // Trace number EDC / Ref QRIS
	PaidAt          time.Time         `gorm:"not null;default:now()" json:"paid_at"`

	// Relations
	Order         *PosOrder         `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	PaymentMethod *PosPaymentMethod `gorm:"foreignKey:PaymentMethodID" json:"payment_method,omitempty"`
}

func (PosOrderPayment) TableName() string {
	return "pos_order_payments"
}
