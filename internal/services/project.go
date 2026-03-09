package services

import (
	"errors"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

// ListProjects returns a paginated list of projects using an explicit JOIN to
// fetch the owner name instead of GORM Preload. Results are scanned into the
// flat ProjectListRow DTO.
func ListProjects(userID string, role string, req *models.PaginationRequest) (int64, []models.ProjectListRow, error) {
	var rows []models.ProjectListRow
	var total int64

	// Build count query depending on role
	countQ := db.DB.Model(&entities.Project{})
	if role != app.UserRoleAdmin {
		countQ = countQ.
			Joins("JOIN project_members ON project_members.project_id = projects.id").
			Where("project_members.user_id = ?", userID)
	}
	if req.Search != "" {
		search := "%" + req.Search + "%"
		countQ = countQ.Where("projects.name LIKE ? OR projects.slug LIKE ? OR projects.description LIKE ?", search, search, search)
	}
	if err := countQ.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// Build data query with explicit JOIN for owner name
	dataQ := db.DB.Table("projects").
		Select(`projects.id, projects.slug, projects.name, projects.description, projects.created_at,
			COALESCE(u.fullname, u.username, '') AS owner_name`).
		Joins("LEFT JOIN project_members pm ON pm.project_id = projects.id AND pm.project_role = 'owner'").
		Joins("LEFT JOIN users u ON u.id = pm.user_id").
		Where("projects.deleted_at IS NULL")
	if role != app.UserRoleAdmin {
		// Non-admin: must be a member of the project (use a subquery to avoid conflict with the owner join)
		dataQ = dataQ.Where("projects.id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID)
	}
	if req.Search != "" {
		search := "%" + req.Search + "%"
		dataQ = dataQ.Where("projects.name LIKE ? OR projects.slug LIKE ? OR projects.description LIKE ?", search, search, search)
	}
	if err := dataQ.Offset(req.GetOffset()).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}

	return total, rows, nil
}

func ListProjectsSimple(userID string, role string) ([]models.ProjectResponse, error) {
	var projects []models.ProjectResponse
	query := db.DB.Model(&entities.Project{}).Select("projects.id, projects.name, projects.slug, projects.description")
	if role == app.UserRoleAdmin {
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
			ProjectRole: app.ProjectRoleOwner,
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

func UpdateProject(projectID string, req *models.UpdateBasicInfoRequest) (*entities.Project, error) {
	project, err := GetProject(projectID)
	if err != nil {
		return nil, err
	}

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

type ProjectMemberWithUser struct {
	entities.ProjectMember
	Username string `gorm:"column:username"`
	Fullname string `gorm:"column:fullname"`
	Email    string `gorm:"column:email"`
}

func listProjectMembersWithUsers(projectID string, page, pageSize int, search string) (int64, []ProjectMemberWithUser, error) {
	var members []ProjectMemberWithUser
	var total int64
	query := db.DB.Table("project_members").
		Joins("JOIN users ON users.id = project_members.user_id").
		Where("project_members.project_id = ?", projectID)

	if search != "" {
		query = query.Where("users.username LIKE ? OR users.email LIKE ? OR users.fullname LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Select("project_members.*, users.username AS username, users.fullname AS fullname, users.email AS email").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&members).Error; err != nil {
		return 0, nil, err
	}
	return total, members, nil
}

func ListProjectMembers(projectID string, page, pageSize int, search string) (int64, []ProjectMemberWithUser, error) {
	return listProjectMembersWithUsers(projectID, page, pageSize, search)
}

func UpdateProjectMemberRole(projectID, userID, role string) error {
	var member entities.ProjectMember
	if err := db.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error; err != nil {
		return err
	}

	if member.ProjectRole == app.ProjectRoleOwner && role != app.ProjectRoleOwner {
		var ownerCount int64
		db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND project_role = ?", projectID, app.ProjectRoleOwner).Count(&ownerCount)
		if ownerCount <= 1 {
			return errors.New("at least one owner is required")
		}
	}

	return db.DB.Model(&member).Update("project_role", role).Error
}

func InviteProjectMembers(projectID string, userIDs []string, role string) error {
	for _, userID := range userIDs {
		var count int64
		db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count)
		if count > 0 {
			if err := UpdateProjectMemberRole(projectID, userID, role); err != nil {
				return err
			}
			continue
		}

		member := &entities.ProjectMember{
			ID:          uuid.New(),
			ProjectID:   projectID,
			UserID:      userID,
			ProjectRole: role,
		}
		if err := db.DB.Create(member).Error; err != nil {
			return err
		}
	}
	return nil
}

func RemoveProjectMember(projectID, userID string) error {
	var member entities.ProjectMember
	if err := db.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error; err != nil {
		return err
	}

	if member.ProjectRole == app.ProjectRoleOwner {
		var ownerCount int64
		db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND project_role = ?", projectID, app.ProjectRoleOwner).Count(&ownerCount)
		if ownerCount <= 1 {
			return errors.New("at least one owner is required")
		}
	}

	return db.DB.Delete(&member).Error
}
