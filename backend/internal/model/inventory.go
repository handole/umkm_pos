package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Supplier represents an external vendor for purchasing inventory.
type Supplier struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_suppliers_tenant_uuid,priority:2" json:"uuid"`
	TenantID      uint64         `gorm:"not null;uniqueIndex:idx_suppliers_tenant_uuid,priority:1;uniqueIndex:idx_suppliers_tenant_code,priority:1" json:"tenant_id"`
	Code          string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_suppliers_tenant_code,priority:2" json:"code"`
	Name          string         `gorm:"type:varchar(255);not null" json:"name"`
	ContactPerson string         `gorm:"type:varchar(100)" json:"contact_person"`
	Phone         string         `gorm:"type:varchar(30)" json:"phone"`
	Email         string         `gorm:"type:varchar(255)" json:"email"`
	Address       string         `gorm:"type:text" json:"address"`
	IsActive      bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (Supplier) TableName() string {
	return "suppliers"
}

// InventoryStock stores real-time stock balance per physical location and product variant.
type InventoryStock struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_stock_tenant_uuid,priority:2" json:"uuid"`
	TenantID   uint64    `gorm:"not null;uniqueIndex:idx_stock_tenant_uuid,priority:1;uniqueIndex:idx_stock_location_variant,priority:1" json:"tenant_id"`
	LocationID uint64    `gorm:"not null;uniqueIndex:idx_stock_location_variant,priority:2" json:"location_id"`
	VariantID  uint64    `gorm:"not null;uniqueIndex:idx_stock_location_variant,priority:3" json:"variant_id"`
	Quantity   int64     `gorm:"not null;default:0" json:"quantity"` // Dalam satuan terkecil / base unit
	UpdatedAt  time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant   *Tenant            `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Location *InventoryLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Variant  *ProductVariant    `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
}

func (InventoryStock) TableName() string {
	return "inventory_stocks"
}

// StockMovement represents the immutable ledger of every inventory mutation.
type StockMovement struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_movements_tenant_uuid,priority:2" json:"uuid"`
	TenantID        uint64    `gorm:"not null;uniqueIndex:idx_movements_tenant_uuid,priority:1;index:idx_movements_audit,priority:1" json:"tenant_id"`
	LocationID      uint64    `gorm:"not null;index:idx_movements_audit,priority:2" json:"location_id"`
	VariantID       uint64    `gorm:"not null;index:idx_movements_audit,priority:3" json:"variant_id"`
	MovementType    string    `gorm:"type:varchar(40);not null" json:"movement_type"` // 'IN_PURCHASE_ORDER', 'OUT_POS_SALE', 'TRANSFER_OUT', 'TRANSFER_IN', 'ADJUSTMENT_OPNAME', 'RETURN_SALE', 'RETURN_SUPPLIER'
	QuantityChange  int64     `gorm:"not null" json:"quantity_change"`                // Positif jika masuk, negatif jika keluar
	BalanceAfter    int64     `gorm:"not null" json:"balance_after"`                  // Saldo stok sesaat setelah baris ini dicatat
	CostPrice       int64     `gorm:"not null" json:"cost_price"`                     // HPP unit saat mutasi
	ReferenceID     *uint64   `gorm:"index" json:"reference_id,omitempty"`
	ReferenceUUID   *uuid.UUID `gorm:"type:uuid;index" json:"reference_uuid,omitempty"`
	ReferenceNumber string    `gorm:"type:varchar(100)" json:"reference_number"` // Nomor Struk / No PO / No Opname
	Notes           string    `gorm:"type:text" json:"notes"`
	CreatedBy       *uint64   `gorm:"index" json:"created_by,omitempty"`
	CreatedAt       time.Time `gorm:"not null;default:now();index:idx_movements_audit,priority:4" json:"created_at"`

	// Relations
	Tenant   *Tenant            `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Location *InventoryLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Variant  *ProductVariant    `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
	User     *User              `gorm:"foreignKey:CreatedBy" json:"user,omitempty"`
}

func (StockMovement) TableName() string {
	return "stock_movements"
}

