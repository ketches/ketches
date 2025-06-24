package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entity"
	"github.com/ketches/ketches/internal/models"
	"gorm.io/gorm"
)

type ProjectService interface {
	ListProjects(ctx context.Context, req *models.ListProjectsRequest) (*models.ListProjectResponse, app.Error)
	AllProjectRefs(ctx context.Context) ([]*models.ProjectRef, app.Error)
	GetProject(ctx context.Context, req *models.GetProjectRequest) (*models.ProjectModel, app.Error)
	GetProjectRef(ctx context.Context, req *models.GetProjectRefRequest) (*models.ProjectRef, app.Error)
	CreateProject(ctx context.Context, req *models.CreateProjectRequest) (*models.ProjectModel, app.Error)
	UpdateProject(ctx context.Context, req *models.UpdateProjectRequest) (*models.ProjectModel, app.Error)
	DeleteProject(ctx context.Context, req *models.DeleteProjectRequest) app.Error
	ListProjectMembers(ctx context.Context, req *models.ListProjectMembersRequest) (*models.ListProjectMembersResponse, app.Error)
	ListAddableProjectMembers(ctx context.Context, projectID string) ([]*models.UserRef, app.Error)
	AddProjectMembers(ctx context.Context, req *models.AddProjectMembersRequest) app.Error
	UpdateProjectMember(ctx context.Context, req *models.UpdateProjectMemberRequest) (*models.ProjectMemberModel, app.Error)
	RemoveProjectMembers(ctx context.Context, req *models.RemoveProjectMembersRequest) app.Error
}

type projectService struct {
	Service
}

func NewProjectService() ProjectService {
	return &projectService{
		Service: LoadService(),
	}
}

func (s *projectService) ListProjects(ctx context.Context, req *models.ListProjectsRequest) (*models.ListProjectResponse, app.Error) {
	projects := []*entity.Project{}
	query := db.Instance().Model(&entity.Project{})
	if !api.IsAdmin(ctx) {
		query = query.Joins("INNER JOIN project_members ON project_members.project_id = projects.id").Where("project_members.user_id = ?", api.UserID(ctx))
	}

	if req.Query != "" {
		query = db.CaseInsensitiveLike(query, req.Query, "projects.slug", "projects.display_name")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("failed to count projects for user %s: %v", api.UserID(ctx), err)
		return nil, app.ErrDatabaseOperationFailed
	}

	if err := req.PagedSQL(query).Find(&projects).Error; err != nil {
		log.Printf("failed to list projects for user %s: %v", api.UserID(ctx), err)
		return nil, app.ErrDatabaseOperationFailed
	}

	result := make([]*models.ProjectModel, 0, len(projects))
	for _, project := range projects {
		result = append(result, &models.ProjectModel{
			ProjectID:   project.ID,
			Slug:        project.Slug,
			DisplayName: project.DisplayName,
			Description: project.Description,
		})
	}

	return &models.ListProjectResponse{
		Total:   total,
		Records: result,
	}, nil
}

