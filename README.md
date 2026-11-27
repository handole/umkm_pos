# UMKM POS & Enterprise Operations Ecosystem

Sistem terpadu untuk operasional **Point of Sale (POS) Kasir**, **Manajemen Stok Gudang**, **Katalog Produk Multi-UOM**, **Mesin Promosi Marketing**, serta **Absensi & Penggajian (Payroll)**.

---

## Ringkasan Arsitektur & Tech Stack

| Komponen | Direktori | Teknologi |
|---|---|---|
| **Backend API** | [`backend/`](backend/) | Go (1.24+), Gin Framework, GORM, JWT, Swagger/OpenAPI (Modular Monolith) |
| **Web Admin** | [`frontend/`](frontend/) | Vue 3, TypeScript, Vite, Pinia, Vue Router, Tailwind CSS v4 |
| **Database & Cache** | [`database/`](database/) | PostgreSQL 16 (auto-init `schema.sql`), Redis 7 |
| **Mobile & Tablet** | [`mobile/`](mobile/) | Flutter (dalam pengembangan / placeholder) |
| **Orkestrasi** | [`docker-compose.yml`](docker-compose.yml) | Docker Compose (Postgres, Redis, Backend API, Web Admin Nginx) |

---

## Rangkuman Dokumentasi

Seluruh spesifikasi dan desain sistem tersedia pada dokumen berikut:

- **[`PRD.md`](PRD.md)** — Product Requirement Document: latar belakang bisnis, arsitektur Modular Monolith, cakupan modul (Auth/RBAC, Produk, Inventaris, Promosi, POS, Payroll), alur transaksi atomik, dan strategi multi-tenant.
- **[`DATABASE.md`](DATABASE.md)** — Dokumentasi skema database PostgreSQL 16+: pola *dual-identifier* (`BIGSERIAL id` internal + `UUID` publik), standar presisi moneter `BIGINT`, ERD lengkap, katalog tabel, dan strategi *row-level locking*.
- **[`database/schema.sql`](database/schema.sql)** — Skrip DDL SQL lengkap yang otomatis dieksekusi saat container PostgreSQL pertama kali dibuat.
- **[`backend/README.md`](backend/README.md)** — Dokumentasi teknis backend: struktur modul `internal/`, matriks RBAC (`SUPERADMIN`, `STORE_MANAGER`, `CASHIER`, `WAREHOUSE`, `HR_FINANCE`), format response JSON, serta panduan endpoint REST API.
- **[`backend/docs/swagger.yaml`](backend/docs/swagger.yaml)** & **[`backend/docs/swagger.json`](backend/docs/swagger.json)** — Spesifikasi OpenAPI / Swagger untuk seluruh endpoint `/api/v1/*`.
- **[`frontend/DESIGN.md`](frontend/DESIGN.md)** — Panduan desain antarmuka Web Admin.

---

## Prasyarat

Pilih sesuai metode menjalankan aplikasi:

- **Menjalankan semua via Docker (Direkomendasikan):**
  - Docker & Docker Compose
- **Menjalankan secara lokal (Local Development):**
  - Docker & Docker Compose (untuk PostgreSQL 16 & Redis 7)
  - Go `1.24+`
  - Node.js & `npm`

---

## Setup Awal (Environment)

1. **Clone repository** dan masuk ke root proyek:
   ```bash
   cd umkm_pos
   ```

2. **Siapkan file `.env` di root** (digunakan oleh `docker-compose.yml`):
   ```env
   APP_ENV=development
   APP_MODE=single-tenant
   PORT=8080
   FRONTEND_PORT=3000
   TZ=Asia/Jakarta

   DEFAULT_TENANT_ID=1
   DEFAULT_TENANT_UUID=00000000-0000-0000-0000-000000000001

   POSTGRES_DB=umkm_pos
   POSTGRES_USER=postgres
   POSTGRES_PASSWORD=postgres
   DB_HOST=postgres
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=postgres
   DB_NAME=umkm_pos
   DB_SSLMODE=disable

   REDIS_HOST=redis
   REDIS_PORT=6379

   JWT_SECRET=ganti-dengan-secret-aman-di-production
   JWT_ACCESS_EXPIRY=2h
   JWT_REFRESH_EXPIRY=168h
   ```

3. **Siapkan file `backend/.env`** (khusus jika menjalankan backend langsung di host dengan `go run`):
   ```bash
   cp backend/.env.example backend/.env
   ```
   > **Catatan:** Pada `backend/.env` untuk local development, pastikan `DB_HOST=localhost` dan `REDIS_HOST=localhost`. Sedangkan pada `.env` root untuk Docker Compose, gunakan `DB_HOST=postgres` dan `REDIS_HOST=redis`.

---

## Cara Menjalankan Aplikasi

### Opsi 1: Menjalankan Semua Service dengan Docker Compose

Cara tercepat untuk menjalankan PostgreSQL, Redis, Backend API, dan Web Admin sekaligus:

```bash
docker compose up -d --build
```

Cek status container:
```bash
docker compose ps
```

Akses aplikasi setelah seluruh container sehat (*healthy*):
- **Web Admin (Nginx):** `http://localhost:3000`
- **Backend API:** `http://localhost:8080`
- **Health Check:** `http://localhost:8080/health` (atau `http://localhost:3000/health`)
- **Swagger / API Docs:** `http://localhost:8080/docs` (atau `http://localhost:3000/docs`)

---

### Opsi 2: Local Development (Hybrid)

Cocok saat melakukan pengembangan aktif pada backend Go dan frontend Vue tanpa harus rebuild container tiap kali ada perubahan kode.

1. **Jalankan PostgreSQL & Redis via Docker Compose:**
   ```bash
   docker compose up -d postgres redis
   ```

2. **Jalankan Backend (Go):**
   Buka terminal baru:
   ```bash
   cd backend
   go run cmd/api/main.go
   ```
   Backend akan berjalan di `http://localhost:8080`.

3. **Jalankan Frontend (Vue + Vite):**
   Buka terminal baru:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```
   Frontend dev server akan berjalan di `http://localhost:5173` (request `/api/*` otomatis di-proxy ke `http://localhost:8080`).

---

## Perintah Berguna

- **Menjalankan unit test backend:**
  ```bash
  cd backend && go test ./...
  ```
- **Build production frontend:**
  ```bash
  cd frontend && npm run build
  ```
- **Melihat log Docker service:**
  ```bash
  docker compose logs -f backend-api
  ```
- **Menghentikan seluruh container:**
  ```bash
  docker compose down
  ```
- **Reset database (hapus container beserta volume data):**
  ```bash
  docker compose down -v
  ```
  > Setelah `down -v`, `database/schema.sql` akan dieksekusi ulang secara otomatis saat container PostgreSQL dijalankan kembali.
