package pos

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/handokodenih/umkm_pos/backend/internal/model"
)

// Usecase defines the business logic contract for POS operations.
type Usecase interface {
	// Shifts
	OpenShift(ctx context.Context, tenantID, cashierID uint64, req OpenShiftRequest) (*ShiftResponse, error)
	GetCurrentShift(ctx context.Context, tenantID, cashierID uint64) (*ShiftResponse, error)
	AddCashLog(ctx context.Context, tenantID, cashierID uint64, req CashLogRequest) (*ShiftCashLogResponse, error)
	CloseShift(ctx context.Context, tenantID, cashierID uint64, req CloseShiftRequest) (*ShiftResponse, error)
	ListShifts(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, limit, offset int) ([]ShiftResponse, int64, error)

	// Payment Methods
	ListPaymentMethods(ctx context.Context, tenantID uint64) ([]PaymentMethodResponse, error)

	// Orders & Checkout
	CheckoutOrder(ctx context.Context, tenantID, cashierID uint64, req CheckoutOrderRequest) (*OrderResponse, error)
	GetOrder(ctx context.Context, tenantID uint64, orderUUID uuid.UUID) (*OrderResponse, error)
	ListOrders(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, status *string, limit, offset int) ([]OrderResponse, int64, error)
	GetOrderReceipt(ctx context.Context, tenantID uint64, orderUUID uuid.UUID) (*OrderReceiptResponse, error)
	VoidOrder(ctx context.Context, tenantID, managerUserID uint64, orderUUID uuid.UUID, req VoidOrderRequest) (*OrderResponse, error)

	// Financial & Sales Reports
	GetSalesReport(ctx context.Context, tenantID uint64, query SalesReportQuery) (*SalesReportSummaryResponse, error)
	ExportSalesReportCSV(ctx context.Context, tenantID uint64, query SalesReportQuery) ([]byte, error)
}

type usecase struct {
	repo Repository
}

// NewUsecase returns an instance of POS Usecase.
func NewUsecase(repo Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) OpenShift(ctx context.Context, tenantID, cashierID uint64, req OpenShiftRequest) (*ShiftResponse, error) {
	existingShift, err := u.repo.FindOpenShiftByCashier(ctx, tenantID, cashierID)
	if err != nil {
		return nil, err
	}
	if existingShift != nil {
		return nil, fmt.Errorf("cashier already has an active open shift (#%s)", existingShift.ShiftNumber)
	}

	shiftNumber := fmt.Sprintf("SHF-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	shift := model.PosShift{
		TenantID:      tenantID,
		OutletID:      req.OutletID,
		CashierUserID: cashierID,
		ShiftNumber:   shiftNumber,
		StartTime:     time.Now(),
		StartingCash:  req.StartingCash,
		ExpectedCash:  req.StartingCash,
		Status:        "OPEN",
		Notes:         req.Notes,
	}

	if err := u.repo.CreateShift(ctx, &shift); err != nil {
		return nil, fmt.Errorf("failed to open shift: %w", err)
	}

	cashierUser, _ := u.repo.FindUserByID(ctx, cashierID)
	cashierName := ""
	if cashierUser != nil {
		cashierName = cashierUser.FullName
	}

	return &ShiftResponse{
		ID:            shift.ID,
		UUID:          shift.UUID,
		ShiftNumber:   shift.ShiftNumber,
		OutletID:      shift.OutletID,
		CashierUserID: shift.CashierUserID,
		CashierName:   cashierName,
		StartTime:     shift.StartTime,
		StartingCash:  shift.StartingCash,
		ExpectedCash:  shift.ExpectedCash,
		Status:        shift.Status,
		Notes:         shift.Notes,
	}, nil
}

