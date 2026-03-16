package services

import (
	"testing"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCollabTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.CollabSprint{},
		&entities.CollabRequirement{},
		&entities.CollabTask{},
		&entities.CollabTestCase{},
		&entities.CollabTestRun{},
		&entities.CollabDefect{},
	))
	db.DB = testDB
}

// --- Requirement depth constraint tests ---

func TestCreateRequirement_ValidDepth(t *testing.T) {
	setupCollabTestDB(t)

	// Create a root requirement (depth 0).
	root, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:    "Root Requirement",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, root.Depth)

	// Create a child requirement (depth 1) — should succeed.
	child, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:               "Child Requirement",
		Status:              models.RequirementStatusTriage,
		Priority:            models.CollabPriorityP2,
		ParentRequirementID: root.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, child.Depth)
	assert.Equal(t, root.ID, child.ParentRequirementID)
}

func TestCreateRequirement_DepthExceeded(t *testing.T) {
	setupCollabTestDB(t)

	// Create root (depth 0).
	root, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:    "Root",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	// Create child (depth 1).
	child, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:               "Child",
		Status:              models.RequirementStatusTriage,
		Priority:            models.CollabPriorityP1,
		ParentRequirementID: root.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, child.Depth)

	// Attempt to create grandchild (depth 2) — should fail.
	_, err = CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:               "Grandchild",
		Status:              models.RequirementStatusTriage,
		Priority:            models.CollabPriorityP1,
		ParentRequirementID: child.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum nesting depth")
}

func TestCreateRequirement_ParentMustBeInSameProject(t *testing.T) {
	setupCollabTestDB(t)

	// Create a requirement in project A.
	parentA, err := CreateRequirement("projA", "user1", &models.CreateRequirementRequest{
		Title:    "Parent in Project A",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	// Try to create a child in project B referencing parent in project A — should fail.
	_, err = CreateRequirement("projB", "user1", &models.CreateRequirementRequest{
		Title:               "Child in Project B",
		Status:              models.RequirementStatusTriage,
		Priority:            models.CollabPriorityP1,
		ParentRequirementID: parentA.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parent requirement not found")
}

// --- Task depth constraint tests ---

func TestCreateTask_ValidDepth(t *testing.T) {
	setupCollabTestDB(t)

	root, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:    "Root Task",
		Status:   models.TaskStatusTodo,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, root.Depth)

	child, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:        "Sub Task",
		Status:       models.TaskStatusTodo,
		Priority:     models.CollabPriorityP2,
		ParentTaskID: root.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, child.Depth)
}

func TestCreateTask_DepthExceeded(t *testing.T) {
	setupCollabTestDB(t)

	root, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:    "Root",
		Status:   models.TaskStatusTodo,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	child, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:        "Child",
		Status:       models.TaskStatusTodo,
		Priority:     models.CollabPriorityP1,
		ParentTaskID: root.ID,
	})
	require.NoError(t, err)

	_, err = CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:        "Grandchild",
		Status:       models.TaskStatusTodo,
		Priority:     models.CollabPriorityP1,
		ParentTaskID: child.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum nesting depth")
}

// --- Task transition tests ---

func TestTransitionTask_ValidTransitions(t *testing.T) {
	setupCollabTestDB(t)

	task, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:    "Transition Test",
		Status:   models.TaskStatusTodo,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusTodo, task.Status)

	// todo -> in_progress
	task, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusInProgress})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusInProgress, task.Status)

	// in_progress -> review
	task, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusReview})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusReview, task.Status)

	// review -> done
	task, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusDone})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusDone, task.Status)

	// done -> in_progress (reopen)
	task, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusInProgress})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusInProgress, task.Status)
}

func TestTransitionTask_InvalidTransition(t *testing.T) {
	setupCollabTestDB(t)

	task, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:    "Invalid Transition",
		Status:   models.TaskStatusTodo,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	// todo -> done (not allowed)
	_, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusDone})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// todo -> review (not allowed)
	_, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusReview})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestTransitionTask_CancelFromAnyActive(t *testing.T) {
	setupCollabTestDB(t)

	// todo -> cancelled
	t1, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title: "Cancel from todo", Status: models.TaskStatusTodo, Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)
	t1, err = TransitionTask("proj1", t1.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusCancelled})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusCancelled, t1.Status)

	// in_progress -> cancelled
	t2, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title: "Cancel from in_progress", Status: models.TaskStatusTodo, Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)
	t2, err = TransitionTask("proj1", t2.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusInProgress})
	require.NoError(t, err)
	t2, err = TransitionTask("proj1", t2.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusCancelled})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusCancelled, t2.Status)

	// review -> cancelled
	t3, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title: "Cancel from review", Status: models.TaskStatusTodo, Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)
	t3, err = TransitionTask("proj1", t3.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusInProgress})
	require.NoError(t, err)
	t3, err = TransitionTask("proj1", t3.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusReview})
	require.NoError(t, err)
	t3, err = TransitionTask("proj1", t3.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusCancelled})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusCancelled, t3.Status)
}