func (s *projectService) AllProjectRefs(ctx context.Context) ([]*models.ProjectRef, app.Error) {
	refs := []*models.ProjectRef{}
	if err := db.Instance().Model(&entity.Project{}).
		Select("projects.id, projects.slug, projects.display_name").
		Joins("INNER JOIN project_members ON project_members.project_id = projects.id").Where("project_members.user_id = ?", api.UserID(ctx)).
		Find(&refs).Error; err != nil {
		log.Printf("failed to list project refs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return refs, nil
}

func (s *projectService) GetProject(ctx context.Context, req *models.GetProjectRequest) (*models.ProjectModel, app.Error) {
	if _, err := s.CheckProjectPermissions(ctx, req.ProjectID); err != nil {
		return nil, err
	}

	project := new(entity.Project)
	if err := db.Instance().First(project, "id = ?", req.ProjectID).Error; err != nil {
		log.Printf("failed to get project %s: %v", req.ProjectID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Project not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.ProjectModel{
		ProjectID:   project.ID,
		Slug:        project.Slug,
		DisplayName: project.DisplayName,
		Description: project.Description,
	}, nil
}

func (s *projectService) GetProjectRef(ctx context.Context, req *models.GetProjectRefRequest) (*models.ProjectRef, app.Error) {
	if _, err := s.CheckProjectPermissions(ctx, req.ProjectID); err != nil {
		return nil, err
	}

	result := &models.ProjectRef{}
	if err := db.Instance().Model(&entity.Project{}).First(result, "id = ?", req.ProjectID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Project not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *projectService) CreateProject(ctx context.Context, req *models.CreateProjectRequest) (*models.ProjectModel, app.Error) {
	operator := req.Operator
	if operator == "" {
		operator = api.UserID(ctx)
	}
	project := &entity.Project{
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		Description: req.Description,
		AuditBase: entity.AuditBase{
			CreatedBy: operator,
			UpdatedBy: operator,
		},
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			log.Printf("failed to create project: %v", err)
			return err
		}

		pm := &entity.ProjectMember{
			ProjectID:   project.ID,
			UserID:      operator,
			ProjectRole: app.ProjectRoleOwner,
		}
		pm.CreatedBy = operator
		pm.UpdatedBy = operator
		if err := tx.Create(pm).Error; err != nil {
			log.Printf("failed to add project owner %s to project %s: %v", operator, project.ID, err)
			if db.IsErrDuplicatedKey(err) {
				return nil
			}
			return err
		}
		return nil
	}); err != nil {
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusConflict, "Project with this slug already exists")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.ProjectModel{
		ProjectID:   project.ID,
		Slug:        project.Slug,
		DisplayName: project.DisplayName,
		Description: project.Description,
	}, nil
}

func (s *projectService) UpdateProject(ctx context.Context, req *models.UpdateProjectRequest) (*models.ProjectModel, app.Error) {
	projectRole, err := s.CheckProjectPermissions(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	if projectRole != app.ProjectRoleOwner {
		return nil, app.NewError(http.StatusForbidden, "You do not have permission to update this project")
	}

	project := &entity.Project{}
	if err := db.Instance().First(project, "id = ?", req.ProjectID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Project not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}
	project.DisplayName = req.DisplayName
	project.Description = req.Description
	project.UpdatedBy = api.UserID(ctx)
	if err := db.Instance().Save(project).Error; err != nil {
		log.Printf("failed to update project %s: %v", req.ProjectID, err)
		return nil, app.ErrDatabaseOperationFailed
	}
	return &models.ProjectModel{
		ProjectID:   project.ID,
		Slug:        project.Slug,
		DisplayName: project.DisplayName,
		Description: project.Description,
	}, nil
}

func (s *projectService) DeleteProject(ctx context.Context, req *models.DeleteProjectRequest) app.Error {
	projectRole, err := s.CheckProjectPermissions(ctx, req.ProjectID)
	if err != nil {
		return err
	}
	if projectRole != app.ProjectRoleOwner {
		return app.NewError(http.StatusForbidden, "You do not have permission to delete this project")
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(entity.Project{}, "id = ?", req.ProjectID).Error; err != nil {
			return err
		}
		if err := tx.Delete(entity.ProjectMember{}, "project_id = ?", req.ProjectID).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Printf("failed to delete project %s: %v", req.ProjectID, err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s *projectService) ListProjectMembers(ctx context.Context, req *models.ListProjectMembersRequest) (*models.ListProjectMembersResponse, app.Error) {
	if _, err := s.CheckProjectPermissions(ctx, req.ProjectID); err != nil {
		return nil, err
	}

	query := db.Instance().Model(&entity.ProjectMember{}).
		Select("project_members.project_id,project_members.user_id,users.username,users.fullname,users.email,users.phone,project_members.project_role,project_members.created_at").
		Joins("LEFT JOIN users ON users.id = project_members.user_id AND project_members.project_id = ?", req.ProjectID).
		Where("project_members.project_id = ?", req.ProjectID)

	if req.Query != "" {
		query = db.CaseInsensitiveLike(query, req.Query, "users.username", "users.fullname", "users.email", "users.phone")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("failed to count project members for project %s: %v", req.ProjectID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	members := []*models.ProjectMemberModel{}
	if err := req.PagedSQL(query).Find(&members).Error; err != nil {
		log.Printf("failed to list project members for project %s: %v", req.ProjectID, err)
		if !db.IsErrRecordNotFound(err) {
			return nil, app.ErrDatabaseOperationFailed
		}
	}

	return &models.ListProjectMembersResponse{
		Total:   total,
		Records: members,
	}, nil
}

func (s *projectService) ListAddableProjectMembers(ctx context.Context, projectID string) ([]*models.UserRef, app.Error) {
	if _, err := s.CheckProjectPermissions(ctx, projectID); err != nil {
		return nil, err
	}

	members := []*models.UserRef{}
	query := db.Instance().Model(&entity.User{}).
		Select("users.id, users.username, users.fullname, users.email, users.phone").
		Joins("LEFT JOIN project_members ON project_members.user_id = users.id AND project_members.project_id = ?", projectID).
		Where("project_members.user_id IS NULL")
	if err := query.Find(&members).Error; err != nil {
		log.Printf("failed to list addable project members for project %s: %v", projectID, err)
		return nil, app.ErrDatabaseOperationFailed
	}
	return members, nil
}

func (s *projectService) AddProjectMembers(ctx context.Context, req *models.AddProjectMembersRequest) app.Error {
	projectRole, err := s.CheckProjectPermissions(ctx, req.ProjectID)
	if err != nil {
		return err
	}

	if projectRole != app.ProjectRoleOwner {
		return app.NewError(http.StatusForbidden, "You do not have permission to add members to this project")
	}

	members := make([]*entity.ProjectMember, 0, len(req.ProjectMemberRoles))
	for _, role := range req.ProjectMemberRoles {
		if !slices.Contains(app.ProjectRoles, role.ProjectRole) {
			return app.NewError(http.StatusBadRequest, fmt.Sprintf("%s is not one of the valid project roles: %v", role.ProjectRole, app.ProjectRoles))
		}
		members = append(members, &entity.ProjectMember{
			ProjectID:   req.ProjectID,
			UserID:      role.UserID,
			ProjectRole: role.ProjectRole,
			AuditBase: entity.AuditBase{
				CreatedBy: api.UserID(ctx),
				UpdatedBy: api.UserID(ctx),
			},
		})
	}

	var failureCount int
	for _, member := range members {
		if err := db.Instance().Create(member).Error; err != nil {
			if db.IsErrDuplicatedKey(err) {
				log.Printf("project member %s already exists in project %s", member.UserID, member.ProjectID)
				continue // Skip if member already exists
			}
			failureCount++
		}
	}
	if failureCount > 0 {
		return app.NewError(http.StatusInternalServerError, fmt.Sprintf("%d members failed to add to project", failureCount))
	}

	return nil
}

func (s *projectService) UpdateProjectMember(ctx context.Context, req *models.UpdateProjectMemberRequest) (*models.ProjectMemberModel, app.Error) {
	projectRole, err := s.CheckProjectPermissions(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	if projectRole != app.ProjectRoleOwner {
		return nil, app.NewError(http.StatusForbidden, "You do not have permission to update members in this project")
	}

	if !slices.Contains(app.ProjectRoles, req.ProjectRole) {
		return nil, app.NewError(http.StatusBadRequest, fmt.Sprintf("%s is not one of the valid project roles: %v", req.ProjectRole, app.ProjectRoles))
	}

	member := &entity.ProjectMember{}
	if err := db.Instance().First(member, "project_id = ? AND user_id = ?", req.ProjectID, req.UserID).Error; err != nil {
		log.Printf("failed to find project member %s in project %s: %v", req.UserID, req.ProjectID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Project member not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	member.ProjectRole = req.ProjectRole
	if err := db.Instance().Save(member).Error; err != nil {
		log.Printf("failed to update project member %s in project %s: %v", req.UserID, req.ProjectID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.ProjectMemberModel{
		ProjectID:   member.ProjectID,
		UserID:      member.UserID,
		ProjectRole: member.ProjectRole,
	}, nil
}

func (s *projectService) RemoveProjectMembers(ctx context.Context, req *models.RemoveProjectMembersRequest) app.Error {
	projectRole, err := s.CheckProjectPermissions(ctx, req.ProjectID)
	if err != nil {
		return err
	}
	if projectRole != app.ProjectRoleOwner {
		return app.NewError(http.StatusForbidden, "You do not have permission to remove members from this project")
	}

	var (
		failureCount int
	)
	for _, userID := range req.UserIDs {
		if err := db.Instance().Delete(&entity.ProjectMember{}, "project_id = ? AND user_id = ?", req.ProjectID, userID).Error; err != nil {
			log.Printf("failed to remove project member %s from project %s: %v", userID, req.ProjectID, err)
			if db.IsErrRecordNotFound(err) {
				continue // Ignore if member not found
			}
			failureCount++
		}
	}

	if failureCount > 0 {
		return app.NewError(http.StatusInternalServerError, fmt.Sprintf("%d members failed to remove from project", failureCount))
	}

	return nil
}
