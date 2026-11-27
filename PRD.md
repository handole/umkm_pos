# Product Requirement Document (PRD)
## Sistem Terpadu POS Kasir, Manajemen Stok Gudang, Katalog Produk, dan Penggajian (Payroll)

---

### Informasi Dokumen
- **Nama Proyek:** UMKM POS & Enterprise Operations Ecosystem
- **Target Platform:** 
  - Backend: Go (Gin Framework) - Modular Monolith Architecture
  - Mobile & Tablet: Flutter (Android Kiosk Kasir & Mobile Warehouse)
  - Web Admin: Vue.js 3 (Vite, Pinia, Tailwind CSS)
  - Database & Infra: PostgreSQL 16, Redis 7, MinIO / S3, Docker Compose
- **Strategi Tenancy:** Hybrid Evolutionary Architecture (Single-Tenant Ready, Multi-Tenant Native by Design)
- **Status Dokumen:** Approved for Implementation

---

## 1. Eksekutif & Arsitektur Strategis

### 1.1 Latar Belakang & Masalah
Operasional bisnis ritel, food & beverage (F&B), dan grosir skala UMKM hingga menengah sering kali terfragmentasi ke dalam aplikasi yang terpisah:
- Kasir menggunakan aplikasi POS pihak ketiga yang tidak terhubung ke gudang pusat.
- Stok gudang dicatat secara manual di buku atau spreadsheet terpisah sehingga data stok kerap selisih.
- Master katalog produk sering kali tidak sinkron antara harga toko dan harga gudang.
- Perhitungan komisi penjualan kasir dan penggajian staf dilakukan manual di akhir bulan, memakan waktu berhari-hari dan rawan kesalahan hitung.

### 1.2 Mengapa Modular Monolith di Go (Gin)
Pendekatan yang dipilih adalah **Modular Monolith** menggunakan Go dan Gin Framework, bukan microservices langsung.
1. **Integritas Transaksi ACID yang Ketat:**
   Alur bisnis utama (misalnya checkout kasir) membutuhkan konsistensi data yang atomik:
   - Mencatat record penjualan (`pos_orders` dan `pos_order_items`).
   - Memotong kuantitas stok di lokasi terkait secara atomik dengan kartu stok (`stock_movements`).
   - Mencatat komisi penjualan bagi kasir/sales yang bertugas ke modul payroll.
   Jika dipecah menjadi microservices sejak awal, alur ini membutuhkan pola konsistensi bertahap (Two-Phase Commit atau Saga Pattern via Message Broker) yang menambah kompleksitas operasional, latensi jaringan, serta risiko data inkonsisten ketika jaringan toko tidak stabil.
2. **Efisiensi Sumber Daya & Kecepatan Eksekusi:**
   Go menghasilkan binary statis tunggal berukuran kecil (15 sampai 25 MB) dengan konsumsi memori rendah (20 sampai 50 MB RAM pada kondisi idle), sangat hemat biaya ketika di-deploy di VPS berbiaya terjangkau.
3. **Batas Domain yang Rapi (Clean Boundaries):**
   Dengan memisahkan struktur internal (`internal/product`, `internal/inventory`, `internal/pos`, `internal/payroll`, `internal/auth`), kode tetap modular. Apabila di masa depan volume transaksi salah satu modul (misal POS) melonjak drastis, modul tersebut dapat diekstrak menjadi service independen tanpa refactor total.

```
                           +--------------------------------------------------+
                           |             Reverse Proxy (Caddy / Nginx)        |
                           +------------------------+-------------------------+
                                                    |
                         +--------------------------+--------------------------+
                         |                                                     |
                         v                                                     v
+------------------------------------+               +------------------------------------+
|  Flutter (Tablet & Mobile App)     |               |  Vue.js 3 (Web Admin Dashboard)    |
|  - Kasir POS (Offline-First Drift) |               |  - Master Data & Supplier PO       |
|  - Scanner Gudang & Stock Opname   |               |  - Stock Opname Approval           |
|  - Absensi Karyawan Kiosk          |               |  - Payroll Review & Financial Rpt  |
+-----------------+------------------+               +-----------------+------------------+
                  |                                                    |
                  +--------------------------+-------------------------+
                                             | REST API /api/v1/*
                                             v
+-----------------------------------------------------------------------------------------+
|                                GO (GIN) MODULAR MONOLITH                                |
|                                                                                         |
|  +-------------------+  +--------------------+  +------------------+  +--------------+  |
|  |  internal/auth    |  |  internal/product  |  | internal/invent. |  | internal/pos |  |
|  |  - RBAC & JWT     |  |  - SKU, Variants   |  | - Multi-Location |  | - Checkout   |  |
|  |  - Tenant Scope   |  |  - Multi-UOM & HPP |  | - Stock Ledger   |  | - Shifts     |  |
|  +-------------------+  +--------------------+  +------------------+  +-------+------+  |
|                                                                               |         |
|  +-------------------------------------------+  +--------------------------+  |         |
|  |  internal/payroll                         |  |  internal/promotion      |  |         |
|  |  - Attendance, Salary, Auto-Commission    |  |  - Rule Evaluator Engine |<-+         |
|  |  - Immutable Finalized Snapshots          |  |  - Date, Outlet & Quota  | (Atomic)   |
|  +-------------------------------------------+  +--------------------------+            |
|                                                                                         |
|  +-----------------------------------------------------------------------------------+  |
|  |  Background Workers (Redis + Asynq): Mass Slip PDF, Report Export, Notifications  |  |
|  +-----------------------------------------------------------------------------------+  |
+--------------------+------------------------------+--------------------+----------------+
                     |                              |                    |
                     v                              v                    v
          +--------------------+          +-------------------+ +-------------------+
          |   PostgreSQL 16    |          |   Redis 7 Cache   | | MinIO / S3 Storage|
          | (ACID Relational)  |          | & Job Queue Broker| | (Media, Invoices, |
          | Multi-Tenant Schema|          | (Catalog & Token) | |  PDF Snapshots)   |
          +--------------------+          +-------------------+ +-------------------+
```

---

## 2. Strategi Evolusi Tenancy (Single-Tenant ke Multi-Tenant)

Agar sistem dapat digunakan sebagai instalasi mandiri (single-tenant untuk satu pemilik bisnis besar) maupun bertransformasi menjadi platform SaaS multi-tenant tanpa perlu merombak ulang skema basis data, arsitektur data harus didesain **Tenant-Aware sejak Hari Pertama**.

