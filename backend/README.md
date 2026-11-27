# Dokumentasi Teknis Backend: UMKM POS & Enterprise Ecosystem
Sistem backend terpadu untuk Point of Sale (POS kasir), manajemen inventaris gudang, katalog produk multi-satuan (multi-UOM), mesin promosi marketing, dan penggajian (payroll). Dibangun menggunakan **Go (Gin Framework)** dengan arsitektur **Modular Monolith**.

---

## 1. Arsitektur & Prinsip Desain

### 1.1 Mengapa Modular Monolith di Go (Gin)
Pendekatan Modular Monolith dipilih untuk menjamin konsistensi transaksi database ACID:
- **Transaksi Atomik Kasir:** Saat kasir melakukan checkout, sistem dalam satu transaksi database tunggal memotong stok di kartu stok (`stock_movements`), mengunci baris stok dengan *pessimistic locking*, memotong kuota promosi, mencatat pesanan penjualan, dan mencatat komisi penjualan bagi kasir/sales.
- **Efisiensi Sumber Daya:** Binary Go statis berukuran kecil (~15 sampai 25 MB) dengan konsumsi memori rendah (~20 sampai 50 MB RAM saat idle), sangat hemat biaya operasional di VPS.
- **Batas Domain Bersih:** Seluruh kode dipisahkan per modul bisnis di dalam direktori `internal/` sehingga terhindar dari keterikatan liar (*spaghetti dependencies*) dan siap diekstrak jika suatu modul membutuhkan skala mandiri di masa depan.

### 1.2 Pola Dual-Identifier (`BIGSERIAL id` vs `UUID uuid`)
- **`ID uint64` (Internal Primary Key):** Digunakan sebagai Foreign Key (FK) antartabel di dalam PostgreSQL. Integer 64-bit (8 bytes) menghasilkan indeks B-Tree yang padat di RAM, mempercepat kueri `JOIN` hingga 3x sampai 4x lipat dibanding UUID.
- **`UUID uuid.UUID` (External Public Key):** Digunakan pada seluruh endpoint REST API publik (contoh: `/api/v1/products/:uuid` atau `/api/v1/pos/orders/:uuid`). Mencegah kebocoran volume data riil bisnis dan serangan enumerasi ID berurutan.

### 1.3 Kesiapan Tenancy (Single-Tenant ke Multi-Tenant)
- Setiap tabel transaksional dan master memiliki kolom `tenant_id uint64 NOT NULL`.
- Pada mode **Single-Tenant**: Aplikasi berjalan dengan default `tenant_id = 1` tanpa perlu mengubah skema database.
- Pada mode **Multi-Tenant (SaaS)**: Middleware secara otomatis membaca identitas tenant dari klaim JWT, subdomain, atau header `X-Tenant-ID`.

### 1.4 Standar Presisi Moneter (`int64` / `BIGINT`)
Seluruh nilai mata uang (harga jual, HPP, diskon, gaji pokok, komisi, dan nominal laci kasir) disimpan dalam satuan rupiah terkecil bertipe `int64` / `BIGINT`. Dilarang keras menggunakan tipe data mengambang (`float32` / `float64`) untuk mencegah distorsi pembulatan pecahan desimal.

---

## 2. Struktur Direktori Proyek

