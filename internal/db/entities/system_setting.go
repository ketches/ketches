package entities

type SystemSetting struct {
	Base
	Key   string `gorm:"type:varchar(128);uniqueIndex;not null"`
	Value string `gorm:"type:text"`
}
