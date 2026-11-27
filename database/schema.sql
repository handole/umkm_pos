-- ==============================================================================
-- DATABASE SCHEMA: UMKM POS & ENTERPRISE OPERATIONS ECOSYSTEM
-- Dialect: PostgreSQL 16+
-- Primary Key Strategy: BIGSERIAL (increment) for internal high-performance FKs
-- Query/External Identifier: UUID (gen_random_uuid) for secure API endpoints
-- Tenancy Model: Shared Database, Shared Schema with tenant_id discriminator
-- Monetary Fields: BIGINT (rupiah terkecil / cents without floating-point error)
-- ==============================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ==============================================================================
-- 1. TENANCY & ORGANIZATIONAL HIERARCHY
-- ==============================================================================

CREATE TABLE tenants (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'SUSPENDED', 'TRIAL'
    subscription_plan VARCHAR(50) NOT NULL DEFAULT 'STANDARD',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;

CREATE TABLE outlets (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    phone VARCHAR(30),
    is_main_branch BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_outlets_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_outlets_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_outlets_tenant_active ON outlets(tenant_id, is_active) WHERE deleted_at IS NULL;

CREATE TABLE inventory_locations (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL, -- e.g. 'Gudang Utama', 'Display Etalase', 'Gudang Transit'
    type VARCHAR(30) NOT NULL,  -- 'WAREHOUSE', 'STORE_FRONT', 'TRANSIT', 'DAMAGED'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_locations_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_locations_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_locations_tenant_outlet ON inventory_locations(tenant_id, outlet_id, type) WHERE deleted_at IS NULL;

-- ==============================================================================
-- 2. AUTHENTICATION & ROLE-BASED ACCESS CONTROL (RBAC)
-- ==============================================================================

CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for system-wide predefined roles
    name VARCHAR(50) NOT NULL,                                -- e.g. 'SUPERADMIN', 'STORE_MANAGER', 'CASHIER', 'WAREHOUSE', 'HR_FINANCE'
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_roles_tenant_name UNIQUE (tenant_id, name),
    CONSTRAINT uq_roles_uuid UNIQUE (uuid)
);

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE, -- e.g. 'pos:checkout', 'inventory:approve-opname', 'payroll:finalize'
    module VARCHAR(50) NOT NULL,       -- 'pos', 'inventory', 'product', 'payroll', 'auth'
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles(id),
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(30),
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    pin_hash VARCHAR(255), -- for quick unlock / POS shift switch / attendance clock-in
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_users_tenant_username UNIQUE (tenant_id, username),
    CONSTRAINT uq_users_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_users_tenant_active ON users(tenant_id, is_active) WHERE deleted_at IS NULL;

CREATE TABLE user_outlets (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id) ON DELETE CASCADE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, outlet_id)
);

-- ==============================================================================
-- 3. PRODUCT CATALOG, MULTI-UOM & COST HISTORY
-- ==============================================================================

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_categories_tenant_slug UNIQUE (tenant_id, slug),
    CONSTRAINT uq_categories_tenant_uuid UNIQUE (tenant_id, uuid)
);

CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    image_url VARCHAR(500),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_products_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_products_tenant_category ON products(tenant_id, category_id) WHERE deleted_at IS NULL;

