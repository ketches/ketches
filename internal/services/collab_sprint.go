package services

import (
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// parseDate parses a date string in "2006-01-02" format.
func parseDate(s string) (*time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, app.NewErrorf("invalid date format: %s", s)
	}
	return &t, nil
}

// ListSprints returns paginated sprints for a project.
func ListSprints(projectID string, params *models.CollabFilterParams) (int64, []entities.CollabSprint, error) {
	params.Validate()

	q := db.DB.Where("project_id = ?", projectID)
	if params.Status != "" {
		statuses := strings.Split(params.Status, ",")
		q = q.Where("status IN ?", statuses)
	}
	if params.Search != "" {
		q = q.Where("name LIKE ?", "%"+params.Search+"%")
	}

	var total int64
	if err := q.Model(&entities.CollabSprint{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var items []entities.CollabSprint
	if err := q.Order("start_date ASC").Offset(params.GetOffset()).Limit(params.PageSize).Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

// GetSprint returns a single sprint by ID scoped to the project.
func GetSprint(projectID, sprintID string) (*entities.CollabSprint, error) {
	var sprint entities.CollabSprint
	if err := db.DB.Where("id = ? AND project_id = ?", sprintID, projectID).First(&sprint).Error; err != nil {
		return nil, err
	}
	return &sprint, nil
}

// CreateSprint creates a new sprint within a project.
func CreateSprint(projectID, userID string, req *models.CreateSprintRequest) (*entities.CollabSprint, error) {
	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return nil, err
	}

	sprint := &entities.CollabSprint{
		Base:      entities.Base{ID: uuid.New()},
		ProjectID: projectID,
		Name:      req.Name,
		Goal:      req.Goal,
		Status:    req.Status,
		StartDate: startDate,
		EndDate:   endDate,
		CreatedBy: userID,
	}
	if err := db.DB.Create(sprint).Error; err != nil {
		return nil, err
	}
	return sprint, nil
}

// UpdateSprint updates an existing sprint.
func UpdateSprint(projectID, sprintID, userID string, req *models.UpdateSprintRequest) (*entities.CollabSprint, error) {
	sprint, err := GetSprint(projectID, sprintID)
	if err != nil {
		return nil, err
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return nil, err
	}

	sprint.Name = req.Name
	sprint.Goal = req.Goal
	sprint.Status = req.Status
	sprint.StartDate = startDate
	sprint.EndDate = endDate
	sprint.UpdatedBy = userID

	if err := db.DB.Save(sprint).Error; err != nil {
		return nil, err
	}
	return sprint, nil
}

// DeleteSprint soft-deletes a sprint by ID scoped to the project.
func DeleteSprint(projectID, sprintID string) error {
	return db.DB.Where("id = ? AND project_id = ?", sprintID, projectID).Delete(&entities.CollabSprint{}).Error
}
