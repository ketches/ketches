package entities

import "time"

// TestRunStatus represents the execution result of a test run.
type TestRunStatus = string

const (
	TestRunStatusPassed  TestRunStatus = "passed"
	TestRunStatusFailed  TestRunStatus = "failed"
	TestRunStatusBlocked TestRunStatus = "blocked"
)

// CollabTestRun represents a single execution record of a test case.
type CollabTestRun struct {
	Base

	ProjectID  string    `gorm:"type:varchar(36);not null;index:idx_collab_tr_project_case;index:idx_collab_tr_project_status"`
	TestCaseID string    `gorm:"type:varchar(36);not null;index:idx_collab_tr_project_case"`
	Status     string    `gorm:"type:varchar(16);not null;index:idx_collab_tr_project_status"`
	ExecutedBy string    `gorm:"type:varchar(36);not null;index"`
	ExecutedAt time.Time `gorm:"not null;index"`
	Comment    string    `gorm:"type:text"`
}

func (CollabTestRun) TableName() string {
	return "collab_test_runs"
}