func TestTransitionTask_NoTransitionFromCancelled(t *testing.T) {
	setupCollabTestDB(t)

	task, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title: "Cancelled task", Status: models.TaskStatusTodo, Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	task, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusCancelled})
	require.NoError(t, err)

	// cancelled -> in_progress (not allowed)
	_, err = TransitionTask("proj1", task.ID, "user1", &models.TaskTransitionRequest{Status: models.TaskStatusInProgress})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no transitions allowed")
}

func TestTransitionRequirement_ValidTransitions(t *testing.T) {
	setupCollabTestDB(t)

	requirement, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:    "Requirement Transition",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusTriage, requirement.Status)

	requirement, err = TransitionRequirement("proj1", requirement.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusConfirmed})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusConfirmed, requirement.Status)

	requirement, err = TransitionRequirement("proj1", requirement.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusInProgress})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusInProgress, requirement.Status)

	requirement, err = TransitionRequirement("proj1", requirement.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusDone})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusDone, requirement.Status)

	requirement, err = TransitionRequirement("proj1", requirement.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusClosed})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusClosed, requirement.Status)

	requirement, err = TransitionRequirement("proj1", requirement.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusInProgress})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusInProgress, requirement.Status)
}

func TestTransitionRequirement_InvalidTransition(t *testing.T) {
	setupCollabTestDB(t)

	requirement, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:    "Invalid Requirement Transition",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	_, err = TransitionRequirement("proj1", requirement.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusDone})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestTransitionRequirement_NoTransitionFromUnknownStatus(t *testing.T) {
	setupCollabTestDB(t)

	requirement, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:    "Unknown Requirement Status",
		Status:   "unknown",
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	_, err = TransitionRequirement("proj1", requirement.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusConfirmed})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no transitions allowed")
}

func TestTransitionRequirement_AllowedRollbacks(t *testing.T) {
	setupCollabTestDB(t)

	// Create and advance to in_progress
	req, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:    "Rollback Test",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusConfirmed})
	require.NoError(t, err)

	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusInProgress})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusInProgress, req.Status)

	// Rollback: in_progress -> confirmed
	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusConfirmed})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusConfirmed, req.Status)

	// Forward again: confirmed -> in_progress -> done
	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusInProgress})
	require.NoError(t, err)

	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusDone})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusDone, req.Status)

	// Rollback: done -> in_progress
	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusInProgress})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusInProgress, req.Status)
}

func TestTransitionRequirement_DisallowDirectClose(t *testing.T) {
	setupCollabTestDB(t)

	// Test disallowed direct close from triage
	req, err := CreateRequirement("proj1", "user1", &models.CreateRequirementRequest{
		Title:    "No Direct Close from Triage",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	_, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusClosed})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// Test disallowed direct close from confirmed
	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusConfirmed})
	require.NoError(t, err)

	_, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusClosed})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// Test disallowed direct close from in_progress
	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusInProgress})
	require.NoError(t, err)

	_, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusClosed})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// Only done -> closed is allowed
	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusDone})
	require.NoError(t, err)

	req, err = TransitionRequirement("proj1", req.ID, "user1", &models.RequirementTransitionRequest{Status: models.RequirementStatusClosed})
	require.NoError(t, err)
	assert.Equal(t, models.RequirementStatusClosed, req.Status)
}


