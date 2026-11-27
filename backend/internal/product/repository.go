package product

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/gorm"
)

// Repository defines database operations for Categories, Products, Variants, and Multi-UOM.
type Repository interface {
	// Category methods
	ListCategories(ctx context.Context, tenantID uint64) ([]model.Category, error)
	GetCategoryByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Category, error)
	GetCategoryByID(ctx context.Context, tenantID uint64, id uint64) (*model.Category, error)
	CreateCategory(ctx context.Context, category *model.Category) error
	UpdateCategory(ctx context.Context, category *model.Category) error
	DeleteCategory(ctx context.Context, tenantID uint64, categoryID uint64) error

	// Product methods
	ListProducts(ctx context.Context, tenantID uint64, filter ProductQueryFilter) ([]model.Product, int64, error)
	GetProductByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Product, error)
	CreateProductWithVariants(ctx context.Context, product *model.Product) error
	UpdateProduct(ctx context.Context, product *model.Product) error
	DeleteProduct(ctx context.Context, tenantID uint64, productUUID uuid.UUID) error
	FindVariantBySKU(ctx context.Context, tenantID uint64, sku string) (*model.ProductVariant, error)
	FindByBarcode(ctx context.Context, tenantID uint64, barcode string) (*model.ProductVariant, *model.ProductUOM, *model.Product, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a product domain repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) ListCategories(ctx context.Context, tenantID uint64) ([]model.Category, error) {
	var categories []model.Category
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND parent_id IS NULL", tenantID).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("name ASC")
		}).
		Order("name ASC").
		Find(&categories).Error
	return categories, err
}

func (r *repository) GetCategoryByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Category, error) {
	var cat model.Category
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		Preload("Children").
		First(&cat).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) GetCategoryByID(ctx context.Context, tenantID uint64, id uint64) (*model.Category, error) {
	var cat model.Category
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&cat).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) CreateCategory(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *repository) UpdateCategory(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

func (r *repository) DeleteCategory(ctx context.Context, tenantID uint64, categoryID uint64) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, categoryID).
		Delete(&model.Category{}).Error
}

func (r *repository) ListProducts(ctx context.Context, tenantID uint64, filter ProductQueryFilter) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Product{}).
		Where("tenant_id = ?", tenantID)

	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != "" {
		searchWildcard := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchWildcard, searchWildcard)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := (filter.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	err := query.
		Preload("Category").
		Preload("Variants.UOMs").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&products).Error

	return products, total, err
}

func (r *repository) GetProductByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Product, error) {
	var p model.Product
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, u).
		Preload("Category").
		Preload("Variants.UOMs").
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) CreateProductWithVariants(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create product header (omit associations so we set TenantID/ProductID explicitly below)
		if err := tx.Omit("Variants").Create(product).Error; err != nil {
			return err
		}

		// 2. Iterate variants
		for i := range product.Variants {
			product.Variants[i].ProductID = product.ID
			product.Variants[i].TenantID = product.TenantID
			if err := tx.Omit("UOMs").Create(&product.Variants[i]).Error; err != nil {
				return err
			}

			// 3. Iterate UOMs
			for j := range product.Variants[i].UOMs {
				product.Variants[i].UOMs[j].VariantID = product.Variants[i].ID
				product.Variants[i].UOMs[j].TenantID = product.TenantID
				if err := tx.Create(&product.Variants[i].UOMs[j]).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *repository) UpdateProduct(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *repository) DeleteProduct(ctx context.Context, tenantID uint64, productUUID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND uuid = ?", tenantID, productUUID).
		Delete(&model.Product{}).Error
}

func (r *repository) FindVariantBySKU(ctx context.Context, tenantID uint64, sku string) (*model.ProductVariant, error) {
	var variant model.ProductVariant
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND sku = ?", tenantID, sku).
		First(&variant).Error
	if err != nil {
		return nil, err
	}
	return &variant, nil
}

// FindByBarcode performs instant barcode matching against retail variant barcode or packaging UOM barcode.
func (r *repository) FindByBarcode(ctx context.Context, tenantID uint64, barcode string) (*model.ProductVariant, *model.ProductUOM, *model.Product, error) {
	if barcode == "" {
		return nil, nil, nil, errors.New("empty barcode")
	}

	// 1. Check if matches retail base variant barcode
	var variant model.ProductVariant
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND barcode = ? AND is_active = true", tenantID, barcode).
		Preload("Product.Category").
		Preload("UOMs").
		First(&variant).Error

	if err == nil {
		// Matched directly to base variant barcode (Pcs)
		var baseUOM *model.ProductUOM
		for _, u := range variant.UOMs {
			if u.IsBaseUOM {
				baseUOM = &u
				break
			}
		}
		return &variant, baseUOM, variant.Product, nil
	}

	// 2. If not matched on variant, check if matches packaging unit barcode (Dus / Karton)
	var uom model.ProductUOM
	err = r.db.WithContext(ctx).
		Where("tenant_id = ? AND barcode = ?", tenantID, barcode).
		First(&uom).Error

	if err == nil {
		// Load variant and product
		var parentVariant model.ProductVariant
		if err := r.db.WithContext(ctx).
			Where("id = ? AND is_active = true", uom.VariantID).
			Preload("Product.Category").
			Preload("UOMs").
			First(&parentVariant).Error; err == nil {
			return &parentVariant, &uom, parentVariant.Product, nil
		}
	}

	return nil, nil, nil, gorm.ErrRecordNotFound
}
