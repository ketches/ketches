package entities

type Project struct {
	UUIDBase
	Slug        string `json:"slug" gorm:"not null;uniqueIndex;size:36"` // Project slug, e.g., 'my-project'
	DisplayName string `json:"displayName" gorm:"not null;size:255"`     // Human-readable name for the project
	Description string `json:"description" gorm:"size:255"`              // Optional description of the project
	AuditBase
}

type ProjectMember struct {
	UUIDBase
	ProjectID   string `json:"project_id" gorm:"not null;uniqueIndex:idx_projectID_userID;index;size:64"` // Project UUID
	UserID      string `json:"user_id" gorm:"not null;uniqueIndex:idx_projectID_userID;index;size:64"`    // User UUID
	ProjectRole string `json:"project_role" gorm:"not null;size:32"`                                      // e.g., 'admin', 'member', 'viewer'
	AuditBase
}

func (ProjectMember) TableName() string {
	return "project_members"
}