### 2.1 Skema Partisi Data: Discriminator Column (`tenant_id`)
Pendekatan yang digunakan adalah **Shared Database, Shared Schema with Tenant Discriminator (`tenant_id`)**.
- Setiap tabel transaksional dan master data wajib memiliki kolom `tenant_id UUID NOT NULL`.
- Setiap primary key atau unique index menyertakan `tenant_id` secara komposit, contoh: `UNIQUE(tenant_id, sku)`.

```sql
-- Format index komposit wajib di setiap tabel entitas
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sku VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    CONSTRAINT uq_products_tenant_sku UNIQUE (tenant_id, sku)
);
CREATE INDEX idx_products_tenant_id ON products(tenant_id);
```

### 2.2 Hierarki Organisasi: Tenant vs Outlet / Branch
Konsep multi-toko tidak boleh disamakan dengan multi-tenant:
1. **Tenant:** Entitas legal/pemilik bisnis (Contoh: "PT Sumber Berkah Retail").
2. **Outlet / Location:** Cabang fisik atau gudang di bawah naungan tenant yang sama (Contoh: "Toko Cabang Surabaya", "Gudang Pusat Cikarang").
- Pada mode **Single-Tenant**: Sistem hanya memiliki 1 record `tenants` (default bootstrap ID `00000000-0000-0000-0000-000000000001`), namun dapat memiliki banyak `outlets` dan `locations`.
- Pada mode **Multi-Tenant (SaaS)**: Setiap pendaftaran tenant baru akan membuat record tenant baru, dan data antartenant terisolasi secara total.

### 2.3 Middleware Tenant Context di Go (Gin)
Resolusi tenant dilakukan di layer middleware HTTP Gin:
1. Middleware memeriksa identitas tenant dari:
   - Claims di dalam JWT token (`tenant_id`).
   - Header request `X-Tenant-ID` (untuk internal/machine-to-machine).
   - Subdomain URL (opsional saat mode SaaS aktif, misal `tokokita.posapp.com`).
2. Jika server dijalankan dengan environment flag `APP_MODE=single-tenant`, middleware secara otomatis menginjeksi default `DEFAULT_TENANT_ID` ke dalam `gin.Context` tanpa memvalidasi subdomain.
3. Di layer repository Go, seluruh query SQL dibungkus dengan parameter `tenant_id`:
   ```go
   func (r *productRepository) FindBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*Product, error) {
       var p Product
       query := `SELECT id, tenant_id, sku, name, price, created_at 
                 FROM products 
                 WHERE tenant_id = $1 AND sku = $2 AND deleted_at IS NULL`
       err := r.db.GetContext(ctx, &p, query, tenantID, sku)
       return &p, err
   }
   ```
4. **Proteksi Tambahan:** PostgreSQL Row Level Security (RLS) dapat diaktifkan pada level database session menggunakan `SET LOCAL app.current_tenant_id = '...'` sebagai lapisan keamanan kedua (*defense in depth*).

---

## 3. Spesifikasi Domain & Modul Backend

### 3.1 Modul Produk (`internal/product`)
Mengelola data barang dagangan, klasifikasi, varian harga, serta satuan bertingkat.

#### Fitur Utama
1. **Pengelompokan & Varian:**
   - Kategori berjenjang (Parent - Child).
   - Atribut varian: Ukuran, Warna, Rasa, atau Tipe Kemasan.
   - Setiap varian memiliki SKU dan Barcode unik.
2. **Multi-Satuan (Multi-UOM) Bertingkat:**
   - Satuan dasar terkecil (*Base Unit*): Contoh `Pcs`.
   - Satuan tingkat menengah: Contoh `Dus` (1 Dus = 12 Pcs).
   - Satuan tingkat besar: Contoh `Karton` (1 Karton = 4 Dus = 48 Pcs).
   - Rasio konversi dikunci secara matematis sehingga saat kasir menjual 1 Dus, sistem memotong 12 Pcs pada kartu stok.
3. **Harga & Riwayat HPP (Harga Pokok Penjualan):**
   - **Tipe Data Moneter:** Seluruh nilai mata uang disimpan dalam tipe data `BIGINT` (satuan rupiah terkecil tanpa desimal) atau `NUMERIC(15,2)`. Dilarang keras menggunakan tipe data `FLOAT` atau `DOUBLE` untuk mencegah pembulatan mengambang (floating-point error).
   - Pemisahan antara harga jual standar dengan tabel riwayat pembelian HPP (`product_cost_history`) dengan metode Moving Average atau FIFO.

#### Data Model Produk
```
+--------------------+       +-----------------------+       +------------------------+
|      tenants       |       |       products        |       |    product_variants    |
+--------------------+       +-----------------------+       +------------------------+
| id (PK)            |<---+  | id (PK)               |<---+  | id (PK)                |
| name               |    +--| tenant_id (FK)        |    +--| product_id (FK)        |
| code               |       | name                  |       | tenant_id (FK)         |
+--------------------+       | category_id (FK)      |       | sku (Unique per tenant)|
                             | base_uom_id (FK)      |       | barcode (Unique)       |
                             | is_active             |       | variant_name           |
                             +-----------------------+       | base_cost_price BIGINT |
                                                             | base_sell_price BIGINT |
                                                             +-----------+------------+
                                                                         |
                                 +---------------------------------------+
                                 |
                                 v
                     +-----------------------+       +------------------------+
                     |     product_uoms      |       |  product_cost_history  |
                     +-----------------------+       +------------------------+
                     | id (PK)               |       | id (PK)                |
                     | variant_id (FK)       |       | variant_id (FK)        |
                     | uom_name (Dus/Karton) |       | tenant_id (FK)         |
                     | conversion_rate INT   |       | reference_id (PO UUID) |
                     | sell_price BIGINT     |       | purchase_price BIGINT  |
                     | barcode VARCHAR       |       | recorded_at TIMESTAMP  |
                     +-----------------------+       +------------------------+
```

---

### 3.2 Modul Inventaris & Stok Gudang (`internal/inventory`)
Mengelola pergerakan fisik barang dengan transparansi jejak audit mutlak.

#### Fitur Utama
1. **Multi-Lokasi:**
   - Memisahkan stok antara Gudang Pusat (Warehouse), Toko Utama (Store/Outlet), dan Rak Display Kasir.
