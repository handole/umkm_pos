package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel provides common fields for entity models.
// Uses uint64 for high-performance internal primary keys,
// and UUID for public API resource identification.
type BaseModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID      uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();not null;uniqueIndex" json:"uuid"`
	CreatedAt time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TenantScopedModel embeds BaseModel and adds TenantID for multi-tenant data isolation.
type TenantScopedModel struct {
	BaseModel
	TenantID uint64 `gorm:"not null;index" json:"tenant_id"`
}
