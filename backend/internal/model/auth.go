package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents a user group with defined authorization levels.
type Role struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	TenantID    *uint64   `gorm:"index;uniqueIndex:idx_roles_tenant_name,priority:1" json:"tenant_id,omitempty"` // NULL for system default roles
	Name        string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_roles_tenant_name,priority:2" json:"name"`
	DisplayName string    `gorm:"type:varchar(100);not null" json:"display_name"`
	Description string    `gorm:"type:text" json:"description"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Tenant      *Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	Users       []User       `gorm:"foreignKey:RoleID" json:"users,omitempty"`
}

func (Role) TableName() string {
	return "roles"
}

// Permission defines a granular action in the system (e.g., 'pos:checkout', 'product:create').
type Permission struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Module      string    `gorm:"type:varchar(50);not null;index" json:"module"` // 'pos', 'inventory', 'product', 'payroll', 'auth', 'promotion'
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Relations
	Roles []Role `gorm:"many2many:role_permissions;" json:"roles,omitempty"`
}

func (Permission) TableName() string {
	return "permissions"
}

// RolePermission is the join table between Role and Permission.
type RolePermission struct {
	RoleID       uint64 `gorm:"primaryKey" json:"role_id"`
	PermissionID uint64 `gorm:"primaryKey" json:"permission_id"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

// User represents an authenticated account in the system.
type User struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_users_tenant_uuid,priority:2" json:"uuid"`
	TenantID     uint64         `gorm:"not null;uniqueIndex:idx_users_tenant_uuid,priority:1;uniqueIndex:idx_users_tenant_username,priority:1" json:"tenant_id"`
	RoleID       uint64         `gorm:"not null;index" json:"role_id"`
	Username     string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_users_tenant_username,priority:2" json:"username"`
	Email        string         `gorm:"type:varchar(255)" json:"email"`
	Phone        string         `gorm:"type:varchar(30)" json:"phone"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	FullName     string         `gorm:"type:varchar(255);not null" json:"full_name"`
	PinHash      string         `gorm:"type:varchar(255)" json:"-"` // Fast unlock for POS shift / kiosk attendance
	IsActive     bool           `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant  *Tenant  `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Role    *Role    `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Outlets []Outlet `gorm:"many2many:user_outlets;" json:"outlets,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// UserOutlet links a User to allowed physical branches/outlets.
type UserOutlet struct {
	UserID    uint64 `gorm:"primaryKey" json:"user_id"`
	OutletID  uint64 `gorm:"primaryKey" json:"outlet_id"`
	IsDefault bool   `gorm:"not null;default:false" json:"is_default"`
}

func (UserOutlet) TableName() string {
	return "user_outlets"
}