func (u *usecase) GetCurrentShift(ctx context.Context, tenantID, cashierID uint64) (*ShiftResponse, error) {
	shift, err := u.repo.FindOpenShiftByCashier(ctx, tenantID, cashierID)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, nil
	}

	cashierName := ""
	if shift.Cashier != nil {
		cashierName = shift.Cashier.FullName
	}

	logs := make([]ShiftCashLogResponse, len(shift.CashLogs))
	for i, cl := range shift.CashLogs {
		logs[i] = ShiftCashLogResponse{
			ID:        cl.ID,
			UUID:      cl.UUID,
			Type:      cl.Type,
			Amount:    cl.Amount,
			Reason:    cl.Reason,
			CreatedAt: cl.CreatedAt,
		}
	}

	return &ShiftResponse{
		ID:              shift.ID,
		UUID:            shift.UUID,
		ShiftNumber:     shift.ShiftNumber,
		OutletID:        shift.OutletID,
		CashierUserID:   shift.CashierUserID,
		CashierName:     cashierName,
		StartTime:       shift.StartTime,
		EndTime:         shift.EndTime,
		StartingCash:    shift.StartingCash,
		ExpectedCash:    shift.ExpectedCash,
		ActualCash:      shift.ActualCash,
		CashDiscrepancy: shift.CashDiscrepancy,
		Status:          shift.Status,
		Notes:           shift.Notes,
		CashLogs:        logs,
	}, nil
}

func (u *usecase) AddCashLog(ctx context.Context, tenantID, cashierID uint64, req CashLogRequest) (*ShiftCashLogResponse, error) {
	shift, err := u.repo.FindOpenShiftByCashier(ctx, tenantID, cashierID)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, errors.New("no active open shift found for this cashier")
	}

	var log model.PosShiftCashLog

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		log = model.PosShiftCashLog{
			ShiftID: shift.ID,
			Type:    req.Type,
			Amount:  req.Amount,
			Reason:  strings.TrimSpace(req.Reason),
		}

		if err := txRepo.CreateCashLog(ctx, &log); err != nil {
			return fmt.Errorf("failed to record cash log: %w", err)
		}

		if req.Type == "CASH_IN" {
			shift.ExpectedCash += req.Amount
		} else {
			if shift.ExpectedCash < req.Amount {
				return fmt.Errorf("cash drawer balance (%d) is insufficient for cash out (%d)", shift.ExpectedCash, req.Amount)
			}
			shift.ExpectedCash -= req.Amount
		}

		if err := txRepo.UpdateShift(ctx, shift); err != nil {
			return fmt.Errorf("failed to update shift expected cash: %w", err)
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &ShiftCashLogResponse{
		ID:        log.ID,
		UUID:      log.UUID,
		Type:      log.Type,
		Amount:    log.Amount,
		Reason:    log.Reason,
		CreatedAt: log.CreatedAt,
	}, nil
}

func (u *usecase) CloseShift(ctx context.Context, tenantID, cashierID uint64, req CloseShiftRequest) (*ShiftResponse, error) {
	shift, err := u.repo.FindOpenShiftByCashier(ctx, tenantID, cashierID)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, errors.New("no active open shift found to close")
	}

	now := time.Now()
	discrepancy := req.ActualCash - shift.ExpectedCash

	shift.Status = "CLOSED"
	shift.EndTime = &now
	shift.ActualCash = &req.ActualCash
	shift.CashDiscrepancy = &discrepancy
	shift.Notes = req.Notes

	if err := u.repo.UpdateShift(ctx, shift); err != nil {
		return nil, fmt.Errorf("failed to close shift: %w", err)
	}

	cashierName := ""
	if shift.Cashier != nil {
		cashierName = shift.Cashier.FullName
	}

	return &ShiftResponse{
		ID:              shift.ID,
		UUID:            shift.UUID,
		ShiftNumber:     shift.ShiftNumber,
		OutletID:        shift.OutletID,
		CashierUserID:   shift.CashierUserID,
		CashierName:     cashierName,
		StartTime:       shift.StartTime,
		EndTime:         shift.EndTime,
		StartingCash:    shift.StartingCash,
		ExpectedCash:    shift.ExpectedCash,
		ActualCash:      shift.ActualCash,
		CashDiscrepancy: shift.CashDiscrepancy,
		Status:          shift.Status,
		Notes:           shift.Notes,
	}, nil
}

