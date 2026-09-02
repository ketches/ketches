package entities

import "time"

type SignupVerificationCode struct {
	Base
	Email             string    `gorm:"type:varchar(128);uniqueIndex:idx_signup_verification_codes_email_unique;not null"`
	CodeHash          string    `gorm:"type:varchar(128);not null"`
	ExpiresAt         time.Time `gorm:"index;not null"`
	ResendAvailableAt time.Time `gorm:"not null"`
	AttemptCount      int       `gorm:"type:int;not null;default:0"`
}
