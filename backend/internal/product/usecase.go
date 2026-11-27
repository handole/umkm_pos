package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrProductNotFound  = errors.New("product not found")
	ErrVariantNotFound  = errors.New("product variant not found")
	ErrSKUAlreadyExists = errors.New("sku is already in use by another variant")
	ErrBarcodeNotFound  = errors.New("barcode is not registered in this tenant")
)

// Usecase defines business logic for product catalog.
type Usecase interface {
	// Category operations
	ListCategories(ctx context.Context, tenantID uint64) ([]CategoryResponse, error)
	CreateCategory(ctx context.Context, tenantID uint64, req CreateCategoryRequest) (*CategoryResponse, error)
	UpdateCategory(ctx context.Context, tenantID uint64, catUUID uuid.UUID, req UpdateCategoryRequest) (*CategoryResponse, error)
	DeleteCategory(ctx context.Context, tenantID uint64, catUUID uuid.UUID) error

	// Product operations
	ListProducts(ctx context.Context, tenantID uint64, filter ProductQueryFilter) ([]ProductResponse, int64, error)
	GetProductByUUID(ctx context.Context, tenantID uint64, productUUID uuid.UUID) (*ProductResponse, error)
	CreateProduct(ctx context.Context, tenantID uint64, req CreateProductRequest) (*ProductResponse, error)
	UpdateProduct(ctx context.Context, tenantID uint64, productUUID uuid.UUID, req UpdateProductRequest) (*ProductResponse, error)
	DeleteProduct(ctx context.Context, tenantID uint64, productUUID uuid.UUID) error

	// Barcode Scanner
	ScanBarcode(ctx context.Context, tenantID uint64, barcode string) (*BarcodeScanResponse, error)
}

type usecase struct {
	repo Repository
}

// NewUsecase instantiates a product domain usecase.
func NewUsecase(repo Repository) Usecase {
	return &usecase{repo: repo}
}

func slugify(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug := reg.ReplaceAllString(lower, "-")
	return strings.Trim(slug, "-")
}

