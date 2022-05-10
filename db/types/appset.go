package types

type AppSet struct {
	Base
	Name      string `json:"name" gorm:"column:name;type:varchar(255);not null;unique"`
	AliasName string `json:"aliasName" gorm:"column:name;type:varchar(255)"`
	Desc      string `json:"desc" gorm:"column:desc;type:longtext"`
}
