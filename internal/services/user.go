package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	defaultBootstrapAdminUsername = "ketches"
	defaultBootstrapAdminPassword = "ketches"
)

var (
	ErrDeleteLastAdmin = errors.New("cannot delete the last admin user")
	ErrDemoteLastAdmin = errors.New("cannot demote the last admin user")
)

func createDefaultProject(tx *gorm.DB, user *entities.User) error {
	displayName := user.Fullname
	if displayName == "" {
		displayName = user.Username
	}

	project := &entities.Project{
		Base:        entities.Base{ID: uuid.New()},
		Slug:        fmt.Sprintf("%s-project", user.Username),
		Name:        fmt.Sprintf("%s's Project", displayName),
		Description: fmt.Sprintf("%s's default project", displayName),
	}

	if err := tx.Create(project).Error; err != nil {
		return err
	}

	member := &entities.ProjectMember{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		UserID:      user.ID,
		ProjectRole: app.ProjectRoleOwner,
	}

	return tx.Create(member).Error
}

func SignUp(req *models.SignUpRequest) (*entities.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &entities.User{
		Base:     entities.Base{ID: uuid.New()},
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Fullname: req.Fullname,
		Role:     app.UserRoleUser,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		return createDefaultProject(tx, user)
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

func SignIn(req *models.SignInRequest) (*entities.User, bool, error) {
	var user entities.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, false, errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, false, errors.New("invalid username or password")
	}

	mustChangePassword := user.Role == app.UserRoleAdmin &&
		user.Username == defaultBootstrapAdminUsername &&
		req.Password == defaultBootstrapAdminPassword

	return &user, mustChangePassword, nil
}

func GetUser(userID string) (*entities.User, error) {
	var user entities.User
	if err := db.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func ListUsers(page, pageSize int, search string) (int64, []entities.User, error) {
	var users []entities.User
	var total int64

	query := db.DB.Model(&entities.User{})

	// Apply search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("username LIKE ? OR fullname LIKE ? OR email LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return 0, nil, err
	}

	return total, users, nil
}

func UpdateUser(userID string, fullname, email, phone string) (*entities.User, error) {
	user, err := GetUser(userID)
	if err != nil {
		return nil, err
	}

	user.Fullname = fullname
	user.Email = email
	user.Phone = phone

	if err := db.DB.Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// countAdmins returns the number of admin users in the system.
func countAdmins() (int64, error) {
	return countAdminsTx(db.DB)
}

func countAdminsTx(tx *gorm.DB) (int64, error) {
	var count int64
	if err := tx.Model(&entities.User{}).Where("role = ?", app.UserRoleAdmin).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func getUserByID(tx *gorm.DB, userID string, unscoped bool) (*entities.User, error) {
	var user entities.User
	query := tx
	if unscoped {
		query = query.Unscoped()
	}
	if err := query.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func listOwnedProjectIDsByUser(tx *gorm.DB, userID string) ([]string, error) {
	projectIDs := make([]string, 0)
	err := tx.Model(&entities.ProjectMember{}).
		Distinct("project_id").
		Where("user_id = ? AND project_role = ?", userID, app.ProjectRoleOwner).
		Pluck("project_id", &projectIDs).Error
	if err != nil {
		return nil, err
	}
	return projectIDs, nil
}

func softDeleteOwnedProjects(tx *gorm.DB, userID string) error {
	projectIDs, err := listOwnedProjectIDsByUser(tx, userID)
	if err != nil {
		return err
	}
	if len(projectIDs) == 0 {
		return nil
	}
	return tx.Where("id IN ?", projectIDs).Delete(&entities.Project{}).Error
}

func restoreOwnedProjects(tx *gorm.DB, userID string) error {
	projectIDs, err := listOwnedProjectIDsByUser(tx, userID)
	if err != nil {
		return err
	}
	if len(projectIDs) == 0 {
		return nil
	}
	return tx.Unscoped().Model(&entities.Project{}).Where("id IN ?", projectIDs).Update("deleted_at", nil).Error
}

func permanentlyDeleteOwnedProjects(tx *gorm.DB, userID string) error {
	projectIDs, err := listOwnedProjectIDsByUser(tx, userID)
	if err != nil {
		return err
	}
	if len(projectIDs) == 0 {
		return nil
	}

	if err := tx.Unscoped().Where("project_id IN ?", projectIDs).Delete(&entities.ProjectMember{}).Error; err != nil {
		return err
	}

	return tx.Unscoped().Where("id IN ?", projectIDs).Delete(&entities.Project{}).Error
}

// EnsureBootstrapAdmin creates the built-in admin account only when no admin exists.
// It never creates a default project for the bootstrap admin.
func EnsureBootstrapAdmin() error {
	adminCount, err := countAdmins()
	if err != nil {
		return err
	}
	if adminCount > 0 {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultBootstrapAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := &entities.User{
		Base:     entities.Base{ID: uuid.New()},
		Username: defaultBootstrapAdminUsername,
		Email:    buildBootstrapAdminEmail(defaultBootstrapAdminUsername),
		Password: string(hashedPassword),
		Fullname: "Ketches Administrator",
		Role:     app.UserRoleAdmin,
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Double check in transaction to avoid duplicate admin creation during concurrent startups.
		var cnt int64
		if err := tx.Model(&entities.User{}).Where("role = ?", app.UserRoleAdmin).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			return nil
		}
		return tx.Create(admin).Error
	})
}

func buildBootstrapAdminEmail(username string) string {
	name := strings.TrimSpace(username)
	if name == "" {
		name = "admin"
	}
	return fmt.Sprintf("%s@local.ketches", name)
}

func DeleteUser(userID string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		user, err := getUserByID(tx, userID, false)
		if err != nil {
			return err
		}

		// Prevent deleting the last admin user.
		if user.Role == app.UserRoleAdmin {
			adminCount, err := countAdminsTx(tx)
			if err != nil {
				return err
			}
			if adminCount <= 1 {
				return ErrDeleteLastAdmin
			}
		}

		if err := softDeleteOwnedProjects(tx, userID); err != nil {
			return err
		}

		return tx.Delete(&entities.User{}, "id = ?", userID).Error
	})
}

func RestoreUser(userID string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		user, err := getUserByID(tx, userID, true)
		if err != nil {
			return err
		}
		if !user.DeletedAt.Valid {
			return errors.New("cannot restore active user")
		}

		if err := restoreOwnedProjects(tx, userID); err != nil {
			return err
		}

		return tx.Unscoped().Model(&entities.User{}).Where("id = ?", userID).Update("deleted_at", nil).Error
	})
}

