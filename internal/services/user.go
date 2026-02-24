package services

import (
	"errors"
	"fmt"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
		ProjectRole: "owner",
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
		Role:     "user",
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

func SignIn(req *models.SignInRequest) (*entities.User, error) {
	var user entities.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	return &user, nil
}

func GetUser(userID string) (*entities.User, error) {
	var user entities.User
	if err := db.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func ListUsers() ([]entities.User, error) {
	var users []entities.User
	if err := db.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
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
	var count int64
	if err := db.DB.Model(&entities.User{}).Where("role = ?", "admin").Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func DeleteUser(userID string) error {
	user, err := GetUser(userID)
	if err != nil {
		return err
	}

	// Prevent deleting the last admin user.
	if user.Role == "admin" {
		adminCount, err := countAdmins()
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return ErrDeleteLastAdmin
		}
	}

	return db.DB.Delete(&entities.User{}, "id = ?", userID).Error
}

func ChangeUserRole(userID string, role string) error {
	user, err := GetUser(userID)
	if err != nil {
		return err
	}

	// Prevent demoting the last admin user to a non-admin role.
	if user.Role == "admin" && role != "admin" {
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
		if req.Role != "admin" && req.Role != "user" {
			req.Role = "user"
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