```
backend/
├── cmd/
│   └── api/
│       └── main.go                  # Bootstrap server HTTP, inisialisasi koneksi & routes
├── docs/
│   ├── swagger.json                 # Spesifikasi OpenAPI / Swagger (JSON)
│   └── swagger.yaml                 # Spesifikasi OpenAPI / Swagger (YAML)
├── internal/
│   ├── auth/                        # Domain Autentikasi & RBAC
│   │   ├── dto.go                   # Request & response data transfer objects
│   │   ├── handler.go               # Gin HTTP handlers
│   │   ├── repository.go            # GORM database queries
│   │   └── usecase.go               # Logika bisnis pendaftaran & login
│   ├── inventory/                   # Domain Stok, Mutasi, Opname, Supplier & Purchase Order
│   ├── middleware/                  # HTTP Middlewares terpusat
│   │   ├── auth.go                  # Verifikasi JWT & injeksi context
│   │   ├── auth_test.go             # Unit test autentikasi & RBAC
│   │   ├── cors.go                  # Cross-Origin Resource Sharing handler
│   │   ├── rbac.go                  # Guard pembatas Role & Permission
│   │   └── tenant.go                # Resolver konteks tenant
│   ├── model/                       # Definisi Entitas GORM & Relasi Database
│   │   ├── auth.go                  # User, Role, Permission, RolePermission, UserOutlet
│   │   ├── base.go                  # BaseModel, TenantScopedModel
│   │   ├── inventory.go             # Stock, Movement, Supplier, PO, Transfer, Opname
│   │   ├── payroll.go               # Employee, Salary, Attendance, Commission, Snapshot
│   │   ├── pos.go                   # Shift, PaymentMethod, Order, OrderItem, Payment
│   │   ├── product.go               # Category, Product, Variant, UOM, CostHistory
│   │   ├── promotion.go             # Customer, Promotion, PromoOutlet, PromoRule, UsageLog
│   │   └── tenant.go                # Tenant, Outlet, InventoryLocation
│   ├── payroll/                     # Domain Absensi Kiosk, Komisi, Penggajian & PDF Slip Gaji
│   │   ├── dto.go
│   │   ├── handler.go
│   │   ├── pdf.go                   # Generator PDF 1.4 slip gaji karyawan
│   │   ├── repository.go
│   │   ├── usecase.go
│   │   └── payroll_test.go
│   ├── platform/
│   │   └── database/
│   │       └── postgres.go          # Pool connection GORM & auto-seed RBAC
│   ├── pos/                         # Domain Kasir POS, Shift, Komisi Otomatis & Laporan Finansial
│   ├── product/                     # Domain Katalog Produk, Varian & Multi-UOM
│   └── promotion/                   # Domain Pelanggan & Mesin Promosi Marketing
├── pkg/
│   ├── hash/                        # Bcrypt password hashing & salted PIN hashing
│   ├── jwt/                         # Token generator, validator & claims parser
│   │   └── jwt_test.go              # Unit test JWT expiration & validation
│   └── response/                    # Envelope format JSON standar
├── .dockerignore
├── .env.example                     # Contoh variabel lingkungan
├── Dockerfile                       # Multi-stage Docker build
├── go.mod                           # Definisi modul & dependensi Go
├── go.sum                           # Checksum hash dependensi
└── README.md                        # Dokumentasi teknis ini
```

---

## 3. Prasyarat & Panduan Menjalankan

### 3.1 Prasyarat Perangkat Lunak
- **Go:** Versi 1.24 atau lebih baru
- **PostgreSQL:** Versi 16 atau lebih baru (dengan ekstensi `pgcrypto`)
- **Redis:** Versi 7 atau lebih baru
- **Docker & Docker Compose** (opsional untuk menjalankan kontainer)

### 3.2 Konfigurasi Environment (`.env`)
Salin berkas konfigurasi default:
```bash
cp .env.example .env
```

Isi variabel lingkungan sesuai kebutuhan:
```env
APP_ENV=development
APP_MODE=single-tenant
PORT=8080

DEFAULT_TENANT_ID=1
DEFAULT_TENANT_UUID=00000000-0000-0000-0000-000000000001

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=umkm_pos
DB_SSLMODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379

JWT_SECRET=umkm-pos-super-secret-key-change-in-production
JWT_ACCESS_EXPIRY=2h
JWT_REFRESH_EXPIRY=168h
```

### 3.3 Menjalankan Secara Lokal (Local Development)

1. **Jalankan Database PostgreSQL & Redis via Docker Compose:**
   Dari root direktori proyek:
   ```bash
   docker compose up -d postgres redis
   ```

2. **Jalankan Aplikasi Backend Go:**
   Masuk ke folder `backend`:
   ```bash
   cd backend
   go run cmd/api/main.go
   ```

Aplikasi akan berjalan dan mendengarkan request di `http://localhost:8080`.

---

## 4. Sistem Hak Akses (RBAC Matrix)

Sistem menggunakan kontrol hak akses berbasis peran (Role) dan izin granular (Permission):