func (u *usecase) ListCategories(ctx context.Context, tenantID uint64) ([]CategoryResponse, error) {
	categories, err := u.repo.ListCategories(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var responses []CategoryResponse
	for _, c := range categories {
		var childrenResponses []CategoryResponse
		for _, ch := range c.Children {
			childrenResponses = append(childrenResponses, CategoryResponse{
				ID:        ch.ID,
				UUID:      ch.UUID,
				ParentID:  ch.ParentID,
				Name:      ch.Name,
				Slug:      ch.Slug,
				IsActive:  ch.IsActive,
				CreatedAt: ch.CreatedAt,
			})
		}

		responses = append(responses, CategoryResponse{
			ID:        c.ID,
			UUID:      c.UUID,
			ParentID:  c.ParentID,
			Name:      c.Name,
			Slug:      c.Slug,
			IsActive:  c.IsActive,
			CreatedAt: c.CreatedAt,
			Children:  childrenResponses,
		})
	}

	return responses, nil
}

func (u *usecase) CreateCategory(ctx context.Context, tenantID uint64, req CreateCategoryRequest) (*CategoryResponse, error) {
	if req.ParentID != nil && *req.ParentID > 0 {
		if _, err := u.repo.GetCategoryByID(ctx, tenantID, *req.ParentID); err != nil {
			return nil, errors.New("parent category does not exist")
		}
	}

	slug := slugify(req.Name)
	if slug == "" {
		slug = "category-" + uuid.New().String()[:8]
	}

	cat := &model.Category{
		TenantID: tenantID,
		ParentID: req.ParentID,
		Name:     req.Name,
		Slug:     slug,
		IsActive: true,
	}

	if err := u.repo.CreateCategory(ctx, cat); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return &CategoryResponse{
		ID:        cat.ID,
		UUID:      cat.UUID,
		ParentID:  cat.ParentID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		IsActive:  cat.IsActive,
		CreatedAt: cat.CreatedAt,
	}, nil
}

func (u *usecase) UpdateCategory(ctx context.Context, tenantID uint64, catUUID uuid.UUID, req UpdateCategoryRequest) (*CategoryResponse, error) {
	cat, err := u.repo.GetCategoryByUUID(ctx, tenantID, catUUID)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	if req.Name != nil && *req.Name != "" {
		cat.Name = *req.Name
		cat.Slug = slugify(*req.Name)
	}
	if req.ParentID != nil {
		cat.ParentID = req.ParentID
	}
	if req.IsActive != nil {
		cat.IsActive = *req.IsActive
	}

	if err := u.repo.UpdateCategory(ctx, cat); err != nil {
		return nil, err
	}

	return &CategoryResponse{
		ID:        cat.ID,
		UUID:      cat.UUID,
		ParentID:  cat.ParentID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		IsActive:  cat.IsActive,
		CreatedAt: cat.CreatedAt,
	}, nil
}

func (u *usecase) DeleteCategory(ctx context.Context, tenantID uint64, catUUID uuid.UUID) error {
	cat, err := u.repo.GetCategoryByUUID(ctx, tenantID, catUUID)
	if err != nil {
		return ErrCategoryNotFound
	}

	return u.repo.DeleteCategory(ctx, tenantID, cat.ID)
}

func mapProductModelToDTO(p *model.Product) ProductResponse {
	var categoryResp *CategoryResponse
	if p.Category != nil {
		categoryResp = &CategoryResponse{
			ID:       p.Category.ID,
			UUID:     p.Category.UUID,
			ParentID: p.Category.ParentID,
			Name:     p.Category.Name,
			Slug:     p.Category.Slug,
			IsActive: p.Category.IsActive,
		}
	}

	var variantResps []ProductVariantResponse
	for _, v := range p.Variants {
		var uomResps []ProductUOMResponse
		for _, u := range v.UOMs {
			uomResps = append(uomResps, ProductUOMResponse{
				ID:             u.ID,
				UUID:           u.UUID,
				UOMName:        u.UOMName,
				ConversionRate: u.ConversionRate,
				SellPrice:      u.SellPrice,
				Barcode:        u.Barcode,
				IsBaseUOM:      u.IsBaseUOM,
			})
		}

		variantResps = append(variantResps, ProductVariantResponse{
			ID:            v.ID,
			UUID:          v.UUID,
			SKU:           v.SKU,
			Barcode:       v.Barcode,
			VariantName:   v.VariantName,
			BaseCostPrice: v.BaseCostPrice,
			BaseSellPrice: v.BaseSellPrice,
			MinStockAlert: v.MinStockAlert,
			IsActive:      v.IsActive,
			UOMs:          uomResps,
		})
	}

	return ProductResponse{
		ID:          p.ID,
		UUID:        p.UUID,
		CategoryID:  p.CategoryID,
		Category:    categoryResp,
		Name:        p.Name,
		Description: p.Description,
		ImageURL:    p.ImageURL,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Variants:    variantResps,
	}
}

func (u *usecase) ListProducts(ctx context.Context, tenantID uint64, filter ProductQueryFilter) ([]ProductResponse, int64, error) {
	products, total, err := u.repo.ListProducts(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}

	var responses []ProductResponse
	for i := range products {
		responses = append(responses, mapProductModelToDTO(&products[i]))
	}

	return responses, total, nil
}

func (u *usecase) GetProductByUUID(ctx context.Context, tenantID uint64, productUUID uuid.UUID) (*ProductResponse, error) {
	p, err := u.repo.GetProductByUUID(ctx, tenantID, productUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	dto := mapProductModelToDTO(p)
	return &dto, nil
}

func (u *usecase) CreateProduct(ctx context.Context, tenantID uint64, req CreateProductRequest) (*ProductResponse, error) {
	// Verify category if set
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if _, err := u.repo.GetCategoryByID(ctx, tenantID, *req.CategoryID); err != nil {
			return nil, errors.New("category does not exist")
		}
	}

	// Verify all SKUs in variants are unique
	for _, v := range req.Variants {
		existing, _ := u.repo.FindVariantBySKU(ctx, tenantID, v.SKU)
		if existing != nil {
			return nil, fmt.Errorf("%w: %s", ErrSKUAlreadyExists, v.SKU)
		}
	}

	product := &model.Product{
		TenantID:    tenantID,
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		IsActive:    true,
	}

	for _, v := range req.Variants {
		variant := model.ProductVariant{
			TenantID:      tenantID,
			SKU:           v.SKU,
			Barcode:       v.Barcode,
			VariantName:   v.VariantName,
			BaseCostPrice: v.BaseCostPrice,
			BaseSellPrice: v.BaseSellPrice,
			MinStockAlert: v.MinStockAlert,
			IsActive:      true,
		}

		hasBaseUOM := false
		for _, uomReq := range v.UOMs {
			if uomReq.IsBaseUOM || uomReq.ConversionRate == 1 {
				hasBaseUOM = true
			}
			variant.UOMs = append(variant.UOMs, model.ProductUOM{
				TenantID:       tenantID,
				UOMName:        uomReq.UOMName,
				ConversionRate: uomReq.ConversionRate,
				SellPrice:      uomReq.SellPrice,
				Barcode:        uomReq.Barcode,
				IsBaseUOM:      uomReq.IsBaseUOM,
			})
		}

		// Ensure default base unit (Pcs) is created if none was specified
		if !hasBaseUOM {
			variant.UOMs = append([]model.ProductUOM{
				{
					TenantID:       tenantID,
					UOMName:        "Pcs",
					ConversionRate: 1,
					SellPrice:      v.BaseSellPrice,
					Barcode:        v.Barcode,
					IsBaseUOM:      true,
				},
			}, variant.UOMs...)
		}

		product.Variants = append(product.Variants, variant)
	}

	if err := u.repo.CreateProductWithVariants(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to save product: %w", err)
	}

	// Fetch full model with relations
	savedProduct, err := u.repo.GetProductByUUID(ctx, tenantID, product.UUID)
	if err != nil {
		return nil, err
	}

	resp := mapProductModelToDTO(savedProduct)
	return &resp, nil
}

func (u *usecase) UpdateProduct(ctx context.Context, tenantID uint64, productUUID uuid.UUID, req UpdateProductRequest) (*ProductResponse, error) {
	p, err := u.repo.GetProductByUUID(ctx, tenantID, productUUID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	if req.Name != nil && *req.Name != "" {
		p.Name = *req.Name
	}
	if req.CategoryID != nil {
		p.CategoryID = req.CategoryID
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.ImageURL != nil {
		p.ImageURL = *req.ImageURL
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if err := u.repo.UpdateProduct(ctx, p); err != nil {
		return nil, err
	}

	resp := mapProductModelToDTO(p)
	return &resp, nil
}

func (u *usecase) DeleteProduct(ctx context.Context, tenantID uint64, productUUID uuid.UUID) error {
	return u.repo.DeleteProduct(ctx, tenantID, productUUID)
}

func (u *usecase) ScanBarcode(ctx context.Context, tenantID uint64, barcode string) (*BarcodeScanResponse, error) {
	cleanCode := strings.TrimSpace(barcode)
	if cleanCode == "" {
		return nil, errors.New("barcode cannot be empty")
	}

	variant, matchedUOM, product, err := u.repo.FindByBarcode(ctx, tenantID, cleanCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBarcodeNotFound
		}
		return nil, err
	}

	productDTO := mapProductModelToDTO(product)

	var variantDTO ProductVariantResponse
	for _, v := range productDTO.Variants {
		if v.ID == variant.ID {
			variantDTO = v
			break
		}
	}

	scanType := "BASE_VARIANT"
	calculatedPrice := variant.BaseSellPrice
	baseUnitMultiplier := 1

	var uomDTO *ProductUOMResponse
	if matchedUOM != nil {
		uomDTO = &ProductUOMResponse{
			ID:             matchedUOM.ID,
			UUID:           matchedUOM.UUID,
			UOMName:        matchedUOM.UOMName,
			ConversionRate: matchedUOM.ConversionRate,
			SellPrice:      matchedUOM.SellPrice,
			Barcode:        matchedUOM.Barcode,
			IsBaseUOM:      matchedUOM.IsBaseUOM,
		}
		if !matchedUOM.IsBaseUOM && matchedUOM.ConversionRate > 1 {
			scanType = "PACKAGING_UOM"
			calculatedPrice = matchedUOM.SellPrice
			baseUnitMultiplier = matchedUOM.ConversionRate
		}
	}

	return &BarcodeScanResponse{
		Product:            productDTO,
		Variant:            variantDTO,
		MatchedUOM:         uomDTO,
		ScanType:           scanType,
		CalculatedPrice:    calculatedPrice,
		BaseUnitMultiplier: baseUnitMultiplier,
	}, nil
}
