package inventory_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/inventory"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
	"gorm.io/gorm"
)

// mockInventoryRepo implements inventory.Repository in-memory for unit testing.
type mockInventoryRepo struct {
	locations     []model.InventoryLocation
	stocks        []model.InventoryStock
	movements     []model.StockMovement
	transfers     []model.StockTransfer
	transferItems []model.StockTransferItem
	opnames       []model.StockOpname
	opnameItems   []model.StockOpnameItem
	opnameScans   []model.StockOpnameScan
	variants      []model.ProductVariant
	uoms          []model.ProductUOM
	suppliers     []model.Supplier
	purchaseOrds  []model.PurchaseOrder

	locSeq uint64
	stkSeq uint64
	movSeq uint64
	trfSeq uint64
	opnSeq uint64
	scnSeq uint64
	itmSeq uint64
}

func newMockInventoryRepo() *mockInventoryRepo {
	return &mockInventoryRepo{}
}

func (m *mockInventoryRepo) WithTransaction(ctx context.Context, fn func(txRepo inventory.Repository) error) error {
	return fn(m)
}

func (m *mockInventoryRepo) CreateLocation(ctx context.Context, loc *model.InventoryLocation) error {
	m.locSeq++
	loc.ID = m.locSeq
	loc.UUID = uuid.New()
	loc.CreatedAt = time.Now()
	loc.UpdatedAt = time.Now()
	m.locations = append(m.locations, *loc)
	return nil
}

func (m *mockInventoryRepo) ListLocations(ctx context.Context, tenantID uint64, outletID *uint64) ([]model.InventoryLocation, error) {
	var res []model.InventoryLocation
	for _, l := range m.locations {
		if l.TenantID == tenantID {
			if outletID != nil && l.OutletID != *outletID {
				continue
			}
			res = append(res, l)
		}
	}
	return res, nil
}