2. **Pola Stock Ledger (Kartu Stok / `stock_movements`):**
   - Dilarang hanya melakukan `UPDATE inventory_stocks SET quantity = quantity - X`.
   - Setiap perubahan kuantitas fisik **wajib** mencatat entri pada `stock_movements` yang immutable (hanya INSERT, tidak ada UPDATE/DELETE).
   - Jenis pergerakan (*movement types*):
     - `IN_PURCHASE_ORDER`: Penerimaan barang dari supplier.
     - `OUT_POS_SALE`: Penjualan langsung di meja kasir.
     - `TRANSFER_OUT` & `TRANSFER_IN`: Mutasi antarlokasi/toko.
     - `ADJUSTMENT_OPNAME`: Koreksi selisih hasil stock opname fisik.
     - `RETURN_SALE`: Retur dari pelanggan.
     - `RETURN_SUPPLIER`: Retur barang rusak ke supplier.
3. **Pencegahan Concurrency Race Condition pada Stok:**
   - Ketika beberapa kasir menjual produk yang sama secara bersamaan, query pemeriksaan stok dilakukan di dalam transaksi database menggunakan **Pessimistic Locking**:
     ```sql
     SELECT id, current_stock 
     FROM inventory_stocks 
     WHERE tenant_id = $1 AND location_id = $2 AND variant_id = $3 
     FOR UPDATE;
     ```
   - Jika kuantitas yang diminta melebihi `current_stock` dan lokasi tidak mengizinkan stok minus (*negative stock protection*), transaksi otomatis dibatalkan (*rollback*) dengan error code `INSUFFICIENT_STOCK`.
4. **Alur Stock Opname & Approval:**
   - Tim gudang mencatat hitungan fisik via mobile app.
   - Sistem mencatat selisih (*variance*) antara stok sistem vs fisik.
   - Status: `DRAFT` -> `SUBMITTED` -> `APPROVED` (oleh Supervisor/Owner).
   - Setelah approved, sistem membuat transaksi `stock_movements` tipe `ADJUSTMENT_OPNAME` secara otomatis.

#### Data Model Stok & Ledger
```sql
CREATE TABLE inventory_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    outlet_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL, -- Contoh: "Gudang Utama", "Display Toko"
    type VARCHAR(30) NOT NULL,  -- 'WAREHOUSE', 'STORE_FRONT', 'TRANSIT'
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE inventory_stocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    location_id UUID NOT NULL REFERENCES inventory_locations(id),
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    quantity BIGINT NOT NULL DEFAULT 0,
    min_stock_alert BIGINT DEFAULT 5,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_stock_location_variant UNIQUE (tenant_id, location_id, variant_id)
);

CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    location_id UUID NOT NULL REFERENCES inventory_locations(id),
    movement_type VARCHAR(40) NOT NULL, -- 'IN_PO', 'OUT_SALE', 'TRANSFER_IN', 'TRANSFER_OUT', 'ADJUSTMENT'
    quantity_change BIGINT NOT NULL,    -- Positif jika bertambah, negatif jika berkurang
    balance_after BIGINT NOT NULL,      -- Saldo stok setelah mutasi ini terjadi
    cost_price BIGINT NOT NULL,         -- Nilai HPP saat mutasi terjadi
    reference_id UUID NOT NULL,         -- ID transaksi POS, ID PO, atau ID Opname
    reference_number VARCHAR(100),      -- Nomor struk / nomor PO
    notes TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX idx_stock_movements_lookup 
ON stock_movements(tenant_id, location_id, variant_id, created_at DESC);
```

---

### 3.3 Modul Kasir POS (`internal/pos`)
Melayani alur kasir berkecepatan tinggi, toleran terhadap gangguan koneksi, dan aman terhadap double charge.

#### Fitur Utama
1. **Manajemen Shift Kasir (Cash Drawer Management):**
   - Buka Shift: Input kas awal / modal laci (*opening float*).
   - Operasional Shift: Catat kas masuk/keluar operasional (*cash in / cash out*).
   - Tutup Shift: Kasir menghitung fisik uang tunai di laci (*closing cash*), sistem menghitung selisih kas (*cash discrepancy*), dan mencetak laporan X/Z report ke printer thermal.
2. **Pola Transaksi Idempotent (Idempotency Key):**
   - Setiap transaksi checkout yang dikirimkan oleh Flutter wajib menyertakan header `Idempotency-Key: <UUIDv4>`.
   - Go Gin menyimpan status request di Redis / database dengan durasi TTL tertentu.
   - Jika koneksi jaringan toko putus sesaat setelah kasir menekan tombol bayar dan aplikasi kasir melakukan pengiriman ulang (*retry*), backend mengenali kunci tersebut dan mengembalikan respons yang sama tanpa memotong stok atau membuat order kedua.
3. **Metode Pembayaran Fleksibel (Split Payment):**
   - Mendukung pembayaran gabungan dalam satu transaksi (Contoh: Total Rp 150.000 dibayar Rp 100.000 Tunai + Rp 50.000 QRIS).
4. **Integrasi Struk Thermal (ESC/POS):**
   - Backend menyediakan payload ringkas untuk parsing cepat oleh driver printer ESC/POS (58mm dan 80mm).

#### Transaksi Atomik Checkout Kasir
Dalam 1 transaksi database tunggal (`db.BeginTx`):
1. Validasi `idempotency_key` (apakah pesanan ini sudah pernah diproses).
2. Kunci baris stok terkait menggunakan `SELECT ... FOR UPDATE`.
3. Kurangi kuantitas stok di tabel `inventory_stocks`.
4. Catat baris kartu stok di tabel `stock_movements`.
5. Simpan record header `pos_orders` dan item `pos_order_items`.
6. Simpan rincian pembayaran di `pos_order_payments`.
7. Hitung komisi penjualan untuk kasir atau tenaga penjual (`salesperson_id`) dan simpan ke entri komisi pending di modul payroll (`payroll_commissions_pending`).
8. Commit transaksi database.

---

### 3.4 Modul Penggajian & Komisi (`internal/payroll`)
Menghubungkan produktivitas penjualan langsung ke kesejahteraan staf toko.

