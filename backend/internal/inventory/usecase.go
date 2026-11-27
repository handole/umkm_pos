package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
)

// Usecase defines the business logic contract for inventory operations.
type Usecase interface {
	// Locations
	CreateLocation(ctx context.Context, tenantID uint64, req CreateLocationRequest) (*LocationResponse, error)
	ListLocations(ctx context.Context, tenantID uint64, outletID *uint64) ([]LocationResponse, error)

	// Stock Balances and Ledger
	GetStocks(ctx context.Context, tenantID uint64, locationID *uint64, lowStockOnly bool, limit, offset int) ([]StockResponse, int64, error)
	GetStockMovements(ctx context.Context, tenantID uint64, filter StockMovementFilterQuery) ([]StockMovementResponse, int64, error)

	// Stock Transfers
	CreateTransfer(ctx context.Context, tenantID, userID uint64, req CreateStockTransferRequest) (*StockTransferResponse, error)
	ReceiveTransfer(ctx context.Context, tenantID, userID uint64, transferUUID uuid.UUID, req ReceiveTransferRequest) (*StockTransferResponse, error)
	GetTransfer(ctx context.Context, tenantID uint64, transferUUID uuid.UUID) (*StockTransferResponse, error)
	ListTransfers(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]StockTransferResponse, int64, error)

	// Stock Opname
	CreateOpname(ctx context.Context, tenantID, userID uint64, req CreateStockOpnameRequest) (*StockOpnameResponse, error)
	GetOpname(ctx context.Context, tenantID uint64, opnameUUID uuid.UUID) (*StockOpnameResponse, error)
	ListOpnames(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]StockOpnameResponse, int64, error)
	ScanOpnameBarcode(ctx context.Context, tenantID, userID uint64, opnameUUID uuid.UUID, req ScanOpnameBarcodeRequest) (*StockOpnameScanResponse, error)
	SubmitOpname(ctx context.Context, tenantID uint64, opnameUUID uuid.UUID) (*StockOpnameResponse, error)
	ApproveOpname(ctx context.Context, tenantID, userID uint64, opnameUUID uuid.UUID) (*StockOpnameResponse, error)

	// Manual Adjustment
	ManualAdjustment(ctx context.Context, tenantID, userID uint64, req ManualAdjustmentRequest) (*StockMovementResponse, error)

	// Suppliers & Purchase Orders
	CreateSupplier(ctx context.Context, tenantID uint64, req CreateSupplierRequest) (*model.Supplier, error)
	ListSuppliers(ctx context.Context, tenantID uint64) ([]model.Supplier, error)
	CreatePurchaseOrder(ctx context.Context, tenantID, userID uint64, req CreatePurchaseOrderRequest) (*model.PurchaseOrder, error)
	GetPurchaseOrder(ctx context.Context, tenantID uint64, poUUID uuid.UUID) (*model.PurchaseOrder, error)
	ListPurchaseOrders(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.PurchaseOrder, int64, error)
	ReceivePurchaseOrder(ctx context.Context, tenantID, userID uint64, poUUID uuid.UUID, req ReceivePurchaseOrderRequest) (*model.PurchaseOrder, error)
}

type usecase struct {
	repo Repository
}

