package entities

// DefectSeverity represents the severity level of a defect.
type DefectSeverity = string

const (
	DefectSeverityCritical DefectSeverity = "critical"
	DefectSeverityHigh     DefectSeverity = "high"
	DefectSeverityMedium   DefectSeverity = "medium"
	DefectSeverityLow      DefectSeverity = "low"
)

// DefectStatus represents the workflow status of a defect.
type DefectStatus = string

const (
	DefectStatusNew           DefectStatus = "new"
	DefectStatusProcessing    DefectStatus = "processing"
	DefectStatusPendingVerify DefectStatus = "pending_verify"
	DefectStatusClosed        DefectStatus = "closed"
	DefectStatusRejected      DefectStatus = "rejected"
)

// CollabDefect represents a defect (bug) tracked within a project.
type CollabDefect struct {
	Base

	ProjectID          string `gorm:"type:varchar(36);not null;index:idx_collab_def_project_status;index:idx_collab_def_project_severity;index:idx_collab_def_project_assignee"`
	RequirementID      string `gorm:"type:varchar(36);index"`
	TaskID             string `gorm:"type:varchar(36);index"`
	TestCaseID         string `gorm:"type:varchar(36);index"`
	TestRunID          string `gorm:"type:varchar(36);index"`
	Title              string `gorm:"type:varchar(200);not null"`
	Description        string `gorm:"type:text;not null"`
	Severity           string `gorm:"type:varchar(16);not null;index:idx_collab_def_project_severity"`
	Status             string `gorm:"type:varchar(24);not null;index:idx_collab_def_project_status"`
	AssigneeID         string `gorm:"type:varchar(36);index:idx_collab_def_project_assignee"`
	ReproductionSteps  string `gorm:"type:text"`
	FixNote            string `gorm:"type:text"`
	RuntimeContextJSON string `gorm:"type:text"`
	CreatedBy          string `gorm:"type:varchar(36);not null;index"`
	UpdatedBy          string `gorm:"type:varchar(36);index"`
}

func (CollabDefect) TableName() string {
	return "collab_defects"
}