// PurchaseOrder represents procurement from suppliers.
type PurchaseOrder struct {
	ID                    uint64              `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                  uuid.UUID           `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_po_tenant_uuid,priority:2" json:"uuid"`
	TenantID              uint64              `gorm:"not null;uniqueIndex:idx_po_tenant_uuid,priority:1;uniqueIndex:idx_po_tenant_number,priority:1" json:"tenant_id"`
	OutletID              uint64              `gorm:"not null;index" json:"outlet_id"`
	SupplierID            uint64              `gorm:"not null;index" json:"supplier_id"`
	DestinationLocationID uint64              `gorm:"not null;index" json:"destination_location_id"`
	PONumber              string              `gorm:"type:varchar(100);not null;uniqueIndex:idx_po_tenant_number,priority:2" json:"po_number"`
	Status                string              `gorm:"type:varchar(30);not null;default:'DRAFT';index" json:"status"` // 'DRAFT', 'SUBMITTED', 'PARTIAL_RECEIVED', 'RECEIVED', 'CANCELLED'
	OrderDate             time.Time           `gorm:"type:date;not null" json:"order_date"`
	ExpectedDate          *time.Time          `gorm:"type:date" json:"expected_date,omitempty"`
	TotalAmount           int64               `gorm:"not null;default:0" json:"total_amount"` // Total nilai belanja (BIGINT)
	Notes                 string              `gorm:"type:text" json:"notes"`
	CreatedBy             uint64              `gorm:"not null;index" json:"created_by"`
	ReceivedBy            *uint64             `gorm:"index" json:"received_by,omitempty"`
	CreatedAt             time.Time           `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt             time.Time           `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant              *Tenant             `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Outlet              *Outlet             `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
	Supplier            *Supplier           `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	DestinationLocation *InventoryLocation  `gorm:"foreignKey:DestinationLocationID" json:"destination_location,omitempty"`
	Items               []PurchaseOrderItem `gorm:"foreignKey:POID" json:"items,omitempty"`
	Creator             *User               `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Receiver            *User               `gorm:"foreignKey:ReceivedBy" json:"receiver,omitempty"`
}

func (PurchaseOrder) TableName() string {
	return "purchase_orders"
}

// PurchaseOrderItem represents individual lines within a Purchase Order.
type PurchaseOrderItem struct {
	ID               uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID             uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	POID             uint64          `gorm:"not null;index" json:"po_id"`
	VariantID        uint64          `gorm:"not null;index" json:"variant_id"`
	UOMID            *uint64         `gorm:"index" json:"uom_id,omitempty"`
	QuantityOrdered  int64           `gorm:"not null" json:"quantity_ordered"`
	QuantityReceived int64           `gorm:"not null;default:0" json:"quantity_received"`
	UnitCost         int64           `gorm:"not null" json:"unit_cost"`
	Subtotal         int64           `gorm:"not null" json:"subtotal"`

	// Relations
	PurchaseOrder *PurchaseOrder  `gorm:"foreignKey:POID" json:"purchase_order,omitempty"`
	Variant       *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
	UOM           *ProductUOM     `gorm:"foreignKey:UOMID" json:"uom,omitempty"`
}

func (PurchaseOrderItem) TableName() string {
	return "purchase_order_items"
}

