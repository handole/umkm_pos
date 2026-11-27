package pos

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines data access operations for POS shifts, payment channels, and orders.
type Repository interface {
	WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error

	// Shifts
	CreateShift(ctx context.Context, shift *model.PosShift) error
	FindOpenShiftByCashier(ctx context.Context, tenantID, cashierID uint64) (*model.PosShift, error)
	FindShiftByID(ctx context.Context, tenantID, shiftID uint64) (*model.PosShift, error)
	UpdateShift(ctx context.Context, shift *model.PosShift) error
	ListShifts(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, limit, offset int) ([]model.PosShift, int64, error)
	CreateCashLog(ctx context.Context, log *model.PosShiftCashLog) error

	// Payment Methods
	ListPaymentMethods(ctx context.Context, tenantID uint64) ([]model.PosPaymentMethod, error)
	FindPaymentMethodByCode(ctx context.Context, tenantID uint64, code string) (*model.PosPaymentMethod, error)
	SeedDefaultPaymentMethods(ctx context.Context, tenantID uint64) error

	// Orders & Checkout
	FindOrderByTenantAndIdempotency(ctx context.Context, tenantID uint64, key uuid.UUID) (*model.PosOrder, error)
	FindOrderByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PosOrder, error)
	CreateOrder(ctx context.Context, order *model.PosOrder) error
	UpdateOrder(ctx context.Context, order *model.PosOrder) error
	CreateOrderItem(ctx context.Context, item *model.PosOrderItem) error
	CreateOrderPayment(ctx context.Context, payment *model.PosOrderPayment) error
	ListOrders(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, status *string, limit, offset int) ([]model.PosOrder, int64, error)

	// Inventory & Stock Ledger integrations
	GetStockWithLock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error)
	SaveStock(ctx context.Context, stock *model.InventoryStock) error
	RecordMovement(ctx context.Context, movement *model.StockMovement) error

	// Location and Entity lookups
	FindLocationByID(ctx context.Context, tenantID, locationID uint64) (*model.InventoryLocation, error)
	FindLocationByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.InventoryLocation, error)
	FindDefaultStoreFrontLocation(ctx context.Context, tenantID, outletID uint64) (*model.InventoryLocation, error)
	FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error)
	FindUOMByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductUOM, error)
	FindOutletByID(ctx context.Context, tenantID, outletID uint64) (*model.Outlet, error)
	FindTenantByID(ctx context.Context, tenantID uint64) (*model.Tenant, error)
	FindUserByID(ctx context.Context, userID uint64) (*model.User, error)

	// Payroll Commission & Reporting integrations
	FindEmployeeByUserID(ctx context.Context, tenantID, userID uint64) (*model.Employee, error)
	CreatePendingCommission(ctx context.Context, comm *model.PayrollCommissionPending) error
	VoidPendingCommissionsByOrder(ctx context.Context, tenantID, orderID uint64) error
	ListOrdersForReport(ctx context.Context, tenantID uint64, outletID *uint64, startDate, endDate *time.Time) ([]model.PosOrder, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new instance of POS repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) WithTransaction(ctx context.Context, fn func(txRepo Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &repository{db: tx}
		return fn(txRepo)
	})
}

func (r *repository) CreateShift(ctx context.Context, shift *model.PosShift) error {
	return r.db.WithContext(ctx).Create(shift).Error
}

func (r *repository) FindOpenShiftByCashier(ctx context.Context, tenantID, cashierID uint64) (*model.PosShift, error) {
	var shift model.PosShift
	err := r.db.WithContext(ctx).
		Preload("Cashier").
		Preload("CashLogs").
		Where("tenant_id = ? AND cashier_user_id = ? AND status = 'OPEN'", tenantID, cashierID).
		First(&shift).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &shift, nil
}

