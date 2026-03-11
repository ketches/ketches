package entities

type OperationLogSensitivity = string

const (
	OperationLogSensitivityPublic    OperationLogSensitivity = "public"
	OperationLogSensitivityInternal  OperationLogSensitivity = "internal"
	OperationLogSensitivitySensitive OperationLogSensitivity = "sensitive"
)

type OperationLogStatus = string

const (
	OperationLogStatusSuccess OperationLogStatus = "success"
	OperationLogStatusFailure OperationLogStatus = "failure"
)

type OperationLog struct {
	Base

	UserID       *string `gorm:"type:varchar(36);index"`
	Username     string  `gorm:"type:varchar(64);index"`
	Action       string  `gorm:"type:varchar(64);not null;index"`
	ResourceType string  `gorm:"type:varchar(64);not null;index"`
	ResourceID   string  `gorm:"type:varchar(64);index"`

	ProjectID *string `gorm:"type:varchar(36);index"`
	EnvID     *string `gorm:"type:varchar(36);index"`
	AppID     *string `gorm:"type:varchar(36);index"`
	RepoID    *string `gorm:"type:varchar(36);index"`

	Status     string `gorm:"type:varchar(16);not null;index"`
	StatusCode int    `gorm:"type:int;not null"`

	Sensitivity    string `gorm:"type:varchar(16);not null;default:'public';index"`
	RequestSummary string `gorm:"type:text"`
	ClientIP       string `gorm:"type:varchar(64)"`
}
