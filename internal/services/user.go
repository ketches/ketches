package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrDeleteLastAdmin                     = errors.New("cannot delete the last admin user")
	ErrDemoteLastAdmin                     = errors.New("cannot demote the last admin user")
	ErrInvalidCurrentPassword              = errors.New("current password is incorrect")
	ErrUsernameAlreadyExists               = errors.New("username already exists")
	ErrEmailAlreadyExists                  = errors.New("email already exists")
	ErrBootstrapAdminPasswordNotConfigured = app.ErrBootstrapAdminPasswordNotConfigured
	ErrPasswordChangeRequired              = errors.New("password change required")
)

const (
	defaultBootstrapAdminUsername = "kadmin"
	defaultBootstrapAdminFullname = "Ketches Admin"
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
	if req == nil {
		return nil, errors.New("signup request is required")
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	email, err := normalizeEmailAddress(req.Email)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &entities.User{
		Base:     entities.Base{ID: uuid.New()},
		Username: strings.TrimSpace(req.Username),
		Email:    email,
		Password: string(hashedPassword),
		Fullname: strings.TrimSpace(req.Fullname),
		Role:     app.UserRoleUser,
	}

	enabled, err := GetPublicSignUpEnabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrPublicSignUpDisabled
	}
	verificationRequired, err := GetSignUpEmailVerificationRequired()
	if err != nil {
		return nil, err
	}
	invalidVerificationCode := false
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if verificationRequired {
			if err := consumeSignupVerificationCode(tx, req.Email, req.VerificationCode); err != nil {
				if errors.Is(err, ErrInvalidVerificationCode) {
					invalidVerificationCode = true
					return nil
				}
				return err
			}
		}
		if err := tx.Create(user).Error; err != nil {
			return mapUserWriteError(err)
		}

		return createDefaultProject(tx, user)
	})

	if err != nil {
		return nil, err
	}
	if invalidVerificationCode {
		return nil, ErrInvalidVerificationCode
	}

	return user, nil
}

func SignIn(req *models.SignInRequest) (*entities.User, bool, error) {
	var user entities.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, false, errors.New("invalid username or password")
	}
	if user.IsLocked {
		return nil, false, ErrAccountLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, false, errors.New("invalid username or password")
	}

	return &user, user.MustChangePassword, nil
}

