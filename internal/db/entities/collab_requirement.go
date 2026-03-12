package entities

// RequirementStatus represents the workflow status of a requirement.
type RequirementStatus = string

const (
	RequirementStatusTriage     RequirementStatus = "triage"
	RequirementStatusConfirmed  RequirementStatus = "confirmed"
	RequirementStatusInProgress RequirementStatus = "in_progress"
	RequirementStatusDone       RequirementStatus = "done"
	RequirementStatusClosed     RequirementStatus = "closed"
)

// PlanningStatus represents the backlog planning status.
type PlanningStatus = string

const (
	PlanningStatusBacklog  PlanningStatus = "backlog"
	PlanningStatusPlanned  PlanningStatus = "planned"
	PlanningStatusInSprint PlanningStatus = "in_sprint"
	PlanningStatusDone     PlanningStatus = "done"
)

// CollabPriority represents priority levels for requirements, tasks, etc.
type CollabPriority = string

const (
	CollabPriorityP0 CollabPriority = "p0"
	CollabPriorityP1 CollabPriority = "p1"
	CollabPriorityP2 CollabPriority = "p2"
	CollabPriorityP3 CollabPriority = "p3"
)

// MaxCollabDepth is the maximum allowed depth for parent-child hierarchies.
const MaxCollabDepth = 1

// CollabRequirement represents a business requirement within a project.
type CollabRequirement struct {
	Base

	ProjectID           string `gorm:"type:varchar(36);not null;index:idx_collab_req_project_status;index:idx_collab_req_project_sprint;index:idx_collab_req_project_rank"`
	SprintID            string `gorm:"type:varchar(36);index:idx_collab_req_project_sprint"`
	Title               string `gorm:"type:varchar(200);not null"`
	Description         string `gorm:"type:text"`
	Status              string `gorm:"type:varchar(24);not null;index:idx_collab_req_project_status"`
	Priority            string `gorm:"type:varchar(8);not null;index"`
	AssigneeID          string `gorm:"type:varchar(36);index"`
	PlanningStatus      string `gorm:"type:varchar(16);not null;index"`
	BacklogRank         int64  `gorm:"type:bigint;not null;index:idx_collab_req_project_rank"`
	ParentRequirementID string `gorm:"type:varchar(36);index:idx_collab_req_parent"`
	Depth               int    `gorm:"type:int;not null;index:idx_collab_req_parent"`
	CreatedBy           string `gorm:"type:varchar(36);not null;index"`
	UpdatedBy           string `gorm:"type:varchar(36);index"`
}

func (CollabRequirement) TableName() string {
	return "collab_requirements"
}