func (u *usecase) ListShifts(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, limit, offset int) ([]ShiftResponse, int64, error) {
	shifts, total, err := u.repo.ListShifts(ctx, tenantID, outletID, cashierID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	res := make([]ShiftResponse, len(shifts))
	for i, s := range shifts {
		cashierName := ""
		if s.Cashier != nil {
			cashierName = s.Cashier.FullName
		}
		res[i] = ShiftResponse{
			ID:              s.ID,
			UUID:            s.UUID,
			ShiftNumber:     s.ShiftNumber,
			OutletID:        s.OutletID,
			CashierUserID:   s.CashierUserID,
			CashierName:     cashierName,
			StartTime:       s.StartTime,
			EndTime:         s.EndTime,
			StartingCash:    s.StartingCash,
			ExpectedCash:    s.ExpectedCash,
			ActualCash:      s.ActualCash,
			CashDiscrepancy: s.CashDiscrepancy,
			Status:          s.Status,
			Notes:           s.Notes,
		}
	}
	return res, total, nil
}

func (u *usecase) ListPaymentMethods(ctx context.Context, tenantID uint64) ([]PaymentMethodResponse, error) {
	methods, err := u.repo.ListPaymentMethods(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(methods) == 0 {
		_ = u.repo.SeedDefaultPaymentMethods(ctx, tenantID)
		methods, _ = u.repo.ListPaymentMethods(ctx, tenantID)
	}

	res := make([]PaymentMethodResponse, len(methods))
	for i, m := range methods {
		res[i] = PaymentMethodResponse{
			ID:                m.ID,
			UUID:              m.UUID,
			Code:              m.Code,
			Name:              m.Name,
			RequiresReference: m.RequiresReference,
			IsActive:          m.IsActive,
		}
	}
	return res, nil
}

func (u *usecase) CheckoutOrder(ctx context.Context, tenantID, cashierID uint64, req CheckoutOrderRequest) (*OrderResponse, error) {
	// 1. Idempotency Check: prevent duplicate charges when network retry happens
	existingOrder, err := u.repo.FindOrderByTenantAndIdempotency(ctx, tenantID, req.IdempotencyKey)
	if err == nil && existingOrder != nil {
		return u.mapOrderToResponse(existingOrder), nil
	}

	// 2. Resolve physical location where stock is deducted
	var location *model.InventoryLocation
	if req.LocationUUID != nil {
		loc, errLoc := u.repo.FindLocationByUUID(ctx, tenantID, *req.LocationUUID)
		if errLoc != nil {
			return nil, fmt.Errorf("specified stock location not found: %w", errLoc)
		}
		location = loc
	} else {
		loc, errLoc := u.repo.FindDefaultStoreFrontLocation(ctx, tenantID, req.OutletID)
		if errLoc != nil {
			return nil, fmt.Errorf("default store location for outlet %d not found: %w", req.OutletID, errLoc)
		}
		location = loc
	}

	// 3. Resolve active shift for this cashier (if any)
	shift, _ := u.repo.FindOpenShiftByCashier(ctx, tenantID, cashierID)
	var shiftID *uint64
	if shift != nil {
		shiftID = &shift.ID
	}

	orderNumber := fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	var createdOrder model.PosOrder

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		var subtotal int64
		orderItems := make([]model.PosOrderItem, 0, len(req.Items))

		// Process line items with pessimistic locking on stock
		for _, itemReq := range req.Items {
			variant, err := txRepo.FindVariantByUUID(ctx, tenantID, itemReq.VariantUUID)
			if err != nil {
				return fmt.Errorf("product variant %s not found: %w", itemReq.VariantUUID, err)
			}

			baseQuantity := itemReq.Quantity
			unitSellPrice := variant.BaseSellPrice
			var uomID *uint64

			if itemReq.UOMUUID != nil {
				uom, errUOM := txRepo.FindUOMByUUID(ctx, tenantID, *itemReq.UOMUUID)
				if errUOM != nil {
					return fmt.Errorf("packaging UOM not found: %w", errUOM)
				}
				rate := int64(uom.ConversionRate)
				if rate <= 0 {
					rate = 1
				}
				baseQuantity = itemReq.Quantity * rate
				unitSellPrice = uom.SellPrice
				uomID = &uom.ID
			}

			// Pessimistic lock on inventory stock row
			stock, err := txRepo.GetStockWithLock(ctx, tenantID, location.ID, variant.ID)
			if err != nil {
				return fmt.Errorf("failed to acquire stock lock for SKU %s: %w", variant.SKU, err)
			}

			if stock.Quantity < baseQuantity {
				return fmt.Errorf("insufficient stock for '%s' (%s): available %d, requested %d", variant.VariantName, variant.SKU, stock.Quantity, baseQuantity)
			}

			// Deduct stock balance
			stock.Quantity -= baseQuantity
			if err := txRepo.SaveStock(ctx, stock); err != nil {
				return fmt.Errorf("failed to deduct stock for SKU %s: %w", variant.SKU, err)
			}

			// Record immutable stock movement ledger
			movement := model.StockMovement{
				TenantID:        tenantID,
				LocationID:      location.ID,
				VariantID:       variant.ID,
				MovementType:    "OUT_POS_SALE",
				QuantityChange:  -baseQuantity,
				BalanceAfter:    stock.Quantity,
				CostPrice:       variant.BaseCostPrice,
				ReferenceNumber: orderNumber,
				Notes:           fmt.Sprintf("Penjualan POS #%s", orderNumber),
				CreatedBy:       &cashierID,
			}
			if err := txRepo.RecordMovement(ctx, &movement); err != nil {
				return fmt.Errorf("failed to record stock movement: %w", err)
			}

			lineSubtotal := (unitSellPrice * itemReq.Quantity) - itemReq.DiscountAmount
			if lineSubtotal < 0 {
				lineSubtotal = 0
			}
			subtotal += lineSubtotal

			orderItems = append(orderItems, model.PosOrderItem{
				VariantID:      variant.ID,
				UOMID:          uomID,
				Quantity:       itemReq.Quantity,
				BaseQuantity:   baseQuantity,
				UnitCostPrice:  variant.BaseCostPrice,
				UnitSellPrice:  unitSellPrice,
				DiscountAmount: itemReq.DiscountAmount,
				Subtotal:       lineSubtotal,
				Notes:          itemReq.Notes,
			})
		}

		// Calculate total order amount
		totalAmount := subtotal - req.DiscountAmount + req.TaxAmount
		if totalAmount < 0 {
			totalAmount = 0
		}

		// Validate payments and calculate change
		var totalPaid int64
		var cashPaid int64
		orderPayments := make([]model.PosOrderPayment, 0, len(req.Payments))

		for _, p := range req.Payments {
			pm, err := txRepo.FindPaymentMethodByCode(ctx, tenantID, p.PaymentMethodCode)
			if err != nil {
				return fmt.Errorf("payment method '%s' is not supported: %w", p.PaymentMethodCode, err)
			}

			if pm.RequiresReference && strings.TrimSpace(p.ReferenceNumber) == "" {
				return fmt.Errorf("reference number is required for payment method '%s'", pm.Name)
			}

			totalPaid += p.Amount
			if pm.Code == "CASH" {
				cashPaid += p.Amount
			}

			orderPayments = append(orderPayments, model.PosOrderPayment{
				PaymentMethodID: pm.ID,
				Amount:          p.Amount,
				ReferenceNumber: p.ReferenceNumber,
				PaidAt:          time.Now(),
			})
		}

		if totalPaid < totalAmount {
			return fmt.Errorf("insufficient payment: total amount %d, paid %d", totalAmount, totalPaid)
		}

		changeAmount := totalPaid - totalAmount

		// If cash was involved and cashier has an open shift, update expected cash in drawer
		if cashPaid > 0 && shift != nil {
			netCashAdded := cashPaid - changeAmount
			shift.ExpectedCash += netCashAdded
			if err := txRepo.UpdateShift(ctx, shift); err != nil {
				return fmt.Errorf("failed to update shift expected cash: %w", err)
			}
		}

		// Create header order
		createdOrder = model.PosOrder{
			TenantID:                tenantID,
			OutletID:                req.OutletID,
			ShiftID:                 shiftID,
			CashierUserID:           cashierID,
			SalespersonUserID:       req.SalespersonUserID,
			CustomerID:              req.CustomerID,
			LocationID:              location.ID,
			OrderNumber:             orderNumber,
			IdempotencyKey:          req.IdempotencyKey,
			Subtotal:                subtotal,
			DiscountAmount:          req.DiscountAmount,
			PromotionDiscountAmount: 0,
			TaxAmount:               req.TaxAmount,
			TotalAmount:             totalAmount,
			PaidAmount:              totalPaid,
			ChangeAmount:            changeAmount,
			Status:                  "COMPLETED",
			OrderedAt:               time.Now(),
		}

		if err := txRepo.CreateOrder(ctx, &createdOrder); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}

		// Save items
		for i := range orderItems {
			orderItems[i].OrderID = createdOrder.ID
			if err := txRepo.CreateOrderItem(ctx, &orderItems[i]); err != nil {
				return fmt.Errorf("failed to save order line item: %w", err)
			}
		}

		// Save payments
		for i := range orderPayments {
			orderPayments[i].OrderID = createdOrder.ID
			if err := txRepo.CreateOrderPayment(ctx, &orderPayments[i]); err != nil {
				return fmt.Errorf("failed to save order payment: %w", err)
			}
		}

		// Auto-calculate salesperson commission into payroll_commissions_pending
		if req.SalespersonUserID != nil {
			emp, err := txRepo.FindEmployeeByUserID(ctx, tenantID, *req.SalespersonUserID)
			if err == nil && emp != nil && emp.SalaryStructure != nil {
				var totalItemQty int64
				for _, item := range orderItems {
					totalItemQty += int64(item.Quantity)
				}
				rateAmt := int64(math.Round(float64(totalAmount) * (emp.SalaryStructure.CommissionRatePercent / 100.0)))
				perItemAmt := totalItemQty * emp.SalaryStructure.CommissionFixedPerItem
				totalComm := rateAmt + perItemAmt
				if totalComm > 0 {
					comm := &model.PayrollCommissionPending{
						TenantID:         tenantID,
						EmployeeID:       emp.ID,
						OrderID:          createdOrder.ID,
						CommissionAmount: totalComm,
						Status:           "PENDING",
					}
					if err := txRepo.CreatePendingCommission(ctx, comm); err != nil {
						return fmt.Errorf("failed to queue sales commission: %w", err)
					}
				}
			}
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	// Reload full order entity with preloads for complete response
	fullOrder, err := u.repo.FindOrderByUUID(ctx, tenantID, createdOrder.UUID)
	if err == nil && fullOrder != nil {
		return u.mapOrderToResponse(fullOrder), nil
	}

	return u.mapOrderToResponse(&createdOrder), nil
}

func (u *usecase) GetOrder(ctx context.Context, tenantID uint64, orderUUID uuid.UUID) (*OrderResponse, error) {
	order, err := u.repo.FindOrderByUUID(ctx, tenantID, orderUUID)
	if err != nil {
		return nil, err
	}
	return u.mapOrderToResponse(order), nil
}

func (u *usecase) ListOrders(ctx context.Context, tenantID uint64, outletID *uint64, cashierID *uint64, status *string, limit, offset int) ([]OrderResponse, int64, error) {
	orders, total, err := u.repo.ListOrders(ctx, tenantID, outletID, cashierID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	res := make([]OrderResponse, len(orders))
	for i := range orders {
		res[i] = *u.mapOrderToResponse(&orders[i])
	}
	return res, total, nil
}

func (u *usecase) GetOrderReceipt(ctx context.Context, tenantID uint64, orderUUID uuid.UUID) (*OrderReceiptResponse, error) {
	order, err := u.repo.FindOrderByUUID(ctx, tenantID, orderUUID)
	if err != nil {
		return nil, err
	}

	tenant, _ := u.repo.FindTenantByID(ctx, tenantID)
	tenantName := "UMKM POS Store"
	if tenant != nil {
		tenantName = tenant.Name
	}

	outletName := ""
	outletAddress := ""
	outletPhone := ""
	if order.Outlet != nil {
		outletName = order.Outlet.Name
		outletAddress = order.Outlet.Address
		outletPhone = order.Outlet.Phone
	}

	cashierName := ""
	if order.Cashier != nil {
		cashierName = order.Cashier.FullName
	}

	receipt := OrderReceiptResponse{}
	receipt.Header.StoreName = tenantName
	receipt.Header.OutletName = outletName
	receipt.Header.Address = outletAddress
	receipt.Header.Phone = outletPhone
	receipt.Header.OrderNumber = order.OrderNumber
	receipt.Header.CashierName = cashierName
	receipt.Header.DateTime = order.OrderedAt.Format("02/01/2006 15:04:05")

	for _, it := range order.Items {
		name := ""
		if it.Variant != nil {
			name = it.Variant.VariantName
			if it.Variant.Product != nil {
				name = it.Variant.Product.Name + " - " + it.Variant.VariantName
			}
		}
		uomName := "Pcs"
		if it.UOM != nil {
			uomName = it.UOM.UOMName
		}

		receipt.Lines = append(receipt.Lines, struct {
			Name      string `json:"name"`
			Quantity  int64  `json:"quantity"`
			UOM       string `json:"uom"`
			UnitPrice int64  `json:"unit_price"`
			Subtotal  int64  `json:"subtotal"`
		}{
			Name:      name,
			Quantity:  it.Quantity,
			UOM:       uomName,
			UnitPrice: it.UnitSellPrice,
			Subtotal:  it.Subtotal,
		})
	}

	receipt.Summary.Subtotal = order.Subtotal
	receipt.Summary.Discount = order.DiscountAmount
	receipt.Summary.Tax = order.TaxAmount
	receipt.Summary.Total = order.TotalAmount
	receipt.Summary.Paid = order.PaidAmount
	receipt.Summary.Change = order.ChangeAmount

	for _, p := range order.Payments {
		pmName := "Payment"
		if p.PaymentMethod != nil {
			pmName = p.PaymentMethod.Name
		}
		receipt.Payments = append(receipt.Payments, struct {
			Method string `json:"method"`
			Amount int64  `json:"amount"`
			Ref    string `json:"ref,omitempty"`
		}{
			Method: pmName,
			Amount: p.Amount,
			Ref:    p.ReferenceNumber,
		})
	}

	receipt.Footer.Notes = "Terima kasih atas kunjungan Anda. Barang yang sudah dibeli tidak dapat ditukar kecuali perjanjian sebelumnya."
	return &receipt, nil
}

func (u *usecase) GetSalesReport(ctx context.Context, tenantID uint64, query SalesReportQuery) (*SalesReportSummaryResponse, error) {
	start, end, err := parseReportDateRange(query.StartDate, query.EndDate)
	if err != nil {
		return nil, err
	}

	orders, err := u.repo.ListOrdersForReport(ctx, tenantID, query.OutletID, start, end)
	if err != nil {
		return nil, err
	}

	summary := &SalesReportSummaryResponse{
		StartDate: query.StartDate,
		EndDate:   query.EndDate,
	}
	pmTotals := make(map[string]*PaymentBreakdownItem)

	for _, o := range orders {
		summary.TotalOrders++
		if o.Status == "VOID" {
			summary.VoidOrders++
			continue
		}
		if o.Status != "COMPLETED" {
			continue
		}
		summary.CompletedOrders++
		summary.GrossSales += o.Subtotal
		summary.TotalDiscount += o.DiscountAmount
		summary.TotalTax += o.TaxAmount
		summary.NetSales += o.TotalAmount

		for _, it := range o.Items {
			summary.TotalCOGS += it.UnitCostPrice * int64(it.BaseQuantity)
		}
		for _, p := range o.Payments {
			code := "UNKNOWN"
			if p.PaymentMethod != nil && p.PaymentMethod.Code != "" {
				code = p.PaymentMethod.Code
			}
			entry, exists := pmTotals[code]
			if !exists {
				entry = &PaymentBreakdownItem{MethodCode: code}
				pmTotals[code] = entry
			}
			netPaid := p.Amount
			if code == "CASH" && o.ChangeAmount > 0 {
				netPaid -= o.ChangeAmount
			}
			entry.Total += netPaid
			entry.Count++
		}
	}

	summary.GrossProfit = summary.NetSales - summary.TotalDiscount - summary.TotalCOGS
	if summary.GrossProfit < 0 {
		summary.GrossProfit = 0
	}
	for _, b := range pmTotals {
		summary.PaymentBreakdown = append(summary.PaymentBreakdown, *b)
	}
	return summary, nil
}

func (u *usecase) ExportSalesReportCSV(ctx context.Context, tenantID uint64, query SalesReportQuery) ([]byte, error) {
	start, end, err := parseReportDateRange(query.StartDate, query.EndDate)
	if err != nil {
		return nil, err
	}

	orders, err := u.repo.ListOrdersForReport(ctx, tenantID, query.OutletID, start, end)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	// UTF-8 BOM for Excel compatibility
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{
		"Order Number", "Ordered At", "Outlet ID", "Status",
		"Subtotal", "Discount", "Tax", "Grand Total",
	})
	for _, o := range orders {
		_ = w.Write([]string{
			o.OrderNumber,
			o.OrderedAt.Format(time.RFC3339),
			fmt.Sprintf("%d", o.OutletID),
			o.Status,
			fmt.Sprintf("%d", o.Subtotal),
			fmt.Sprintf("%d", o.DiscountAmount),
			fmt.Sprintf("%d", o.TaxAmount),
			fmt.Sprintf("%d", o.TotalAmount),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parseReportDateRange(startStr, endStr string) (*time.Time, *time.Time, error) {
	var startPtr, endPtr *time.Time
	if startStr != "" {
		t, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return nil, nil, errors.New("invalid start_date format, expected YYYY-MM-DD")
		}
		startPtr = &t
	}
	if endStr != "" {
		t, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return nil, nil, errors.New("invalid end_date format, expected YYYY-MM-DD")
		}
		endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		endPtr = &endOfDay
	}
	return startPtr, endPtr, nil
}

func (u *usecase) VoidOrder(ctx context.Context, tenantID, managerUserID uint64, orderUUID uuid.UUID, req VoidOrderRequest) (*OrderResponse, error) {
	order, err := u.repo.FindOrderByUUID(ctx, tenantID, orderUUID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	if order.Status != "COMPLETED" {
		return nil, fmt.Errorf("order cannot be voided because status is already '%s'", order.Status)
	}

	errTx := u.repo.WithTransaction(ctx, func(txRepo Repository) error {
		// Restore stock for each item in order
		for _, item := range order.Items {
			stock, err := txRepo.GetStockWithLock(ctx, tenantID, order.LocationID, item.VariantID)
			if err != nil {
				return fmt.Errorf("failed to lock stock for void restoration: %w", err)
			}

			stock.Quantity += item.BaseQuantity
			if err := txRepo.SaveStock(ctx, stock); err != nil {
				return fmt.Errorf("failed to restore stock: %w", err)
			}

			movement := model.StockMovement{
				TenantID:        tenantID,
				LocationID:      order.LocationID,
				VariantID:       item.VariantID,
				MovementType:    "RETURN_SALE",
				QuantityChange:  item.BaseQuantity,
				BalanceAfter:    stock.Quantity,
				CostPrice:       item.UnitCostPrice,
				ReferenceNumber: order.OrderNumber,
				Notes:           fmt.Sprintf("Void order #%s: %s", order.OrderNumber, req.Reason),
				CreatedBy:       &managerUserID,
			}
			if err := txRepo.RecordMovement(ctx, &movement); err != nil {
				return fmt.Errorf("failed to record void movement: %w", err)
			}
		}

		// Revert cash from shift if applicable
		if order.ShiftID != nil {
			shift, err := txRepo.FindShiftByID(ctx, tenantID, *order.ShiftID)
			if err == nil && shift != nil && shift.Status == "OPEN" {
				var cashPaid int64
				for _, p := range order.Payments {
					if p.PaymentMethod != nil && p.PaymentMethod.Code == "CASH" {
						cashPaid += p.Amount
					}
				}
				if cashPaid > 0 {
					netCash := cashPaid - order.ChangeAmount
					shift.ExpectedCash -= netCash
					_ = txRepo.UpdateShift(ctx, shift)
				}
			}
		}

		order.Status = "VOID"
		if err := txRepo.UpdateOrder(ctx, order); err != nil {
			return fmt.Errorf("failed to update order status to VOID: %w", err)
		}
		if err := txRepo.VoidPendingCommissionsByOrder(ctx, tenantID, order.ID); err != nil {
			return fmt.Errorf("failed to void pending commissions: %w", err)
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return u.mapOrderToResponse(order), nil
}

func (u *usecase) mapOrderToResponse(order *model.PosOrder) *OrderResponse {
	outletName := ""
	if order.Outlet != nil {
		outletName = order.Outlet.Name
	}
	cashierName := ""
	if order.Cashier != nil {
		cashierName = order.Cashier.FullName
	}

	items := make([]OrderItemResponse, len(order.Items))
	for i, it := range order.Items {
		sku := ""
		vname := ""
		if it.Variant != nil {
			sku = it.Variant.SKU
			vname = it.Variant.VariantName
		}
		uomName := "Pcs"
		if it.UOM != nil {
			uomName = it.UOM.UOMName
		}

		items[i] = OrderItemResponse{
			ID:            it.ID,
			UUID:          it.UUID,
			VariantID:     it.VariantID,
			VariantSKU:    sku,
			VariantName:   vname,
			UOMName:       uomName,
			Quantity:      it.Quantity,
			BaseQuantity:  it.BaseQuantity,
			UnitCostPrice: it.UnitCostPrice,
			UnitSellPrice: it.UnitSellPrice,
			Subtotal:      it.Subtotal,
			Notes:         it.Notes,
		}
	}

	payments := make([]OrderPaymentResponse, len(order.Payments))
	for i, p := range order.Payments {
		pmCode := ""
		pmName := ""
		if p.PaymentMethod != nil {
			pmCode = p.PaymentMethod.Code
			pmName = p.PaymentMethod.Name
		}
		payments[i] = OrderPaymentResponse{
			ID:                p.ID,
			UUID:              p.UUID,
			PaymentMethodCode: pmCode,
			PaymentMethodName: pmName,
			Amount:            p.Amount,
			ReferenceNumber:   p.ReferenceNumber,
			PaidAt:            p.PaidAt,
		}
	}

	return &OrderResponse{
		ID:                      order.ID,
		UUID:                    order.UUID,
		OrderNumber:             order.OrderNumber,
		IdempotencyKey:          order.IdempotencyKey,
		OutletID:                order.OutletID,
		OutletName:              outletName,
		CashierUserID:           order.CashierUserID,
		CashierName:             cashierName,
		Subtotal:                order.Subtotal,
		DiscountAmount:          order.DiscountAmount,
		PromotionDiscountAmount: order.PromotionDiscountAmount,
		TaxAmount:               order.TaxAmount,
		TotalAmount:             order.TotalAmount,
		PaidAmount:              order.PaidAmount,
		ChangeAmount:            order.ChangeAmount,
		Status:                  order.Status,
		OrderedAt:               order.OrderedAt,
		Items:                   items,
		Payments:                payments,
	}
}
