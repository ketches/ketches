package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// lockForUpdate serializes read/modify/write operations on PostgreSQL and
// MySQL. SQLite has no row-level locking clause, so the transaction itself is
// the strongest portable primitive available there.
func lockForUpdate(tx *gorm.DB) *gorm.DB {
	if tx == nil || tx.Dialector.Name() == "sqlite" {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}

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
		Select(`projects.id, projects.slug, projects.name, projects.description, projects.collaboration_enabled, projects.created_at,
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
	query := db.DB.Model(&entities.Project{}).Select("projects.id, projects.name, projects.slug, projects.description, projects.collaboration_enabled")
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
		Base:                 entities.Base{ID: uuid.New()},
		Slug:                 req.Slug,
		Name:                 req.Name,
		Description:          req.Description,
		CollaborationEnabled: req.CollaborationEnabled,
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
	project.CollaborationEnabled = req.CollaborationEnabled

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

func PermanentlyDeleteProject(projectID string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := loadRecycleBinProject(tx, projectID, actor); err != nil {
			return err
		}
		if err := claimRecycleBinDeletionTargets(tx, newRecycleBinDeletionTarget(
			recycleBinResourceProject, projectID, &entities.Project{}, "project",
		)); err != nil {
			return err
		}
		return validateProjectPermanentDeletionTx(tx, projectID)
	}); err != nil {
		return err
	}
	if err := cleanupProjectNamespaces(context.Background(), projectID); err != nil {
		return err
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := loadRecycleBinProject(tx, projectID, actor); err != nil {
			return err
		}
		return permanentlyDeleteProjectTx(tx, projectID)
	})
}

func cleanupProjectNamespaces(ctx context.Context, projectID string) error {
	if !db.DB.Migrator().HasTable(&entities.Env{}) {
		return nil
	}
	var envs []entities.Env
	if err := db.DB.Unscoped().Where("project_id = ?", projectID).Find(&envs).Error; err != nil {
		return err
	}
	for _, env := range envs {
		if env.ClusterID == "" || env.ClusterNamespace == "" {
			continue
		}
		err := core.DeleteNamespace(ctx, env.ClusterID, env.ClusterNamespace)
		if k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func permanentlyDeleteProjectTx(tx *gorm.DB, projectID string) error {
	var project entities.Project
	if err := tx.Unscoped().First(&project, "id = ?", projectID).Error; err != nil {
		return err
	}
	if !project.DeletedAt.Valid {
		return app.WrapErrorf(ErrRecycleBinResourceActive, "project %s", projectID)
	}

	if err := deleteProjectOwnedRecordsTx(context.Background(), tx, projectID); err != nil {
		return err
	}
	if err := tx.Unscoped().Delete(&entities.Project{}, "id = ?", projectID).Error; err != nil {
		return err
	}
	return deleteRecycleBinDeletionClaim(tx, recycleBinResourceProject, projectID)
}

func IsProjectMember(projectID, userID string) (bool, error) {
	var count int64
	if err := db.DB.Model(&entities.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func RestoreProject(projectID string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return restoreProjectTx(tx, projectID)
	})
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

var ErrInvalidProjectRole = errors.New("invalid project role")

func validateProjectRole(role string) error {
	switch role {
	case app.ProjectRoleOwner, app.ProjectRoleDeveloper, app.ProjectRoleViewer:
		return nil
	default:
		return ErrInvalidProjectRole
	}
}

func UpdateProjectMemberRole(projectID, userID, role string) error {
	if err := validateProjectRole(role); err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the project before counting owners. All owner mutations use the
		// same lock order, preventing two concurrent demotions from both
		// observing the last owner.
		var project entities.Project
		if err := lockForUpdate(tx).First(&project, "id = ?", projectID).Error; err != nil {
			return err
		}

		var member entities.ProjectMember
		if err := lockForUpdate(tx).Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error; err != nil {
			return err
		}
		if member.ProjectRole == app.ProjectRoleOwner && role != app.ProjectRoleOwner {
			var ownerCount int64
			if err := tx.Model(&entities.ProjectMember{}).
				Where("project_id = ? AND project_role = ?", projectID, app.ProjectRoleOwner).
				Count(&ownerCount).Error; err != nil {
				return err
			}
			if ownerCount <= 1 {
				return errors.New("at least one owner is required")
			}
		}
		return tx.Model(&entities.ProjectMember{}).
			Where("id = ?", member.ID).
			Update("project_role", role).Error
	})
}

func InviteProjectMembers(projectID string, userIDs []string, role string, senderID string) error {
	if err := validateProjectRole(role); err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Serializing invitations by project makes the member/pending-notification
		// decision atomic and prevents duplicate invitations under concurrent
		// requests.
		var project entities.Project
		if err := lockForUpdate(tx).First(&project, "id = ?", projectID).Error; err != nil {
			return err
		}

		actionDataBytes, err := json.Marshal(map[string]string{"role": role})
		if err != nil {
			return err
		}
		actionData := string(actionDataBytes)
		for _, userID := range userIDs {
			var member entities.ProjectMember
			memberErr := lockForUpdate(tx).
				Where("project_id = ? AND user_id = ?", projectID, userID).
				First(&member).Error
			if memberErr == nil {
				if member.ProjectRole == app.ProjectRoleOwner && role != app.ProjectRoleOwner {
					var ownerCount int64
					if err := tx.Model(&entities.ProjectMember{}).
						Where("project_id = ? AND project_role = ?", projectID, app.ProjectRoleOwner).
						Count(&ownerCount).Error; err != nil {
						return err
					}
					if ownerCount <= 1 {
						return errors.New("at least one owner is required")
					}
				}
				if err := tx.Model(&entities.ProjectMember{}).
					Where("id = ?", member.ID).
					Update("project_role", role).Error; err != nil {
					return err
				}
				continue
			}
			if !errors.Is(memberErr, gorm.ErrRecordNotFound) {
				return memberErr
			}

			var invitation entities.Notification
			pendingErr := lockForUpdate(tx).
				Where("recipient_id = ? AND resource_id = ? AND event_type = ? AND status = ?", userID, projectID, "project_invitation", "pending").
				First(&invitation).Error
			switch {
			case pendingErr == nil:
				if err := tx.Model(&entities.Notification{}).Where("id = ?", invitation.ID).Update("action_data", actionData).Error; err != nil {
					return err
				}
			case errors.Is(pendingErr, gorm.ErrRecordNotFound):
				notification := newNotification(
					userID,
					senderID,
					"invitation",
					"project_invitation",
					"Project Invitation",
					"You have been invited to join project \""+project.Name+"\"",
					"project",
					projectID,
					projectID,
					actionData,
				)
				if err := tx.Create(notification).Error; err != nil {
					return err
				}
			default:
				return pendingErr
			}
		}
		return nil
	})
}

// ListInvitableUsers returns users who can be invited to a project.
// It excludes: admin-role users, existing project members, and users with
// pending invitations. Users who refused a previous invitation ARE included.
func ListInvitableUsers(projectID string, search string) ([]entities.User, error) {
	var users []entities.User
	query := db.DB.Model(&entities.User{}).
		Where("role != ?", app.UserRoleAdmin).
		Where("id NOT IN (?)",
			db.DB.Model(&entities.ProjectMember{}).Select("user_id").Where("project_id = ?", projectID),
		).
		Where("id NOT IN (?)",
			db.DB.Model(&entities.Notification{}).Select("recipient_id").
				Where("resource_id = ? AND event_type = 'project_invitation' AND status = 'pending'", projectID),
		)

	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("username LIKE ? OR fullname LIKE ? OR email LIKE ?", pattern, pattern, pattern)
	}

	if err := query.Order("username").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func RemoveProjectMember(projectID, userID string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var project entities.Project
		if err := lockForUpdate(tx).First(&project, "id = ?", projectID).Error; err != nil {
			return err
		}
		var member entities.ProjectMember
		if err := lockForUpdate(tx).Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error; err != nil {
			return err
		}
		if member.ProjectRole == app.ProjectRoleOwner {
			var ownerCount int64
			if err := tx.Model(&entities.ProjectMember{}).
				Where("project_id = ? AND project_role = ?", projectID, app.ProjectRoleOwner).
				Count(&ownerCount).Error; err != nil {
				return err
			}
			if ownerCount <= 1 {
				return errors.New("at least one owner is required")
			}
		}
		return tx.Delete(&entities.ProjectMember{}, "id = ?", member.ID).Error
	})
}
