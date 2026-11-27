package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category organizes products in hierarchical parent-child structures.
type Category struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID      uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_categories_tenant_uuid,priority:2" json:"uuid"`
	TenantID  uint64    `gorm:"not null;uniqueIndex:idx_categories_tenant_uuid,priority:1;uniqueIndex:idx_categories_tenant_slug,priority:1" json:"tenant_id"`
	ParentID  *uint64   `gorm:"index" json:"parent_id,omitempty"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Slug      string    `gorm:"type:varchar(120);not null;uniqueIndex:idx_categories_tenant_slug,priority:2" json:"slug"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant   *Tenant    `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

func (Category) TableName() string {
	return "categories"
}

// Product represents a master catalog item.
type Product struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_products_tenant_uuid,priority:2" json:"uuid"`
	TenantID    uint64         `gorm:"not null;uniqueIndex:idx_products_tenant_uuid,priority:1" json:"tenant_id"`
	CategoryID  *uint64        `gorm:"index" json:"category_id,omitempty"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	ImageURL    string         `gorm:"type:varchar(500)" json:"image_url"`
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant   *Tenant          `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Category *Category        `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
}

func (Product) TableName() string {
	return "products"
}

// ProductVariant represents a distinct sellable/stockable SKU (e.g., Size, Color, Volume).
type ProductVariant struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_variants_tenant_uuid,priority:2" json:"uuid"`
	TenantID      uint64         `gorm:"not null;uniqueIndex:idx_variants_tenant_uuid,priority:1;uniqueIndex:idx_variants_tenant_sku,priority:1;uniqueIndex:idx_variants_tenant_barcode,priority:1" json:"tenant_id"`
	ProductID     uint64         `gorm:"not null;index" json:"product_id"`
	SKU           string         `gorm:"type:varchar(64);not null;uniqueIndex:idx_variants_tenant_sku,priority:2" json:"sku"`
	Barcode       string         `gorm:"type:varchar(64);uniqueIndex:idx_variants_tenant_barcode,priority:2" json:"barcode"` // Retail EAN-13 / Code-128 barcode
	VariantName   string         `gorm:"type:varchar(150);not null" json:"variant_name"`                                     // 'Standard', 'Red - XL', '500ml'
	BaseCostPrice int64          `gorm:"not null;default:0" json:"base_cost_price"`                                          // HPP satuan terkecil (BIGINT)
	BaseSellPrice int64          `gorm:"not null;default:0" json:"base_sell_price"`                                          // Harga jual satuan terkecil (BIGINT)
	MinStockAlert int64          `gorm:"not null;default:5" json:"min_stock_alert"`
	IsActive      bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant      *Tenant              `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Product     *Product             `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	UOMs        []ProductUOM         `gorm:"foreignKey:VariantID" json:"uoms,omitempty"`
	CostHistory []ProductCostHistory `gorm:"foreignKey:VariantID" json:"cost_history,omitempty"`
	Stocks      []InventoryStock     `gorm:"foreignKey:VariantID" json:"stocks,omitempty"`
}

func (ProductVariant) TableName() string {
	return "product_variants"
}

// ProductUOM represents multi-level unit of measurement (Pcs, Box, Carton) and its package barcode.
type ProductUOM struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_product_uoms_tenant_uuid,priority:2" json:"uuid"`
	TenantID       uint64    `gorm:"not null;uniqueIndex:idx_product_uoms_tenant_uuid,priority:1;index" json:"tenant_id"`
	VariantID      uint64    `gorm:"not null;uniqueIndex:idx_product_uoms_variant_name,priority:1" json:"variant_id"`
	UOMName        string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_product_uoms_variant_name,priority:2" json:"uom_name"` // 'Pcs', 'Dus', 'Karton'
	ConversionRate int       `gorm:"not null;default:1" json:"conversion_rate"`                                                       // Ratio to base unit (e.g. 1 Dus = 12 Pcs)
	SellPrice      int64     `gorm:"not null" json:"sell_price"`                                                                      // Price for this packaging unit (BIGINT)
	Barcode        string    `gorm:"type:varchar(64);index" json:"barcode"`                                                           // Outer box barcode (ITF-14 / Code-128)
	IsBaseUOM      bool      `gorm:"not null;default:false" json:"is_base_uom"`
	CreatedAt      time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Variant *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
}

func (ProductUOM) TableName() string {
	return "product_uoms"
}

// ProductCostHistory records purchasing and HPP recalculation history (Moving Average).
type ProductCostHistory struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_cost_history_tenant_uuid,priority:2" json:"uuid"`
	TenantID          uint64    `gorm:"not null;uniqueIndex:idx_cost_history_tenant_uuid,priority:1;index" json:"tenant_id"`
	VariantID         uint64    `gorm:"not null;index" json:"variant_id"`
	ReferenceType     string    `gorm:"type:varchar(50);not null" json:"reference_type"` // 'PURCHASE_ORDER', 'INITIAL_IMPORT', 'MANUAL_ADJUSTMENT'
	ReferenceID       *uint64   `gorm:"index" json:"reference_id,omitempty"`
	PurchasePrice     int64     `gorm:"not null" json:"purchase_price"` // Harga beli per unit dasar
	PreviousCostPrice int64     `gorm:"not null" json:"previous_cost_price"`
	NewCostPrice      int64     `gorm:"not null" json:"new_cost_price"` // HPP moving average baru
	RecordedAt        time.Time `gorm:"not null;default:now();index" json:"recorded_at"`

	// Relations
	Variant *ProductVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
}

func (ProductCostHistory) TableName() string {
	return "product_cost_history"
}