func TestCreateTask_ParentMustBeInSameProject(t *testing.T) {
	setupCollabTestDB(t)

	parentA, err := CreateTask("projA", "user1", &models.CreateTaskRequest{
		Title:    "Parent task in Project A",
		Status:   models.TaskStatusTodo,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	_, err = CreateTask("projB", "user1", &models.CreateTaskRequest{
		Title:        "Child task in Project B",
		Status:       models.TaskStatusTodo,
		Priority:     models.CollabPriorityP1,
		ParentTaskID: parentA.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parent task not found")
}

func TestCreateTestCase_RejectsCrossProjectRequirementLink(t *testing.T) {
	setupCollabTestDB(t)

	reqA, err := CreateRequirement("projA", "user1", &models.CreateRequirementRequest{
		Title:    "Requirement A",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	_, err = CreateTestCase("projB", "user1", &models.CreateTestCaseRequest{
		Title:          "Case with cross-project requirement",
		Steps:          "Step",
		ExpectedResult: "Result",
		RequirementID:  reqA.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cross-project")
}

func TestCreateTestRun_RejectsCrossProjectTestCaseLink(t *testing.T) {
	setupCollabTestDB(t)

	tcA, err := CreateTestCase("projA", "user1", &models.CreateTestCaseRequest{
		Title:          "Case A",
		Steps:          "Step",
		ExpectedResult: "Result",
	})
	require.NoError(t, err)

	_, err = CreateTestRun("projB", tcA.ID, "user1", &models.CreateTestRunRequest{
		Status: models.TestRunStatusPassed,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cross-project")
}

func TestCreateDefect_AutoCreatesTaskWhenNoUpstreamLinkProvided(t *testing.T) {
	setupCollabTestDB(t)

	defect, err := CreateDefect("proj1", "user1", &models.CreateDefectRequest{
		Title:       "Defect without links",
		Description: "desc",
		Severity:    models.DefectSeverityHigh,
		Status:      models.DefectStatusNew,
	})
	require.NoError(t, err)
	require.NotEmpty(t, defect.TaskID)
	assert.Equal(t, "proj1", defect.ProjectID)
	assert.Equal(t, models.DefectStatusNew, defect.Status)

	task, err := GetTask("proj1", defect.TaskID)
	require.NoError(t, err)
	assert.Equal(t, "[Defect] Defect without links", task.Title)
	assert.Equal(t, "desc", task.Description)
	assert.Equal(t, models.TaskStatusTodo, task.Status)
	assert.Equal(t, models.CollabPriorityP1, task.Priority)
	assert.Equal(t, "user1", task.CreatedBy)
}

func TestCreateDefect_RejectsCrossProjectLink(t *testing.T) {
	setupCollabTestDB(t)

	reqA, err := CreateRequirement("projA", "user1", &models.CreateRequirementRequest{
		Title:    "Requirement A",
		Status:   models.RequirementStatusTriage,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	_, err = CreateDefect("projB", "user1", &models.CreateDefectRequest{
		Title:         "Cross project defect",
		Description:   "desc",
		Severity:      models.DefectSeverityHigh,
		Status:        models.DefectStatusNew,
		RequirementID: reqA.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cross-project")
}

func TestTransitionDefect_ValidAndInvalidTransitions(t *testing.T) {
	setupCollabTestDB(t)

	task, err := CreateTask("proj1", "user1", &models.CreateTaskRequest{
		Title:    "Task for defect",
		Status:   models.TaskStatusTodo,
		Priority: models.CollabPriorityP1,
	})
	require.NoError(t, err)

	defect, err := CreateDefect("proj1", "user1", &models.CreateDefectRequest{
		Title:       "Defect transition",
		Description: "desc",
		Severity:    models.DefectSeverityHigh,
		Status:      models.DefectStatusNew,
		TaskID:      task.ID,
	})
	require.NoError(t, err)

	defect, err = TransitionDefect("proj1", defect.ID, "user1", &models.DefectTransitionRequest{Status: models.DefectStatusProcessing})
	require.NoError(t, err)
	assert.Equal(t, models.DefectStatusProcessing, defect.Status)

	defect, err = TransitionDefect("proj1", defect.ID, "user1", &models.DefectTransitionRequest{Status: models.DefectStatusPendingVerify})
	require.NoError(t, err)
	assert.Equal(t, models.DefectStatusPendingVerify, defect.Status)

	defect, err = TransitionDefect("proj1", defect.ID, "user1", &models.DefectTransitionRequest{Status: models.DefectStatusClosed})
	require.NoError(t, err)
	assert.Equal(t, models.DefectStatusClosed, defect.Status)

	defect, err = TransitionDefect("proj1", defect.ID, "user1", &models.DefectTransitionRequest{Status: models.DefectStatusProcessing})
	require.NoError(t, err)
	assert.Equal(t, models.DefectStatusProcessing, defect.Status)

	_, err = TransitionDefect("proj1", defect.ID, "user1", &models.DefectTransitionRequest{Status: models.DefectStatusClosed})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}