func (r *repository) FindShiftByID(ctx context.Context, tenantID, shiftID uint64) (*model.PosShift, error) {
	var shift model.PosShift
	err := r.db.WithContext(ctx).
		Preload("Cashier").
		Preload("CashLogs").
		Where("tenant_id = ? AND id = ?", tenantID, shiftID).
		First(&shift).Error
	if err != nil {
		return nil, err
	}
	return &shift, nil
}

func (r *repository) UpdateShift(ctx context.Context, shift *model.PosShift) error {
	return r.db.WithContext(ctx).Save(shift).Error
}

func (r *repository) ListShifts(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, limit, offset int) ([]model.PosShift, int64, error) {
	var shifts []model.PosShift
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.PosShift{}).
		Preload("Cashier").
		Where("tenant_id = ?", tenantID)

	if outletID != nil {
		query = query.Where("outlet_id = ?", *outletID)
	}
	if cashierID != nil {
		query = query.Where("cashier_user_id = ?", *cashierID)
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
		Find(&shifts).Error

	return shifts, total, err
}

func (r *repository) CreateCashLog(ctx context.Context, log *model.PosShiftCashLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *repository) ListPaymentMethods(ctx context.Context, tenantID uint64) ([]model.PosPaymentMethod, error) {
	var methods []model.PosPaymentMethod
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("id ASC").
		Find(&methods).Error
	return methods, err
}

func (r *repository) FindPaymentMethodByCode(ctx context.Context, tenantID uint64, code string) (*model.PosPaymentMethod, error) {
	var method model.PosPaymentMethod
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ? AND is_active = ?", tenantID, code, true).
		First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (r *repository) SeedDefaultPaymentMethods(ctx context.Context, tenantID uint64) error {
	defaults := []model.PosPaymentMethod{
		{TenantID: tenantID, Code: "CASH", Name: "Uang Tunai", RequiresReference: false, IsActive: true},
		{TenantID: tenantID, Code: "QRIS", Name: "QRIS Dinamis / Statis", RequiresReference: true, IsActive: true},
		{TenantID: tenantID, Code: "DEBIT_CARD", Name: "Kartu Debit (EDC)", RequiresReference: true, IsActive: true},
		{TenantID: tenantID, Code: "CREDIT_CARD", Name: "Kartu Kredit (EDC)", RequiresReference: true, IsActive: true},
		{TenantID: tenantID, Code: "BANK_TRANSFER", Name: "Transfer Bank Langsung", RequiresReference: true, IsActive: true},
	}

	for _, d := range defaults {
		var existing model.PosPaymentMethod
		err := r.db.WithContext(ctx).Where("tenant_id = ? AND code = ?", tenantID, d.Code).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if errCreate := r.db.WithContext(ctx).Create(&d).Error; errCreate != nil {
				return errCreate
			}
		}
	}
	return nil
}

func (r *repository) FindOrderByTenantAndIdempotency(ctx context.Context, tenantID uint64, key uuid.UUID) (*model.PosOrder, error) {
	var order model.PosOrder
	err := r.db.WithContext(ctx).
		Preload("Outlet").
		Preload("Cashier").
		Preload("Items.Variant.Product").
		Preload("Items.UOM").
		Preload("Payments.PaymentMethod").
		Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *repository) FindOrderByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PosOrder, error) {
	var order model.PosOrder
	err := r.db.WithContext(ctx).
		Preload("Outlet").
		Preload("Cashier").
		Preload("Items.Variant.Product").
		Preload("Items.UOM").
		Preload("Payments.PaymentMethod").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *repository) CreateOrder(ctx context.Context, order *model.PosOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *repository) UpdateOrder(ctx context.Context, order *model.PosOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *repository) CreateOrderItem(ctx context.Context, item *model.PosOrderItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *repository) CreateOrderPayment(ctx context.Context, payment *model.PosOrderPayment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *repository) ListOrders(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, status *string, limit, offset int) ([]model.PosOrder, int64, error) {
	var orders []model.PosOrder
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.PosOrder{}).
		Preload("Outlet").
		Preload("Cashier").
		Preload("Items.Variant").
		Preload("Payments.PaymentMethod").
		Where("tenant_id = ?", tenantID)

	if outletID != nil {
		query = query.Where("outlet_id = ?", *outletID)
	}
	if cashierID != nil {
		query = query.Where("cashier_user_id = ?", *cashierID)
	}
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
		Find(&orders).Error

	return orders, total, err
}

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

