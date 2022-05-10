package types

type Tenant struct {
	Base
	Name string `json:"name" gorm:"column:name;type:varchar(255);not null;unique"`
	Desc string `json:"desc" gorm:"column:desc;type:longtext"`
}