#### Fitur Utama
1. **Kalkulasi Komisi Otomatis:**
   - Aturan komisi dapat dikonfigurasi fleksibel:
     - Persentase dari total penjualan bersih (misal 1% dari omzet).
     - Nominal tetap per item produk tertentu (misal Rp 2.000 per botol parfum yang terjual).
   - Record komisi otomatis tercipta pada saat checkout POS berhasil.
2. **Manajemen Absensi Terintegrasi:**
   - Staf kasir atau gudang melakukan clock-in / clock-out langsung dari tablet/HP.
   - Data absensi mencatat jam kerja, keterlambatan (menit late), dan lembur yang tervalidasi.
3. **Komponen Penggajian:**
   - **Pendapatan (+):** Gaji Pokok, Tunjangan Jabatan, Uang Makan/Transport, Komisi Penjualan, Upah Lembur.
   - **Potongan (-):** Kasbon/Pinjaman Karyawan, Penalti Terlambat, BPJS Ketenagakerjaan/Kesehatan, PPh 21.
4. **Snapshot Payroll Saat Finalisasi (Wajib):**
   - **Prinsip Immutability:** Ketika periode penggajian bulanan di-`FINALIZE`, sistem membuat salinan permanen ke tabel `payroll_snapshots` dan `payroll_item_snapshots`.
   - **Alasan Teknis:** Jika di bulan depan ada kenaikan gaji pokok karyawan atau perubahan skema komisi master, data slip gaji bulan lalu **tidak boleh berubah sama sekali**.
5. **Distribusi Slip Gaji:**
   - Background worker (Redis + Asynq) membuat dokumen PDF slip gaji massal, menyimpannya di MinIO/S3, dan mengirimkan notifikasi via WhatsApp/Email.

---

### 3.5 Modul Autentikasi & RBAC (`internal/auth`)
Menjamin keamanan akses berbasis peran di seluruh channel (Flutter & Web).

#### Role Matrix
| Role | POS Kasir & Promo | Scanner Gudang | Absensi Kiosk | Web Master Produk | Web Stok & PO | Web Marketing & Promo | Web Payroll & Finansial |
|---|---|---|---|---|---|---|---|
| **Owner / Super Admin** | Full Access | Full Access | Manage | Full Access | Full Access | Full Access | Full Access |
| **Store Manager / SPV** | Full Access | Void / Approve | Monitor | Read Only | Request Mutasi | Create & Manage | Review Absensi |
| **Kasir (Cashier)** | Input, Bayar, Apply Promo | No Access | Clock-In/Out | Read Only | Read Display | View Active Promos | View Slip Sendiri |
| **Warehouse Staff** | No Access | Scan & Opname | Clock-In/Out | No Access | Input Terima/Kirim | No Access | View Slip Sendiri |
| **HR & Finance Admin** | No Access | No Access | Manage | Read Only | View Laporan Nilai | View Promo ROI | Full Payroll & Slip |

---

### 3.6 Modul Marketing & Mesin Promosi (`internal/promotion`)
Mengelola seluruh kampanye promosi, diskon otomatis, voucher/kupon, dan loyalitas pelanggan secara fleksibel dengan kendali parameter granular.

#### 1. Parameter Waktu & Kalender
Promosi dapat dibatasi dan dijadwalkan berdasarkan multi-dimensi waktu:
- **Rentang Tanggal:** Tanggal mulai (`start_date`) dan tanggal berakhir (`end_date`).
- **Bulan Tertentu:** Berlaku pada bulan spesifik (contoh: Bulan Ramadhan, Promo Akhir Tahun Desember).
- **Tanggal Bulanan (Payday):** Berlaku pada tanggal tertentu setiap bulan (contoh: Promo Gajian tanggal 25 sampai 30 setiap bulan).
- **Hari dalam Seminggu:** Pilihan hari tertentu (contoh: Weekend Sale khusus Sabtu dan Minggu, atau Flash Monday khusus hari Senin).
- **Jam Tertentu (Happy Hour):** Jam mulai dan selesai (contoh: Happy Hour pukul 14:00 sampai 17:00 WIB).

#### 2. Parameter Target Produk & Kategori
- **Lingkup Sasaran (Target Scope):**
  - `ENTIRE_TRANSACTION`: Diskon memotong subtotal nota belanja secara keseluruhan.
  - `SPECIFIC_CATEGORIES`: Diskon hanya berlaku untuk produk di dalam kategori terpilih (contoh: Kategori Makanan Ringan atau Kategori Minuman Dingin).
  - `SPECIFIC_PRODUCTS`: Diskon hanya berlaku untuk varian/SKU tertentu (contoh: Kopi Susu 500ml).
- **Aturan Inklusi & Eksklusi Granular (`promotion_rules`):**
  - Mampu menggabungkan aturan inklusi dan pengecualian (contoh: Diskon 20% untuk semua produk kategori Bakery KECUALI produk dengan SKU tertentu yang sedang langka).

#### 3. Parameter Multi-Tenant & Multi-Outlet
- **Tenant Scope:** Setiap promosi terikat kuat pada `tenant_id` masing-masing sehingga aman dan terisolasi total dalam ekosistem multi-tenant.
- **Outlet Scope (`promotion_outlets`):**
  - Berlaku di seluruh cabang toko milik tenant.
  - Atau hanya berlaku di cabang fisik tertentu (contoh: Promo Diskon 30% Grand Opening khusus Cabang Surabaya Rungkut).

#### 4. Mekanisme & Tipe Diskon
1. **Persentase (`PERCENTAGE`):** Potongan dalam persen (contoh: 15%) dilengkapi batas maksimal diskon (`max_discount_amount`, contoh: maksimal potongan Rp 30.000).
2. **Nominal Tetap (`FIXED_AMOUNT`):** Potongan harga tunai langsung (contoh: Potongan Rp 10.000 per nota atau per produk terpilih).
3. **Beli X Dapat Y (`BUY_X_GET_Y`):** Beli barang A sejumlah X, mendapatkan gratis barang B sejumlah Y (contoh: Beli 2 Botol Sirup gratis 1 Gelas Kaca).
4. **Diskon Bertingkat Kuantitas (`TIERED_QUANTITY`):**
   - Beli 1 sampai 2 item: Harga normal.
   - Beli 3 sampai 5 item: Diskon 5%.
   - Beli 6 item atau lebih: Diskon 10%.
