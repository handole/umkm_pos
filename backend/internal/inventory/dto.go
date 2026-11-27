package inventory

import (
	"time"

	"github.com/google/uuid"
)

// CreateLocationRequest creates a new warehouse, shelf, or storage location.
type CreateLocationRequest struct {
	OutletID uint64 `json:"outlet_id" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required,oneof=WAREHOUSE STORE_FRONT TRANSIT DAMAGED"`
}

// LocationResponse outputs storage location details.
type LocationResponse struct {
	ID        uint64    `json:"id"`
	UUID      uuid.UUID `json:"uuid"`
	OutletID  uint64    `json:"outlet_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// StockResponse outputs current real-time stock balance per location and variant.
type StockResponse struct {
	ID           uint64    `json:"id"`
	UUID         uuid.UUID `json:"uuid"`
	LocationID   uint64    `json:"location_id"`
	LocationName string    `json:"location_name"`
	VariantID    uint64    `json:"variant_id"`
	VariantSKU   string    `json:"variant_sku"`
	VariantName  string    `json:"variant_name"`
	ProductName  string    `json:"product_name"`
	Quantity     int64     `json:"quantity"` // Dalam satuan terkecil (base unit)
	UpdatedAt    time.Time `json:"updated_at"`
}

// StockMovementFilterQuery handles filtering for stock ledger records.
type StockMovementFilterQuery struct {
	LocationID   *uint64 `form:"location_id"`
	VariantID    *uint64 `form:"variant_id"`
	MovementType *string `form:"movement_type"`
	Limit        int     `form:"limit,default=20"`
	Offset       int     `form:"offset,default=0"`
}

// StockMovementResponse represents an immutable ledger audit entry.
type StockMovementResponse struct {
	ID              uint64    `json:"id"`
	UUID            uuid.UUID `json:"uuid"`
	LocationID      uint64    `json:"location_id"`
	LocationName    string    `json:"location_name"`
	VariantID       uint64    `json:"variant_id"`
	VariantSKU      string    `json:"variant_sku"`
	VariantName     string    `json:"variant_name"`
	MovementType    string    `json:"movement_type"`
	QuantityChange  int64     `json:"quantity_change"`
	BalanceAfter    int64     `json:"balance_after"`
	CostPrice       int64     `json:"cost_price"`
	ReferenceNumber string    `json:"reference_number"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateStockTransferItemRequest represents a line item in transfer request.
type CreateStockTransferItemRequest struct {
	VariantUUID  uuid.UUID `json:"variant_uuid" binding:"required"`
	QuantitySent int64     `json:"quantity_sent" binding:"required,gt=0"`
}

// CreateStockTransferRequest submits a new inventory transfer between locations.
type CreateStockTransferRequest struct {
	SourceLocationUUID      uuid.UUID                        `json:"source_location_uuid" binding:"required"`
	DestinationLocationUUID uuid.UUID                        `json:"destination_location_uuid" binding:"required"`
	Notes                   string                           `json:"notes"`
	Items                   []CreateStockTransferItemRequest `json:"items" binding:"required,min=1,dive"`
}

// StockTransferItemResponse outputs transfer line item details.
type StockTransferItemResponse struct {
	ID               uint64    `json:"id"`
	UUID             uuid.UUID `json:"uuid"`
	VariantID        uint64    `json:"variant_id"`
	VariantSKU       string    `json:"variant_sku"`
	VariantName      string    `json:"variant_name"`
	QuantitySent     int64     `json:"quantity_sent"`
	QuantityReceived int64     `json:"quantity_received"`
}

// StockTransferResponse outputs full transfer record.
type StockTransferResponse struct {
	ID                      uint64                      `json:"id"`
	UUID                    uuid.UUID                   `json:"uuid"`
	TransferNumber          string                      `json:"transfer_number"`
	SourceLocationID        uint64                      `json:"source_location_id"`
	SourceLocationName      string                      `json:"source_location_name"`
	DestinationLocationID   uint64                      `json:"destination_location_id"`
	DestinationLocationName string                      `json:"destination_location_name"`
	Status                  string                      `json:"status"` // PENDING, IN_TRANSIT, RECEIVED, REJECTED
	Notes                   string                      `json:"notes"`
	CreatedAt               time.Time                   `json:"created_at"`
	Items                   []StockTransferItemResponse `json:"items"`
}

// ReceiveTransferItemRequest represents receiving quantities per item.
type ReceiveTransferItemRequest struct {
	VariantUUID      uuid.UUID `json:"variant_uuid" binding:"required"`
	QuantityReceived int64     `json:"quantity_received" binding:"required,gte=0"`
}

// ReceiveTransferRequest confirms arrival of inventory at destination.
type ReceiveTransferRequest struct {
	Items []ReceiveTransferItemRequest `json:"items" binding:"required,min=1,dive"`
}

// CreateStockOpnameRequest starts an inventory audit session.
type CreateStockOpnameRequest struct {
	LocationUUID uuid.UUID `json:"location_uuid" binding:"required"`
	Notes        string    `json:"notes"`
}

// ScanOpnameBarcodeRequest accepts barcode scans during physical audit.
type ScanOpnameBarcodeRequest struct {
	Barcode  string `json:"barcode" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,gt=0"`
	DeviceID string `json:"device_id"`
}

// StockOpnameScanResponse logs individual scan events.
type StockOpnameScanResponse struct {
	ID                  uint64    `json:"id"`
	UUID                uuid.UUID `json:"uuid"`
	ScannedBarcode      string    `json:"scanned_barcode"`
	VariantSKU          string    `json:"variant_sku"`
	VariantName         string    `json:"variant_name"`
	QuantityCounted     int64     `json:"quantity_counted"`
	BaseQuantityCounted int64     `json:"base_quantity_counted"`
	ScannedAt           time.Time `json:"scanned_at"`
	DeviceID            string    `json:"device_id"`
}

// StockOpnameItemResponse shows physical vs system discrepancy.
type StockOpnameItemResponse struct {
	ID                 uint64    `json:"id"`
	UUID               uuid.UUID `json:"uuid"`
	VariantID          uint64    `json:"variant_id"`
	VariantSKU         string    `json:"variant_sku"`
	VariantName        string    `json:"variant_name"`
	SystemQuantity     int64     `json:"system_quantity"`
	PhysicalQuantity   int64     `json:"physical_quantity"`
	DifferenceQuantity int64     `json:"difference_quantity"` // physical - system
	CostPrice          int64     `json:"cost_price"`
	Notes              string    `json:"notes"`
}

// StockOpnameResponse outputs complete stock opname document.
type StockOpnameResponse struct {
	ID           uint64                    `json:"id"`
	UUID         uuid.UUID                 `json:"uuid"`
	OpnameNumber string                    `json:"opname_number"`
	LocationID   uint64                    `json:"location_id"`
	LocationName string                    `json:"location_name"`
	Status       string                    `json:"status"` // DRAFT, SUBMITTED, APPROVED, REJECTED
	Notes        string                    `json:"notes"`
	CreatedAt    time.Time                 `json:"created_at"`
	Items        []StockOpnameItemResponse `json:"items"`
	Scans        []StockOpnameScanResponse `json:"scans,omitempty"`
}

// ManualAdjustmentRequest applies an immediate manual inventory adjustment.
type ManualAdjustmentRequest struct {
	LocationUUID   uuid.UUID `json:"location_uuid" binding:"required"`
	VariantUUID    uuid.UUID `json:"variant_uuid" binding:"required"`
	QuantityChange int64     `json:"quantity_change" binding:"required,ne=0"` // Positive for addition, negative for deduction
	CostPrice      int64     `json:"cost_price" binding:"gte=0"`
	Notes          string    `json:"notes" binding:"required"`
}

// CreateSupplierRequest represents payload to register a supplier.
type CreateSupplierRequest struct {
	Code        string `json:"code" binding:"required,max=50"`
	Name        string `json:"name" binding:"required,max=255"`
	ContactName string `json:"contact_name" binding:"max=255"`
	Phone       string `json:"phone" binding:"max=30"`
	Email       string `json:"email" binding:"max=255"`
	Address     string `json:"address"`
}

// CreatePurchaseOrderItemRequest represents a line item in a new Purchase Order.
type CreatePurchaseOrderItemRequest struct {
	VariantUUID     uuid.UUID  `json:"variant_uuid" binding:"required"`
	UOMUUID         *uuid.UUID `json:"uom_uuid,omitempty"`
	QuantityOrdered int64      `json:"quantity_ordered" binding:"required,gt=0"`
	UnitCost        int64      `json:"unit_cost" binding:"required,gte=0"`
}

// CreatePurchaseOrderRequest represents payload to create a Purchase Order.
type CreatePurchaseOrderRequest struct {
	OutletID                uint64                           `json:"outlet_id" binding:"required"`
	SupplierUUID            uuid.UUID                        `json:"supplier_uuid" binding:"required"`
	DestinationLocationUUID uuid.UUID                        `json:"destination_location_uuid" binding:"required"`
	OrderDate               string                           `json:"order_date" binding:"required"` // YYYY-MM-DD
	ExpectedDate            string                           `json:"expected_date"`               // YYYY-MM-DD
	Notes                   string                           `json:"notes"`
	Items                   []CreatePurchaseOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// ReceivePurchaseOrderItemRequest specifies received quantity for a PO line.
type ReceivePurchaseOrderItemRequest struct {
	ItemUUID         uuid.UUID `json:"item_uuid" binding:"required"`
	QuantityReceived int64     `json:"quantity_received" binding:"required,gt=0"`
}

// ReceivePurchaseOrderRequest records incoming goods from a supplier into stock.
type ReceivePurchaseOrderRequest struct {
	Items []ReceivePurchaseOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}
