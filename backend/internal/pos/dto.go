package pos

import (
	"time"

	"github.com/google/uuid"
)

// OpenShiftRequest starts a new cashier shift session.
type OpenShiftRequest struct {
	OutletID     uint64 `json:"outlet_id" binding:"required"`
	StartingCash int64  `json:"starting_cash" binding:"gte=0"`
	Notes        string `json:"notes"`
}

// CloseShiftRequest finalizes an active cashier shift session.
type CloseShiftRequest struct {
	ActualCash int64  `json:"actual_cash" binding:"gte=0"`
	Notes      string `json:"notes"`
}

// CashLogRequest records cash-in or cash-out within an active shift.
type CashLogRequest struct {
	Type   string `json:"type" binding:"required,oneof=CASH_IN CASH_OUT"`
	Amount int64  `json:"amount" binding:"required,gt=0"`
	Reason string `json:"reason" binding:"required"`
}

// ShiftCashLogResponse outputs shift cash movement entries.
type ShiftCashLogResponse struct {
	ID        uint64    `json:"id"`
	UUID      uuid.UUID `json:"uuid"`
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// ShiftResponse represents cashier shift details.
type ShiftResponse struct {
	ID              uint64                 `json:"id"`
	UUID            uuid.UUID              `json:"uuid"`
	ShiftNumber     string                 `json:"shift_number"`
	OutletID        uint64                 `json:"outlet_id"`
	CashierUserID   uint64                 `json:"cashier_user_id"`
	CashierName     string                 `json:"cashier_name"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	StartingCash    int64                  `json:"starting_cash"`
	ExpectedCash    int64                  `json:"expected_cash"`
	ActualCash      *int64                 `json:"actual_cash,omitempty"`
	CashDiscrepancy *int64                 `json:"cash_discrepancy,omitempty"`
	Status          string                 `json:"status"` // OPEN, CLOSED
	Notes           string                 `json:"notes"`
	CashLogs        []ShiftCashLogResponse `json:"cash_logs,omitempty"`
}

// PaymentMethodResponse represents available payment channel.
type PaymentMethodResponse struct {
	ID                uint64    `json:"id"`
	UUID              uuid.UUID `json:"uuid"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	RequiresReference bool      `json:"requires_reference"`
	IsActive          bool      `json:"is_active"`
}

// CheckoutItemRequest represents an individual line item in cart.
type CheckoutItemRequest struct {
	VariantUUID    uuid.UUID  `json:"variant_uuid" binding:"required"`
	UOMUUID        *uuid.UUID `json:"uom_uuid,omitempty"`
	Quantity       int64      `json:"quantity" binding:"required,gt=0"`
	DiscountAmount int64      `json:"discount_amount" binding:"gte=0"`
	Notes          string     `json:"notes"`
}

// CheckoutPaymentRequest represents split payment distribution.
type CheckoutPaymentRequest struct {
	PaymentMethodCode string `json:"payment_method_code" binding:"required"`
	Amount            int64  `json:"amount" binding:"required,gt=0"`
	ReferenceNumber   string `json:"reference_number"`
}

// CheckoutOrderRequest defines payload for POS order checkout.
type CheckoutOrderRequest struct {
	IdempotencyKey      uuid.UUID                `json:"idempotency_key" binding:"required"`
	OutletID            uint64                   `json:"outlet_id" binding:"required"`
	LocationUUID        *uuid.UUID               `json:"location_uuid,omitempty"` // Gudang/rak toko asal pemotongan stok
	CustomerID          *uint64                  `json:"customer_id,omitempty"`
	SalespersonUserID   *uint64                  `json:"salesperson_user_id,omitempty"`
	DiscountAmount      int64                    `json:"discount_amount" binding:"gte=0"`
	TaxAmount           int64                    `json:"tax_amount" binding:"gte=0"`
	CouponCode          string                   `json:"coupon_code"`
	Items               []CheckoutItemRequest    `json:"items" binding:"required,min=1,dive"`
	Payments            []CheckoutPaymentRequest `json:"payments" binding:"required,min=1,dive"`
}

// OrderItemResponse represents purchased line item in order response.
type OrderItemResponse struct {
	ID            uint64    `json:"id"`
	UUID          uuid.UUID `json:"uuid"`
	VariantID     uint64    `json:"variant_id"`
	VariantSKU    string    `json:"variant_sku"`
	VariantName   string    `json:"variant_name"`
	UOMName       string    `json:"uom_name"`
	Quantity      int64     `json:"quantity"`
	BaseQuantity  int64     `json:"base_quantity"`
	UnitCostPrice int64     `json:"unit_cost_price"`
	UnitSellPrice int64     `json:"unit_sell_price"`
	Subtotal      int64     `json:"subtotal"`
	Notes         string    `json:"notes"`
}

// OrderPaymentResponse represents payment breakdown in order response.
type OrderPaymentResponse struct {
	ID                uint64    `json:"id"`
	UUID              uuid.UUID `json:"uuid"`
	PaymentMethodCode string    `json:"payment_method_code"`
	PaymentMethodName string    `json:"payment_method_name"`
	Amount            int64     `json:"amount"`
	ReferenceNumber   string    `json:"reference_number"`
	PaidAt            time.Time `json:"paid_at"`
}

// OrderResponse represents complete POS order details.
type OrderResponse struct {
	ID                      uint64                 `json:"id"`
	UUID                    uuid.UUID              `json:"uuid"`
	OrderNumber             string                 `json:"order_number"`
	IdempotencyKey          uuid.UUID              `json:"idempotency_key"`
	OutletID                uint64                 `json:"outlet_id"`
	OutletName              string                 `json:"outlet_name"`
	CashierUserID           uint64                 `json:"cashier_user_id"`
	CashierName             string                 `json:"cashier_name"`
	Subtotal                int64                  `json:"subtotal"`
	DiscountAmount          int64                  `json:"discount_amount"`
	PromotionDiscountAmount int64                  `json:"promotion_discount_amount"`
	TaxAmount               int64                  `json:"tax_amount"`
	TotalAmount             int64                  `json:"total_amount"`
	PaidAmount              int64                  `json:"paid_amount"`
	ChangeAmount            int64                  `json:"change_amount"`
	Status                  string                 `json:"status"` // COMPLETED, VOID
	OrderedAt               time.Time              `json:"ordered_at"`
	Items                   []OrderItemResponse    `json:"items"`
	Payments                []OrderPaymentResponse `json:"payments"`
}

// OrderReceiptResponse provides formatted print payload for ESC/POS thermal printers.
type OrderReceiptResponse struct {
	Header struct {
		StoreName    string `json:"store_name"`
		OutletName   string `json:"outlet_name"`
		Address      string `json:"address"`
		Phone        string `json:"phone"`
		OrderNumber  string `json:"order_number"`
		CashierName  string `json:"cashier_name"`
		DateTime     string `json:"date_time"`
	} `json:"header"`
	Lines []struct {
		Name      string `json:"name"`
		Quantity  int64  `json:"quantity"`
		UOM       string `json:"uom"`
		UnitPrice int64  `json:"unit_price"`
		Subtotal  int64  `json:"subtotal"`
	} `json:"lines"`
	Summary struct {
		Subtotal     int64 `json:"subtotal"`
		Discount     int64 `json:"discount"`
		Tax          int64 `json:"tax"`
		Total        int64 `json:"total"`
		Paid         int64 `json:"paid"`
		Change       int64 `json:"change"`
	} `json:"summary"`
	Payments []struct {
		Method string `json:"method"`
		Amount int64  `json:"amount"`
		Ref    string `json:"ref,omitempty"`
	} `json:"payments"`
	Footer struct {
		Notes string `json:"notes"`
	} `json:"footer"`
}

// VoidOrderRequest allows authorized manager to void a transaction.
type VoidOrderRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// SalesReportQuery filters financial & sales reports.
type SalesReportQuery struct {
	OutletID  *uint64 `form:"outlet_id"`
	StartDate string  `form:"start_date"` // YYYY-MM-DD
	EndDate   string  `form:"end_date"`   // YYYY-MM-DD
}

// PaymentBreakdownItem summarizes sales per payment method.
type PaymentBreakdownItem struct {
	MethodCode string `json:"method_code"`
	Total      int64  `json:"total"`
	Count      int64  `json:"count"`
}

// SalesReportSummaryResponse provides aggregated revenue, COGS, and gross profit.
type SalesReportSummaryResponse struct {
	StartDate        string                 `json:"start_date"`
	EndDate          string                 `json:"end_date"`
	TotalOrders      int64                  `json:"total_orders"`
	CompletedOrders  int64                  `json:"completed_orders"`
	VoidOrders       int64                  `json:"void_orders"`
	GrossSales       int64                  `json:"gross_sales"`
	TotalDiscount    int64                  `json:"total_discount"`
	TotalTax         int64                  `json:"total_tax"`
	NetSales         int64                  `json:"net_sales"`
	TotalCOGS        int64                  `json:"total_cogs"`
	GrossProfit      int64                  `json:"gross_profit"`
	PaymentBreakdown []PaymentBreakdownItem `json:"payment_breakdown"`
}