func (m *mockInventoryRepo) FindLocationByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.InventoryLocation, error) {
	for _, l := range m.locations {
		if l.TenantID == tenantID && l.UUID == u {
			return &l, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) FindLocationByID(ctx context.Context, tenantID uint64, id uint64) (*model.InventoryLocation, error) {
	for _, l := range m.locations {
		if l.TenantID == tenantID && l.ID == id {
			return &l, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) GetStock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error) {
	for _, s := range m.stocks {
		if s.TenantID == tenantID && s.LocationID == locationID && s.VariantID == variantID {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockInventoryRepo) GetStockWithLock(ctx context.Context, tenantID, locationID, variantID uint64) (*model.InventoryStock, error) {
	for i, s := range m.stocks {
		if s.TenantID == tenantID && s.LocationID == locationID && s.VariantID == variantID {
			return &m.stocks[i], nil
		}
	}
	// Inisialisasi saldo jika belum ada
	m.stkSeq++
	newStock := model.InventoryStock{
		ID:         m.stkSeq,
		UUID:       uuid.New(),
		TenantID:   tenantID,
		LocationID: locationID,
		VariantID:  variantID,
		Quantity:   0,
		UpdatedAt:  time.Now(),
	}
	m.stocks = append(m.stocks, newStock)
	return &m.stocks[len(m.stocks)-1], nil
}

func (m *mockInventoryRepo) SaveStock(ctx context.Context, stock *model.InventoryStock) error {
	for i, s := range m.stocks {
		if s.ID == stock.ID || (s.TenantID == stock.TenantID && s.LocationID == stock.LocationID && s.VariantID == stock.VariantID) {
			m.stocks[i].Quantity = stock.Quantity
			m.stocks[i].UpdatedAt = time.Now()
			return nil
		}
	}
	m.stkSeq++
	stock.ID = m.stkSeq
	if stock.UUID == uuid.Nil {
		stock.UUID = uuid.New()
	}
	stock.UpdatedAt = time.Now()
	m.stocks = append(m.stocks, *stock)
	return nil
}

func (m *mockInventoryRepo) ListStocks(ctx context.Context, tenantID uint64, locationID *uint64, lowStockOnly bool, limit, offset int) ([]model.InventoryStock, int64, error) {
	var res []model.InventoryStock
	for _, s := range m.stocks {
		if s.TenantID == tenantID {
			if locationID != nil && s.LocationID != *locationID {
				continue
			}
			res = append(res, s)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockInventoryRepo) RecordMovement(ctx context.Context, movement *model.StockMovement) error {
	m.movSeq++
	movement.ID = m.movSeq
	movement.UUID = uuid.New()
	movement.CreatedAt = time.Now()
	m.movements = append(m.movements, *movement)
	return nil
}

func (m *mockInventoryRepo) ListMovements(ctx context.Context, tenantID uint64, filter inventory.StockMovementFilterQuery) ([]model.StockMovement, int64, error) {
	var res []model.StockMovement
	for _, mov := range m.movements {
		if mov.TenantID == tenantID {
			res = append(res, mov)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockInventoryRepo) CreateTransfer(ctx context.Context, t *model.StockTransfer) error {
	m.trfSeq++
	t.ID = m.trfSeq
	t.UUID = uuid.New()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	m.transfers = append(m.transfers, *t)
	return nil
}

func (m *mockInventoryRepo) FindTransferByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.StockTransfer, error) {
	for _, t := range m.transfers {
		if t.TenantID == tenantID && t.UUID == u {
			cp := t
			var items []model.StockTransferItem
			for _, it := range m.transferItems {
				if it.TransferID == cp.ID {
					for _, v := range m.variants {
						if v.ID == it.VariantID {
							itCopy := it
							itCopy.Variant = &v
							items = append(items, itCopy)
							break
						}
					}
				}
			}
			cp.Items = items
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) UpdateTransfer(ctx context.Context, t *model.StockTransfer) error {
	for i, tr := range m.transfers {
		if tr.ID == t.ID {
			m.transfers[i] = *t
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) CreateTransferItem(ctx context.Context, item *model.StockTransferItem) error {
	m.itmSeq++
	item.ID = m.itmSeq
	item.UUID = uuid.New()
	m.transferItems = append(m.transferItems, *item)
	return nil
}

func (m *mockInventoryRepo) UpdateTransferItem(ctx context.Context, item *model.StockTransferItem) error {
	for i, it := range m.transferItems {
		if it.ID == item.ID {
			m.transferItems[i] = *item
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) ListTransfers(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.StockTransfer, int64, error) {
	var res []model.StockTransfer
	for _, t := range m.transfers {
		if t.TenantID == tenantID {
			if status != nil && *status != "" && t.Status != *status {
				continue
			}
			res = append(res, t)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockInventoryRepo) CreateOpname(ctx context.Context, o *model.StockOpname) error {
	m.opnSeq++
	o.ID = m.opnSeq
	o.UUID = uuid.New()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	m.opnames = append(m.opnames, *o)
	return nil
}

func (m *mockInventoryRepo) FindOpnameByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.StockOpname, error) {
	for _, o := range m.opnames {
		if o.TenantID == tenantID && o.UUID == u {
			cp := o
			var items []model.StockOpnameItem
			for _, it := range m.opnameItems {
				if it.OpnameID == cp.ID {
					itCopy := it
					for _, v := range m.variants {
						if v.ID == it.VariantID {
							itCopy.Variant = &v
							break
						}
					}
					items = append(items, itCopy)
				}
			}
			cp.Items = items

			var scans []model.StockOpnameScan
			for _, sc := range m.opnameScans {
				if sc.OpnameID == cp.ID {
					scCopy := sc
					for _, v := range m.variants {
						if v.ID == sc.VariantID {
							scCopy.Variant = &v
							break
						}
					}
					scans = append(scans, scCopy)
				}
			}
			cp.Scans = scans

			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) UpdateOpname(ctx context.Context, o *model.StockOpname) error {
	for i, op := range m.opnames {
		if op.ID == o.ID {
			m.opnames[i] = *o
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) ListOpnames(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.StockOpname, int64, error) {
	var res []model.StockOpname
	for _, o := range m.opnames {
		if o.TenantID == tenantID {
			if status != nil && *status != "" && o.Status != *status {
				continue
			}
			res = append(res, o)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockInventoryRepo) RecordOpnameScan(ctx context.Context, s *model.StockOpnameScan) error {
	m.scnSeq++
	s.ID = m.scnSeq
	s.UUID = uuid.New()
	s.ScannedAt = time.Now()
	m.opnameScans = append(m.opnameScans, *s)
	return nil
}

func (m *mockInventoryRepo) FindOpnameItem(ctx context.Context, opnameID, variantID uint64) (*model.StockOpnameItem, error) {
	for _, it := range m.opnameItems {
		if it.OpnameID == opnameID && it.VariantID == variantID {
			return &it, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) SaveOpnameItem(ctx context.Context, item *model.StockOpnameItem) error {
	for i, it := range m.opnameItems {
		if it.OpnameID == item.OpnameID && it.VariantID == item.VariantID {
			m.opnameItems[i] = *item
			return nil
		}
	}
	m.itmSeq++
	item.ID = m.itmSeq
	item.UUID = uuid.New()
	m.opnameItems = append(m.opnameItems, *item)
	return nil
}

func (m *mockInventoryRepo) FindVariantByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.ProductVariant, error) {
	for _, v := range m.variants {
		if v.TenantID == tenantID && v.UUID == u {
			return &v, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) FindBarcode(ctx context.Context, tenantID uint64, barcode string) (*model.ProductVariant, *model.ProductUOM, int, error) {
	for _, v := range m.variants {
		if v.TenantID == tenantID && v.Barcode == barcode {
			return &v, nil, 1, nil
		}
	}
	for _, u := range m.uoms {
		if u.TenantID == tenantID && u.Barcode == barcode {
			for _, v := range m.variants {
				if v.ID == u.VariantID {
					rate := u.ConversionRate
					if rate <= 0 {
						rate = 1
					}
					return &v, &u, rate, nil
				}
			}
		}
	}
	return nil, nil, 0, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) CreateSupplier(ctx context.Context, s *model.Supplier) error {
	s.ID = uint64(len(m.suppliers) + 1)
	s.UUID = uuid.New()
	m.suppliers = append(m.suppliers, *s)
	return nil
}

func (m *mockInventoryRepo) ListSuppliers(ctx context.Context, tenantID uint64) ([]model.Supplier, error) {
	var res []model.Supplier
	for _, s := range m.suppliers {
		if s.TenantID == tenantID {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *mockInventoryRepo) FindSupplierByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.Supplier, error) {
	for _, s := range m.suppliers {
		if s.TenantID == tenantID && s.UUID == u {
			sc := s
			return &sc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) CreatePurchaseOrder(ctx context.Context, po *model.PurchaseOrder) error {
	po.ID = uint64(len(m.purchaseOrds) + 1)
	po.UUID = uuid.New()
	for i := range po.Items {
		po.Items[i].ID = uint64(i + 1)
		po.Items[i].UUID = uuid.New()
		po.Items[i].POID = po.ID
	}
	m.purchaseOrds = append(m.purchaseOrds, *po)
	return nil
}

func (m *mockInventoryRepo) FindPurchaseOrderByUUID(ctx context.Context, tenantID uint64, u uuid.UUID) (*model.PurchaseOrder, error) {
	for _, po := range m.purchaseOrds {
		if po.TenantID == tenantID && po.UUID == u {
			cp := po
			cp.Items = append([]model.PurchaseOrderItem(nil), po.Items...)
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) ListPurchaseOrders(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.PurchaseOrder, int64, error) {
	var res []model.PurchaseOrder
	for _, po := range m.purchaseOrds {
		if po.TenantID == tenantID && (status == nil || *status == "" || po.Status == *status) {
			res = append(res, po)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockInventoryRepo) UpdatePurchaseOrder(ctx context.Context, po *model.PurchaseOrder) error {
	for i := range m.purchaseOrds {
		if m.purchaseOrds[i].ID == po.ID {
			m.purchaseOrds[i] = *po
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockInventoryRepo) UpdatePurchaseOrderItem(ctx context.Context, item *model.PurchaseOrderItem) error {
	for i := range m.purchaseOrds {
		if m.purchaseOrds[i].ID == item.POID {
			for j := range m.purchaseOrds[i].Items {
				if m.purchaseOrds[i].Items[j].ID == item.ID {
					m.purchaseOrds[i].Items[j] = *item
					return nil
				}
			}
		}
	}
	return gorm.ErrRecordNotFound
}

func setupInventoryTestApp(repo inventory.Repository) (*gin.Engine, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtService := pkgjwt.NewService("test-secret-key-for-inventory-tests", time.Hour, 24*time.Hour)
	tokens, _ := jwtService.GenerateTokenPair(
		1,
		uuid.New(),
		1,
		uuid.New(),
		"manager_budi",
		"STORE_MANAGER",
		[]string{
			"inventory:read",
			"inventory:transfer",
			"inventory:opname",
			"inventory:approve-opname",
		},
		[]uint64{1},
	)
	token := tokens.AccessToken

	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyTenantID, uint64(1))
		c.Next()
	})

	apiV1 := r.Group("/api/v1")
	uc := inventory.NewUsecase(repo)
	handler := inventory.NewHandler(uc)
	handler.RegisterRoutes(apiV1, middleware.Authenticate(jwtService))

	return r, token
}

func TestInventoryLocationsAndManualAdjustment(t *testing.T) {
	repo := newMockInventoryRepo()
	r, token := setupInventoryTestApp(repo)

	// 1. Create Gudang Utama location
	createLocPayload := inventory.CreateLocationRequest{
		OutletID: 1,
		Code:     "GDG-01",
		Name:     "Gudang Utama",
		Type:     "WAREHOUSE",
	}
	body, _ := json.Marshal(createLocPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/locations", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var locRes struct {
		Data inventory.LocationResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &locRes)
	if locRes.Data.Code != "GDG-01" {
		t.Fatalf("expected code GDG-01, got %s", locRes.Data.Code)
	}

	// 2. Setup mock product variant
	variantUUID := uuid.New()
	mockVariant := model.ProductVariant{
		ID:            101,
		UUID:          variantUUID,
		TenantID:      1,
		SKU:           "IND-GRG-01",
		Barcode:       "8998866110011",
		VariantName:   "Indomie Goreng Original",
		BaseCostPrice: 2800,
		BaseSellPrice: 3500,
	}
	repo.variants = append(repo.variants, mockVariant)

	// 3. Apply manual adjustment: initialize stock +100
	adjPayload := inventory.ManualAdjustmentRequest{
		LocationUUID:   locRes.Data.UUID,
		VariantUUID:    variantUUID,
		QuantityChange: 100,
		CostPrice:      2800,
		Notes:          "Initial stock balance opname setup",
	}
	bodyAdj, _ := json.Marshal(adjPayload)
	reqAdj := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/adjustments", bytes.NewBuffer(bodyAdj))
	reqAdj.Header.Set("Authorization", "Bearer "+token)
	reqAdj.Header.Set("Content-Type", "application/json")
	wAdj := httptest.NewRecorder()
	r.ServeHTTP(wAdj, reqAdj)

	if wAdj.Code != http.StatusOK {
		t.Fatalf("expected adjustment status 200, got %d: %s", wAdj.Code, wAdj.Body.String())
	}

	// 4. Verify stock balances
	reqStock := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stocks", nil)
	reqStock.Header.Set("Authorization", "Bearer "+token)
	wStock := httptest.NewRecorder()
	r.ServeHTTP(wStock, reqStock)

	if wStock.Code != http.StatusOK {
		t.Fatalf("expected stock list status 200, got %d: %s", wStock.Code, wStock.Body.String())
	}

	var stockRes struct {
		Data []inventory.StockResponse `json:"data"`
	}
	json.Unmarshal(wStock.Body.Bytes(), &stockRes)
	if len(stockRes.Data) != 1 || stockRes.Data[0].Quantity != 100 {
		t.Fatalf("expected stock quantity 100, got %+v", stockRes.Data)
	}
}

func TestStockTransferWorkflow(t *testing.T) {
	repo := newMockInventoryRepo()
	r, token := setupInventoryTestApp(repo)

	// Prepare source (Gudang) and destination (Toko)
	locSource := model.InventoryLocation{
		ID: 1, UUID: uuid.New(), TenantID: 1, OutletID: 1, Code: "GUDANG", Name: "Gudang Pusat", Type: "WAREHOUSE", IsActive: true,
	}
	locDest := model.InventoryLocation{
		ID: 2, UUID: uuid.New(), TenantID: 1, OutletID: 1, Code: "TOKO", Name: "Toko Depan Kasir", Type: "STORE_FRONT", IsActive: true,
	}
	repo.locations = append(repo.locations, locSource, locDest)

	variantUUID := uuid.New()
	mockVariant := model.ProductVariant{
		ID: 201, UUID: variantUUID, TenantID: 1, SKU: "KOPI-01", VariantName: "Kopi Kapal Api", BaseCostPrice: 1500,
	}
	repo.variants = append(repo.variants, mockVariant)

	// Initial stock at Gudang = 50
	repo.stocks = append(repo.stocks, model.InventoryStock{
		ID: 1, UUID: uuid.New(), TenantID: 1, LocationID: locSource.ID, VariantID: mockVariant.ID, Quantity: 50,
	})

	// 1. Create Stock Transfer of 20 units
	trfPayload := inventory.CreateStockTransferRequest{
		SourceLocationUUID:      locSource.UUID,
		DestinationLocationUUID: locDest.UUID,
		Notes:                   "Restock rak kasir toko depan",
		Items: []inventory.CreateStockTransferItemRequest{
			{
				VariantUUID:  variantUUID,
				QuantitySent: 20,
			},
		},
	}
	bodyTrf, _ := json.Marshal(trfPayload)
	reqTrf := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBuffer(bodyTrf))
	reqTrf.Header.Set("Authorization", "Bearer "+token)
	reqTrf.Header.Set("Content-Type", "application/json")
	wTrf := httptest.NewRecorder()
	r.ServeHTTP(wTrf, reqTrf)

	if wTrf.Code != http.StatusCreated {
		t.Fatalf("expected transfer creation 201, got %d: %s", wTrf.Code, wTrf.Body.String())
	}

	var trfRes struct {
		Data inventory.StockTransferResponse `json:"data"`
	}
	json.Unmarshal(wTrf.Body.Bytes(), &trfRes)
	if trfRes.Data.Status != "IN_TRANSIT" {
		t.Fatalf("expected transfer status IN_TRANSIT, got %s", trfRes.Data.Status)
	}

	// Verify source stock is deducted: 50 - 20 = 30
	srcStock, _ := repo.GetStock(context.Background(), 1, locSource.ID, mockVariant.ID)
	if srcStock.Quantity != 30 {
		t.Fatalf("expected source stock 30, got %d", srcStock.Quantity)
	}

	// 2. Receive Transfer at destination
	recvPayload := inventory.ReceiveTransferRequest{
		Items: []inventory.ReceiveTransferItemRequest{
			{
				VariantUUID:      variantUUID,
				QuantityReceived: 20,
			},
		},
	}
	bodyRecv, _ := json.Marshal(recvPayload)
	recvUrl := fmt.Sprintf("/api/v1/inventory/transfers/%s/receive", trfRes.Data.UUID)
	reqRecv := httptest.NewRequest(http.MethodPost, recvUrl, bytes.NewBuffer(bodyRecv))
	reqRecv.Header.Set("Authorization", "Bearer "+token)
	reqRecv.Header.Set("Content-Type", "application/json")
	wRecv := httptest.NewRecorder()
	r.ServeHTTP(wRecv, reqRecv)

	if wRecv.Code != http.StatusOK {
		t.Fatalf("expected receive status 200, got %d: %s", wRecv.Code, wRecv.Body.String())
	}

	// Verify destination stock increased to 20
	destStock, _ := repo.GetStock(context.Background(), 1, locDest.ID, mockVariant.ID)
	if destStock.Quantity != 20 {
		t.Fatalf("expected destination stock 20, got %d", destStock.Quantity)
	}
}

func TestStockOpnameDualBarcodeScanAndApproval(t *testing.T) {
	repo := newMockInventoryRepo()
	r, token := setupInventoryTestApp(repo)

	loc := model.InventoryLocation{
		ID: 1, UUID: uuid.New(), TenantID: 1, OutletID: 1, Code: "GDG", Name: "Gudang", Type: "WAREHOUSE", IsActive: true,
	}
	repo.locations = append(repo.locations, loc)

	variantUUID := uuid.New()
	mockVariant := model.ProductVariant{
		ID:            301,
		UUID:          variantUUID,
		TenantID:      1,
		SKU:           "MIE-01",
		Barcode:       "899111000", // Barcode retail sachet
		VariantName:   "Mie Instan Goreng",
		BaseCostPrice: 3000,
	}
	repo.variants = append(repo.variants, mockVariant)

	// UOM Karton (isi 40 pcs) dengan barcode kemasan kardus
	mockUOM := model.ProductUOM{
		ID:             1,
		UUID:           uuid.New(),
		TenantID:       1,
		VariantID:      mockVariant.ID,
		UOMName:        "Dus",
		ConversionRate: 40,
		Barcode:        "0899111000999", // Barcode luar kardus
	}
	repo.uoms = append(repo.uoms, mockUOM)

	// Saldo sistem saat ini = 10 pcs
	repo.stocks = append(repo.stocks, model.InventoryStock{
		ID: 1, UUID: uuid.New(), TenantID: 1, LocationID: loc.ID, VariantID: mockVariant.ID, Quantity: 10,
	})

	// 1. Create Stock Opname Session
	createOpnPayload := inventory.CreateStockOpnameRequest{
		LocationUUID: loc.UUID,
		Notes:        "Audit fisik akhir bulan gudang",
	}
	bodyOpn, _ := json.Marshal(createOpnPayload)
	reqOpn := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/opnames", bytes.NewBuffer(bodyOpn))
	reqOpn.Header.Set("Authorization", "Bearer "+token)
	reqOpn.Header.Set("Content-Type", "application/json")
	wOpn := httptest.NewRecorder()
	r.ServeHTTP(wOpn, reqOpn)

	if wOpn.Code != http.StatusCreated {
		t.Fatalf("expected opname create 201, got %d: %s", wOpn.Code, wOpn.Body.String())
	}

	var opnRes struct {
		Data inventory.StockOpnameResponse `json:"data"`
	}
	json.Unmarshal(wOpn.Body.Bytes(), &opnRes)
	opnameUUID := opnRes.Data.UUID

	// 2. Scan barcode eceran: 5 sachet
	scanSachetPayload := inventory.ScanOpnameBarcodeRequest{
		Barcode:  "899111000",
		Quantity: 5,
		DeviceID: "MOBILE-SCANNER-01",
	}
	bodySachet, _ := json.Marshal(scanSachetPayload)
	scanUrl := fmt.Sprintf("/api/v1/inventory/opnames/%s/scan", opnameUUID)
	reqScan1 := httptest.NewRequest(http.MethodPost, scanUrl, bytes.NewBuffer(bodySachet))
	reqScan1.Header.Set("Authorization", "Bearer "+token)
	reqScan1.Header.Set("Content-Type", "application/json")
	wScan1 := httptest.NewRecorder()
	r.ServeHTTP(wScan1, reqScan1)

	if wScan1.Code != http.StatusOK {
		t.Fatalf("expected scan 1 status 200, got %d: %s", wScan1.Code, wScan1.Body.String())
	}

	// 3. Scan barcode kardus luar (1 Dus = 40 pcs)
	scanBoxPayload := inventory.ScanOpnameBarcodeRequest{
		Barcode:  "0899111000999",
		Quantity: 1, // 1 Dus
		DeviceID: "MOBILE-SCANNER-01",
	}
	bodyBox, _ := json.Marshal(scanBoxPayload)
	reqScan2 := httptest.NewRequest(http.MethodPost, scanUrl, bytes.NewBuffer(bodyBox))
	reqScan2.Header.Set("Authorization", "Bearer "+token)
	reqScan2.Header.Set("Content-Type", "application/json")
	wScan2 := httptest.NewRecorder()
	r.ServeHTTP(wScan2, reqScan2)

	if wScan2.Code != http.StatusOK {
		t.Fatalf("expected scan 2 status 200, got %d: %s", wScan2.Code, wScan2.Body.String())
	}

	var scan2Res struct {
		Data inventory.StockOpnameScanResponse `json:"data"`
	}
	json.Unmarshal(wScan2.Body.Bytes(), &scan2Res)
	// Base quantity harus 40
	if scan2Res.Data.BaseQuantityCounted != 40 {
		t.Fatalf("expected base quantity 40 for 1 Dus, got %d", scan2Res.Data.BaseQuantityCounted)
	}

	// 4. Submit opname
	submitUrl := fmt.Sprintf("/api/v1/inventory/opnames/%s/submit", opnameUUID)
	reqSubmit := httptest.NewRequest(http.MethodPost, submitUrl, nil)
	reqSubmit.Header.Set("Authorization", "Bearer "+token)
	wSubmit := httptest.NewRecorder()
	r.ServeHTTP(wSubmit, reqSubmit)

	if wSubmit.Code != http.StatusOK {
		t.Fatalf("expected submit status 200, got %d: %s", wSubmit.Code, wSubmit.Body.String())
	}

	// 5. Supervisor approves opname
	approveUrl := fmt.Sprintf("/api/v1/inventory/opnames/%s/approve", opnameUUID)
	reqApprove := httptest.NewRequest(http.MethodPost, approveUrl, nil)
	reqApprove.Header.Set("Authorization", "Bearer "+token)
	wApprove := httptest.NewRecorder()
	r.ServeHTTP(wApprove, reqApprove)

	if wApprove.Code != http.StatusOK {
		t.Fatalf("expected approve status 200, got %d: %s", wApprove.Code, wApprove.Body.String())
	}

	// Total fisik: 5 + 40 = 45. Sistem awal = 10. Selisih = +35.
	// Stock di database harus terupdate menjadi 45.
	finalStock, _ := repo.GetStock(context.Background(), 1, loc.ID, mockVariant.ID)
	if finalStock.Quantity != 45 {
		t.Fatalf("expected adjusted stock to be 45, got %d", finalStock.Quantity)
	}
}

func TestSupplierAndPurchaseOrderReceiving(t *testing.T) {
	ctx := context.Background()
	repo := newMockInventoryRepo()
	uc := inventory.NewUsecase(repo)

	loc := model.InventoryLocation{ID: 1, UUID: uuid.New(), TenantID: 1, OutletID: 1, Name: "Gudang Utama", Code: "WH-01", Type: "WAREHOUSE"}
	variant := model.ProductVariant{ID: 1, UUID: uuid.New(), TenantID: 1, SKU: "SKU-PO", VariantName: "Beras 5kg"}
	repo.locations = append(repo.locations, loc)
	repo.variants = append(repo.variants, variant)

	sup, err := uc.CreateSupplier(ctx, 1, inventory.CreateSupplierRequest{
		Code: "SUP-01",
		Name: "PT Sumber Pangan",
	})
	if err != nil {
		t.Fatalf("CreateSupplier failed: %v", err)
	}

	po, err := uc.CreatePurchaseOrder(ctx, 1, 1, inventory.CreatePurchaseOrderRequest{
		OutletID:                1,
		SupplierUUID:            sup.UUID,
		DestinationLocationUUID: loc.UUID,
		OrderDate:               "2026-11-25",
		Items: []inventory.CreatePurchaseOrderItemRequest{
			{VariantUUID: variant.UUID, QuantityOrdered: 50, UnitCost: 60000},
		},
	})
	if err != nil || po.TotalAmount != 3000000 {
		t.Fatalf("CreatePurchaseOrder failed: err=%v total=%d", err, po.TotalAmount)
	}

	// Partial receive 20
	poPartial, err := uc.ReceivePurchaseOrder(ctx, 1, 1, po.UUID, inventory.ReceivePurchaseOrderRequest{
		Items: []inventory.ReceivePurchaseOrderItemRequest{
			{ItemUUID: po.Items[0].UUID, QuantityReceived: 20},
		},
	})
	if err != nil || poPartial.Status != "PARTIAL_RECEIVED" {
		t.Fatalf("expected PARTIAL_RECEIVED, got %v / %s", err, poPartial.Status)
	}

	// Receive remaining 30
	poFull, err := uc.ReceivePurchaseOrder(ctx, 1, 1, po.UUID, inventory.ReceivePurchaseOrderRequest{
		Items: []inventory.ReceivePurchaseOrderItemRequest{
			{ItemUUID: po.Items[0].UUID, QuantityReceived: 30},
		},
	})
	if err != nil || poFull.Status != "RECEIVED" {
		t.Fatalf("expected RECEIVED, got %v / %s", err, poFull.Status)
	}

	stk, _ := repo.GetStock(ctx, 1, loc.ID, variant.ID)
	if stk.Quantity != 50 {
		t.Fatalf("expected stock 50 after PO receive, got %d", stk.Quantity)
	}
}