5. **Harga Khusus (`SPECIAL_PRICE`):** Menetapkan harga coret langsung pada periode promo (contoh: Harga normal Rp 35.000 menjadi Rp 25.000 pada jam Happy Hour).

#### 5. Pemicu Promosi (Trigger Mechanism)
- **Otomatis (`AUTOMATIC`):** Mesin promosi di POS secara otomatis mendeteksi dan mengaplikasikan promo jika isi keranjang kasir memenuhi seluruh syarat waktu, outlet, minimum belanja, dan produk tanpa kasir perlu mengingat atau menginput kode apapun.
- **Kode Kupon / Voucher (`COUPON_CODE`):** Membutuhkan input kode voucher oleh kasir atau pelanggan (contoh: `GAJIANHEMAT`, `PROMOVIP10`).

#### 6. Batasan Kuota & Anggaran Subsidi (Budget Control)
- **Total Kuota (`total_quota`):** Kuota maksimal pemakaian promosi di seluruh transaksi (contoh: Terbatas untuk 500 transaksi pertama).
- **Kuota Harian (`daily_quota`):** Membatasi pemakaian per hari agar tidak habis di awal periode (contoh: Maksimal 50 transaksi per hari).
- **Kuota Per Pelanggan (`per_customer_quota`):** Dibatasi maksimal 1x atau 2x penggunaan per nomor telepon pelanggan / member.
- **Batas Anggaran Subsidi Toko (`total_budget`):** Pemilik toko dapat mengalokasikan plafon anggaran promosi (contoh: Anggaran promo Rp 5.000.000). Setiap kali kasir memberikan diskon, nilai `current_spend_amount` bertambah. Saat plafon tercapai, promo otomatis nonaktif.

#### 7. Mesin Evaluasi Promosi di Backend Go (Rule Evaluation Pipeline)
Saat transaksi POS divalidasi, modul `internal/promotion` menjalankan pipeline evaluasi berurutan:
1. **Filter Waktu & Kalender:** Cek apakah saat ini berada dalam rentang tanggal, hari, bulan, dan jam yang valid.
2. **Filter Outlet:** Cek apakah outlet transaksi masuk dalam daftar `promotion_outlets`.
3. **Filter Kuota & Anggaran:** Cek apakah `total_quota`, `daily_quota`, dan `total_budget` masih mencukupi menggunakan transaksi database.
4. **Filter Keranjang Belanja:** Evaluasi item belanjaan terhadap syarat minimum belanja (`min_order_amount`), minimum kuantitas (`min_quantity`), serta kecocokan kategori/produk target.
5. **Kalkulasi Nilai Potongan:** Hitung nilai diskon terbaik yang menguntungkan pelanggan (jika ada lebih dari 1 promo non-stackable, sistem memilih potongan terbesar) dan kembalikan rincian potongan per item serta potongan header nota.

---

## 4. Arsitektur Frontend: Flutter vs Vue.js

Untuk efisiensi pengembangan dan pengalaman pengguna terbaik, antarmuka dibagi berdasarkan konteks kerja pengguna di lapangan.

```
+-----------------------------------------------------------------------------------------+
|                                    ARSITEKTUR CLIENT                                    |
+----------------------------------------------------+------------------------------------+
|  FLUTTER (Android Tablet & Mobile)                 |  VUE.JS 3 (Web Admin Dashboard)    |
|  Konteks: Operasional Lapangan Cepat & Hardware    |  Konteks: Backoffice, Analitik, PC |
+----------------------------------------------------+------------------------------------+
| 1. KASIR POS (Tablet Landscape 10"-12"):           | 1. MASTER DATA:                    |
|    - Split-screen: Katalog kiri, Keranjang kanan   |    - Bulk Import/Export Excel CSV  |
|    - Offline-First Sync via Drift (SQLite)         |    - Konfigurasi Multi-UOM & Harga |
|    - Cetak Struk ESC/POS (Bluetooth/USB/LAN)       |                                    |
|                                                    | 2. PENGADAAN & GUDANG:             |
| 2. WAREHOUSE SCANNER (HP Android):                 |    - Pembuatan Purchase Order (PO) |
|    - Kamera barcode / Bluetooth 2D Scanner         |    - Approval Selisih Stock Opname |
|    - Good Receiving & Mutasi Antar Rak             |    - Kartu Stok Audit Trail Viewer |
|                                                    |                                    |
| 3. ABSENSI STAF (Kiosk Mode):                      | 3. PAYROLL & KEUANGAN:             |
|    - Clock-in/Clock-out via PIN & Foto Liveness    |    - Rekap Absensi & Lembur        |
|    - Verifikasi lokasi GPS / Geofencing Outlet     |    - Approval Komisi Kasir         |
|                                                    |    - Generate & Mass Export Slip   |
+----------------------------------------------------+------------------------------------+
```

### 4.1 Flutter: Spesifikasi Kasir, Gudang, & Integrasi Barcode Scanner
1. **Layout Kasir Landscape (Tablet 10 - 12 Inch):**
   - Sisi Kiri (60% Lebar): Grid item dengan thumbnail foto, pencarian cepat instan (debounce 150ms), filter kategori, dan indikator status barcode scanner hardware.
   - Sisi Kanan (40% Lebar): Keranjang belanja aktif, kuantitas item, diskon per item/total, voucher promo, rincian pajak, dan tombol checkout besar dengan shortcut nominal uang pas.

2. **Pola Barcode Ganda (Dual-Barcode: Eceran vs Kemasan Grosir):**
   - **Barcode Eceran (`product_variants.barcode`):** Barcode EAN-13/Code-128 pada kemasan satuan terkecil (sachet, botol, bungkus). Saat dipindai di kasir, langsung menambahkan 1 Pcs ke keranjang.
   - **Barcode Kemasan Grosir (`product_uoms.barcode`):** Barcode ITF-14/Code-128 pada kardus/dus luar. Saat dipindai di kasir, langsung menambahkan 1 Dus (dengan harga jual grosir) dan secara otomatis memotong kuantitas stok dasar (misal 1 Dus = 24 Pcs) pada kartu stok.
   - **Keuntungan di Gudang:** Saat stock opname atau penerimaan barang (PO), staf gudang cukup memindai barcode kardus tanpa perlu membongkar isi fisik satu per satu.

