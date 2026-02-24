package services

import (
	"errors"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListProjects(userID string, role string) ([]entities.Project, error) {
	var projects []entities.Project
	if role == "admin" {
		if err := db.DB.Find(&projects).Error; err != nil {
			return nil, err
		}
	} else {
		err := db.DB.Joins("JOIN project_members ON project_members.project_id = projects.id").
			Where("project_members.user_id = ?", userID).
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

func RestoreProject(projectID string) error {
	return db.DB.Unscoped().Model(&entities.Project{}).Where("id = ?", projectID).Update("deleted_at", nil).Error
}

func ListProjectMembers(projectID string) ([]entities.ProjectMember, error) {
	var members []entities.ProjectMember
	if err := db.DB.Preload("User").Where("project_id = ?", projectID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
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