| Role | Deskripsi Peran | Akses Modul | Daftar Permission Bawaan |
|---|---|---|---|
| **`SUPERADMIN`** | Pemilik Usaha / Super Admin | Seluruh Modul (Akses Penuh Tanpa Batas) | Memiliki bypass universal ke seluruh permission |
| **`STORE_MANAGER`** | Supervisor / Manajer Toko | Backoffice Toko, Review Absensi, Approval Opname, Void Kasir | `product:read`, `inventory:*`, `pos:*`, `promotion:manage`, `payroll:read` |
| **`CASHIER`** | Kasir (Frontliner) | Kasir POS Tablet, Buka/Tutup Shift, Aplikasi Diskon | `product:read`, `pos:checkout`, `pos:shift` |
| **`WAREHOUSE`** | Staf Gudang | Receiving Barang PO, Mutasi Antar Rak, Scan Stock Opname | `product:read`, `inventory:read`, `inventory:transfer`, `inventory:opname` |
| **`HR_FINANCE`** | HR & Keuangan | Absensi, Lembur, Komisi Penjualan, Finalisasi Gaji Bulanan | `product:read`, `inventory:read`, `payroll:read`, `payroll:finalize` |

---

## 5. Dokumentasi RESTful API

Seluruh respons API dibungkus dalam format envelope JSON standar:
```json
{
  "success": true,
  "code": 200,
  "message": "Operation description",
  "data": {},
  "error": null
}
```

### 5.1 Endpoint Publik (Authentication)

#### 1. Registrasi User Baru
- **Method:** `POST`
- **URL:** `/api/v1/auth/register`
- **Headers:** `Content-Type: application/json`
- **Request Body:**
  ```json
  {
    "username": "kasir_deni",
    "email": "deni@tokoberkah.com",
    "phone": "08123456789",
    "password": "Password123!",
    "full_name": "Deni Handoko",
    "role_name": "CASHIER",
    "pin": "123456",
    "tenant_code": "default",
    "outlet_ids": [1]
  }
  ```
- **Response (201 Created):**
  ```json
  {
    "success": true,
    "code": 201,
    "message": "User registered successfully",
    "data": {
      "token": {
        "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "expires_at": "2026-11-22T13:00:00Z"
      },
      "user": {
        "id": 1,
        "uuid": "7c88b901-b541-4775-9e66-6b22eb8ec982",
        "tenant_id": 1,
        "username": "kasir_deni",
        "full_name": "Deni Handoko",
        "email": "deni@tokoberkah.com",
        "role": "CASHIER",
        "is_active": true
      },
      "tenant": {
        "id": 1,
        "uuid": "00000000-0000-0000-0000-000000000001",
        "code": "default",
        "name": "UMKM Retail Store (Default)"
      },
      "permissions": ["product:read", "pos:checkout", "pos:shift"],
      "outlets": [
        {
          "id": 1,
          "code": "MAIN-01",
          "name": "Toko Pusat (Utama)",
          "is_main_branch": true
        }
      ]
    }
  }
  ```

#### 2. Login User
- **Method:** `POST`
- **URL:** `/api/v1/auth/login`
- **Request Body:**
  ```json
  {
    "username": "kasir_deni",
    "password": "Password123!"
  }
  ```

#### 3. Quick PIN Login (Pertukaran Kasir POS Cepat)
- **Method:** `POST`
- **URL:** `/api/v1/auth/login/pin`
- **Request Body:**
  ```json
  {
    "username": "kasir_deni",
    "pin": "123456"
  }
  ```

---

### 5.2 Endpoint Terproteksi (Protected Endpoints)

Seluruh permintaan ke endpoint terproteksi wajib menyertakan token JWT pada header:
```http
Authorization: Bearer <access_token>
```

#### 1. Profil Pengguna yang Sedang Login
- **Method:** `GET`
- **URL:** `/api/v1/auth/me`
- **Headers:** `Authorization: Bearer <access_token>`

#### 2. Uji Akses Berdasarkan Role & Permission
- `GET /api/v1/admin/stats` : Dibatasi untuk role `SUPERADMIN` dan `STORE_MANAGER`.
- `GET /api/v1/pos/ping` : Dibatasi untuk permission `pos:checkout`.
- `GET /api/v1/inventory/ping` : Dibatasi untuk permission `inventory:opname`.
- `GET /api/v1/payroll/ping` : Dibatasi untuk permission `payroll:finalize`.

