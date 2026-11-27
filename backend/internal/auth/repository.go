package auth

import (
	"context"
	"errors"

	"github.com/handokodenih/umkm_pos/backend/internal/model"
	"gorm.io/gorm"
)

// Repository interface defines database operations for the Auth domain.
type Repository interface {
	FindUserByUsernameOrEmail(ctx context.Context, tenantID uint64, identifier string) (*model.User, error)
	FindUserByID(ctx context.Context, userID uint64) (*model.User, error)
	FindRoleByName(ctx context.Context, tenantID *uint64, roleName string) (*model.Role, error)
	FindTenantByCode(ctx context.Context, code string) (*model.Tenant, error)
	FindTenantByID(ctx context.Context, id uint64) (*model.Tenant, error)
	CreateTenant(ctx context.Context, tenant *model.Tenant) error
	CreateUser(ctx context.Context, user *model.User, outletIDs []uint64) error
	GetRolePermissions(ctx context.Context, roleID uint64) ([]string, error)
	GetUserOutlets(ctx context.Context, userID uint64) ([]model.Outlet, error)
	EnsureDefaultTenantAndOutlet(ctx context.Context) (*model.Tenant, *model.Outlet, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new Auth repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindUserByUsernameOrEmail(ctx context.Context, tenantID uint64, identifier string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Tenant").
		Where("tenant_id = ? AND (username = ? OR email = ?)", tenantID, identifier, identifier).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) FindUserByID(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Tenant").
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) FindRoleByName(ctx context.Context, tenantID *uint64, roleName string) (*model.Role, error) {
	var role model.Role
	query := r.db.WithContext(ctx).Where("name = ?", roleName)
	if tenantID != nil && *tenantID > 0 {
		query = query.Where("tenant_id = ? OR tenant_id IS NULL", *tenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}

	err := query.Order("tenant_id DESC NULLS LAST").First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *repository) FindTenantByCode(ctx context.Context, code string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *repository) FindTenantByID(ctx context.Context, id uint64) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *repository) CreateTenant(ctx context.Context, tenant *model.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *repository) CreateUser(ctx context.Context, user *model.User, outletIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		if len(outletIDs) > 0 {
			var userOutlets []model.UserOutlet
			for i, oID := range outletIDs {
				userOutlets = append(userOutlets, model.UserOutlet{
					UserID:    user.ID,
					OutletID:  oID,
					IsDefault: i == 0,
				})
			}
			if err := tx.Create(&userOutlets).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) GetRolePermissions(ctx context.Context, roleID uint64) ([]string, error) {
	var permissions []string
	err := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Pluck("permissions.name", &permissions).Error
	return permissions, err
}

func (r *repository) GetUserOutlets(ctx context.Context, userID uint64) ([]model.Outlet, error) {
	var outlets []model.Outlet
	err := r.db.WithContext(ctx).
		Table("outlets").
		Joins("JOIN user_outlets ON user_outlets.outlet_id = outlets.id").
		Where("user_outlets.user_id = ? AND outlets.deleted_at IS NULL", userID).
		Find(&outlets).Error
	return outlets, err
}

func (r *repository) EnsureDefaultTenantAndOutlet(ctx context.Context) (*model.Tenant, *model.Outlet, error) {
	var tenant model.Tenant
	err := r.db.WithContext(ctx).Where("id = 1 OR code = 'default'").First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tenant = model.Tenant{
				Code:             "default",
				Name:             "UMKM Retail Store (Default)",
				Status:           "ACTIVE",
				SubscriptionPlan: "ENTERPRISE",
			}
			if err := r.db.WithContext(ctx).Create(&tenant).Error; err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	var outlet model.Outlet
	err = r.db.WithContext(ctx).Where("tenant_id = ? AND is_main_branch = true", tenant.ID).First(&outlet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			outlet = model.Outlet{
				TenantID:     tenant.ID,
				Code:         "MAIN-01",
				Name:         "Toko Pusat (Utama)",
				Address:      "Jl. Operasional No. 1",
				IsMainBranch: true,
				IsActive:     true,
			}
			if err := r.db.WithContext(ctx).Create(&outlet).Error; err != nil {
				return nil, nil, err
			}

			// Also create default warehouse and store front locations
			whLocation := model.InventoryLocation{
				TenantID: tenant.ID,
				OutletID: outlet.ID,
				Code:     "GUDANG-PUSAT",
				Name:     "Gudang Utama",
				Type:     "WAREHOUSE",
				IsActive: true,
			}
			r.db.WithContext(ctx).Create(&whLocation)

			storeLocation := model.InventoryLocation{
				TenantID: tenant.ID,
				OutletID: outlet.ID,
				Code:     "DISPLAY-KASIR",
				Name:     "Display Depan Kasir",
				Type:     "STORE_FRONT",
				IsActive: true,
			}
			r.db.WithContext(ctx).Create(&storeLocation)
		} else {
			return nil, nil, err
		}
	}

	return &tenant, &outlet, nil
}
