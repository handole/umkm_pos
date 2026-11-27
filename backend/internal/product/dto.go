package product

import (
	"time"

	"github.com/google/uuid"
)

// CreateCategoryRequest payload to create a new category.
type CreateCategoryRequest struct {
	Name     string  `json:"name" binding:"required"`
	ParentID *uint64 `json:"parent_id,omitempty"`
}

// UpdateCategoryRequest payload to edit a category.
type UpdateCategoryRequest struct {
	Name     *string `json:"name,omitempty"`
	ParentID *uint64 `json:"parent_id,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// CategoryResponse represents category data returned to client.
type CategoryResponse struct {
	ID        uint64             `json:"id"`
	UUID      uuid.UUID          `json:"uuid"`
	ParentID  *uint64            `json:"parent_id,omitempty"`
	Name      string             `json:"name"`
	Slug      string             `json:"slug"`
	IsActive  bool               `json:"is_active"`
	CreatedAt time.Time          `json:"created_at"`
	Children  []CategoryResponse `json:"children,omitempty"`
}

// CreateUOMRequest creates packaging unit of measure.
type CreateUOMRequest struct {
	UOMName        string `json:"uom_name" binding:"required"` // 'Pcs', 'Dus', 'Karton'
	ConversionRate int    `json:"conversion_rate" binding:"required,gte=1"` // 1 Dus = 12 Pcs
	SellPrice      int64  `json:"sell_price" binding:"required,gte=0"`
	Barcode        string `json:"barcode"` // Barcode pada kemasan kardus
	IsBaseUOM      bool   `json:"is_base_uom"`
}

// CreateVariantRequest creates SKU variant under a product.
type CreateVariantRequest struct {
	SKU           string             `json:"sku" binding:"required"`
	Barcode       string             `json:"barcode"` // Retail EAN-13 barcode
	VariantName   string             `json:"variant_name" binding:"required"` // 'Standard', 'Merah - L', '500ml'
	BaseCostPrice int64              `json:"base_cost_price" binding:"gte=0"`
	BaseSellPrice int64              `json:"base_sell_price" binding:"required,gte=0"`
	MinStockAlert int64              `json:"min_stock_alert"`
	UOMs          []CreateUOMRequest `json:"uoms"`
}

// CreateProductRequest creates a master product with initial variants and UOMs.
type CreateProductRequest struct {
	Name        string                 `json:"name" binding:"required"`
	CategoryID  *uint64                `json:"category_id,omitempty"`
	Description string                 `json:"description"`
	ImageURL    string                 `json:"image_url"`
	Variants    []CreateVariantRequest `json:"variants" binding:"required,min=1,dive"`
}

// UpdateProductRequest updates master product details.
type UpdateProductRequest struct {
	Name        *string `json:"name,omitempty"`
	CategoryID  *uint64 `json:"category_id,omitempty"`
	Description *string `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// ProductUOMResponse output for packaging unit.
type ProductUOMResponse struct {
	ID             uint64    `json:"id"`
	UUID           uuid.UUID `json:"uuid"`
	UOMName        string    `json:"uom_name"`
	ConversionRate int       `json:"conversion_rate"`
	SellPrice      int64     `json:"sell_price"`
	Barcode        string    `json:"barcode,omitempty"`
	IsBaseUOM      bool      `json:"is_base_uom"`
}

// ProductVariantResponse output for a SKU variant.
type ProductVariantResponse struct {
	ID            uint64               `json:"id"`
	UUID          uuid.UUID            `json:"uuid"`
	SKU           string               `json:"sku"`
	Barcode       string               `json:"barcode,omitempty"`
	VariantName   string               `json:"variant_name"`
	BaseCostPrice int64                `json:"base_cost_price"`
	BaseSellPrice int64                `json:"base_sell_price"`
	MinStockAlert int64                `json:"min_stock_alert"`
	IsActive      bool                 `json:"is_active"`
	UOMs          []ProductUOMResponse `json:"uoms,omitempty"`
}

// ProductResponse output for master product.
type ProductResponse struct {
	ID          uint64                   `json:"id"`
	UUID        uuid.UUID                `json:"uuid"`
	CategoryID  *uint64                  `json:"category_id,omitempty"`
	Category    *CategoryResponse        `json:"category,omitempty"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	ImageURL    string                   `json:"image_url,omitempty"`
	IsActive    bool                     `json:"is_active"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	Variants    []ProductVariantResponse `json:"variants,omitempty"`
}

// ProductQueryFilter parameters for product catalog list.
type ProductQueryFilter struct {
	Page       int     `form:"page,default=1"`
	Limit      int     `form:"limit,default=20"`
	Search     string  `form:"search"`
	CategoryID *uint64 `form:"category_id"`
	IsActive   *bool   `form:"is_active"`
}

// BarcodeScanResponse optimized output for POS cashier and warehouse barcode scanner.
type BarcodeScanResponse struct {
	Product            ProductResponse         `json:"product"`
	Variant            ProductVariantResponse  `json:"variant"`
	MatchedUOM         *ProductUOMResponse     `json:"matched_uom,omitempty"`
	ScanType           string                  `json:"scan_type"` // 'BASE_VARIANT' or 'PACKAGING_UOM'
	CalculatedPrice    int64                   `json:"calculated_price"`
	BaseUnitMultiplier int                     `json:"base_unit_multiplier"` // e.g. 1 if Pcs, 12 if Dus
}