// StockTransfer represents relocation of inventory between physical locations.
type StockTransfer struct {
	ID                    uint64              `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                  uuid.UUID           `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_transfers_tenant_uuid,priority:2" json:"uuid"`
	TenantID              uint64              `gorm:"not null;uniqueIndex:idx_transfers_tenant_uuid,priority:1;uniqueIndex:idx_transfers_tenant_number,priority:1" json:"tenant_id"`
	TransferNumber        string              `gorm:"type:varchar(100);not null;uniqueIndex:idx_transfers_tenant_number,priority:2" json:"transfer_number"`
	SourceLocationID      uint64              `gorm:"not null;index" json:"source_location_id"`
	DestinationLocationID uint64              `gorm:"not null;index" json:"destination_location_id"`
	Status                string              `gorm:"type:varchar(30);not null;default:'PENDING'" json:"status"` // 'PENDING', 'IN_TRANSIT', 'RECEIVED', 'REJECTED'
	CreatedBy             uint64              `gorm:"not null;index" json:"created_by"`
	ReceivedBy            *uint64             `gorm:"index" json:"received_by,omitempty"`
	Notes                 string              `gorm:"type:text" json:"notes"`
	CreatedAt             time.Time           `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt             time.Time           `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant              *Tenant             `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	SourceLocation      *InventoryLocation  `gorm:"foreignKey:SourceLocationID" json:"source_location,omitempty"`
	DestinationLocation *InventoryLocation  `gorm:"foreignKey:DestinationLocationID" json:"destination_location,omitempty"`
	Items               []StockTransferItem `gorm:"foreignKey:TransferID" json:"items,omitempty"`
}

func (StockTransfer) TableName() string {
	return "stock_transfers"
}

// StockTransferItem represents line items in a transfer.
type StockTransferItem struct {
	ID               uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID             uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	TransferID       uint64          `gorm:"not null;index" json:"transfer_id"`
	VariantID        uint64          `gorm:"not null;index" json:"variant_id"`
	QuantitySent     int64           `gorm:"not null" json:"quantity_sent"`
	QuantityReceived int64           `gorm:"not null;default:0" json:"quantity_received"`

	// Relations
	Transfer *StockTransfer  `gorm:"foreignKey:TransferID" json:"transfer,omitempty"`
	Variant  *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
}

func (StockTransferItem) TableName() string {
	return "stock_transfer_items"
}

// StockOpname represents physical stock auditing.
type StockOpname struct {
	ID           uint64             `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID         uuid.UUID          `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_opname_tenant_uuid,priority:2" json:"uuid"`
	TenantID     uint64             `gorm:"not null;uniqueIndex:idx_opname_tenant_uuid,priority:1;uniqueIndex:idx_opname_tenant_number,priority:1" json:"tenant_id"`
	LocationID   uint64             `gorm:"not null;index" json:"location_id"`
	OpnameNumber string             `gorm:"type:varchar(100);not null;uniqueIndex:idx_opname_tenant_number,priority:2" json:"opname_number"`
	Status       string             `gorm:"type:varchar(30);not null;default:'DRAFT'" json:"status"` // 'DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED'
	PerformedBy  uint64             `gorm:"not null;index" json:"performed_by"`
	ApprovedBy   *uint64            `gorm:"index" json:"approved_by,omitempty"`
	Notes        string             `gorm:"type:text" json:"notes"`
	CreatedAt    time.Time          `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time          `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant    *Tenant            `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Location  *InventoryLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Items     []StockOpnameItem  `gorm:"foreignKey:OpnameID" json:"items,omitempty"`
	Scans     []StockOpnameScan  `gorm:"foreignKey:OpnameID" json:"scans,omitempty"`
	Inspector *User              `gorm:"foreignKey:PerformedBy" json:"inspector,omitempty"`
	Approver  *User              `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}

func (StockOpname) TableName() string {
	return "stock_opnames"
}

// StockOpnameItem represents discrepancy calculation lines in a stock audit.
type StockOpnameItem struct {
	ID                 uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID               uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	OpnameID           uint64          `gorm:"not null;index" json:"opname_id"`
	VariantID          uint64          `gorm:"not null;index" json:"variant_id"`
	SystemQuantity     int64           `gorm:"not null" json:"system_quantity"`
	PhysicalQuantity   int64           `gorm:"not null" json:"physical_quantity"`
	DifferenceQuantity int64           `gorm:"not null" json:"difference_quantity"` // physical - system
	CostPrice          int64           `gorm:"not null" json:"cost_price"`
	Notes              string          `gorm:"type:text" json:"notes"`

	// Relations
	Opname  *StockOpname    `gorm:"foreignKey:OpnameID" json:"opname,omitempty"`
	Variant *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
}

func (StockOpnameItem) TableName() string {
	return "stock_opname_items"
}

// StockOpnameScan records individual barcode scan events during stock audit.
type StockOpnameScan struct {
	ID                  uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	OpnameID            uint64          `gorm:"not null;index" json:"opname_id"`
	VariantID           uint64          `gorm:"not null;index" json:"variant_id"`
	UOMID               *uint64         `gorm:"index" json:"uom_id,omitempty"`
	ScannedBarcode      string          `gorm:"type:varchar(64);not null" json:"scanned_barcode"`
	QuantityCounted     int64           `gorm:"not null" json:"quantity_counted"`      // Hitungan pada UOM terpindai
	BaseQuantityCounted int64           `gorm:"not null" json:"base_quantity_counted"`  // Dikonversi ke satuan terkecil
	ScannedBy           uint64          `gorm:"not null;index" json:"scanned_by"`
	ScannedAt           time.Time       `gorm:"not null;default:now();index" json:"scanned_at"`
	DeviceID            string          `gorm:"type:varchar(100)" json:"device_id"`

	// Relations
	Opname  *StockOpname    `gorm:"foreignKey:OpnameID" json:"opname,omitempty"`
	Variant *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
	UOM     *ProductUOM     `gorm:"foreignKey:UOMID" json:"uom,omitempty"`
	Scanner *User           `gorm:"foreignKey:ScannedBy" json:"scanner,omitempty"`
}

func (StockOpnameScan) TableName() string {
	return "stock_opname_scans"
}
