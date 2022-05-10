package types

type App struct {
	Base
	Name      string    `json:"name" gorm:"column:name;type:varchar(255);not null;unique"`
	AliasName string    `json:"aliasName" gorm:"column:name;type:varchar(255)"`
	Desc      string    `json:"desc" gorm:"column:desc;type:longtext"`
	BuildFrom BuildFrom `json:"buildFrom" gorm:"column:build_from;type:tinyint(1);not null"`
	Image     string    `json:"image" gorm:"column:image;type:varchar(255)"`
	AppSetID  uint32    `json:"appSetID" gorm:"column:app_set_id;not null"`
}

type BuildFrom uint8

const (
	_ BuildFrom = iota
	BuildFromImage
	BuildFromSource
)
