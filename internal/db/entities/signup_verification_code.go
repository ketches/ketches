package entities

import "time"

type SignupVerificationCode struct {
	Base
	Email             string    `gorm:"type:varchar(128);index;not null"`
	CodeHash          string    `gorm:"type:varchar(128);not null"`
	ExpiresAt         time.Time `gorm:"index;not null"`
	ResendAvailableAt time.Time `gorm:"not null"`
}
