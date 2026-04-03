package services

import (
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

var allowedRequirementTransitions = map[string][]string{
	// Forward path: triage -> confirmed -> in_progress -> done -> closed
	entities.RequirementStatusTriage:     {entities.RequirementStatusConfirmed},
	entities.RequirementStatusConfirmed:  {entities.RequirementStatusInProgress},
	entities.RequirementStatusInProgress: {entities.RequirementStatusDone, entities.RequirementStatusConfirmed},
	entities.RequirementStatusDone:       {entities.RequirementStatusClosed, entities.RequirementStatusInProgress},
	entities.RequirementStatusClosed:     {entities.RequirementStatusInProgress},
}

// ListRequirements returns paginated requirements for a project.
func ListRequirements(projectID string, params *models.CollabFilterParams) (int64, []entities.CollabRequirement, error) {
	params.Validate()

	q := db.DB.Where("project_id = ?", projectID)
	if params.Status != "" {
		q = q.Where("status = ?", params.Status)
	}
	if params.SprintID != "" {
		q = q.Where("sprint_id = ?", params.SprintID)
	}
	if params.AssigneeID != "" {
		q = q.Where("assignee_id = ?", params.AssigneeID)
	}
	if params.Priority != "" {
		q = q.Where("priority = ?", params.Priority)
	}
	if params.Search != "" {
		q = q.Where("title LIKE ?", "%"+params.Search+"%")
	}

	var total int64
	if err := q.Model(&entities.CollabRequirement{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var items []entities.CollabRequirement
	if err := q.Order("backlog_rank ASC, created_at ASC").Offset(params.GetOffset()).Limit(params.PageSize).Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

func ListBacklog(projectID string, params *models.CollabFilterParams) (int64, []entities.CollabRequirement, error) {
	params.Validate()

	q := db.DB.Where("project_id = ? AND planning_status = ?", projectID, entities.PlanningStatusBacklog)
	if params.Status != "" {
		q = q.Where("status = ?", params.Status)
	}
	if params.AssigneeID != "" {
		q = q.Where("assignee_id = ?", params.AssigneeID)
	}
	if params.Priority != "" {
		q = q.Where("priority = ?", params.Priority)
	}
	if params.Search != "" {
		q = q.Where("title LIKE ?", "%"+params.Search+"%")
	}

	var total int64
	if err := q.Model(&entities.CollabRequirement{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var items []entities.CollabRequirement
	if err := q.Order("backlog_rank ASC, created_at ASC").Offset(params.GetOffset()).Limit(params.PageSize).Find(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}

// GetRequirement returns a single requirement by ID scoped to the project.
func GetRequirement(projectID, reqID string) (*entities.CollabRequirement, error) {
	var req entities.CollabRequirement
	if err := db.DB.Where("id = ? AND project_id = ?", reqID, projectID).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

// CreateRequirement creates a new requirement within a project.
func CreateRequirement(projectID, userID string, req *models.CreateRequirementRequest) (*entities.CollabRequirement, error) {
	depth := 0
	if req.ParentRequirementID != "" {
		parent, err := GetRequirement(projectID, req.ParentRequirementID)
		if err != nil {
			return nil, app.WrapErrorf(err, "parent requirement not found: %w", err)
		}
		if parent.Depth >= entities.MaxCollabDepth {
			return nil, app.NewErrorf("maximum nesting depth (%d) exceeded", entities.MaxCollabDepth)
		}
		depth = parent.Depth + 1
	}

	// Determine planning status based on sprint assignment.
	planningStatus := entities.PlanningStatusBacklog
	if req.SprintID != "" {
		planningStatus = entities.PlanningStatusInSprint
	}

	entity := &entities.CollabRequirement{
		Base:                entities.Base{ID: uuid.New()},
		ProjectID:           projectID,
		SprintID:            req.SprintID,
		Title:               req.Title,
		Description:         req.Description,
		Status:              req.Status,
		Priority:            req.Priority,
		AssigneeID:          req.AssigneeID,
		PlanningStatus:      planningStatus,
		ParentRequirementID: req.ParentRequirementID,
		Depth:               depth,
		CreatedBy:           userID,
	}
	if err := db.DB.Create(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// UpdateRequirement updates an existing requirement.
func UpdateRequirement(projectID, reqID, userID string, req *models.UpdateRequirementRequest) (*entities.CollabRequirement, error) {
	entity, err := GetRequirement(projectID, reqID)
	if err != nil {
		return nil, err
	}

	entity.Title = req.Title
	entity.Description = req.Description
	entity.Status = req.Status
	entity.Priority = req.Priority
	entity.AssigneeID = req.AssigneeID
	entity.SprintID = req.SprintID
	entity.UpdatedBy = userID

	if req.PlanningStatus != "" {
		entity.PlanningStatus = req.PlanningStatus
	}

	if err := db.DB.Save(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// DeleteRequirement soft-deletes a requirement by ID scoped to the project.
func DeleteRequirement(projectID, reqID string) error {
	return db.DB.Where("id = ? AND project_id = ?", reqID, projectID).Delete(&entities.CollabRequirement{}).Error
}

func TransitionRequirement(projectID, reqID, userID string, req *models.RequirementTransitionRequest) (*entities.CollabRequirement, error) {
	entity, err := GetRequirement(projectID, reqID)
	if err != nil {
		return nil, err
	}

	targets, ok := allowedRequirementTransitions[entity.Status]
	if !ok {
		return nil, app.NewErrorf("no transitions allowed from status %q", entity.Status)
	}

	allowed := false
	for _, t := range targets {
		if t == req.Status {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, app.NewErrorf("transition from %q to %q is not allowed", entity.Status, req.Status)
	}

	entity.Status = req.Status
	entity.UpdatedBy = userID
	if err := db.DB.Save(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// BacklogReorder updates the backlog_rank of multiple requirements in one transaction.
func BacklogReorder(projectID string, req *models.BacklogReorderRequest) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if err := tx.Model(&entities.CollabRequirement{}).
				Where("id = ? AND project_id = ?", item.RequirementID, projectID).
				Update("backlog_rank", item.Rank).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BacklogPlanToSprint moves requirements from backlog into a sprint.
func BacklogPlanToSprint(projectID string, req *models.BacklogPlanToSprintRequest) error {
	return db.DB.Model(&entities.CollabRequirement{}).
		Where("id IN ? AND project_id = ?", req.RequirementIDs, projectID).
		Updates(map[string]any{
			"sprint_id":       req.SprintID,
			"planning_status": entities.PlanningStatusInSprint,
		}).Error
}

// BacklogReturn moves requirements from a sprint back to the backlog.
func BacklogReturn(projectID string, req *models.BacklogReturnRequest) error {
	return db.DB.Model(&entities.CollabRequirement{}).
		Where("id IN ? AND project_id = ?", req.RequirementIDs, projectID).
		Updates(map[string]any{
			"sprint_id":       "",
			"planning_status": entities.PlanningStatusBacklog,
		}).Error
}
