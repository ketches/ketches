package types

import "time"

type Base struct {
	ID        uint32    `json:"id" gorm:"primary_key"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
	CreatedBy uint32    `json:"createdBy" gorm:"column:created_by"`
	UpdatedBy uint32    `json:"updatedBy" gorm:"column:updated_by"`
}