---

### 5.3 Modul Master Produk & Dual-Barcode Scanner

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| `GET` | `/api/v1/categories` | `product:read` | Menampilkan struktur kategori hierarkis |
| `POST` | `/api/v1/categories` | `product:create` | Menambah kategori produk baru |
| `PUT` | `/api/v1/categories/:uuid` | `product:update` | Mengubah nama atau status kategori |
| `DELETE` | `/api/v1/categories/:uuid` | `product:delete` | Menghapus kategori |
| `GET` | `/api/v1/products` | `product:read` | Pencarian produk dengan pagination, filter kategori, dan status |
| `POST` | `/api/v1/products` | `product:create` | Membuat produk baru beserta varian SKU dan multi-UOM bertingkat |
| `GET` | `/api/v1/products/:uuid` | `product:read` | Detail produk beserta varian dan satuan kemasan UOM |
| `PUT` | `/api/v1/products/:uuid` | `product:update` | Mengubah informasi master produk |
| `DELETE` | `/api/v1/products/:uuid` | `product:delete` | Soft delete produk |
| `GET` | `/api/v1/products/barcode/:code` | `product:read` | Endpoint pemindaian dual-barcode (EAN-13 eceran atau ITF-14 kemasan dus) |

---

### 5.4 Modul Manajemen Lokasi & Stok Gudang (Inventory)

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| `POST` | `/api/v1/inventory/locations` | `inventory:transfer` | Menambah lokasi fisik penyimpanan baru (WAREHOUSE, STORE_FRONT, TRANSIT, DAMAGED) |
| `GET` | `/api/v1/inventory/locations` | `inventory:read` | Menampilkan daftar lokasi penyimpanan per outlet |
| `GET` | `/api/v1/inventory/stocks` | `inventory:read` | Menampilkan saldo stok real-time per lokasi dan varian (mendukung filter low-stock alert) |
| `GET` | `/api/v1/inventory/movements` | `inventory:read` | Kartu stok / mutasi ledger append-only immutable untuk jejak audit |
| `POST` | `/api/v1/inventory/transfers` | `inventory:transfer` | Menginisiasi mutasi stok antar lokasi, memotong stok asal secara atomik (status `IN_TRANSIT`) |
| `GET` | `/api/v1/inventory/transfers` | `inventory:read` | Daftar mutasi transfer antar gudang |
| `GET` | `/api/v1/inventory/transfers/:uuid` | `inventory:read` | Detail permintaan dan item mutasi transfer |
| `POST` | `/api/v1/inventory/transfers/:uuid/receive` | `inventory:transfer` | Penerimaan barang mutasi di lokasi tujuan, menambah stok dan mencatat ledger (status `RECEIVED`) |
| `POST` | `/api/v1/inventory/opnames` | `inventory:opname` | Membuka sesi audit fisik / stock opname baru (status `DRAFT`) |
| `GET` | `/api/v1/inventory/opnames` | `inventory:read` | Daftar riwayat sesi stock opname |
| `GET` | `/api/v1/inventory/opnames/:uuid` | `inventory:read` | Detail sesi opname, daftar selisih fisik vs sistem, dan log pemindaian |
| `POST` | `/api/v1/inventory/opnames/:uuid/scan` | `inventory:opname` | Pemindaian barcode fisik mobile (otomatis mengonversi barcode kardus menjadi base quantity) |
| `POST` | `/api/v1/inventory/opnames/:uuid/submit` | `inventory:opname` | Mengirim hasil audit untuk ditinjau supervisor (status `SUBMITTED`) |
| `POST` | `/api/v1/inventory/opnames/:uuid/approve` | `inventory:approve-opname` | Persetujuan manajer: menyesuaikan saldo stok dan mencatat selisih ke kartu stok (status `APPROVED`) |
| `POST` | `/api/v1/inventory/adjustments` | `inventory:approve-opname` | Penyesuaian stok manual langsung untuk barang rusak atau pembukuan saldo awal |
| `POST` | `/api/v1/inventory/suppliers` | `inventory:transfer` | Mendaftarkan data supplier baru |
| `GET` | `/api/v1/inventory/suppliers` | `inventory:read` | Daftar supplier aktif per tenant |
| `POST` | `/api/v1/inventory/purchase-orders` | `inventory:transfer` | Membuat Purchase Order (PO) ke supplier |
| `GET` | `/api/v1/inventory/purchase-orders` | `inventory:read` | Daftar Purchase Order dengan filter status |
| `GET` | `/api/v1/inventory/purchase-orders/:uuid` | `inventory:read` | Detail Purchase Order beserta item pesanan |
| `POST` | `/api/v1/inventory/purchase-orders/:uuid/receive` | `inventory:transfer` | Penerimaan barang PO (parsial/penuh), otomatis menambah stok & mencatat kartu stok `IN` |

