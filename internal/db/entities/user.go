package entities

type User struct {
	Base
	Username     string `gorm:"type:varchar(64);uniqueIndex;not null"`
	Email        string `gorm:"type:varchar(128);uniqueIndex;not null"`
	Password     string `gorm:"type:varchar(128);not null"`
	Fullname     string `gorm:"type:varchar(64)"`
	Phone        string `gorm:"type:varchar(32)"`
	Gender       int    `gorm:"type:int;default:0"`
	Role         string `gorm:"type:varchar(16);default:'user'"`
	RefreshToken string `gorm:"type:text"`
}
