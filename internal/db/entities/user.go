package entities

import "time"

type User struct {
	Base
	Username           string     `gorm:"type:varchar(64);uniqueIndex;not null"`
	Email              string     `gorm:"type:varchar(128);uniqueIndex;not null"`
	Password           string     `gorm:"type:varchar(128);not null"`
	Fullname           string     `gorm:"type:varchar(64)"`
	Bio                string     `gorm:"type:text"`
	Phone              string     `gorm:"type:varchar(32)"`
	Gender             int        `gorm:"type:int;default:0"`
	Role               string     `gorm:"type:varchar(16);default:'user'"`
	RefreshToken       string     `gorm:"type:text"`
	MustChangePassword bool       `gorm:"type:boolean;not null;default:false"`
	IsLocked           bool       `gorm:"type:boolean;default:false"`
	LockedAt           *time.Time `gorm:"index"`
	LockedReason       string     `gorm:"type:varchar(255)"`
}