func PermanentlyDeleteUser(userID string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		user, err := getUserByID(tx, userID, true)
		if err != nil {
			return err
		}
		if !user.DeletedAt.Valid {
			return errors.New("cannot permanently delete active user")
		}

		if err := permanentlyDeleteOwnedProjects(tx, userID); err != nil {
			return err
		}

		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&entities.ProjectMember{}).Error; err != nil {
			return err
		}

		return tx.Unscoped().Delete(&entities.User{}, "id = ?", userID).Error
	})
}

func ChangeUserRole(userID string, role string) error {
	user, err := GetUser(userID)
	if err != nil {
		return err
	}

	// Prevent demoting the last admin user to a non-admin role.
	if user.Role == app.UserRoleAdmin && role != app.UserRoleAdmin {
		adminCount, err := countAdmins()
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return ErrDemoteLastAdmin
		}
	}

	return db.DB.Model(&entities.User{}).Where("id = ?", userID).Update("role", role).Error
}

func CreateUser(req *models.CreateUserRequest) (*entities.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &entities.User{
		Base:     entities.Base{ID: uuid.New()},
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Fullname: req.Fullname,
		Phone:    req.Phone,
		Role:     req.Role,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		return createDefaultProject(tx, user)
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

// BatchImportUsers imports multiple users at once
func BatchImportUsers(requests []models.CreateUserRequest) (*models.BatchImportResponse, error) {
	response := &models.BatchImportResponse{
		Succeeded: 0,
		Failed:    0,
		Errors:    []models.ImportError{},
		Users:     []models.UserResponse{},
	}

	for i, req := range requests {
		// Validate role
		if req.Role != app.UserRoleAdmin && req.Role != app.UserRoleUser {
			req.Role = app.UserRoleUser
		}

		user, err := CreateUser(&req)
		if err != nil {
			response.Failed++
			response.Errors = append(response.Errors, models.ImportError{
				Index:   i,
				Message: fmt.Sprintf("failed to create user %s: %v", req.Username, err),
			})
			continue
		}

		response.Succeeded++
		response.Users = append(response.Users, models.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Fullname:  user.Fullname,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		})
	}

	return response, nil
}
