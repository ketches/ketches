package models

// Collaboration module shared constants and filter models.

// Sprint status constants.
const (
	SprintStatusPlanned = "planned"
	SprintStatusActive  = "active"
	SprintStatusClosed  = "closed"
)

// Requirement status constants.
const (
	RequirementStatusTriage     = "triage"
	RequirementStatusConfirmed  = "confirmed"
	RequirementStatusInProgress = "in_progress"
	RequirementStatusDone       = "done"
	RequirementStatusClosed     = "closed"
)

// Planning status constants (backlog workflow).
const (
	PlanningStatusBacklog  = "backlog"
	PlanningStatusPlanned  = "planned"
	PlanningStatusInSprint = "in_sprint"
	PlanningStatusDone     = "done"
)

// Task status constants.
const (
	TaskStatusTodo       = "todo"
	TaskStatusInProgress = "in_progress"
	TaskStatusReview     = "review"
	TaskStatusDone       = "done"
	TaskStatusCancelled  = "cancelled"
)

// Test run status constants.
const (
	TestRunStatusPassed  = "passed"
	TestRunStatusFailed  = "failed"
	TestRunStatusBlocked = "blocked"
)

// Defect severity constants.
const (
	DefectSeverityCritical = "critical"
	DefectSeverityHigh     = "high"
	DefectSeverityMedium   = "medium"
	DefectSeverityLow      = "low"
)

// Defect status constants.
const (
	DefectStatusNew           = "new"
	DefectStatusProcessing    = "processing"
	DefectStatusPendingVerify = "pending_verify"
	DefectStatusClosed        = "closed"
	DefectStatusRejected      = "rejected"
)

// Priority constants shared across requirements and tasks.
const (
	CollabPriorityP0 = "p0"
	CollabPriorityP1 = "p1"
	CollabPriorityP2 = "p2"
	CollabPriorityP3 = "p3"
)

// MaxCollabDepth is the maximum allowed depth for parent-child hierarchies.
const MaxCollabDepth = 1

// CollabFilterParams holds common filter parameters for collaboration list queries.
type CollabFilterParams struct {
	PaginationRequest
	Status     string `form:"status" json:"status"`
	AssigneeID string `form:"assignee_id" json:"assignee_id"`
	SprintID   string `form:"sprint_id" json:"sprint_id"`
	Priority   string `form:"priority" json:"priority"`
}
