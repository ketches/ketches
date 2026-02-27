package services

import (
	"errors"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListProjects(userID string, role string, req *models.PaginationRequest) (int64, []entities.Project, error) {
	var projects []entities.Project
	var total int64

	// Build base query depending on role
	var baseQuery *gorm.DB
	if role == "admin" {
		baseQuery = db.DB.Model(&entities.Project{})
	} else {
		baseQuery = db.DB.Model(&entities.Project{}).
			Joins("JOIN project_members ON project_members.project_id = projects.id").
			Where("project_members.user_id = ?", userID)
	}

	// Apply search filter
	if req.Search != "" {
		search := "%" + req.Search + "%"
		baseQuery = baseQuery.Where("projects.name LIKE ? OR projects.slug LIKE ? OR projects.description LIKE ?", search, search, search)
	}

	// Count total before pagination
	if err := baseQuery.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// Apply pagination and preload members with user info
	err := baseQuery.
		Select("projects.*").
		Offset(req.GetOffset()).
		Limit(req.PageSize).
		Preload("Members", "project_role = ?", "owner").Preload("Members.User").
		Find(&projects).Error
	if err != nil {
		return 0, nil, err
	}

	return total, projects, nil
}

func ListProjectsSimple(userID string, role string) ([]entities.Project, error) {
	var projects []entities.Project
	query := db.DB.Select("projects.id, projects.name, projects.slug, projects.description")
	if role == "admin" {
		if err := query.Order("name").Find(&projects).Error; err != nil {
			return nil, err
		}
	} else {
		err := query.Joins("JOIN project_members ON project_members.project_id = projects.id").
			Where("project_members.user_id = ?", userID).
			Order("name").
			Find(&projects).Error
		if err != nil {
			return nil, err
		}
	}
	return projects, nil
}

func CreateProject(req *models.CreateProjectRequest, userID string) (*entities.Project, error) {
	var existing entities.Project
	if err := db.DB.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		return nil, errors.New("project with this slug already exists")
	}

	project := &entities.Project{
		Base:        entities.Base{ID: uuid.New()},
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}

		member := &entities.ProjectMember{
			ID:          uuid.New(),
			ProjectID:   project.ID,
			UserID:      userID,
			ProjectRole: "owner",
		}

		return tx.Create(member).Error
	})

	if err != nil {
		return nil, err
	}

	return project, nil
}

func GetProject(projectID string) (*entities.Project, error) {
	var project entities.Project
	if err := db.DB.First(&project, "id = ?", projectID).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func UpdateProject(projectID string, req *models.CreateProjectRequest) (*entities.Project, error) {
	project, err := GetProject(projectID)
	if err != nil {
		return nil, err
	}

	if project.Slug != req.Slug {
		var existing entities.Project
		if err := db.DB.Where("slug = ? AND id != ?", req.Slug, projectID).First(&existing).Error; err == nil {
			return nil, errors.New("project with this slug already exists")
		}
	}

	project.Slug = req.Slug
	project.Name = req.Name
	project.Description = req.Description

	if err := db.DB.Save(project).Error; err != nil {
		return nil, err
	}
	return project, nil
}

func DeleteProject(projectID string) error {
	var envCount int64
	if err := db.DB.Model(&entities.Env{}).Where("project_id = ?", projectID).Count(&envCount).Error; err != nil {
		return err
	}

	if envCount > 0 {
		return errors.New("cannot delete project: it contains environments. Please delete all environments first or move them to recycle bin")
	}

	return db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error
}

func PermanentlyDeleteProject(projectID string) error {
	var project entities.Project
	if err := db.DB.Unscoped().First(&project, "id = ?", projectID).Error; err != nil {
		return err
	}

	return db.DB.Unscoped().Delete(&entities.Project{}, "id = ?", projectID).Error
}

func IsProjectMember(projectID, userID string) (bool, error) {
	var count int64
	if err := db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func RestoreProject(projectID string) error {
	return db.DB.Unscoped().Model(&entities.Project{}).Where("id = ?", projectID).Update("deleted_at", nil).Error
}

func ListProjectMembers(projectID string, page, pageSize int, search string) (int64, []entities.ProjectMember, error) {
	var members []entities.ProjectMember
	var total int64
	query := db.DB.Model(&entities.ProjectMember{}).Where("project_id = ?", projectID)

	if search != "" {
		query = query.Joins("User").Where("\"User\".username LIKE ? OR \"User\".email LIKE ? OR \"User\".fullname LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Preload("User").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&members).Error; err != nil {
		return 0, nil, err
	}
	return total, members, nil
}

func UpdateProjectMemberRole(projectID, userID, role string) error {
	var member entities.ProjectMember
	if err := db.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error; err != nil {
		return err
	}

	if member.ProjectRole == "owner" && role != "owner" {
		var ownerCount int64
		db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND project_role = ?", projectID, "owner").Count(&ownerCount)
		if ownerCount <= 1 {
			return errors.New("at least one owner is required")
		}
	}

	return db.DB.Model(&member).Update("project_role", role).Error
}

func AddProjectMember(projectID, userID, role string) error {
	var count int64
	db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count)
	if count > 0 {
		return UpdateProjectMemberRole(projectID, userID, role)
	}

	member := &entities.ProjectMember{
		ID:          uuid.New(),
		ProjectID:   projectID,
		UserID:      userID,
		ProjectRole: role,
	}
	return db.DB.Create(member).Error
}

func RemoveProjectMember(projectID, userID string) error {
	var member entities.ProjectMember
	if err := db.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error; err != nil {
		return err
	}

	if member.ProjectRole == "owner" {
		var ownerCount int64
		db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND project_role = ?", projectID, "owner").Count(&ownerCount)
		if ownerCount <= 1 {
			return errors.New("at least one owner is required")
		}
	}

	return db.DB.Delete(&member).Error
}