---

### 5.5 Modul Kasir POS & Transaksi Checkout

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| `POST` | `/api/v1/pos/shifts/open` | `pos:shift` | Membuka shift kasir dengan modal awal kas fisik |
| `GET` | `/api/v1/pos/shifts/current` | `pos:shift` | Mengecek status dan saldo shift aktif kasir yang sedang login |
| `POST` | `/api/v1/pos/shifts/cash-log` | `pos:shift` | Mencatat kas masuk / kas keluar kecil (petty cash) selama shift aktif |
| `POST` | `/api/v1/pos/shifts/close` | `pos:shift` | Menutup shift kasir, menghitung selisih kas fisik vs sistem |
| `GET` | `/api/v1/pos/shifts` | `pos:shift` | Rekap riwayat seluruh sesi shift kasir |
| `GET` | `/api/v1/pos/payment-methods` | `pos:checkout` | Daftar metode pembayaran aktif (CASH, QRIS, DEBIT, CREDIT, TRANSFER) |
| `POST` | `/api/v1/pos/checkout` | `pos:checkout` | Transaksi checkout kasir dengan idempotency key, split payment, penguncian stok, dan kalkulasi komisi otomatis |
| `POST` | `/api/v1/pos/orders` | `pos:checkout` | Alias endpoint checkout pesanan kasir |
| `GET` | `/api/v1/pos/orders` | `pos:checkout` | Daftar histori transaksi penjualan |
| `GET` | `/api/v1/pos/orders/:uuid` | `pos:checkout` | Detail item, diskon, dan metode pembayaran pesanan |
| `GET` | `/api/v1/pos/orders/:uuid/receipt` | `pos:checkout` | Data struk belanja siap cetak ke thermal printer ESC/POS (58mm/80mm) |
| `POST` | `/api/v1/pos/orders/:uuid/void` | `pos:void` | Pembatalan transaksi pesanan: memulihkan stok, saldo kas laci, dan membatalkan komisi `PENDING` |
| `GET` | `/api/v1/pos/reports/summary` | `payroll:read` | Ringkasan laporan penjualan, HPP, laba kotor, dan breakdown metode pembayaran |
| `GET` | `/api/v1/pos/reports/export` | `payroll:read` | Ekspor laporan transaksi penjualan dalam format CSV (kompatibel dengan Excel) |

---

### 5.6 Modul Promosi & Loyalitas Pelanggan (Marketing Engine)

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| `POST` | `/api/v1/customers` | `pos:checkout` | Pendaftaran pelanggan / member baru |
| `GET` | `/api/v1/customers` | `pos:checkout` | Pencarian daftar pelanggan berdasarkan nama, nomor HP, atau kode |
| `GET` | `/api/v1/customers/:uuid` | `pos:checkout` | Detail profil pelanggan, tier loyalitas, dan total belanja |
| `PUT` | `/api/v1/customers/:uuid` | `pos:checkout` | Mengubah informasi pelanggan |
| `POST` | `/api/v1/promotions` | `promotion:manage` | Membuat kampanye promosi baru dengan konfigurasi waktu, target produk/kategori, kuota, dan budget |
| `GET` | `/api/v1/promotions` | `promotion:manage` | Menampilkan daftar promosi aktif dan histori kampanye |
| `GET` | `/api/v1/promotions/:uuid` | `promotion:manage` | Detail kampanye promosi beserta daftar aturan produk dan outlet |
| `PUT` | `/api/v1/promotions/:uuid` | `promotion:manage` | Mengubah parameter promosi |
| `DELETE` | `/api/v1/promotions/:uuid` | `promotion:manage` | Menghapus atau menonaktifkan kampanye promosi |
| `POST` | `/api/v1/promotions/evaluate` | `pos:checkout` | Engine evaluator kalkulasi diskon otomatis dan validasi kupon belanja |

