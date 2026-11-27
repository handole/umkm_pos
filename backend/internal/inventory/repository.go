package inventory

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines data access operations for inventory management.
type Repository interface {
	WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error

	// Locations
	CreateLocation(ctx context.Context, loc *model.InventoryLocation) error
	ListLocations(ctx context.Context, tenantID uint64, outletID *uint64) ([]model.InventoryLocation, error)
	FindLocationByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.InventoryLocation, error)
	FindLocationByID(ctx context.Context, tenantID uint64, id uint64) (*model.InventoryLocation, error)

	// Stock Balances
	GetStock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error)
	GetStockWithLock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error)
	SaveStock(ctx context.Context, stock *model.InventoryStock) error
	ListStocks(ctx context.Context, tenantID uint64, locationID *uint64, lowStockOnly bool, limit, offset int) ([]model.InventoryStock, int64, error)

	// Stock Movement Ledger
	RecordMovement(ctx context.Context, m *model.StockMovement) error
	ListMovements(ctx context.Context, tenantID uint64, filter StockMovementFilterQuery) ([]model.StockMovement, int64, error)

	// Stock Transfers
	CreateTransfer(ctx context.Context, t *model.StockTransfer) error
	FindTransferByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.StockTransfer, error)
	UpdateTransfer(ctx context.Context, t *model.StockTransfer) error
	CreateTransferItem(ctx context.Context, item *model.StockTransferItem) error
	UpdateTransferItem(ctx context.Context, item *model.StockTransferItem) error
	ListTransfers(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.StockTransfer, int64, error)

	// Stock Opname
	CreateOpname(ctx context.Context, o *model.StockOpname) error
	FindOpnameByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.StockOpname, error)
	UpdateOpname(ctx context.Context, o *model.StockOpname) error
	ListOpnames(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.StockOpname, int64, error)
	RecordOpnameScan(ctx context.Context, s *model.StockOpnameScan) error
	FindOpnameItem(ctx context.Context, opnameID, variantID uint64) (*model.StockOpnameItem, error)
	SaveOpnameItem(ctx context.Context, item *model.StockOpnameItem) error

	// Product Variant and Barcode Lookups
	FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error)
	FindBarcode(ctx context.Context, tenantID uint64, barcode string) (*model.ProductVariant, *model.ProductUOM, int, error)

	// Suppliers & Purchase Orders
	CreateSupplier(ctx context.Context, s *model.Supplier) error
	ListSuppliers(ctx context.Context, tenantID uint64) ([]model.Supplier, error)
	FindSupplierByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Supplier, error)
	CreatePurchaseOrder(ctx context.Context, po *model.PurchaseOrder) error
	FindPurchaseOrderByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PurchaseOrder, error)
	ListPurchaseOrders(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.PurchaseOrder, int64, error)
	UpdatePurchaseOrder(ctx context.Context, po *model.PurchaseOrder) error
	UpdatePurchaseOrderItem(ctx context.Context, item *model.PurchaseOrderItem) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new instance of inventory Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &repository{db: tx}
		return fn(txRepo)
	})
}

func (r *repository) CreateLocation(ctx context.Context, loc *model.InventoryLocation) error {
	return r.db.WithContext(ctx).Create(loc).Error
}

func (r *repository) ListLocations(ctx context.Context, tenantID uint64, outletID *uint64) ([]model.InventoryLocation, error) {
	var locations []model.InventoryLocation
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if outletID != nil {
		query = query.Where("outlet_id = ?", *outletID)
	}
	err := query.Order("id ASC").Find(&locations).Error
	return locations, err
}

func (r *repository) FindLocationByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.InventoryLocation, error) {
	var loc model.InventoryLocation
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *repository) FindLocationByID(ctx context.Context, tenantID uint64, id uint64) (*model.InventoryLocation, error) {
	var loc model.InventoryLocation
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *repository) GetStock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error) {
	var stock model.InventoryStock
	err := r.db.WithContext(ctx).
		Preload("Variant").
		Preload("Location").
		Where("tenant_id = ? AND location_id = ? AND variant_id = ?", tenantID, locationID, variantID).
		First(&stock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &stock, nil
}

// GetStockWithLock queries the stock record with pessimistic row-level locking (SELECT ... FOR UPDATE).
func (r *repository) GetStockWithLock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error) {
	var stock model.InventoryStock
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND location_id = ? AND variant_id = ?", tenantID, locationID, variantID).
		First(&stock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newStock := model.InventoryStock{
				TenantID:   tenantID,
				LocationID: locationID,
				VariantID:  variantID,
				Quantity:   0,
			}
			if errCreate := r.db.WithContext(ctx).Create(&newStock).Error; errCreate != nil {
				return nil, errCreate
			}
			return &newStock, nil
		}
		return nil, err
	}
	return &stock, nil
}

func (r *repository) SaveStock(ctx context.Context, stock *model.InventoryStock) error {
	return r.db.WithContext(ctx).Save(stock).Error
}