func GetUser(userID string) (*entities.User, error) {
	var user entities.User
	if err := db.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByUsername(username string) (*entities.User, error) {
	var user entities.User
	if err := db.DB.Where("username = ?", strings.TrimSpace(username)).First(&user).Error; err != nil {
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

func GetCurrentUserProfile(userID string) (*entities.User, error) {
	return GetUser(userID)
}

func UpdateCurrentUserProfile(userID, fullname, email, bio string) (*entities.User, error) {
	user, err := GetUser(userID)
	if err != nil {
		return nil, err
	}

	user.Fullname = strings.TrimSpace(fullname)
	user.Email = strings.TrimSpace(email)
	user.Bio = strings.TrimSpace(bio)

	if err := db.DB.Save(user).Error; err != nil {
		return nil, mapUserWriteError(err)
	}
	return user, nil
}

func UpdateUser(userID, fullname, email, bio, phone string) (*entities.User, error) {
	user, err := GetUser(userID)
	if err != nil {
		return nil, err
	}

	user.Fullname = fullname
	user.Email = email
	user.Bio = bio
	user.Phone = phone

	if err := db.DB.Save(user).Error; err != nil {
		return nil, mapUserWriteError(err)
	}
	return user, nil
}

func ChangeCurrentUserPassword(userID, currentPassword, newPassword string) error {
	user, err := GetUser(userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return ErrInvalidCurrentPassword
	}

	return ChangeUserPassword(userID, newPassword)
}

func ChangeUserPassword(userID, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return db.DB.Model(&entities.User{}).Where("id = ?", userID).Updates(map[string]any{
		"password":             string(hashedPassword),
		"refresh_token":        "",
		"must_change_password": false,
	}).Error
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

	result := tx.Unscoped().Model(&entities.Project{}).
		Where("projects.id IN ? AND projects.deleted_at IS NOT NULL", projectIDs).
		Where(`NOT EXISTS (
			SELECT 1
			FROM recycle_bin_deletion_claims AS claims
			WHERE claims.resource_type = ? AND claims.resource_id = projects.id
		)`, recycleBinResourceProject).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	for _, projectID := range projectIDs {
		claimed, err := hasRecycleBinDeletionClaim(tx, recycleBinResourceProject, projectID)
		if err != nil {
			return err
		}
		if claimed {
			return app.WrapErrorf(ErrRecycleBinResourceDeleting, "project %s", projectID)
		}
	}
	return nil
}

func permanentlyDeleteOwnedProjects(tx *gorm.DB, userID string) error {
	projectIDs, err := listOwnedProjectIDsByUser(tx, userID)
	if err != nil {
		return err
	}
	if len(projectIDs) == 0 {
		return nil
	}

	for _, projectID := range projectIDs {
		if err := permanentlyDeleteProjectTx(tx, projectID); err != nil {
			return err
		}
	}
	return nil
}

// EnsureBootstrapAdmin creates the bootstrap admin account only when it is
// needed and when no admin exists. It never creates a default project for the
// bootstrap admin.
func EnsureBootstrapAdmin() error {
	adminCount, err := countAdmins()
	if err != nil {
		return err
	}
	if adminCount > 0 {
		return nil
	}

	bootstrapUsername := resolveBootstrapAdminUsername()
	bootstrapPassword := app.Config.BootstrapAdminPassword
	if strings.TrimSpace(bootstrapPassword) == "" {
		return ErrBootstrapAdminPasswordNotConfigured
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(bootstrapPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := &entities.User{
		Base:               entities.Base{ID: uuid.New()},
		Username:           bootstrapUsername,
		Email:              buildBootstrapAdminEmail(bootstrapUsername),
		Password:           string(hashedPassword),
		Fullname:           buildBootstrapAdminFullname(bootstrapUsername),
		Role:               app.UserRoleAdmin,
		MustChangePassword: true,
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

func resolveBootstrapAdminUsername() string {
	if username := strings.TrimSpace(app.Config.BootstrapAdminUsername); username != "" {
		return username
	}
	return defaultBootstrapAdminUsername
}

func buildBootstrapAdminFullname(username string) string {
	if strings.TrimSpace(app.Config.BootstrapAdminUsername) == "" {
		return defaultBootstrapAdminFullname
	}

	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, username)
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return defaultBootstrapAdminFullname
	}

	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}

	return strings.Join(parts, " ")
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
		err := restoreUserTx(tx, userID)
		if errors.Is(err, ErrRecycleBinResourceActive) {
			return app.WrapError("cannot restore active user", err)
		}
		return err
	})
}

func PermanentlyDeleteUser(userID string) error {
	var ownedProjectIDs []string
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		user, err := getUserByID(tx, userID, true)
		if err != nil {
			return err
		}
		if !user.DeletedAt.Valid {
			return errors.New("cannot permanently delete active user")
		}
		ownedProjectIDs, err = listOwnedProjectIDsByUser(tx, userID)
		if err != nil {
			return err
		}

		targets := make([]recycleBinDeletionTarget, 0, len(ownedProjectIDs)+1)
		targets = append(targets, newRecycleBinDeletionTarget(
			recycleBinResourceUser, userID, &entities.User{}, "user",
		))
		for _, projectID := range ownedProjectIDs {
			targets = append(targets, newRecycleBinDeletionTarget(
				recycleBinResourceProject, projectID, &entities.Project{}, "project",
			))
		}
		if err := claimRecycleBinDeletionTargets(tx, targets...); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	for _, projectID := range ownedProjectIDs {
		if err := cleanupProjectNamespaces(context.Background(), projectID); err != nil {
			return err
		}
	}

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
		if err := deleteUserOwnedRecordsTx(tx, userID); err != nil {
			return err
		}

		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&entities.ProjectMember{}).Error; err != nil {
			return err
		}

		if err := tx.Unscoped().Delete(&entities.User{}, "id = ?", userID).Error; err != nil {
			return err
		}
		return deleteRecycleBinDeletionClaim(tx, recycleBinResourceUser, userID)
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
	if req == nil {
		return nil, errors.New("create user request is required")
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	email, err := normalizeEmailAddress(req.Email)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &entities.User{
		Base:     entities.Base{ID: uuid.New()},
		Username: strings.TrimSpace(req.Username),
		Email:    email,
		Password: string(hashedPassword),
		Fullname: strings.TrimSpace(req.Fullname),
		Phone:    strings.TrimSpace(req.Phone),
		Role:     req.Role,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return mapUserWriteError(err)
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

func SetUserLockState(userID string, locked bool, reason string) (*entities.User, error) {
	if _, err := GetUser(userID); err != nil {
		return nil, err
	}

	updates := map[string]any{
		"is_locked":     locked,
		"locked_reason": strings.TrimSpace(reason),
	}
	if locked {
		now := currentTime()
		updates["locked_at"] = &now
		updates["refresh_token"] = ""
	} else {
		updates["locked_at"] = nil
		updates["locked_reason"] = ""
	}

	if err := db.DB.Model(&entities.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetUser(userID)
}

func mapUserWriteError(err error) error {
	if err == nil {
		return nil
	}

	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "users.username"), strings.Contains(lower, "username"):
		if isUniqueConstraintError(lower) {
			return ErrUsernameAlreadyExists
		}
	case strings.Contains(lower, "users.email"), strings.Contains(lower, "email"):
		if isUniqueConstraintError(lower) {
			return ErrEmailAlreadyExists
		}
	}

	if isUniqueConstraintError(lower) {
		return errors.New("user already exists")
	}
	return err
}

func isUniqueConstraintError(lowerErr string) bool {
	return strings.Contains(lowerErr, "unique constraint") ||
		strings.Contains(lowerErr, "duplicate key") ||
		strings.Contains(lowerErr, "error 1062")
}