func (r *repository) RecordMovement(ctx context.Context, movement *model.StockMovement) error {
	return r.db.WithContext(ctx).Create(movement).Error
}

func (r *repository) FindLocationByID(ctx context.Context, tenantID, locationID uint64) (*model.InventoryLocation, error) {
	var loc model.InventoryLocation
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, locationID).First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *repository) FindLocationByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.InventoryLocation, error) {
	var loc model.InventoryLocation
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND uuid = ?", tenantID, u).First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *repository) FindDefaultStoreFrontLocation(ctx context.Context, tenantID, outletID uint64) (*model.InventoryLocation, error) {
	var loc model.InventoryLocation
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND outlet_id = ? AND type = 'STORE_FRONT' AND is_active = ?", tenantID, outletID, true).
		First(&loc).Error
	if err == nil {
		return &loc, nil
	}

	// Fallback to warehouse or any active location in outlet
	errFallback := r.db.WithContext(ctx).
		Where("tenant_id = ? AND outlet_id = ? AND is_active = ?", tenantID, outletID, true).
		First(&loc).Error
	if errFallback != nil {
		return nil, errFallback
	}
	return &loc, nil
}

func (r *repository) FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error) {
	var variant model.ProductVariant
	err := r.db.WithContext(ctx).
		Preload("Product").
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&variant).Error
	if err != nil {
		return nil, err
	}
	return &variant, nil
}

func (r *repository) FindUOMByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductUOM, error) {
	var uom model.ProductUOM
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		First(&uom).Error
	if err != nil {
		return nil, err
	}
	return &uom, nil
}

func (r *repository) FindOutletByID(ctx context.Context, tenantID, outletID uint64) (*model.Outlet, error) {
	var outlet model.Outlet
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, outletID).
		First(&outlet).Error
	if err != nil {
		return nil, err
	}
	return &outlet, nil
}

func (r *repository) FindTenantByID(ctx context.Context, tenantID uint64) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.WithContext(ctx).
		Where("id = ?", tenantID).
		First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *repository) FindUserByID(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) FindEmployeeByUserID(ctx context.Context, tenantID, userID uint64) (*model.Employee, error) {
	var emp model.Employee
	err := r.db.WithContext(ctx).
		Preload("SalaryStructure").
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", tenantID, userID, true).
		First(&emp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

func (r *repository) CreatePendingCommission(ctx context.Context, comm *model.PayrollCommissionPending) error {
	return r.db.WithContext(ctx).Create(comm).Error
}

func (r *repository) VoidPendingCommissionsByOrder(ctx context.Context, tenantID, orderID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.PayrollCommissionPending{}).
		Where("tenant_id = ? AND order_id = ? AND status = ?", tenantID, orderID, "PENDING").
		Update("status", "VOID").Error
}

func (r *repository) ListOrdersForReport(ctx context.Context, tenantID uint64, outletID *uint64, startDate, endDate *time.Time) ([]model.PosOrder, error) {
	db := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Payments").
		Preload("Payments.PaymentMethod").
		Where("tenant_id = ?", tenantID)

	if outletID != nil {
		db = db.Where("outlet_id = ?", *outletID)
	}
	if startDate != nil {
		db = db.Where("ordered_at >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("ordered_at <= ?", *endDate)
	}

	var orders []model.PosOrder
	err := db.Order("ordered_at ASC").Find(&orders).Error
	return orders, err
}
