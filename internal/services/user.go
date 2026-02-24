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

func DeleteUser(userID string) error {
	return db.DB.Delete(&entities.User{}, "id = ?", userID).Error
}

func ChangeUserRole(userID string, role string) error {
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
