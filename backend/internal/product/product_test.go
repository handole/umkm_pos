package product_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"github.com/handokodenih/umkm_pos/backend/internal/product"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
	"gorm.io/gorm"
)

// mockRepository implements product.Repository in-memory for testing.
type mockRepository struct {
	categories []model.Category
	products   []model.Product
	variants   []model.ProductVariant
	uoms       []model.ProductUOM
	catSeq     uint64
	prodSeq    uint64
	varSeq     uint64
	uomSeq     uint64
}

func newMockRepository() *mockRepository {
	return &mockRepository{}
}

func (m *mockRepository) ListCategories(ctx context.Context, tenantID uint64) ([]model.Category, error) {
	var result []model.Category
	for _, c := range m.categories {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockRepository) GetCategoryByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Category, error) {
	for _, c := range m.categories {
		if c.TenantID == tenantID && c.UUID == u {
			return &c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockRepository) GetCategoryByID(ctx context.Context, tenantID uint64, id uint64) (*model.Category, error) {
	for _, c := range m.categories {
		if c.TenantID == tenantID && c.ID == id {
			return &c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockRepository) CreateCategory(ctx context.Context, category *model.Category) error {
	m.catSeq++
	category.ID = m.catSeq
	category.UUID = uuid.New()
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()
	m.categories = append(m.categories, *category)
	return nil
}

func (m *mockRepository) UpdateCategory(ctx context.Context, category *model.Category) error {
	for i, c := range m.categories {
		if c.ID == category.ID {
			m.categories[i] = *category
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockRepository) DeleteCategory(ctx context.Context, tenantID uint64, categoryID uint64) error {
	for i, c := range m.categories {
		if c.TenantID == tenantID && c.ID == categoryID {
			m.categories = append(m.categories[:i], m.categories[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) ListProducts(ctx context.Context, tenantID uint64, filter product.ProductQueryFilter) ([]model.Product, int64, error) {
	var result []model.Product
	for _, p := range m.products {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockRepository) GetProductByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Product, error) {
	for _, p := range m.products {
		if p.TenantID == tenantID && p.UUID == u {
			return &p, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockRepository) CreateProductWithVariants(ctx context.Context, p *model.Product) error {
	m.prodSeq++
	p.ID = m.prodSeq
	p.UUID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	for i := range p.Variants {
		m.varSeq++
		p.Variants[i].ID = m.varSeq
		p.Variants[i].UUID = uuid.New()
		p.Variants[i].ProductID = p.ID
		p.Variants[i].TenantID = p.TenantID
		m.variants = append(m.variants, p.Variants[i])

		for j := range p.Variants[i].UOMs {
			m.uomSeq++
			p.Variants[i].UOMs[j].ID = m.uomSeq
			p.Variants[i].UOMs[j].UUID = uuid.New()
			p.Variants[i].UOMs[j].VariantID = p.Variants[i].ID
			p.Variants[i].UOMs[j].TenantID = p.TenantID
			m.uoms = append(m.uoms, p.Variants[i].UOMs[j])
		}
	}

	m.products = append(m.products, *p)
	return nil
}

func (m *mockRepository) UpdateProduct(ctx context.Context, p *model.Product) error {
	for i, prod := range m.products {
		if prod.ID == p.ID {
			m.products[i] = *p
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockRepository) DeleteProduct(ctx context.Context, tenantID uint64, productUUID uuid.UUID) error {
	for i, p := range m.products {
		if p.TenantID == tenantID && p.UUID == productUUID {
			m.products = append(m.products[:i], m.products[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) FindVariantBySKU(ctx context.Context, tenantID uint64, sku string) (*model.ProductVariant, error) {
	for _, v := range m.variants {
		if v.TenantID == tenantID && v.SKU == sku {
			return &v, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockRepository) FindByBarcode(ctx context.Context, tenantID uint64, barcode string) (*model.ProductVariant, *model.ProductUOM, *model.Product, error) {
	// 1. Check variant barcode (retail eceran)
	for _, v := range m.variants {
		if v.TenantID == tenantID && v.Barcode == barcode {
			// Find parent product
			for _, p := range m.products {
				if p.ID == v.ProductID {
					var matchedUOM *model.ProductUOM
					for _, u := range v.UOMs {
						if u.IsBaseUOM {
							matchedUOM = &u
							break
						}
					}
					return &v, matchedUOM, &p, nil
				}
			}
		}
	}

	// 2. Check packaging UOM barcode (kardus / dus)
	for _, u := range m.uoms {
		if u.TenantID == tenantID && u.Barcode == barcode {
			for _, v := range m.variants {
				if v.ID == u.VariantID {
					for _, p := range m.products {
						if p.ID == v.ProductID {
							return &v, &u, &p, nil
						}
					}
				}
			}
		}
	}

	return nil, nil, nil, gorm.ErrRecordNotFound
}

func setupProductTestRouter(repo product.Repository, jwtService *pkgjwt.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ResolveTenantContext())

	apiV1 := r.Group("/api/v1")
	usecase := product.NewUsecase(repo)
	handler := product.NewHandler(usecase)
	handler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

	return r
}

func TestProductAndBarcodeScanFlow(t *testing.T) {
	jwtService := pkgjwt.NewService("test-secret", time.Hour, 24*time.Hour)
	repo := newMockRepository()
	router := setupProductTestRouter(repo, jwtService)

	// Create tokens for manager with full permissions
	managerTokens, _ := jwtService.GenerateTokenPair(
		1,
		uuid.New(),
		1,
		uuid.New(),
		"manager_agus",
		"STORE_MANAGER",
		[]string{"product:read", "product:create", "product:update", "product:delete"},
		[]uint64{1},
	)

	// Create tokens for cashier with read-only product permission
	cashierTokens, _ := jwtService.GenerateTokenPair(
		2,
		uuid.New(),
		1,
		uuid.New(),
		"kasir_budi",
		"CASHIER",
		[]string{"product:read", "pos:checkout"},
		[]uint64{1},
	)

	// 1. Create a Category
	catPayload := product.CreateCategoryRequest{
		Name: "Makanan Instan",
	}
	body, _ := json.Marshal(catPayload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+managerTokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for Category, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. Cashier tries to create a product -> Should fail 403 Forbidden
	prodPayload := product.CreateProductRequest{
		Name: "Indomie Goreng Spesial",
		Variants: []product.CreateVariantRequest{
			{
				SKU:           "IND-001",
				Barcode:       "089686010118", // Barcode eceran sachet
				VariantName:   "Original 85g",
				BaseCostPrice: 3000,
				BaseSellPrice: 3500,
				UOMs: []product.CreateUOMRequest{
					{
						UOMName:        "Pcs",
						ConversionRate: 1,
						SellPrice:      3500,
						Barcode:        "089686010118",
						IsBaseUOM:      true,
					},
					{
						UOMName:        "Dus",
						ConversionRate: 40,
						SellPrice:      135000,
						Barcode:        "089686010199", // Barcode kemasan kardus luar
						IsBaseUOM:      false,
					},
				},
			},
		},
	}
	body, _ = json.Marshal(prodPayload)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+cashierTokens.AccessToken) // Cashier doesn't have product:create
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden for Cashier creating product, got %d", w.Code)
	}

	// 3. Manager creates the product -> Should succeed 201 Created
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+managerTokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for Product, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 4. Barcode Scan Test 1: Scan Retail Eceran Barcode (089686010118) by Cashier
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/products/barcode/089686010118", nil)
	req.Header.Set("Authorization", "Bearer "+cashierTokens.AccessToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for Barcode scan, got %d. Body: %s", w.Code, w.Body.String())
	}

	var scanResult struct {
		Success bool                        `json:"success"`
		Data    product.BarcodeScanResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &scanResult)

	if scanResult.Data.ScanType != "BASE_VARIANT" {
		t.Errorf("Expected ScanType BASE_VARIANT, got %s", scanResult.Data.ScanType)
	}
	if scanResult.Data.CalculatedPrice != 3500 {
		t.Errorf("Expected CalculatedPrice 3500, got %d", scanResult.Data.CalculatedPrice)
	}
	if scanResult.Data.BaseUnitMultiplier != 1 {
		t.Errorf("Expected BaseUnitMultiplier 1, got %d", scanResult.Data.BaseUnitMultiplier)
	}

	// 5. Barcode Scan Test 2: Scan Packaging Dus Barcode (089686010199) by Cashier/Warehouse
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/products/barcode/089686010199", nil)
	req.Header.Set("Authorization", "Bearer "+cashierTokens.AccessToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for Packaging Barcode scan, got %d. Body: %s", w.Code, w.Body.String())
	}

	var boxScanResult struct {
		Success bool                        `json:"success"`
		Data    product.BarcodeScanResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &boxScanResult)

	if boxScanResult.Data.ScanType != "PACKAGING_UOM" {
		t.Errorf("Expected ScanType PACKAGING_UOM, got %s", boxScanResult.Data.ScanType)
	}
	if boxScanResult.Data.CalculatedPrice != 135000 {
		t.Errorf("Expected CalculatedPrice 135000 (Dus price), got %d", boxScanResult.Data.CalculatedPrice)
	}
	if boxScanResult.Data.BaseUnitMultiplier != 40 {
		t.Errorf("Expected BaseUnitMultiplier 40 (40 Pcs per Dus), got %d", boxScanResult.Data.BaseUnitMultiplier)
	}

	// 6. Barcode Scan Test 3: Scan Non-Existent Barcode -> Should return 404 Not Found
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/products/barcode/999999999999", nil)
	req.Header.Set("Authorization", "Bearer "+cashierTokens.AccessToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found for unregistered barcode, got %d", w.Code)
	}
}
