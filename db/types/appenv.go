package types

type AppEnv struct {
	Base
	AppID  uint32 `json:"appID" gorm:"column:app_id"`
	Key    string `json:"key" gorm:"column:key;type:varchar(255);not null"`
	Value  string `json:"value" gorm:"column:value;type:longtext"`
	Desc   string `json:"desc" gorm:"column:desc;type:varchar(255)"`
	Masked bool   `json:"masked" gorm:"column:masked;type:tinyint(1);not null"`
}