---

### 5.7 Modul Absensi, Komisi & Penggajian (Payroll)

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| `POST` | `/api/v1/payroll/employees` | `payroll:finalize` | Registrasi karyawan baru beserta struktur gaji awal (opsional) |
| `GET` | `/api/v1/payroll/employees` | `payroll:read` | Daftar karyawan dengan filter outlet, status aktif, dan pencarian |
| `GET` | `/api/v1/payroll/employees/:uuid` | `payroll:read` | Detail karyawan beserta struktur gaji |
| `PUT` | `/api/v1/payroll/employees/:uuid` | `payroll:finalize` | Memperbarui profil dan status karyawan |
| `PUT` | `/api/v1/payroll/employees/:uuid/salary-structure` | `payroll:finalize` | Mengatur gaji pokok, tunjangan, tarif komisi, dan potongan BPJS |
| `POST` | `/api/v1/payroll/attendance/clock-in` | Authenticated | Clock-in absensi kiosk (verifikasi PIN, foto liveness, dan menit keterlambatan) |
| `POST` | `/api/v1/payroll/attendance/clock-out` | Authenticated | Clock-out absensi kiosk (foto dan pencatatan menit lembur) |
| `POST` | `/api/v1/payroll/attendance` | `payroll:finalize` | Input atau koreksi absensi manual oleh HR/Admin (`PRESENT`, `LATE`, `ABSENT`, `LEAVE`, `SICK`) |
| `GET` | `/api/v1/payroll/attendance` | `payroll:read` | Rekapitulasi log absensi karyawan berdasarkan rentang tanggal |
| `POST` | `/api/v1/payroll/commissions/calculate` | `payroll:finalize` | Kalkulasi dan antrean komisi penjualan secara idempoten per order |
| `GET` | `/api/v1/payroll/commissions` | `payroll:read` | Daftar komisi penjualan (`PENDING`, `PAID`, `VOID`) |
| `POST` | `/api/v1/payroll/commissions/:uuid/void` | `payroll:finalize` | Membatalkan entri komisi yang masih `PENDING` |
| `POST` | `/api/v1/payroll/periods` | `payroll:finalize` | Membuka periode penggajian bulanan baru (status `DRAFT`) |
| `GET` | `/api/v1/payroll/periods` | `payroll:read` | Daftar periode penggajian |
| `GET` | `/api/v1/payroll/periods/:uuid` | `payroll:read` | Detail periode penggajian (beserta snapshot jika sudah `FINALIZED`) |
| `GET` | `/api/v1/payroll/periods/:uuid/preview` | `payroll:read` | Simulasi perhitungan gaji bersih seluruh karyawan sebelum finalisasi |
| `POST` | `/api/v1/payroll/periods/:uuid/finalize` | `payroll:finalize` | Mengunci periode, membuat `payroll_snapshots` immutable, dan menandai komisi `PAID` |
| `GET` | `/api/v1/payroll/slips` | `payroll:read` | Daftar slip gaji final yang telah dikunci |
| `GET` | `/api/v1/payroll/slips/:uuid` | `payroll:read` | Detail slip gaji beserta rincian item pendapatan dan potongan |
| `GET` | `/api/v1/payroll/slips/:uuid/pdf` | `payroll:read` | Mengunduh dokumen slip gaji format PDF |

---

### 5.8 Dokumentasi Swagger / OpenAPI

- **Swagger UI:** `GET /docs`
- **OpenAPI Spec (JSON):** `GET /api/v1/docs/swagger.json`
- **File Statis:** [docs/swagger.json](docs/swagger.json) & [docs/swagger.yaml](docs/swagger.yaml)

Untuk men-generate ulang spesifikasi Swagger setelah mengubah anotasi handler:
```bash
cd backend
swag init -g cmd/api/main.go -o docs --outputTypes json,yaml --parseInternal
```

---

## 6. Pengujian Unit & Otomasi (Testing)

