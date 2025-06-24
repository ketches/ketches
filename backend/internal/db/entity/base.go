package entity

import (
	"time"

	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

// IDBase is the base entity for all database entities with uint64 ID as primary key.
type IDBase struct {
	ID uint64 `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"` // ID is the primary key of the entity, auto-incremented by GORM.
}

// UUIDBase is the base entity for all database entities with UUID as primary key.
type UUIDBase struct {
	ID string `gorm:"primary_key;column:id;size:36" json:"id"` // ID is the primary key of the entity, represented as a UUID string.
}

// BeforeCreate is a gorm hook to generate a new UUID before creating a new UUIDBase entity.
func (b *UUIDBase) BeforeCreate(tx *gorm.DB) error {
	b.ID = uuid.New()
	return nil
}

// AuditBase is the base entity for all database entities with audit fields.
type AuditBase struct {
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"` // Time when the record was created
	CreatedBy string    `gorm:"column:created_by" json:"created_by"` // User who created the record
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"` // Time when the record was last updated
	UpdatedBy string    `gorm:"column:updated_by" json:"updated_by"` // User who last updated the record
}