3. **Arsitektur Pemindai Barcode di Kasir POS (Tablet Flutter):**
   - **Hardware Scanner Support:** Mendukung barcode scanner eksternal USB OTG (tipe pistol atau omnidirectional meja) dan Bluetooth barcode scanner.
   - **Invisible Keystroke Buffer (HID Keyboard Wedge):** Memanfaatkan global hardware keyboard listener (`HardwareKeyboard` / `KeyEvent`) di Flutter untuk menangkap aliran karakter cepat yang diakhiri tombol `Enter` (`\n`). Kasir dapat langsung menembakkan scanner ke produk kapan saja tanpa harus mengarahkan kursor atau mengklik kolom input pencarian terlebih dahulu.
   - **Auto-Increment pada Scan Duplikat:** Jika produk yang sama dipindai berulang kali, kuantitas item di keranjang belanja bertambah secara otomatis (+1) tanpa membuka dialog pop-up konfirmasi.
   - **Pencarian Cepat Sub-50ms (Offline-First Drift SQLite):** Pencocokan barcode dilakukan langsung pada indeks SQLite lokal tablet (`idx_variants_barcode`), sehingga produk langsung muncul dalam hitungan milidetik sekalipun koneksi internet toko sedang terputus.
   - **Audio & Haptic Feedback:** Nada "Beep" tinggi untuk pemindaian sukses, dan nada "Buzz" peringatan panjang jika barcode tidak terdaftar di sistem.

4. **Arsitektur Pemindai Barcode di Mobile Gudang (Stock Opname & Good Receiving):**
   - **Kamera HP dengan Google ML Kit / Mobile Scanner:** Pemindaian barcode 1D dan 2D cepat langsung dari kamera smartphone Android staf gudang.
   - **Continuous Scanning Mode:** Kamera tetap aktif dengan visualisasi bounding box hijau dan tombol cepat toggle lampu senter (*torch*) untuk area lorong rak gudang yang gelap.
   - **Dukungan Wearable Ring Scanner:** Staf gudang dapat menghubungkan scanner cincin jari via Bluetooth SPP/HID, sehingga kedua tangan tetap bebas untuk memindahkan kardus barang.
   - **Mode Operasional Stock Opname:**
     - *Mode Akumulasi Cepat (+1 per scan):* Untuk pemindaian cepat barang eceran di rak display.
     - *Mode Batch & Prompt Kuantitas:* Saat memindai kardus besar atau pallet, muncul modal keypad angka cepat untuk memasukkan jumlah koli (contoh: scan kardus -> input 5 dus -> sistem mencatat 60 Pcs).
     - *Metode Blind Opname:* Petugas gudang tidak diperlihatkan angka stok sistem di layar HP (mencegah bias hitung atau kecurangan), sistem hanya mencatat kondisi riil fisik.
     - *Audit Log Pemindaian (`stock_opname_scans`):* Setiap pemindaian terekam secara detail mencakup barcode fisik yang discan, timestamp, user, dan device ID.
     - *Offline Scan Buffer:* Staf gudang yang melakukan opname di sudut gudang tanpa jangkauan Wi-Fi atau sinyal seluler tetap dapat melakukan scanning tanpa henti. Data tersimpan di SQLite lokal dan otomatis tersinkronisasi saat perangkat kembali ke area berjarak sinyal.

5. **Offline-First POS Engine & Hardware Printer:**
   - Katalog produk dan antrean offline (`offline_orders_queue`) menggunakan Drift (SQLite).
   - Cetak struk kasir thermal ESC/POS via Bluetooth, USB, dan Network TCP/IP port 9100.

### 4.2 Vue.js 3: Spesifikasi Web Backoffice & Analitik
1. **Teknologi Stack:**
   - Vue 3 dengan Composition API (`<script setup>`).
   - Vite sebagai build tool berkecepatan tinggi.
   - Pinia untuk state management terpusat.
   - Tailwind CSS dengan library komponen teruji (Headless UI / Shadcn Vue) tanpa template bento-box yang generik.
2. **Fitur Layar Lebar & Mass Data Processing:**
   - Bulk upload katalog produk via file Excel/CSV dengan validasi skema di sisi browser sebelum dikirim ke server.
   - Form pembuatan Purchase Order (PO) berjenjang ke supplier dengan kalkulasi otomatis subtotal dan pajak.
   - Dashboard Audit Trail Kartu Stok dengan filter mendalam (Rentang tanggal, Lokasi gudang, Tipe pergerakan, dan Akun pelaksana).
   - Modul Review Penggajian: Tabel rekapitulasi gaji bulanan interaktif, penyesuaian manual bonus/potongan sebelum finalisasi, serta tombol aksi "Finalize & Mass Generate PDF".

---

## 5. Arsitektur Teknis Backend Go & Infrastruktur

### 5.1 Struktur Direktori Modular Monolith Go
```
cmd/
  api/
    main.go                   # Inisialisasi HTTP server, database, worker, dan routes
  worker/
    main.go                   # Inisialisasi background worker Asynq
internal/
  platform/                   # Komponen infrastruktur bersama
    database/                 # PostgreSQL pool connection & migrations
    redis/                    # Redis client & caching helper
    storage/                  # MinIO / AWS S3 client wrapper
    middleware/               # Tenant context, Auth JWT, RBAC, Logger, CORS
    idempotency/              # Handler idempotency key validation
  auth/                       # Domain Auth & Role Management
    delivery/http/
    repository/
    usecase/
    domain.go
  product/                    # Domain Master Produk & UOM
    delivery/http/
    repository/
    usecase/
    domain.go
  inventory/                  # Domain Stok, Gudang, & Kartu Stok
    delivery/http/
    repository/
    usecase/
    domain.go
  pos/                        # Domain Kasir, Shift, & Transaksi
    delivery/http/
    repository/
    usecase/
    domain.go
  payroll/                    # Domain Absensi, Komisi, & Penggajian
    delivery/http/
    repository/
    usecase/
    domain.go
  promotion/                  # Domain Marketing, Diskon & Mesin Promosi
    delivery/http/
    repository/
    usecase/
    evaluator/                # Rules Engine (Date, Scope, Tier, Quota)
    domain.go
migrations/                   # File SQL migrasi versi golang-migrate
docker-compose.yml            # Orkestrasi lokal & staging
Dockerfile                    # Multi-stage build binary Go
```