// NewUsecase returns an instance of inventory Usecase.
func NewUsecase(repo Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) CreateLocation(ctx context.Context, tenantID uint64, req CreateLocationRequest) (*LocationResponse, error) {
	loc := model.InventoryLocation{
		TenantID: tenantID,
		OutletID: req.OutletID,
		Code:     strings.ToUpper(strings.TrimSpace(req.Code)),
		Name:     strings.TrimSpace(req.Name),
		Type:     req.Type,
		IsActive: true,
	}

	if err := u.repo.CreateLocation(ctx, &loc); err != nil {
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	return &LocationResponse{
		ID:        loc.ID,
		UUID:      loc.UUID,
		OutletID:  loc.OutletID,
		Code:      loc.Code,
		Name:      loc.Name,
		Type:      loc.Type,
		IsActive:  loc.IsActive,
		CreatedAt: loc.CreatedAt,
	}, nil
}

func (u *usecase) ListLocations(ctx context.Context, tenantID uint64, outletID *uint64) ([]LocationResponse, error) {
	locations, err := u.repo.ListLocations(ctx, tenantID, outletID)
	if err != nil {
		return nil, err
	}

	res := make([]LocationResponse, len(locations))
	for i, l := range locations {
		res[i] = LocationResponse{
			ID:        l.ID,
			UUID:      l.UUID,
			OutletID:  l.OutletID,
			Code:      l.Code,
			Name:      l.Name,
			Type:      l.Type,
			IsActive:  l.IsActive,
			CreatedAt: l.CreatedAt,
		}
	}
	return res, nil
}

func (u *usecase) GetStocks(ctx context.Context, tenantID uint64, locationID *uint64, lowStockOnly bool, limit, offset int) ([]StockResponse, int64, error) {
	stocks, total, err := u.repo.ListStocks(ctx, tenantID, locationID, lowStockOnly, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	res := make([]StockResponse, len(stocks))
	for i, s := range stocks {
		locationName := ""
		if s.Location != nil {
			locationName = s.Location.Name
		}
		variantSKU := ""
		variantName := ""
		productName := ""
		if s.Variant != nil {
			variantSKU = s.Variant.SKU
			variantName = s.Variant.VariantName
			if s.Variant.Product != nil {
				productName = s.Variant.Product.Name
			}
		}

		res[i] = StockResponse{
			ID:           s.ID,
			UUID:         s.UUID,
			LocationID:   s.LocationID,
			LocationName: locationName,
			VariantID:    s.VariantID,
			VariantSKU:   variantSKU,
			VariantName:  variantName,
			ProductName:  productName,
			Quantity:     s.Quantity,
			UpdatedAt:    s.UpdatedAt,
		}
	}
	return res, total, nil
}

func (u *usecase) GetStockMovements(ctx context.Context, tenantID uint64, filter StockMovementFilterQuery) ([]StockMovementResponse, int64, error) {
	movements, total, err := u.repo.ListMovements(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}

	res := make([]StockMovementResponse, len(movements))
	for i, m := range movements {
		locationName := ""
		if m.Location != nil {
			locationName = m.Location.Name
		}
		variantSKU := ""
		variantName := ""
		if m.Variant != nil {
			variantSKU = m.Variant.SKU
			variantName = m.Variant.VariantName
		}

		res[i] = StockMovementResponse{
			ID:              m.ID,
			UUID:            m.UUID,
			LocationID:      m.LocationID,
			LocationName:    locationName,
			VariantID:       m.VariantID,
			VariantSKU:      variantSKU,
			VariantName:     variantName,
			MovementType:    m.MovementType,
			QuantityChange:  m.QuantityChange,
			BalanceAfter:    m.BalanceAfter,
			CostPrice:       m.CostPrice,
			ReferenceNumber: m.ReferenceNumber,
			Notes:           m.Notes,
			CreatedAt:       m.CreatedAt,
		}
	}
	return res, total, nil
}

func (u *usecase) CreateTransfer(ctx context.Context, tenantID, userID uint64, req CreateStockTransferRequest) (*StockTransferResponse, error) {
	if req.SourceLocationUUID == req.DestinationLocationUUID {
		return nil, errors.New("source and destination location must be different")
	}

	sourceLoc, err := u.repo.FindLocationByUUID(ctx, tenantID, req.SourceLocationUUID)
	if err != nil {
		return nil, fmt.Errorf("source location not found: %w", err)
	}

	destLoc, err := u.repo.FindLocationByUUID(ctx, tenantID, req.DestinationLocationUUID)
	if err != nil {
		return nil, fmt.Errorf("destination location not found: %w", err)
	}

	transferNumber := fmt.Sprintf("TRF-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	var transfer model.StockTransfer
	responseItems := make([]StockTransferItemResponse, 0, len(req.Items))

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		transfer = model.StockTransfer{
			TenantID:              tenantID,
			TransferNumber:        transferNumber,
			SourceLocationID:      sourceLoc.ID,
			DestinationLocationID: destLoc.ID,
			Status:                "IN_TRANSIT",
			CreatedBy:             userID,
			Notes:                 req.Notes,
		}

		if err := txRepo.CreateTransfer(ctx, &transfer); err != nil {
			return fmt.Errorf("failed to create stock transfer: %w", err)
		}

		for _, itemReq := range req.Items {
			variant, err := txRepo.FindVariantByUUID(ctx, tenantID, itemReq.VariantUUID)
			if err != nil {
				return fmt.Errorf("variant %s not found: %w", itemReq.VariantUUID, err)
			}

			// Pessimistic lock on source stock
			sourceStock, err := txRepo.GetStockWithLock(ctx, tenantID, sourceLoc.ID, variant.ID)
			if err != nil {
				return fmt.Errorf("failed to acquire source stock lock: %w", err)
			}

			if sourceStock.Quantity < itemReq.QuantitySent {
				return fmt.Errorf("insufficient stock for variant %s: available %d, requested %d", variant.SKU, sourceStock.Quantity, itemReq.QuantitySent)
			}

			// Deduct stock from source location
			sourceStock.Quantity -= itemReq.QuantitySent
			if err := txRepo.SaveStock(ctx, sourceStock); err != nil {
				return fmt.Errorf("failed to deduct source stock: %w", err)
			}

			// Append immutable stock movement ledger
			movement := model.StockMovement{
				TenantID:        tenantID,
				LocationID:      sourceLoc.ID,
				VariantID:       variant.ID,
				MovementType:    "TRANSFER_OUT",
				QuantityChange:  -itemReq.QuantitySent,
				BalanceAfter:    sourceStock.Quantity,
				CostPrice:       variant.BaseCostPrice,
				ReferenceNumber: transferNumber,
				Notes:           fmt.Sprintf("Transfer to %s (%s)", destLoc.Name, destLoc.Code),
				CreatedBy:       &userID,
			}
			if err := txRepo.RecordMovement(ctx, &movement); err != nil {
				return fmt.Errorf("failed to record transfer movement: %w", err)
			}

			// Create line item record
			item := model.StockTransferItem{
				TransferID:       transfer.ID,
				VariantID:        variant.ID,
				QuantitySent:     itemReq.QuantitySent,
				QuantityReceived: 0,
			}
			if err := txRepo.CreateTransferItem(ctx, &item); err != nil {
				return fmt.Errorf("failed to save transfer line item: %w", err)
			}

			responseItems = append(responseItems, StockTransferItemResponse{
				ID:               item.ID,
				UUID:             item.UUID,
				VariantID:        variant.ID,
				VariantSKU:       variant.SKU,
				VariantName:      variant.VariantName,
				QuantitySent:     item.QuantitySent,
				QuantityReceived: item.QuantityReceived,
			})
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &StockTransferResponse{
		ID:                      transfer.ID,
		UUID:                    transfer.UUID,
		TransferNumber:          transfer.TransferNumber,
		SourceLocationID:        sourceLoc.ID,
		SourceLocationName:      sourceLoc.Name,
		DestinationLocationID:   destLoc.ID,
		DestinationLocationName: destLoc.Name,
		Status:                  transfer.Status,
		Notes:                   transfer.Notes,
		CreatedAt:               transfer.CreatedAt,
		Items:                   responseItems,
	}, nil
}

func (u *usecase) ReceiveTransfer(ctx context.Context, tenantID, userID uint64, transferUUID uuid.UUID, req ReceiveTransferRequest) (*StockTransferResponse, error) {
	transfer, err := u.repo.FindTransferByUUID(ctx, tenantID, transferUUID)
	if err != nil {
		return nil, fmt.Errorf("transfer not found: %w", err)
	}

	if transfer.Status != "IN_TRANSIT" {
		return nil, fmt.Errorf("transfer cannot be received because status is '%s'", transfer.Status)
	}

	receivedMap := make(map[uuid.UUID]int64)
	for _, it := range req.Items {
		receivedMap[it.VariantUUID] = it.QuantityReceived
	}

	responseItems := make([]StockTransferItemResponse, 0, len(transfer.Items))

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		for i := range transfer.Items {
			item := &transfer.Items[i]
			variant := item.Variant
			if variant == nil {
				v, err := txRepo.FindVariantByUUID(ctx, tenantID, item.UUID)
				if err != nil {
					return fmt.Errorf("variant for item not found: %w", err)
				}
				variant = v
			}

			receivedQty, ok := receivedMap[variant.UUID]
			if !ok {
				receivedQty = item.QuantitySent
			}

			// Pessimistic lock on destination stock
			destStock, err := txRepo.GetStockWithLock(ctx, tenantID, transfer.DestinationLocationID, variant.ID)
			if err != nil {
				return fmt.Errorf("failed to lock destination stock: %w", err)
			}

			// Increase stock at destination location
			destStock.Quantity += receivedQty
			if err := txRepo.SaveStock(ctx, destStock); err != nil {
				return fmt.Errorf("failed to save destination stock: %w", err)
			}

			// Append immutable stock movement ledger
			movement := model.StockMovement{
				TenantID:        tenantID,
				LocationID:      transfer.DestinationLocationID,
				VariantID:       variant.ID,
				MovementType:    "TRANSFER_IN",
				QuantityChange:  receivedQty,
				BalanceAfter:    destStock.Quantity,
				CostPrice:       variant.BaseCostPrice,
				ReferenceNumber: transfer.TransferNumber,
				Notes:           fmt.Sprintf("Received from transfer %s", transfer.TransferNumber),
				CreatedBy:       &userID,
			}
			if err := txRepo.RecordMovement(ctx, &movement); err != nil {
				return fmt.Errorf("failed to record transfer movement: %w", err)
			}

			item.QuantityReceived = receivedQty
			if err := txRepo.UpdateTransferItem(ctx, item); err != nil {
				return fmt.Errorf("failed to update transfer item: %w", err)
			}

			responseItems = append(responseItems, StockTransferItemResponse{
				ID:               item.ID,
				UUID:             item.UUID,
				VariantID:        variant.ID,
				VariantSKU:       variant.SKU,
				VariantName:      variant.VariantName,
				QuantitySent:     item.QuantitySent,
				QuantityReceived: item.QuantityReceived,
			})
		}

		transfer.Status = "RECEIVED"
		transfer.ReceivedBy = &userID
		if err := txRepo.UpdateTransfer(ctx, transfer); err != nil {
			return fmt.Errorf("failed to update transfer status: %w", err)
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	sourceName := ""
	if transfer.SourceLocation != nil {
		sourceName = transfer.SourceLocation.Name
	}
	destName := ""
	if transfer.DestinationLocation != nil {
		destName = transfer.DestinationLocation.Name
	}

	return &StockTransferResponse{
		ID:                      transfer.ID,
		UUID:                    transfer.UUID,
		TransferNumber:          transfer.TransferNumber,
		SourceLocationID:        transfer.SourceLocationID,
		SourceLocationName:      sourceName,
		DestinationLocationID:   transfer.DestinationLocationID,
		DestinationLocationName: destName,
		Status:                  transfer.Status,
		Notes:                   transfer.Notes,
		CreatedAt:               transfer.CreatedAt,
		Items:                   responseItems,
	}, nil
}

func (u *usecase) GetTransfer(ctx context.Context, tenantID uint64, transferUUID uuid.UUID) (*StockTransferResponse, error) {
	transfer, err := u.repo.FindTransferByUUID(ctx, tenantID, transferUUID)
	if err != nil {
		return nil, err
	}

	items := make([]StockTransferItemResponse, len(transfer.Items))
	for i, it := range transfer.Items {
		sku := ""
		vname := ""
		if it.Variant != nil {
			sku = it.Variant.SKU
			vname = it.Variant.VariantName
		}
		items[i] = StockTransferItemResponse{
			ID:               it.ID,
			UUID:             it.UUID,
			VariantID:        it.VariantID,
			VariantSKU:       sku,
			VariantName:      vname,
			QuantitySent:     it.QuantitySent,
			QuantityReceived: it.QuantityReceived,
		}
	}

	sourceName := ""
	if transfer.SourceLocation != nil {
		sourceName = transfer.SourceLocation.Name
	}
	destName := ""
	if transfer.DestinationLocation != nil {
		destName = transfer.DestinationLocation.Name
	}

	return &StockTransferResponse{
		ID:                      transfer.ID,
		UUID:                    transfer.UUID,
		TransferNumber:          transfer.TransferNumber,
		SourceLocationID:        transfer.SourceLocationID,
		SourceLocationName:      sourceName,
		DestinationLocationID:   transfer.DestinationLocationID,
		DestinationLocationName: destName,
		Status:                  transfer.Status,
		Notes:                   transfer.Notes,
		CreatedAt:               transfer.CreatedAt,
		Items:                   items,
	}, nil
}

func (u *usecase) ListTransfers(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]StockTransferResponse, int64, error) {
	transfers, total, err := u.repo.ListTransfers(ctx, tenantID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	res := make([]StockTransferResponse, len(transfers))
	for i, t := range transfers {
		sourceName := ""
		if t.SourceLocation != nil {
			sourceName = t.SourceLocation.Name
		}
		destName := ""
		if t.DestinationLocation != nil {
			destName = t.DestinationLocation.Name
		}

		items := make([]StockTransferItemResponse, len(t.Items))
		for j, it := range t.Items {
			sku := ""
			vname := ""
			if it.Variant != nil {
				sku = it.Variant.SKU
				vname = it.Variant.VariantName
			}
			items[j] = StockTransferItemResponse{
				ID:               it.ID,
				UUID:             it.UUID,
				VariantID:        it.VariantID,
				VariantSKU:       sku,
				VariantName:      vname,
				QuantitySent:     it.QuantitySent,
				QuantityReceived: it.QuantityReceived,
			}
		}

		res[i] = StockTransferResponse{
			ID:                      t.ID,
			UUID:                    t.UUID,
			TransferNumber:          t.TransferNumber,
			SourceLocationID:        t.SourceLocationID,
			SourceLocationName:      sourceName,
			DestinationLocationID:   t.DestinationLocationID,
			DestinationLocationName: destName,
			Status:                  t.Status,
			Notes:                   t.Notes,
			CreatedAt:               t.CreatedAt,
			Items:                   items,
		}
	}
	return res, total, nil
}

func (u *usecase) CreateOpname(ctx context.Context, tenantID, userID uint64, req CreateStockOpnameRequest) (*StockOpnameResponse, error) {
	loc, err := u.repo.FindLocationByUUID(ctx, tenantID, req.LocationUUID)
	if err != nil {
		return nil, fmt.Errorf("location not found: %w", err)
	}

	opnameNumber := fmt.Sprintf("OPN-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	opname := model.StockOpname{
		TenantID:     tenantID,
		LocationID:   loc.ID,
		OpnameNumber: opnameNumber,
		Status:       "DRAFT",
		PerformedBy:  userID,
		Notes:        req.Notes,
	}

	if err := u.repo.CreateOpname(ctx, &opname); err != nil {
		return nil, fmt.Errorf("failed to create stock opname: %w", err)
	}

	return &StockOpnameResponse{
		ID:           opname.ID,
		UUID:         opname.UUID,
		OpnameNumber: opname.OpnameNumber,
		LocationID:   loc.ID,
		LocationName: loc.Name,
		Status:       opname.Status,
		Notes:        opname.Notes,
		CreatedAt:    opname.CreatedAt,
		Items:        []StockOpnameItemResponse{},
	}, nil
}

func (u *usecase) GetOpname(ctx context.Context, tenantID uint64, opnameUUID uuid.UUID) (*StockOpnameResponse, error) {
	opname, err := u.repo.FindOpnameByUUID(ctx, tenantID, opnameUUID)
	if err != nil {
		return nil, err
	}

	locationName := ""
	if opname.Location != nil {
		locationName = opname.Location.Name
	}

	items := make([]StockOpnameItemResponse, len(opname.Items))
	for i, it := range opname.Items {
		sku := ""
		vname := ""
		if it.Variant != nil {
			sku = it.Variant.SKU
			vname = it.Variant.VariantName
		}
		items[i] = StockOpnameItemResponse{
			ID:                 it.ID,
			UUID:               it.UUID,
			VariantID:          it.VariantID,
			VariantSKU:         sku,
			VariantName:        vname,
			SystemQuantity:     it.SystemQuantity,
			PhysicalQuantity:   it.PhysicalQuantity,
			DifferenceQuantity: it.DifferenceQuantity,
			CostPrice:          it.CostPrice,
			Notes:              it.Notes,
		}
	}

	scans := make([]StockOpnameScanResponse, len(opname.Scans))
	for i, sc := range opname.Scans {
		sku := ""
		vname := ""
		if sc.Variant != nil {
			sku = sc.Variant.SKU
			vname = sc.Variant.VariantName
		}
		scans[i] = StockOpnameScanResponse{
			ID:                  sc.ID,
			UUID:                sc.UUID,
			ScannedBarcode:      sc.ScannedBarcode,
			VariantSKU:          sku,
			VariantName:         vname,
			QuantityCounted:     sc.QuantityCounted,
			BaseQuantityCounted: sc.BaseQuantityCounted,
			ScannedAt:           sc.ScannedAt,
			DeviceID:            sc.DeviceID,
		}
	}

	return &StockOpnameResponse{
		ID:           opname.ID,
		UUID:         opname.UUID,
		OpnameNumber: opname.OpnameNumber,
		LocationID:   opname.LocationID,
		LocationName: locationName,
		Status:       opname.Status,
		Notes:        opname.Notes,
		CreatedAt:    opname.CreatedAt,
		Items:        items,
		Scans:        scans,
	}, nil
}

func (u *usecase) ListOpnames(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]StockOpnameResponse, int64, error) {
	opnames, total, err := u.repo.ListOpnames(ctx, tenantID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	res := make([]StockOpnameResponse, len(opnames))
	for i, o := range opnames {
		locName := ""
		if o.Location != nil {
			locName = o.Location.Name
		}
		res[i] = StockOpnameResponse{
			ID:           o.ID,
			UUID:         o.UUID,
			OpnameNumber: o.OpnameNumber,
			LocationID:   o.LocationID,
			LocationName: locName,
			Status:       o.Status,
			Notes:        o.Notes,
			CreatedAt:    o.CreatedAt,
			Items:        []StockOpnameItemResponse{},
		}
	}
	return res, total, nil
}

func (u *usecase) ScanOpnameBarcode(ctx context.Context, tenantID, userID uint64, opnameUUID uuid.UUID, req ScanOpnameBarcodeRequest) (*StockOpnameScanResponse, error) {
	opname, err := u.repo.FindOpnameByUUID(ctx, tenantID, opnameUUID)
	if err != nil {
		return nil, fmt.Errorf("opname session not found: %w", err)
	}

	if opname.Status != "DRAFT" {
		return nil, fmt.Errorf("cannot scan for opname with status '%s'", opname.Status)
	}

	// Dual-barcode lookup: retail sachet barcode or outer box UOM barcode
	variant, uom, multiplier, err := u.repo.FindBarcode(ctx, tenantID, strings.TrimSpace(req.Barcode))
	if err != nil {
		return nil, fmt.Errorf("barcode '%s' not registered in tenant: %w", req.Barcode, err)
	}

	baseQuantity := req.Quantity * int64(multiplier)

	var uomID *uint64
	if uom != nil {
		uomID = &uom.ID
	}

	var scanRecord model.StockOpnameScan

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		scanRecord = model.StockOpnameScan{
			OpnameID:            opname.ID,
			VariantID:           variant.ID,
			UOMID:               uomID,
			ScannedBarcode:      req.Barcode,
			QuantityCounted:     req.Quantity,
			BaseQuantityCounted: baseQuantity,
			ScannedBy:           userID,
			DeviceID:            req.DeviceID,
		}

		if err := txRepo.RecordOpnameScan(ctx, &scanRecord); err != nil {
			return fmt.Errorf("failed to record scan log: %w", err)
		}

		existingItem, errItem := txRepo.FindOpnameItem(ctx, opname.ID, variant.ID)
		if errItem != nil || existingItem == nil {
			// First scan for this variant in this opname session: inspect current system stock
			systemStock, _ := txRepo.GetStock(ctx, tenantID, opname.LocationID, variant.ID)
			systemQty := int64(0)
			if systemStock != nil {
				systemQty = systemStock.Quantity
			}

			opnameItem := model.StockOpnameItem{
				OpnameID:           opname.ID,
				VariantID:          variant.ID,
				SystemQuantity:     systemQty,
				PhysicalQuantity:   baseQuantity,
				DifferenceQuantity: baseQuantity - systemQty,
				CostPrice:          variant.BaseCostPrice,
			}
			if err := txRepo.SaveOpnameItem(ctx, &opnameItem); err != nil {
				return fmt.Errorf("failed to create opname item summary: %w", err)
			}
		} else {
			existingItem.PhysicalQuantity += baseQuantity
			existingItem.DifferenceQuantity = existingItem.PhysicalQuantity - existingItem.SystemQuantity
			if err := txRepo.SaveOpnameItem(ctx, existingItem); err != nil {
				return fmt.Errorf("failed to update opname item summary: %w", err)
			}
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &StockOpnameScanResponse{
		ID:                  scanRecord.ID,
		UUID:                scanRecord.UUID,
		ScannedBarcode:      scanRecord.ScannedBarcode,
		VariantSKU:          variant.SKU,
		VariantName:         variant.VariantName,
		QuantityCounted:     scanRecord.QuantityCounted,
		BaseQuantityCounted: scanRecord.BaseQuantityCounted,
		ScannedAt:           scanRecord.ScannedAt,
		DeviceID:            scanRecord.DeviceID,
	}, nil
}

func (u *usecase) SubmitOpname(ctx context.Context, tenantID uint64, opnameUUID uuid.UUID) (*StockOpnameResponse, error) {
	opname, err := u.repo.FindOpnameByUUID(ctx, tenantID, opnameUUID)
	if err != nil {
		return nil, err
	}

	if opname.Status != "DRAFT" {
		return nil, fmt.Errorf("cannot submit opname with status '%s'", opname.Status)
	}

	opname.Status = "SUBMITTED"
	if err := u.repo.UpdateOpname(ctx, opname); err != nil {
		return nil, err
	}

	return u.GetOpname(ctx, tenantID, opnameUUID)
}

func (u *usecase) ApproveOpname(ctx context.Context, tenantID, userID uint64, opnameUUID uuid.UUID) (*StockOpnameResponse, error) {
	opname, err := u.repo.FindOpnameByUUID(ctx, tenantID, opnameUUID)
	if err != nil {
		return nil, err
	}

	if opname.Status != "SUBMITTED" && opname.Status != "DRAFT" {
		return nil, fmt.Errorf("opname already finalized with status '%s'", opname.Status)
	}

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		for _, item := range opname.Items {
			if item.DifferenceQuantity == 0 {
				continue
			}

			// Pessimistic lock stock row
			stock, err := txRepo.GetStockWithLock(ctx, tenantID, opname.LocationID, item.VariantID)
			if err != nil {
				return fmt.Errorf("failed to lock stock for opname adjustment: %w", err)
			}

			// Overwrite stock balance with confirmed physical count
			stock.Quantity = item.PhysicalQuantity
			if err := txRepo.SaveStock(ctx, stock); err != nil {
				return fmt.Errorf("failed to adjust stock balance: %w", err)
			}

			// Append immutable stock movement ledger
			movement := model.StockMovement{
				TenantID:        tenantID,
				LocationID:      opname.LocationID,
				VariantID:       item.VariantID,
				MovementType:    "ADJUSTMENT_OPNAME",
				QuantityChange:  item.DifferenceQuantity,
				BalanceAfter:    item.PhysicalQuantity,
				CostPrice:       item.CostPrice,
				ReferenceNumber: opname.OpnameNumber,
				Notes:           fmt.Sprintf("Stock opname audit adjustment (%+d)", item.DifferenceQuantity),
				CreatedBy:       &userID,
			}
			if err := txRepo.RecordMovement(ctx, &movement); err != nil {
				return fmt.Errorf("failed to record opname ledger adjustment: %w", err)
			}
		}

		opname.Status = "APPROVED"
		opname.ApprovedBy = &userID
		if err := txRepo.UpdateOpname(ctx, opname); err != nil {
			return fmt.Errorf("failed to approve opname: %w", err)
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return u.GetOpname(ctx, tenantID, opnameUUID)
}

func (u *usecase) ManualAdjustment(ctx context.Context, tenantID, userID uint64, req ManualAdjustmentRequest) (*StockMovementResponse, error) {
	loc, err := u.repo.FindLocationByUUID(ctx, tenantID, req.LocationUUID)
	if err != nil {
		return nil, fmt.Errorf("location not found: %w", err)
	}

	variant, err := u.repo.FindVariantByUUID(ctx, tenantID, req.VariantUUID)
	if err != nil {
		return nil, fmt.Errorf("variant not found: %w", err)
	}

	var movement model.StockMovement

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		stock, err := txRepo.GetStockWithLock(ctx, tenantID, loc.ID, variant.ID)
		if err != nil {
			return fmt.Errorf("failed to lock stock: %w", err)
		}

		newQuantity := stock.Quantity + req.QuantityChange
		if newQuantity < 0 {
			return fmt.Errorf("stock cannot be negative: current %d, change %d", stock.Quantity, req.QuantityChange)
		}

		stock.Quantity = newQuantity
		if err := txRepo.SaveStock(ctx, stock); err != nil {
			return fmt.Errorf("failed to update stock: %w", err)
		}

		costPrice := req.CostPrice
		if costPrice == 0 {
			costPrice = variant.BaseCostPrice
		}

		adjNumber := fmt.Sprintf("ADJ-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
		movement = model.StockMovement{
			TenantID:        tenantID,
			LocationID:      loc.ID,
			VariantID:       variant.ID,
			MovementType:    "MANUAL_ADJUSTMENT",
			QuantityChange:  req.QuantityChange,
			BalanceAfter:    stock.Quantity,
			CostPrice:       costPrice,
			ReferenceNumber: adjNumber,
			Notes:           req.Notes,
			CreatedBy:       &userID,
		}

		if err := txRepo.RecordMovement(ctx, &movement); err != nil {
			return fmt.Errorf("failed to record movement: %w", err)
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &StockMovementResponse{
		ID:              movement.ID,
		UUID:            movement.UUID,
		LocationID:      loc.ID,
		LocationName:    loc.Name,
		VariantID:       variant.ID,
		VariantSKU:      variant.SKU,
		VariantName:     variant.VariantName,
		MovementType:    movement.MovementType,
		QuantityChange:  movement.QuantityChange,
		BalanceAfter:    movement.BalanceAfter,
		CostPrice:       movement.CostPrice,
		ReferenceNumber: movement.ReferenceNumber,
		Notes:           movement.Notes,
		CreatedAt:       movement.CreatedAt,
	}, nil
}

func (u *usecase) CreateSupplier(ctx context.Context, tenantID uint64, req CreateSupplierRequest) (*model.Supplier, error) {
	s := &model.Supplier{
		TenantID:      tenantID,
		Code:          req.Code,
		Name:          req.Name,
		ContactPerson: req.ContactName,
		Phone:         req.Phone,
		Email:       req.Email,
		Address:     req.Address,
		IsActive:    true,
	}
	if err := u.repo.CreateSupplier(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *usecase) ListSuppliers(ctx context.Context, tenantID uint64) ([]model.Supplier, error) {
	return u.repo.ListSuppliers(ctx, tenantID)
}

func (u *usecase) CreatePurchaseOrder(ctx context.Context, tenantID, userID uint64, req CreatePurchaseOrderRequest) (*model.PurchaseOrder, error) {
	supplier, err := u.repo.FindSupplierByUUID(ctx, tenantID, req.SupplierUUID)
	if err != nil {
		return nil, errors.New("supplier not found")
	}
	destLoc, err := u.repo.FindLocationByUUID(ctx, tenantID, req.DestinationLocationUUID)
	if err != nil {
		return nil, errors.New("destination location not found")
	}
	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		return nil, errors.New("invalid order_date format, expected YYYY-MM-DD")
	}
	var expectedDate *time.Time
	if req.ExpectedDate != "" {
		parsed, err := time.Parse("2006-01-02", req.ExpectedDate)
		if err != nil {
			return nil, errors.New("invalid expected_date format, expected YYYY-MM-DD")
		}
		expectedDate = &parsed
	}

	var totalAmount int64
	poItems := make([]model.PurchaseOrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		variant, err := u.repo.FindVariantByUUID(ctx, tenantID, it.VariantUUID)
		if err != nil {
			return nil, fmt.Errorf("variant %s not found", it.VariantUUID)
		}
		sub := it.QuantityOrdered * it.UnitCost
		totalAmount += sub
		poItems = append(poItems, model.PurchaseOrderItem{
			VariantID:       variant.ID,
			QuantityOrdered: it.QuantityOrdered,
			UnitCost:        it.UnitCost,
			Subtotal:        sub,
		})
	}

	po := &model.PurchaseOrder{
		TenantID:              tenantID,
		OutletID:              req.OutletID,
		SupplierID:            supplier.ID,
		DestinationLocationID: destLoc.ID,
		PONumber:              fmt.Sprintf("PO-%s-%s", time.Now().Format("20060102"), strings.ToUpper(uuid.New().String()[:6])),
		Status:                "SUBMITTED",
		OrderDate:             orderDate,
		ExpectedDate:          expectedDate,
		TotalAmount:           totalAmount,
		Notes:                 req.Notes,
		CreatedBy:             userID,
		Items:                 poItems,
	}

	if err := u.repo.CreatePurchaseOrder(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

func (u *usecase) GetPurchaseOrder(ctx context.Context, tenantID uint64, poUUID uuid.UUID) (*model.PurchaseOrder, error) {
	return u.repo.FindPurchaseOrderByUUID(ctx, tenantID, poUUID)
}

func (u *usecase) ListPurchaseOrders(ctx context.Context, tenantID uint64, status *string, limit, offset int) ([]model.PurchaseOrder, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return u.repo.ListPurchaseOrders(ctx, tenantID, status, limit, offset)
}

func (u *usecase) ReceivePurchaseOrder(ctx context.Context, tenantID, userID uint64, poUUID uuid.UUID, req ReceivePurchaseOrderRequest) (*model.PurchaseOrder, error) {
	var updatedPO *model.PurchaseOrder

	err := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		po, err := txRepo.FindPurchaseOrderByUUID(ctx, tenantID, poUUID)
		if err != nil {
			return errors.New("purchase order not found")
		}
		if po.Status == "RECEIVED" || po.Status == "CANCELLED" {
			return fmt.Errorf("purchase order cannot be received in status '%s'", po.Status)
		}

		itemByUUID := make(map[uuid.UUID]*model.PurchaseOrderItem, len(po.Items))
		for i := range po.Items {
			itemByUUID[po.Items[i].UUID] = &po.Items[i]
		}

		for _, rec := range req.Items {
			poItem, ok := itemByUUID[rec.ItemUUID]
			if !ok {
				return fmt.Errorf("purchase order item %s not found", rec.ItemUUID)
			}
			if poItem.QuantityReceived+rec.QuantityReceived > poItem.QuantityOrdered {
				return fmt.Errorf("received quantity exceeds ordered quantity for item %s", rec.ItemUUID)
			}

			poItem.QuantityReceived += rec.QuantityReceived
			if err := txRepo.UpdatePurchaseOrderItem(ctx, poItem); err != nil {
				return err
			}

			stock, err := txRepo.GetStockWithLock(ctx, tenantID, po.DestinationLocationID, poItem.VariantID)
			if err != nil {
				return err
			}
			stock.Quantity += rec.QuantityReceived
			if err := txRepo.SaveStock(ctx, stock); err != nil {
				return err
			}

			movement := model.StockMovement{
				TenantID:        tenantID,
				LocationID:      po.DestinationLocationID,
				VariantID:       poItem.VariantID,
				MovementType:    "IN",
				QuantityChange:  rec.QuantityReceived,
				BalanceAfter:    stock.Quantity,
				CostPrice:       poItem.UnitCost,
				ReferenceNumber: po.PONumber,
				Notes:           fmt.Sprintf("Penerimaan PO %s", po.PONumber),
				CreatedBy:       &userID,
			}
			if err := txRepo.RecordMovement(ctx, &movement); err != nil {
				return err
			}
		}

		allReceived := true
		for _, it := range po.Items {
			if it.QuantityReceived < it.QuantityOrdered {
				allReceived = false
				break
			}
		}
		if allReceived {
			po.Status = "RECEIVED"
		} else {
			po.Status = "PARTIAL_RECEIVED"
		}
		po.ReceivedBy = &userID
		po.UpdatedAt = time.Now().UTC()

		if err := txRepo.UpdatePurchaseOrder(ctx, po); err != nil {
			return err
		}
		updatedPO = po
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updatedPO, nil
}
