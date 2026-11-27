package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant represents the top-level business organization or legal entity.
type Tenant struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	Code             string         `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name             string         `gorm:"type:varchar(255);not null" json:"name"`
	Status           string         `gorm:"type:varchar(30);not null;default:'ACTIVE'" json:"status"` // ACTIVE, SUSPENDED, TRIAL
	SubscriptionPlan string         `gorm:"type:varchar(50);not null;default:'STANDARD'" json:"subscription_plan"`
	CreatedAt        time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Outlets []Outlet `gorm:"foreignKey:TenantID" json:"outlets,omitempty"`
	Users   []User   `gorm:"foreignKey:TenantID" json:"users,omitempty"`
}

func (Tenant) TableName() string {
	return "tenants"
}

// Outlet represents a physical store branch or sales outlet under a Tenant.
type Outlet struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_outlets_tenant_uuid,priority:2" json:"uuid"`
	TenantID     uint64         `gorm:"not null;uniqueIndex:idx_outlets_tenant_uuid,priority:1;uniqueIndex:idx_outlets_tenant_code,priority:1" json:"tenant_id"`
	Code         string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_outlets_tenant_code,priority:2" json:"code"`
	Name         string         `gorm:"type:varchar(255);not null" json:"name"`
	Address      string         `gorm:"type:text" json:"address"`
	Phone        string         `gorm:"type:varchar(30)" json:"phone"`
	IsMainBranch bool           `gorm:"not null;default:false" json:"is_main_branch"`
	IsActive     bool           `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant    *Tenant             `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Locations []InventoryLocation `gorm:"foreignKey:OutletID" json:"locations,omitempty"`
}

func (Outlet) TableName() string {
	return "outlets"
}

// InventoryLocation represents a physical storage point (Warehouse, Front Shelf, Transit).
type InventoryLocation struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID      uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex:idx_locations_tenant_uuid,priority:2" json:"uuid"`
	TenantID  uint64         `gorm:"not null;uniqueIndex:idx_locations_tenant_uuid,priority:1;uniqueIndex:idx_locations_tenant_code,priority:1" json:"tenant_id"`
	OutletID  uint64         `gorm:"not null;index" json:"outlet_id"`
	Code      string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_locations_tenant_code,priority:2" json:"code"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	Type      string         `gorm:"type:varchar(30);not null" json:"type"` // WAREHOUSE, STORE_FRONT, TRANSIT, DAMAGED
	IsActive  bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Outlet *Outlet `gorm:"foreignKey:OutletID" json:"outlet,omitempty"`
}

func (InventoryLocation) TableName() string {
	return "inventory_locations"
}