### 5.2 Background Worker Pool (Redis + Asynq)
Untuk menjaga response time HTTP Gin di bawah 100ms, tugas berat dialihkan ke queue worker:
1. **Pembuatan Dokumen PDF Slip Gaji:** Menggunakan library Go PDF (`gofpdf` atau Chromium headless) untuk merender slip gaji terenkripsi.
2. **Ekspor Laporan Finansial:** Pembuatan file Excel (.xlsx) dengan ratusan ribu baris data transaksi penjualan untuk diunduh oleh Finance.
3. **Pengiriman Notifikasi Eksternal:** Integrasi webhook WhatsApp Gateway dan Email SMTP untuk pengiriman nota digital atau slip gaji karyawan.

### 5.3 Topologi Docker & Orkestrasi
File `docker-compose.yml` mencakup 6 service utama:
1. **`reverse-proxy` (Caddy atau Nginx):**
   - Menangani otomatisasi SSL/TLS (Let's Encrypt).
   - Routing: Request `/api/*` diteruskan ke `backend-api:8080`, request non-API disajikan dari static build `web-admin`.
2. **`backend-api` (Go Gin):**
   - Multi-stage Dockerfile: Stage 1 menggunakan `golang:1.23-alpine` untuk kompilasi binary biner statis, Stage 2 menyalin biner ke image minimal `alpine:3.20` atau `distroless`.
3. **`web-admin` (Vue.js):**
   - Multi-stage Dockerfile: Stage 1 menjalankan `npm run build`, Stage 2 menyajikan folder `/dist` melalui container Nginx Alpine berukuran mini.
4. **`postgres`:** PostgreSQL 16 Alpine dengan volume terpasang (`postgres_data`) dan optimasi parameter memori (`shared_buffers=512MB`, `work_mem=16MB`).
5. **`redis`:** Redis 7 Alpine untuk caching dan queue broker dengan mode persistensi AOF (*Append-Only File*).
6. **`db-backup`:** Container cron berkala yang mengeksekusi `pg_dump` terkompresi setiap hari pukul 02:00 pagi dan menyimpannya ke MinIO/S3 eksternal.

---

## 6. Spesifikasi API Kritis (REST Endpoints)

Seluruh response API menggunakan envelope JSON terstandarisasi:
```json
{
  "success": true,
  "code": 200,
  "message": "Operation completed successfully",
  "data": {},
  "error": null
}
```

### 6.1 Daftar Endpoint Inti
| Modul | Method | Endpoint | Deskripsi | Keterangan Header / Concurrency |
|---|---|---|---|---|
| **Auth** | `POST` | `/api/v1/auth/login` | Login user & issue JWT | Body: username, password, tenant_code |
| **Product** | `GET` | `/api/v1/products` | List produk dengan pagination & filter | Cache di Redis (TTL 10 menit) |
| **Product** | `POST` | `/api/v1/products` | Buat produk baru beserta varian & UOM | Invalidasi cache produk di Redis |
| **Inventory**| `GET` | `/api/v1/inventory/stocks` | Cek stok terkini per lokasi | Mengambil dari `inventory_stocks` |
| **Inventory**| `POST` | `/api/v1/inventory/movements`| Buat mutasi manual / kartu stok | Transaksi database ACID |
| **Inventory**| `POST` | `/api/v1/inventory/opname` | Submit hasil stock opname fisik | Status: Draft / Need Approval |
| **POS** | `POST` | `/api/v1/pos/shifts/open` | Buka shift kasir & isi modal laci | Validasi status shift aktif |
| **POS** | `POST` | `/api/v1/pos/orders/checkout`| Eksekusi pembayaran kasir | **Wajib:** `Idempotency-Key` (UUIDv4) |
| **POS** | `POST` | `/api/v1/pos/shifts/close`| Tutup shift kasir & input uang fisik | Cetak X/Z Report data |
| **Promotion**| `GET`  | `/api/v1/promotions` | List promosi aktif per outlet | Scoped by tenant_id & outlet_id |
| **Promotion**| `POST` | `/api/v1/promotions` | Buat kampanye promosi baru | Parameter: Waktu, Produk, Kuota, Budget |
| **Promotion**| `POST` | `/api/v1/promotions/validate` | Validasi & kalkulasi promo keranjang POS | Input: Cart items, outlet, kode kupon |
| **Customer** | `POST` | `/api/v1/customers` | Daftarkan member/pelanggan baru | Nomor telepon unik per tenant |
| **Payroll** | `POST` | `/api/v1/payroll/attendance`| Clock-in/out staf toko | Simpan foto & koordinat GPS |
| **Payroll** | `GET` | `/api/v1/payroll/periods` | Daftar periode penggajian bulanan | Filter status: DRAFT / FINALIZED |
| **Payroll** | `POST` | `/api/v1/payroll/finalize`| Kunci periode & snapshot data gaji | Tidak dapat diedit setelah finalize |
| **Payroll** | `GET` | `/api/v1/payroll/slips/:id/pdf`| Unduh slip gaji terenkripsi PDF | Di-generate oleh background worker |

---

## 7. Roadmap Implementasi Phased MVP

Agar proyek berjalan terarah dan tidak mengalami *over-engineering*, pembangunan dibagi menjadi 5 fase sistematis:

```
+---------------------------------------------------------------------------------------+
|                                  ROADMAP PENGEMBANGAN                                 |
+-------------------+-------------------+-------------------+-------------------+-------+
| FASE 1 (Bulan 1)  | FASE 2 (Bulan 2)  | FASE 3 (Bulan 3)  | FASE 4 (Bulan 4)  | FASE 5|
| Pondasi Inti      | Mesin Uang (POS)  | Gudang Lanjutan   | HR & Penggajian   | SaaS  |
+-------------------+-------------------+-------------------+-------------------+-------+
| - Skema Tenancy   | - Kasir Tablet    | - PO Supplier &   | - Absensi Kiosk   | - Reg |
|   & PostgreSQL    |   Flutter split   |   Good Receiving  |   PIN / Foto      |   SaaS|
| - Auth & RBAC     |   screen          | - Mutasi Stok     | - Auto-Komisi POS | - Sub-|
| - Master Produk,  | - ESC/POS Printer |   Antar Gudang    | - Hitung Gaji &   |   scrip|
|   Varian, & UOM   | - Transaksi Kasir | - Mobile Opname   |   Snapshot        |   tion|
| - Stock Ledger    |   Pessimistic Lock|   App Android     | - Background      | - Sub-|
|   dasar (In/Out)  | - Shift Kasir     | - Rekonsiliasi &  |   Worker PDF      |   dom-|
| - Web Admin Vue 3 | - Offline-First   |   Approval di Web |   Slip Massal     |   ain |
|   dasar           |   Drift SQLite    |                   |                   |   Iso |
+-------------------+-------------------+-------------------+-------------------+-------+
```

### 7.1 Fase 1: Pondasi Basis Data, Auth, Produk, & Stok Dasar
- **Target:** Backend Go Gin siap melayani master data untuk Web Admin Vue.js.
- **Deliverable:**
  - Database PostgreSQL terpasang dengan ekstensi UUID dan skema `tenant_id` terpadu.
  - Modul `internal/auth` dengan JWT authentication dan RBAC middleware.
  - Modul `internal/product` lengkap dengan kategori, varian, dan satuan berjenjang (multi-UOM).
  - Modul `internal/inventory` dasar: Pembuatan lokasi gudang/outlet dan pencatatan manual mutasi masuk/keluar pada `stock_movements`.
  - Web Admin Vue 3: Halaman Login, Manajemen Produk, dan Pemantauan Stok.

### 7.2 Fase 2: Mesin Uang (POS Kasir Flutter, Promosi Dasar & Integrasi Hardware)
- **Target:** Toko dapat melayani transaksi pembayaran kasir di lapangan secara instan dengan dukungan kalkulasi promo otomatis.
- **Deliverable:**
  - Aplikasi Flutter Tablet Kasir: Desain split-screen landscape.
  - Alur transaksi checkout atomic di Go Gin dengan `SELECT ... FOR UPDATE` dan header `Idempotency-Key`.
  - Integrasi Mesin Promosi POS: Diskon otomatis, input voucher/kupon belanja, dan proteksi kuota promo secara real-time.
  - Integrasi thermal receipt printer via Bluetooth dan USB OTG.
  - Modul manajemen shift kasir (Buka Shift, Kas Masuk/Keluar, Tutup Shift, Selisih Kas).
  - Integrasi database lokal Drift (SQLite) untuk kapabilitas offline-first POS.

### 7.3 Fase 3: Logistik Gudang Lanjutan & Kampanye Marketing Web
- **Target:** Alur pengadaan barang tertib dan manajemen kampanye promosi berparameter lengkap di Web Admin.
- **Deliverable:**
  - Fitur Purchase Order (PO) ke supplier di Web Admin: Pembaharuan HPP otomatis saat barang diterima.
  - Fitur Marketing & Promo Builder di Web Admin: Pengaturan jadwal kalender (tanggal, bulan, hari, happy hour), target produk/kategori, alokasi outlet, dan kontrol budget subsidi diskon.
  - Aplikasi Mobile Gudang di Flutter Android: Scan barcode continuous via kamera atau Bluetooth scanner untuk receiving barang.
  - Fitur Mutasi Stok antargudang dengan status `IN_TRANSIT` hingga diverifikasi diterima oleh cabang penerima.
  - Alur Stock Opname berjenjang: Input selisih via mobile, approval selisih oleh supervisor di Web Admin, dan penyesuaian otomatis kartu stok.

### 7.4 Fase 4: Sumber Daya Manusia, Komisi Otomatis, & Penggajian
- **Target:** Otomatisasi penggajian staf toko dan eliminasi kalkulasi manual di akhir bulan.
- **Deliverable:**
  - Fitur Kiosk Absensi karyawan pada tablet kasir (PIN/Foto selfie).
  - Kalkulasi otomatis komisi kasir dari setiap transaksi penjualan POS yang berhasil.
  - Halaman Rekap Payroll di Web Admin: Review absensi, jam lembur, kasbon, dan komisi.
  - Fitur Finalisasi Penggajian: Pembuatan `payroll_snapshots` yang permanen dan immutable.
  - Background Worker Asynq: Ekspor massal slip gaji format PDF dan pengiriman notifikasi otomatis.

### 7.5 Fase 5: Evolusi Menuju Multi-Tenant SaaS
- **Target:** Sistem siap dikomersialkan sebagai layanan cloud SaaS berlangganan untuk banyak pemilik usaha.
- **Deliverable:**
  - Endpoint registrasi tenant mandiri (*self-service registration*) yang otomatis membuat default tenant, outlet default, dan user owner.
  - Integrasi Payment Gateway untuk penagihan langganan bulanan (*subscription billing*).
  - Subdomain routing otomatis di Caddy/Nginx (misal `tokoabc.poscloud.id`).
  - Verifikasi isolasi data ketat via automated security testing dan audit kueri PostgreSQL Row Level Security (RLS).

---

## 8. Standar Keandalan, Keamanan, & Non-Functional Requirements (NFR)

1. **Performa & Latensi:**
   - Latensi checkout transaksi POS di Go Gin harus berada di bawah 250 milidetik pada beban 95th percentile (p95).
   - Katalog produk lokal di tablet Flutter harus mampu memuat 10.000 SKU dalam waktu kurang dari 500 milidetik menggunakan indeks SQLite lokal.
2. **Keamanan Data (Data Security):**
   - Password karyawan dienkripsi menggunakan algoritma `bcrypt` dengan cost factor 12 atau `argon2id`.
   - Seluruh komunikasi data antara Flutter, Vue.js, dan Go Gin wajib menggunakan protokol HTTPS (TLS 1.3).
   - Hak akses data strictly scoped per `tenant_id` dan `outlet_id`. Setiap query SQL diverifikasi agar tidak membocorkan data tenant lain.
3. **Penyimpanan Berkas (Object Storage):**
   - Gambar produk, bukti transfer kasir, dan arsip PDF slip gaji disimpan pada MinIO/S3 dengan nama file berbasis hash UUID acak.
   - Akses file privat (seperti slip gaji) menggunakan URL berbatas waktu (*Pre-Signed URL*) dengan masa kedaluwarsa maksimal 15 menit.
4. **Strategi Pemulihan Bencana (Disaster Recovery):**
   - Container `db-backup` membuat dump database setiap hari dengan rotasi retensi: 7 backup harian, 4 backup mingguan, dan 12 backup bulanan.
   - Write-Ahead Logging (WAL) archiving diaktifkan pada PostgreSQL untuk kapabilitas Point-in-Time Recovery (PITR).