func (r *repository) ListStocks(ctx context.Context, tenantID uint64, locationID *uint64, lowStockOnly bool, limit, offset int) ([]model.InventoryStock, int64, error) {
	var stocks []model.InventoryStock
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.InventoryStock{}).
		Preload("Location").
		Preload("Variant.Product").
		Where("inventory_stocks.tenant_id = ?", tenantID)

	if locationID != nil {
		query = query.Where("location_id = ?", *locationID)
	}

	if lowStockOnly {
		query = query.Joins("JOIN product_variants ON product_variants.id = inventory_stocks.variant_id").
			Where("inventory_stocks.quantity <= product_variants.min_stock_alert")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("inventory_stocks.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&stocks).Error

	return stocks, total, err
}

func (r *repository) RecordMovement(ctx context.Context, m *model.StockMovement) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *repository) ListMovements(ctx context.Context, tenantID uint64, filter StockMovementFilterQuery) ([]model.StockMovement, int64, error) {
	var movements []model.StockMovement
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.StockMovement{}).
		Preload("Location").
		Preload("Variant").
		Where("stock_movements.tenant_id = ?", tenantID)

	if filter.LocationID != nil {
		query = query.Where("location_id = ?", *filter.LocationID)
	}
	if filter.VariantID != nil {
		query = query.Where("variant_id = ?", *filter.VariantID)
	}
	if filter.MovementType != nil && *filter.MovementType != "" {
		query = query.Where("movement_type = ?", *filter.MovementType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	err := query.Order("id DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&movements).Error

	return movements, total, err
}

func (r *repository) CreateTransfer(ctx context.Context, t *model.StockTransfer) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *repository) FindTransferByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.StockTransfer, error) {
	var transfer model.StockTransfer
	err := r.db.WithContext(ctx).
		Preload("SourceLocation").
		Preload("DestinationLocation").
		Preload("Items.Variant").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&transfer).Error
	if err != nil {
		return nil, err
	}
	return &transfer, nil
}

func (r *repository) UpdateTransfer(ctx context.Context, t *model.StockTransfer) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *repository) CreateTransferItem(ctx context.Context, item *model.StockTransferItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *repository) UpdateTransferItem(ctx context.Context, item *model.StockTransferItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *repository) ListTransfers(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.StockTransfer, int64, error) {
	var transfers []model.StockTransfer
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.StockTransfer{}).
		Preload("SourceLocation").
		Preload("DestinationLocation").
		Preload("Items.Variant").
		Where("tenant_id = ?", tenantID)

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	err := query.Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&transfers).Error

	return transfers, total, err
}

func (r *repository) CreateOpname(ctx context.Context, o *model.StockOpname) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *repository) FindOpnameByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.StockOpname, error) {
	var opname model.StockOpname
	err := r.db.WithContext(ctx).
		Preload("Location").
		Preload("Items.Variant").
		Preload("Scans.Variant").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&opname).Error
	if err != nil {
		return nil, err
	}
	return &opname, nil
}

func (r *repository) UpdateOpname(ctx context.Context, o *model.StockOpname) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *repository) ListOpnames(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.StockOpname, int64, error) {
	var opnames []model.StockOpname
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.StockOpname{}).
		Preload("Location").
		Where("tenant_id = ?", tenantID)

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	err := query.Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&opnames).Error

	return opnames, total, err
}

func (r *repository) RecordOpnameScan(ctx context.Context, s *model.StockOpnameScan) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *repository) FindOpnameItem(ctx context.Context, opnameID, variantID uint64) (*model.StockOpnameItem, error) {
	var item model.StockOpnameItem
	err := r.db.WithContext(ctx).
		Where("opname_id = ? AND variant_id = ?", opnameID, variantID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) SaveOpnameItem(ctx context.Context, item *model.StockOpnameItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *repository) FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error) {
	var variant model.ProductVariant
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&variant).Error
	if err != nil {
		return nil, err
	}
	return &variant, nil
}

// FindBarcode searches dual barcodes: retail unit or packaging carton box.
func (r *repository) FindBarcode(ctx context.Context, tenantID uint64, barcode string) (*model.ProductVariant, *model.ProductUOM, int, error) {
	var variant model.ProductVariant
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND barcode = ? AND is_active = ?", tenantID, barcode, true).
		First(&variant).Error
	if err == nil {
		return &variant, nil, 1, nil
	}

	var uom model.ProductUOM
	errUOM := r.db.WithContext(ctx).
		Preload("Variant").
		Where("tenant_id = ? AND barcode = ?", tenantID, barcode).
		First(&uom).Error
	if errUOM == nil && uom.Variant != nil {
		rate := uom.ConversionRate
		if rate <= 0 {
			rate = 1
		}
		return uom.Variant, &uom, rate, nil
	}

	return nil, nil, 0, gorm.ErrRecordNotFound
}

func (r *repository) CreateSupplier(ctx context.Context, s *model.Supplier) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *repository) ListSuppliers(ctx context.Context, tenantID uint64) ([]model.Supplier, error) {
	var suppliers []model.Supplier
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("name ASC").Find(&suppliers).Error
	return suppliers, err
}

func (r *repository) FindSupplierByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Supplier, error) {
	var s model.Supplier
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) CreatePurchaseOrder(ctx context.Context, po *model.PurchaseOrder) error {
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *repository) FindPurchaseOrderByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	err := r.db.WithContext(ctx).
		Preload("Supplier").
		Preload("DestinationLocation").
		Preload("Items").
		Preload("Items.Variant").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&po).Error
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *repository) ListPurchaseOrders(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.PurchaseOrder, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.PurchaseOrder{}).Where("tenant_id = ?", tenantID)
	if status != nil && *status != "" {
		db = db.Where("status = ?", *status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var pos []model.PurchaseOrder
	err := db.Preload("Supplier").Preload("DestinationLocation").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&pos).Error
	return pos, total, err
}

func (r *repository) UpdatePurchaseOrder(ctx context.Context, po *model.PurchaseOrder) error {
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *repository) UpdatePurchaseOrderItem(ctx context.Context, item *model.PurchaseOrderItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}
