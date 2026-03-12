package entities

// CollabTestCase represents a test case definition within a project.
type CollabTestCase struct {
	Base

	ProjectID      string `gorm:"type:varchar(36);not null;index:idx_collab_tc_project_req;index:idx_collab_tc_project_task"`
	RequirementID  string `gorm:"type:varchar(36);index:idx_collab_tc_project_req"`
	TaskID         string `gorm:"type:varchar(36);index:idx_collab_tc_project_task"`
	Title          string `gorm:"type:varchar(200);not null"`
	Precondition   string `gorm:"type:text"`
	Steps          string `gorm:"type:text;not null"`
	ExpectedResult string `gorm:"type:text;not null"`
	CreatedBy      string `gorm:"type:varchar(36);not null;index"`
	UpdatedBy      string `gorm:"type:varchar(36);index"`
}

func (CollabTestCase) TableName() string {
	return "collab_test_cases"
}