CREATE TABLE product_variants (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku VARCHAR(64) NOT NULL,
    barcode VARCHAR(64),
    variant_name VARCHAR(150) NOT NULL, -- e.g. 'Standard', 'Merah - L', '500ml'
    base_cost_price BIGINT NOT NULL DEFAULT 0, -- HPP satuan terkecil (base unit)
    base_sell_price BIGINT NOT NULL DEFAULT 0, -- Harga jual satuan terkecil
    min_stock_alert BIGINT NOT NULL DEFAULT 5,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_variants_tenant_sku UNIQUE (tenant_id, sku),
    CONSTRAINT uq_variants_tenant_barcode UNIQUE (tenant_id, barcode),
    CONSTRAINT uq_variants_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_variants_lookup ON product_variants(tenant_id, product_id, is_active) WHERE deleted_at IS NULL;

CREATE TABLE product_uoms (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    uom_name VARCHAR(50) NOT NULL, -- e.g. 'Pcs', 'Dus', 'Karton'
    conversion_rate INT NOT NULL DEFAULT 1, -- Jumlah satuan dasar terkecil, e.g. 1 Dus = 12 Pcs
    sell_price BIGINT NOT NULL, -- Harga jual spesifik satuan ini
    barcode VARCHAR(64),
    is_base_uom BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_product_uoms_variant_name UNIQUE (variant_id, uom_name),
    CONSTRAINT uq_product_uoms_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_product_uoms_variant ON product_uoms(variant_id);
CREATE INDEX idx_product_uoms_barcode ON product_uoms(tenant_id, barcode) WHERE barcode IS NOT NULL;

CREATE TABLE product_cost_history (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    reference_type VARCHAR(50) NOT NULL, -- 'PURCHASE_ORDER', 'INITIAL_IMPORT', 'MANUAL_ADJUSTMENT'
    reference_id BIGINT,
    purchase_price BIGINT NOT NULL, -- Harga beli per satuan dasar pada transaksi ini
    previous_cost_price BIGINT NOT NULL,
    new_cost_price BIGINT NOT NULL, -- HPP rata-rata bergerak (Moving Average)
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cost_history_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_cost_history_lookup ON product_cost_history(tenant_id, variant_id, recorded_at DESC);

-- ==============================================================================
-- 4. INVENTORY, SUPPLIER, PO, MUTASI & STOCK LEDGER
-- ==============================================================================

CREATE TABLE suppliers (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    contact_person VARCHAR(100),
    phone VARCHAR(30),
    email VARCHAR(255),
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_suppliers_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_suppliers_tenant_uuid UNIQUE (tenant_id, uuid)
);

CREATE TABLE inventory_stocks (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    location_id BIGINT NOT NULL REFERENCES inventory_locations(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity BIGINT NOT NULL DEFAULT 0, -- Kuantitas dalam satuan dasar terkecil
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_stock_location_variant UNIQUE (tenant_id, location_id, variant_id),
    CONSTRAINT uq_stock_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_stocks_lookup ON inventory_stocks(tenant_id, location_id, variant_id);

CREATE TABLE stock_movements (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    location_id BIGINT NOT NULL REFERENCES inventory_locations(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    movement_type VARCHAR(40) NOT NULL, -- 'IN_PURCHASE_ORDER', 'OUT_POS_SALE', 'TRANSFER_OUT', 'TRANSFER_IN', 'ADJUSTMENT_OPNAME', 'RETURN_SALE', 'RETURN_SUPPLIER'
    quantity_change BIGINT NOT NULL,    -- Positif jika barang masuk, negatif jika keluar
    balance_after BIGINT NOT NULL,      -- Saldo stok setelah mutasi ini terjadi
    cost_price BIGINT NOT NULL,         -- Nilai HPP saat mutasi dicatat
    reference_id BIGINT,                -- ID internal relasi (orders, po, opname)
    reference_uuid UUID,                -- UUID dokumen terkait untuk audit publik
    reference_number VARCHAR(100),      -- Nomor struk / nomor PO / nomor opname
    notes TEXT,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_stock_movements_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_stock_movements_audit ON stock_movements(tenant_id, location_id, variant_id, created_at DESC);

CREATE TABLE purchase_orders (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id),
    supplier_id BIGINT NOT NULL REFERENCES suppliers(id),
    destination_location_id BIGINT NOT NULL REFERENCES inventory_locations(id),
    po_number VARCHAR(100) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'SUBMITTED', 'PARTIAL_RECEIVED', 'RECEIVED', 'CANCELLED'
    order_date DATE NOT NULL,
    expected_date DATE,
    total_amount BIGINT NOT NULL DEFAULT 0,
    notes TEXT,
    created_by BIGINT NOT NULL REFERENCES users(id),
    received_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_po_tenant_number UNIQUE (tenant_id, po_number),
    CONSTRAINT uq_po_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_po_tenant_status ON purchase_orders(tenant_id, status);

CREATE TABLE purchase_order_items (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    po_id BIGINT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id),
    uom_id BIGINT REFERENCES product_uoms(id),
    quantity_ordered BIGINT NOT NULL,
    quantity_received BIGINT NOT NULL DEFAULT 0,
    unit_cost BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,
    CONSTRAINT uq_po_items_tenant_uuid UNIQUE (uuid)
);
CREATE INDEX idx_po_items_po ON purchase_order_items(po_id);

CREATE TABLE stock_transfers (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transfer_number VARCHAR(100) NOT NULL,
    source_location_id BIGINT NOT NULL REFERENCES inventory_locations(id),
    destination_location_id BIGINT NOT NULL REFERENCES inventory_locations(id),
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'IN_TRANSIT', 'RECEIVED', 'REJECTED'
    created_by BIGINT NOT NULL REFERENCES users(id),
    received_by BIGINT REFERENCES users(id),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_transfer_tenant_number UNIQUE (tenant_id, transfer_number),
    CONSTRAINT uq_transfer_tenant_uuid UNIQUE (tenant_id, uuid)
);

CREATE TABLE stock_transfer_items (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    transfer_id BIGINT NOT NULL REFERENCES stock_transfers(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id),
    quantity_sent BIGINT NOT NULL,
    quantity_received BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT uq_transfer_items_uuid UNIQUE (uuid)
);
CREATE INDEX idx_transfer_items_transfer ON stock_transfer_items(transfer_id);

CREATE TABLE stock_opnames (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    location_id BIGINT NOT NULL REFERENCES inventory_locations(id),
    opname_number VARCHAR(100) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED'
    performed_by BIGINT NOT NULL REFERENCES users(id),
    approved_by BIGINT REFERENCES users(id),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_opname_tenant_number UNIQUE (tenant_id, opname_number),
    CONSTRAINT uq_opname_tenant_uuid UNIQUE (tenant_id, uuid)
);

CREATE TABLE stock_opname_items (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    opname_id BIGINT NOT NULL REFERENCES stock_opnames(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id),
    system_quantity BIGINT NOT NULL,
    physical_quantity BIGINT NOT NULL,
    difference_quantity BIGINT NOT NULL, -- physical_quantity - system_quantity
    cost_price BIGINT NOT NULL,
    notes TEXT,
    CONSTRAINT uq_opname_items_uuid UNIQUE (uuid)
);
CREATE INDEX idx_opname_items_opname ON stock_opname_items(opname_id);

-- Log event audit pemindaian barcode fisik saat opname berlangsung
CREATE TABLE stock_opname_scans (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    opname_id BIGINT NOT NULL REFERENCES stock_opnames(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id),
    uom_id BIGINT REFERENCES product_uoms(id),
    scanned_barcode VARCHAR(64) NOT NULL,
    quantity_counted BIGINT NOT NULL,      -- Nilai hitung pada pemindaian ini dalam UOM terkait
    base_quantity_counted BIGINT NOT NULL, -- Dikonversi ke satuan dasar untuk rekapitulasi
    scanned_by BIGINT NOT NULL REFERENCES users(id),
    scanned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    device_id VARCHAR(100),
    CONSTRAINT uq_opname_scans_uuid UNIQUE (uuid)
);
CREATE INDEX idx_opname_scans_opname ON stock_opname_scans(opname_id, scanned_at DESC);

-- ==============================================================================
-- 5. CUSTOMERS & MARKETING PROMOTION ENGINE
-- ==============================================================================

CREATE TABLE customers (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(30),
    email VARCHAR(255),
    member_tier VARCHAR(50) NOT NULL DEFAULT 'REGULAR', -- 'REGULAR', 'SILVER', 'GOLD', 'PLATINUM'
    total_spend BIGINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_customers_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_customers_tenant_phone UNIQUE (tenant_id, phone),
    CONSTRAINT uq_customers_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_customers_lookup ON customers(tenant_id, phone, is_active) WHERE deleted_at IS NULL;

CREATE TABLE promotions (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL, -- e.g. 'PAYDAY-NOV', 'DISKONKOPI20', 'FLASH-SALE'
    name VARCHAR(255) NOT NULL,
    description TEXT,
    promo_type VARCHAR(30) NOT NULL DEFAULT 'AUTOMATIC', -- 'AUTOMATIC', 'COUPON_CODE'
    discount_type VARCHAR(30) NOT NULL,                  -- 'PERCENTAGE', 'FIXED_AMOUNT', 'BUY_X_GET_Y', 'SPECIAL_PRICE'
    discount_value BIGINT NOT NULL,                      -- Persentase (misal 15 untuk 15%) atau nominal rupiah (misal 10000)
    max_discount_amount BIGINT,                          -- Batas maksimal potongan rupiah untuk tipe persentase
    min_order_amount BIGINT NOT NULL DEFAULT 0,          -- Syarat minimal nilai belanja nota
    min_quantity BIGINT NOT NULL DEFAULT 1,              -- Syarat minimal kuantitas item
    target_scope VARCHAR(30) NOT NULL DEFAULT 'ENTIRE_TRANSACTION', -- 'ENTIRE_TRANSACTION', 'SPECIFIC_PRODUCTS', 'SPECIFIC_CATEGORIES'
    
    -- Parameter Waktu & Kalender
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    start_time TIME,                                     -- Happy Hour jam mulai, e.g. '14:00:00'
    end_time TIME,                                       -- Happy Hour jam selesai, e.g. '17:00:00'
    days_of_week VARCHAR(50) NOT NULL DEFAULT 'ALL',     -- 'ALL', '1,2,3,4,5' (Weekday), '6,7' (Weekend)
    days_of_month VARCHAR(100) NOT NULL DEFAULT 'ALL',   -- 'ALL', '25,26,27,28,29,30' (Payday promo)
    applicable_months VARCHAR(50) NOT NULL DEFAULT 'ALL',-- 'ALL', '3,4' (Ramadhan), '12' (Year End)

    -- Parameter Kuota & Anggaran
    total_quota INT,                                     -- Maksimal total transaksi (NULL = unlimited)
    daily_quota INT,                                     -- Maksimal transaksi per hari
    per_customer_quota INT NOT NULL DEFAULT 1,           -- Maksimal pemakaian per pelanggan/member
    current_usage_count INT NOT NULL DEFAULT 0,          -- Akumulasi pemakaian transaksi sukses
    total_budget BIGINT,                                 -- Plafon anggaran subsidi diskon dari owner
    current_spend_amount BIGINT NOT NULL DEFAULT 0,      -- Akumulasi rupiah subsidi diskon yang telah keluar

    is_stackable BOOLEAN NOT NULL DEFAULT FALSE,         -- Boleh digabung dengan promo lain
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_promotions_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_promotions_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_promotions_schedule ON promotions(tenant_id, start_date, end_date, is_active) WHERE deleted_at IS NULL;

-- Pembatasan outlet yang memberlakukan promosi (jika kosong = berlaku untuk semua outlet tenant)
CREATE TABLE promotion_outlets (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    promotion_id BIGINT NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id) ON DELETE CASCADE,
    CONSTRAINT uq_promotion_outlets UNIQUE (promotion_id, outlet_id),
    CONSTRAINT uq_promotion_outlets_uuid UNIQUE (uuid)
);

-- Aturan Inklusi / Eksklusi Produk / Kategori & Buy X Get Y & Tiering
CREATE TABLE promotion_rules (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    promotion_id BIGINT NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
    rule_type VARCHAR(30) NOT NULL, -- 'INCLUDE_CATEGORY', 'EXCLUDE_CATEGORY', 'INCLUDE_PRODUCT', 'EXCLUDE_PRODUCT', 'BUY_X_GET_Y', 'TIERED_QTY'
    category_id BIGINT REFERENCES categories(id) ON DELETE CASCADE,
    variant_id BIGINT REFERENCES product_variants(id) ON DELETE CASCADE,
    buy_variant_id BIGINT REFERENCES product_variants(id) ON DELETE CASCADE,     -- Untuk Buy X Get Y (varian syarat)
    buy_quantity INT,                                                            -- Beli sebanyak X
    reward_variant_id BIGINT REFERENCES product_variants(id) ON DELETE CASCADE,  -- Varian hadiah gratis
    reward_quantity INT,                                                         -- Dapat gratis sebanyak Y
    tier_min_qty INT,                                                            -- Batas bawah kuantitas tier
    tier_max_qty INT,                                                            -- Batas atas kuantitas tier
    tier_discount_value BIGINT,                                                  -- Nilai diskon khusus tier
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_promotion_rules_uuid UNIQUE (uuid)
);
CREATE INDEX idx_promotion_rules_promo ON promotion_rules(promotion_id, rule_type);

CREATE TABLE promotion_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    promotion_id BIGINT NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL, -- Relasi ke pos_orders.id
    customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL,
    discount_amount BIGINT NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_promotion_usage_logs_uuid UNIQUE (uuid)
);
CREATE INDEX idx_promotion_usage_audit ON promotion_usage_logs(tenant_id, promotion_id, customer_id, used_at DESC);

-- ==============================================================================
-- 6. POS (POINT OF SALE), SHIFT, CASH DRAWER & CHECKOUT
-- ==============================================================================

CREATE TABLE pos_shifts (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id),
    cashier_user_id BIGINT NOT NULL REFERENCES users(id),
    shift_number VARCHAR(100) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE,
    starting_cash BIGINT NOT NULL DEFAULT 0,   -- Modal kas kecil awal laci
    expected_cash BIGINT NOT NULL DEFAULT 0,   -- Saldo sistem (awal + sales cash + in - out)
    actual_cash BIGINT,                       -- Uang fisik saat tutup shift
    cash_discrepancy BIGINT,                  -- Selisih kas fisik vs sistem
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN', -- 'OPEN', 'CLOSED'
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pos_shifts_tenant_number UNIQUE (tenant_id, shift_number),
    CONSTRAINT uq_pos_shifts_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_shifts_cashier ON pos_shifts(tenant_id, cashier_user_id, status);

CREATE TABLE pos_shift_cash_logs (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    shift_id BIGINT NOT NULL REFERENCES pos_shifts(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- 'CASH_IN', 'CASH_OUT'
    amount BIGINT NOT NULL,
    reason VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_shift_cash_logs_uuid UNIQUE (uuid)
);

CREATE TABLE pos_payment_methods (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL, -- 'CASH', 'QRIS', 'DEBIT_CARD', 'CREDIT_CARD', 'BANK_TRANSFER'
    name VARCHAR(100) NOT NULL,
    requires_reference BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payment_methods_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_payment_methods_tenant_uuid UNIQUE (tenant_id, uuid)
);

CREATE TABLE pos_orders (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id),
    shift_id BIGINT REFERENCES pos_shifts(id),
    cashier_user_id BIGINT NOT NULL REFERENCES users(id),
    salesperson_user_id BIGINT REFERENCES users(id), -- Digunakan untuk kalkulasi komisi staf
    customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL, -- Member / pelanggan jika terdaftar
    promotion_id BIGINT REFERENCES promotions(id) ON DELETE SET NULL, -- Promosi nota yang diaplikasikan
    coupon_code VARCHAR(50), -- Kode kupon / voucher jika ada
    location_id BIGINT NOT NULL REFERENCES inventory_locations(id), -- Lokasi fisik pemotongan stok
    order_number VARCHAR(100) NOT NULL,
    idempotency_key UUID NOT NULL, -- Mencegah double-order saat retry client jaringan toko
    subtotal BIGINT NOT NULL,
    discount_amount BIGINT NOT NULL DEFAULT 0, -- Total diskon keseluruhan
    promotion_discount_amount BIGINT NOT NULL DEFAULT 0, -- Diskon spesifik dari mesin promosi
    tax_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL,
    paid_amount BIGINT NOT NULL,
    change_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'COMPLETED', -- 'COMPLETED', 'VOID', 'REFUNDED'
    ordered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pos_orders_tenant_number UNIQUE (tenant_id, order_number),
    CONSTRAINT uq_pos_orders_tenant_idempotency UNIQUE (tenant_id, idempotency_key),
    CONSTRAINT uq_pos_orders_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_pos_orders_lookup ON pos_orders(tenant_id, outlet_id, ordered_at DESC);
CREATE INDEX idx_pos_orders_salesperson ON pos_orders(tenant_id, salesperson_user_id, ordered_at DESC);
CREATE INDEX idx_pos_orders_customer ON pos_orders(tenant_id, customer_id, ordered_at DESC);
CREATE INDEX idx_pos_orders_promotion ON pos_orders(tenant_id, promotion_id);

CREATE TABLE pos_order_items (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    order_id BIGINT NOT NULL REFERENCES pos_orders(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variants(id),
    uom_id BIGINT REFERENCES product_uoms(id),
    promotion_id BIGINT REFERENCES promotions(id) ON DELETE SET NULL, -- Promosi spesifik item ini jika ada
    quantity BIGINT NOT NULL,      -- Kuantitas transaksi berdasarkan UOM yang dipilih
    base_quantity BIGINT NOT NULL, -- Konversi ke satuan dasar untuk mutasi kartu stok
    unit_cost_price BIGINT NOT NULL, -- HPP saat transaksi terjadi untuk kalkulasi laba kotor
    unit_sell_price BIGINT NOT NULL,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    promotion_discount_amount BIGINT NOT NULL DEFAULT 0,
    subtotal BIGINT NOT NULL,
    notes TEXT,
    CONSTRAINT uq_pos_order_items_uuid UNIQUE (uuid)
);
CREATE INDEX idx_pos_order_items_order ON pos_order_items(order_id);
CREATE INDEX idx_pos_order_items_variant ON pos_order_items(variant_id);

CREATE TABLE pos_order_payments (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    order_id BIGINT NOT NULL REFERENCES pos_orders(id) ON DELETE CASCADE,
    payment_method_id BIGINT NOT NULL REFERENCES pos_payment_methods(id),
    amount BIGINT NOT NULL,
    reference_number VARCHAR(100), -- No Approval EDC / Ref QRIS / Bukti Transfer
    paid_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pos_order_payments_uuid UNIQUE (uuid)
);
CREATE INDEX idx_pos_order_payments_order ON pos_order_payments(order_id);

-- ==============================================================================
-- 7. HR, EMPLOYEES, ATTENDANCE & PAYROLL
-- ==============================================================================

CREATE TABLE employees (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL, -- Relasi ke akun login jika karyawan memiliki akses sistem
    outlet_id BIGINT NOT NULL REFERENCES outlets(id),
    employee_code VARCHAR(50) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    job_title VARCHAR(100) NOT NULL,
    phone VARCHAR(30),
    bank_name VARCHAR(100),
    bank_account_number VARCHAR(100),
    bank_account_holder VARCHAR(255),
    joined_date DATE NOT NULL,
    employment_status VARCHAR(30) NOT NULL DEFAULT 'PERMANENT', -- 'PERMANENT', 'CONTRACT', 'PROBATION'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_employees_tenant_code UNIQUE (tenant_id, employee_code),
    CONSTRAINT uq_employees_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_employees_tenant_active ON employees(tenant_id, outlet_id, is_active) WHERE deleted_at IS NULL;

CREATE TABLE salary_structures (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL UNIQUE REFERENCES employees(id) ON DELETE CASCADE,
    basic_salary BIGINT NOT NULL DEFAULT 0,
    fixed_allowance BIGINT NOT NULL DEFAULT 0,
    meal_transport_allowance BIGINT NOT NULL DEFAULT 0,
    commission_rate_percent NUMERIC(5,2) NOT NULL DEFAULT 0, -- % komisi penjualan POS
    commission_fixed_per_item BIGINT NOT NULL DEFAULT 0,     -- Nilai komisi flat per item
    bpjs_health_deduction BIGINT NOT NULL DEFAULT 0,
    bpjs_employment_deduction BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_salary_structures_tenant_uuid UNIQUE (tenant_id, uuid)
);

CREATE TABLE attendance_logs (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    outlet_id BIGINT NOT NULL REFERENCES outlets(id),
    log_date DATE NOT NULL,
    clock_in TIMESTAMP WITH TIME ZONE,
    clock_out TIMESTAMP WITH TIME ZONE,
    clock_in_photo_url VARCHAR(500),
    clock_out_photo_url VARCHAR(500),
    late_minutes INT NOT NULL DEFAULT 0,
    overtime_minutes INT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'PRESENT', -- 'PRESENT', 'LATE', 'ABSENT', 'LEAVE', 'SICK'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_attendance_employee_date UNIQUE (tenant_id, employee_id, log_date),
    CONSTRAINT uq_attendance_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_attendance_period ON attendance_logs(tenant_id, employee_id, log_date);

CREATE TABLE payroll_commissions_pending (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL REFERENCES pos_orders(id) ON DELETE CASCADE,
    commission_amount BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'CALCULATED', 'PAID', 'VOID'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_commissions_pending_uuid UNIQUE (uuid)
);
CREATE INDEX idx_commissions_employee_status ON payroll_commissions_pending(tenant_id, employee_id, status);

CREATE TABLE payroll_periods (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_name VARCHAR(100) NOT NULL, -- e.g. 'November 2026'
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    payment_date DATE NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'LOCKED', 'FINALIZED'
    total_payout BIGINT NOT NULL DEFAULT 0,
    finalized_by BIGINT REFERENCES users(id),
    finalized_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payroll_period_tenant_name UNIQUE (tenant_id, period_name),
    CONSTRAINT uq_payroll_period_tenant_uuid UNIQUE (tenant_id, uuid)
);

-- Snapshot data gaji karyawan saat periode di-FINALIZE (Data historis terkunci permanen)
CREATE TABLE payroll_snapshots (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_id BIGINT NOT NULL REFERENCES payroll_periods(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    employee_code VARCHAR(50) NOT NULL,
    employee_name VARCHAR(255) NOT NULL,
    job_title VARCHAR(100) NOT NULL,
    bank_name VARCHAR(100),
    bank_account_number VARCHAR(100),
    basic_salary BIGINT NOT NULL,
    total_allowances BIGINT NOT NULL DEFAULT 0,
    total_overtime_pay BIGINT NOT NULL DEFAULT 0,
    total_commissions BIGINT NOT NULL DEFAULT 0,
    total_deductions BIGINT NOT NULL DEFAULT 0, -- Kasbon, keterlambatan, BPJS, pajak
    net_salary BIGINT NOT NULL,
    slip_pdf_url VARCHAR(500),
    status VARCHAR(30) NOT NULL DEFAULT 'FINALIZED',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payroll_snapshot_period_employee UNIQUE (period_id, employee_id),
    CONSTRAINT uq_payroll_snapshot_tenant_uuid UNIQUE (tenant_id, uuid)
);
CREATE INDEX idx_payroll_snapshots_period ON payroll_snapshots(tenant_id, period_id);

-- Rincian baris pendapatan dan potongan pada slip gaji
CREATE TABLE payroll_item_snapshots (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    snapshot_id BIGINT NOT NULL REFERENCES payroll_snapshots(id) ON DELETE CASCADE,
    item_type VARCHAR(20) NOT NULL, -- 'EARNING', 'DEDUCTION'
    component_name VARCHAR(100) NOT NULL, -- e.g. 'Gaji Pokok', 'Komisi POS', 'Potongan Kasbon'
    amount BIGINT NOT NULL,
    notes VARCHAR(255),
    CONSTRAINT uq_payroll_item_snapshots_uuid UNIQUE (uuid)
);
CREATE INDEX idx_payroll_item_snapshots ON payroll_item_snapshots(snapshot_id);
