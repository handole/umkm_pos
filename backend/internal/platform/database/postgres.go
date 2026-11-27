package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config contains PostgreSQL connection parameters.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// LoadConfigFromEnv builds database config from environment variables with safe defaults.
func LoadConfigFromEnv() Config {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	pass := os.Getenv("DB_PASSWORD")
	if pass == "" {
		pass = "postgres"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "umkm_pos"
	}
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	return Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		DBName:   dbname,
		SSLMode:  sslmode,
	}
}

// NewPostgresDB establishes a connection pool to PostgreSQL via GORM.
func NewPostgresDB(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Jakarta",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("PostgreSQL connection pool initialized successfully")
	return db, nil
}

// SeedDefaultRolesAndPermissions ensures baseline RBAC data exists.
func SeedDefaultRolesAndPermissions(db *gorm.DB) error {
	defaultPermissions := []model.Permission{
		{Name: "product:read", Module: "product", Description: "Melihat daftar dan detail produk"},
		{Name: "product:create", Module: "product", Description: "Membuat produk baru"},
		{Name: "product:update", Module: "product", Description: "Mengubah data produk dan harga"},
		{Name: "product:delete", Module: "product", Description: "Menghapus produk"},
		{Name: "inventory:read", Module: "inventory", Description: "Melihat saldo stok dan kartu stok"},
		{Name: "inventory:transfer", Module: "inventory", Description: "Membuat permohonan mutasi antar gudang"},
		{Name: "inventory:opname", Module: "inventory", Description: "Melakukan scan hitung fisik stock opname"},
		{Name: "inventory:approve-opname", Module: "inventory", Description: "Menyetujui penyesuaian selisih stock opname"},
		{Name: "pos:checkout", Module: "pos", Description: "Melakukan transaksi pembayaran kasir"},
		{Name: "pos:shift", Module: "pos", Description: "Buka dan tutup laci shift kasir"},
		{Name: "pos:void", Module: "pos", Description: "Membatalkan transaksi pesanan kasir"},
		{Name: "promotion:manage", Module: "promotion", Description: "Membuat dan mengelola kampanye promosi"},
		{Name: "payroll:read", Module: "payroll", Description: "Melihat rekapitulasi absensi dan gaji"},
		{Name: "payroll:finalize", Module: "payroll", Description: "Mengunci dan memfinalisasi slip gaji bulanan"},
		{Name: "user:manage", Module: "auth", Description: "Mengelola staf dan hak akses pengguna"},
	}

	for _, p := range defaultPermissions {
		var existing model.Permission
		if err := db.Where("name = ?", p.Name).First(&existing).Error; err != nil {
			if err := db.Create(&p).Error; err != nil {
				return err
			}
		}
	}

	// Default system roles
	systemRoles := []struct {
		Name        string
		DisplayName string
		Description string
		PermNames   []string
	}{
		{
			Name:        "SUPERADMIN",
			DisplayName: "Super Administrator / Owner",
			Description: "Akses penuh ke seluruh modul sistem",
			PermNames: []string{
				"product:read", "product:create", "product:update", "product:delete",
				"inventory:read", "inventory:transfer", "inventory:opname", "inventory:approve-opname",
				"pos:checkout", "pos:shift", "pos:void",
				"promotion:manage",
				"payroll:read", "payroll:finalize",
				"user:manage",
			},
		},
		{
			Name:        "STORE_MANAGER",
			DisplayName: "Store Manager / Supervisor",
			Description: "Pengawasan operasional toko, approval opname, dan void kasir",
			PermNames: []string{
				"product:read", "inventory:read", "inventory:transfer", "inventory:opname", "inventory:approve-opname",
				"pos:checkout", "pos:shift", "pos:void",
				"promotion:manage",
				"payroll:read",
			},
		},
		{
			Name:        "CASHIER",
			DisplayName: "Kasir (Frontliner)",
			Description: "Operasional kasir, buka/tutup shift, dan aplikasi promosi",
			PermNames: []string{
				"product:read", "pos:checkout", "pos:shift",
			},
		},
		{
			Name:        "WAREHOUSE",
			DisplayName: "Staf Gudang",
			Description: "Penerimaan barang PO, mutasi rak, dan scan fisik stock opname",
			PermNames: []string{
				"product:read", "inventory:read", "inventory:transfer", "inventory:opname",
			},
		},
		{
			Name:        "HR_FINANCE",
			DisplayName: "HR & Keuangan",
			Description: "Penggajian, absensi, kalkulasi komisi, dan laporan keuangan",
			PermNames: []string{
				"product:read", "inventory:read", "payroll:read", "payroll:finalize",
			},
		},
	}

	for _, r := range systemRoles {
		var role model.Role
		if err := db.Where("name = ? AND tenant_id IS NULL", r.Name).First(&role).Error; err != nil {
			newRole := model.Role{
				Name:        r.Name,
				DisplayName: r.DisplayName,
				Description: r.Description,
				IsSystem:    true,
			}
			if err := db.Create(&newRole).Error; err != nil {
				return err
			}

			// Assign permissions
			var perms []model.Permission
			db.Where("name IN ?", r.PermNames).Find(&perms)
			db.Model(&newRole).Association("Permissions").Replace(perms)
		}
	}

	return nil
}