Untuk menjalankan seluruh rangkaian pengujian unit (unit test) dan integrasi middleware:
```bash
cd backend
go test -v ./...
```

Hasil pengujian yang berhasil:
```
=== RUN   TestAuthAndRBACMiddleware
--- PASS: TestAuthAndRBACMiddleware (0.00s)
=== RUN   TestJWTGenerateAndValidate
--- PASS: TestJWTGenerateAndValidate (0.00s)
=== RUN   TestJWTExpired
--- PASS: TestJWTExpired (0.01s)
=== RUN   TestProductAndBarcodeScanFlow
--- PASS: TestProductAndBarcodeScanFlow (0.00s)
=== RUN   TestInventoryLocationsAndManualAdjustment
--- PASS: TestInventoryLocationsAndManualAdjustment (0.00s)
=== RUN   TestStockTransferWorkflow
--- PASS: TestStockTransferWorkflow (0.00s)
=== RUN   TestStockOpnameDualBarcodeScanAndApproval
--- PASS: TestStockOpnameDualBarcodeScanAndApproval (0.00s)
=== RUN   TestSupplierAndPurchaseOrderReceiving
--- PASS: TestSupplierAndPurchaseOrderReceiving (0.00s)
=== RUN   TestPosShiftLifecycleAndCashLogs
--- PASS: TestPosShiftLifecycleAndCashLogs (0.00s)
=== RUN   TestPosCheckoutIdempotencyAndPessimisticStock
--- PASS: TestPosCheckoutIdempotencyAndPessimisticStock (0.00s)
=== RUN   TestPosSplitPaymentThermalReceiptAndVoid
--- PASS: TestPosSplitPaymentThermalReceiptAndVoid (0.00s)
=== RUN   TestAutoCommissionAndSalesReport
--- PASS: TestAutoCommissionAndSalesReport (0.00s)
=== RUN   TestCustomerRegistrationAndListing
--- PASS: TestCustomerRegistrationAndListing (0.00s)
=== RUN   TestPromotionDateAndTimeSchedulingHappyHour
--- PASS: TestPromotionDateAndTimeSchedulingHappyHour (0.00s)
=== RUN   TestCouponCodeValidationAndTargetedProductScope
--- PASS: TestCouponCodeValidationAndTargetedProductScope (0.00s)
=== RUN   TestAttendanceAndPayrollLifecycle
--- PASS: TestAttendanceAndPayrollLifecycle (0.00s)
=== RUN   TestPayrollHTTPRoutesAndRBAC
--- PASS: TestPayrollHTTPRoutesAndRBAC (0.00s)
PASS
ok      github.com/handokodenih/umkm_pos/backend/internal/inventory     0.699s
ok      github.com/handokodenih/umkm_pos/backend/internal/middleware    0.252s
ok      github.com/handokodenih/umkm_pos/backend/internal/payroll       1.149s
ok      github.com/handokodenih/umkm_pos/backend/internal/pos           1.579s
ok      github.com/handokodenih/umkm_pos/backend/internal/product       1.983s
ok      github.com/handokodenih/umkm_pos/backend/internal/promotion     2.375s
ok      github.com/handokodenih/umkm_pos/backend/pkg/jwt                1.397s
```

---

## 7. Deployment Menggunakan Docker

### 7.1 Membangun Image Docker Mandiri
```bash
cd backend
docker build -t umkm-pos-backend:latest .
```

Karakteristik Image:
- Menggunakan Alpine Linux 3.20 (~15 sampai 25 MB).
- Pengguna non-root `appuser`.
- Zona waktu Indonesia Barat (`TZ=Asia/Jakarta`).
- Healthcheck otomatis pada port 8080.

### 7.2 Menjalankan via Docker Compose
Dari direktori utama proyek:
```bash
docker compose up --build -d
```
Berkas `docker-compose.yml` akan menginisialisasi:
1. Container `umkm_pos_postgres` (PostgreSQL 16) dengan inisialisasi skema otomatis dari `database/schema.sql`.
2. Container `umkm_pos_redis` (Redis 7) dengan persistensi AOF.
3. Container `umkm_pos_backend` (Go Gin API Server) yang terhubung ke jaringan bridge internal `app_network`.
