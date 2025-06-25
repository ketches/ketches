/*
Copyright 2025 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"unicode"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	emailRgx = regexp.MustCompile(`^[a-zA-Z0-9._-]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	phoneRgx = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type UserService interface {
	List(ctx context.Context, req *models.ListUsersRequest) (*models.ListUsersResponse, app.Error)
	Get(ctx context.Context, req *models.GetUserProfileRequest) (*models.UserModel, app.Error)
	SignUp(ctx context.Context, req *models.UserSignUpRequest) (*models.UserModel, app.Error)
	SignIn(ctx context.Context, req *models.UserSignInRequest) (*models.UserModel, app.Error)
	SignOut(ctx context.Context, req *models.UserSignOutRequest, refreshToken string) app.Error
	RefreshToken(ctx context.Context, refreshToken string) (*models.UserModel, app.Error)
	Update(ctx context.Context, req *models.UserUpdateRequest) (*models.UserModel, app.Error)
	Rename(ctx context.Context, req *models.UserRenameRequest) (*models.UserModel, app.Error)
	ChangeRole(ctx context.Context, req *models.UserChangeRoleRequest) (*models.UserModel, app.Error)
	ResetPassword(ctx context.Context, req *models.UserResetPasswordRequest) (*models.UserModel, app.Error)
	Delete(ctx context.Context, req *models.DeleteUserRequest) app.Error
}

type userService struct {
	Service
}

func NewUserService() UserService {
	return &userService{
		Service: LoadService(),
	}
}

func (s *userService) List(ctx context.Context, req *models.ListUsersRequest) (*models.ListUsersResponse, app.Error) {
	query := db.Instance().Model(&entities.User{})

	if req.Query != "" {
		query = db.CaseInsensitiveLike(query, req.Query, "username", "fullname")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("failed to count users: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	users := []entities.User{}
	if err := query.Find(&users).Error; err != nil {
		return nil, app.ErrDatabaseOperationFailed
	}

	result := &models.ListUsersResponse{
		Total:   total,
		Records: make([]*models.UserModel, 0, len(users)),
	}
	for _, user := range users {
		result.Records = append(result.Records, &models.UserModel{
			UserID:   user.ID,
			Username: user.Username,
			Email:    user.Email,
			Fullname: user.Fullname,
			Gender:   user.Gender,
			Phone:    user.Phone,
			Role:     user.Role,
		})
	}

	return result, nil
}

func (s *userService) Get(ctx context.Context, req *models.GetUserProfileRequest) (*models.UserModel, app.Error) {
	user := new(entities.User)
	if err := db.Instance().First(user, "id = ?", req.UserID).Error; err != nil {
		log.Printf("user %s does not exist: %v\n", req.UserID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, fmt.Sprintf("user %s does not exist", req.UserID))
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.UserModel{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Fullname: user.Fullname,
		Gender:   user.Gender,
		Phone:    user.Phone,
		Role:     user.Role,
	}, nil
}

func (s *userService) SignUp(ctx context.Context, req *models.UserSignUpRequest) (*models.UserModel, app.Error) {
	if err := s.validateSignUpUser(req); err != nil {
		return nil, app.NewError(http.StatusBadRequest, fmt.Sprintf("invalid user sign up request: %v", err))
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, app.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to generate password hash: %v", err))
	}

	user := &entities.User{
		Username: req.Username,
		Email:    req.Email,
		Fullname: req.Fullname,
		Password: string(passwordHashBytes),
		Phone:    req.Phone,
		Role:     req.Role,
	}

	if err := db.Instance().Create(user).Error; err != nil {
		log.Printf("failed to sign up user: %v\n", err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusConflict, fmt.Sprintf("username or email already exists"))
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	// Create a default project for signed-up user
	go func() {
		NewProjectService().CreateProject(ctx, &models.CreateProjectRequest{
			Slug:        fmt.Sprintf("%s-%s", user.Username, user.ID[0:5]),
			DisplayName: fmt.Sprintf("%s's Personal Project", user.Fullname),
			Description: fmt.Sprintf("%s's personal project, automatically created upon user sign up", user.Fullname),
			Operator:    user.ID,
		})
	}()

	return &models.UserModel{
		UserID:   user.ID,
		Username: user.Username,
		Fullname: user.Fullname,
		Email:    user.Email,
		Phone:    user.Phone,
		Role:     user.Role,
	}, nil
}

func (s *userService) SignIn(ctx context.Context, req *models.UserSignInRequest) (*models.UserModel, app.Error) {
	user := new(entities.User)
	if err := db.Instance().First(user, "username = ?", req.Username).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			if err := db.Instance().First(user, "email = ?", req.Username).Error; err != nil {
				if db.IsErrRecordNotFound(err) {
					return nil, app.NewError(http.StatusNotFound, fmt.Sprintf("user %s does not exist", req.Username))
				} else {
					log.Printf("failed to get user by email %s: %v\n", req.Username, err)
					return nil, app.ErrDatabaseOperationFailed
				}
			}
		} else {
			log.Printf("failed to get user by username %s: %v\n", req.Username, err)
			return nil, app.ErrDatabaseOperationFailed
		}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, app.NewError(http.StatusUnauthorized, "incorrect username or password")
	}

	// Generate access token
	accessToken, expiresAt, err := app.GenerateToken(app.TokenClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		UserRole:  user.Role,
		TokenType: app.TokenTypeAccessToken,
	})
	if err != nil {
		log.Printf("failed to generate user %s access token: %v", user.ID, err)
		return nil, app.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to generate user %s access token", user.ID))
	}

	// Generate refresh token
	refreshToken, expiresAt, err := app.GenerateToken(app.TokenClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		UserRole:  user.Role,
		TokenType: app.TokenTypeRefreshToken,
	})
	if err != nil {
		log.Printf("failed to generate user %s refresh token: %v", user.ID, err)
		return nil, app.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to generate user %s refresh token", user.ID))
	}

	if err := db.Instance().Create(&entities.UserToken{
		UserID:    user.ID,
		Token:     refreshToken,
		TokenType: string(app.TokenTypeRefreshToken),
		ExpiresAt: expiresAt.Unix(),
		AuditBase: entities.AuditBase{
			CreatedBy: user.ID,
		},
	}).Error; err != nil {
		log.Printf("failed to create user %s refresh token: %v", user.ID, err)
		return nil, app.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to create user %s refresh token", user.ID))
	}

	return &models.UserModel{
		UserID:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Fullname:     user.Fullname,
		Gender:       user.Gender,
		Phone:        user.Phone,
		Role:         user.Role,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *userService) SignOut(ctx context.Context, req *models.UserSignOutRequest, refreshToken string) app.Error {
	if req.UserID == "" {
		return app.NewError(http.StatusBadRequest, "user id is required")
	}
	if refreshToken == "" {
		return app.NewError(http.StatusBadRequest, "refresh token is required")
	}

	if err := db.Instance().Delete(&entities.UserToken{}, "user_id = ? AND token = ? AND token_type = ?", req.UserID, refreshToken, app.TokenTypeRefreshToken).Error; err != nil {
		log.Printf("failed to delete user %s refresh token: %v\n", req.UserID, err)
		return app.ErrDatabaseOperationFailed
	}

	log.Printf("user %s signed out successfully", req.UserID)
	return nil
}

func (s *userService) Update(ctx context.Context, req *models.UserUpdateRequest) (*models.UserModel, app.Error) {
	if req.UserID == "" {
		return nil, app.NewError(http.StatusBadRequest, "user id is required")
	}

	if req.Email == "" {
		return nil, app.NewError(http.StatusBadRequest, "email is required")
	}

	user := new(entities.User)
	if err := db.Instance().First(user, "id = ?", req.UserID).Error; err != nil {
		log.Printf("failed to get user %s: %v\n", req.UserID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, fmt.Sprintf("user %s does not exist", req.UserID))
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	user.Email = req.Email
	user.Fullname = req.Fullname
	user.Phone = req.Phone
	user.Gender = req.Gender

	if err := db.Instance().Save(user).Error; err != nil {
		log.Printf("failed to update user %s: %v\n", req.UserID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.UserModel{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Fullname: user.Fullname,
		Gender:   user.Gender,
		Phone:    user.Phone,
		Role:     user.Role,
	}, nil
}

func (s *userService) Rename(ctx context.Context, req *models.UserRenameRequest) (*models.UserModel, app.Error) {
	if req.UserID == "" {
		return nil, app.NewError(http.StatusBadRequest, "user id is required")
	}

	if req.NewUsername == "" {
		return nil, app.NewError(http.StatusBadRequest, "new username is required")
	}

	if req.Password == "" {
		return nil, app.NewError(http.StatusBadRequest, "password is required")
	}

	user := new(entities.User)
	if err := db.Instance().First(user, "id = ?", req.UserID).Error; err != nil {
		log.Printf("failed to get user %s: %v\n", req.UserID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, fmt.Sprintf("user %s does not exist", req.UserID))
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, app.NewError(http.StatusUnauthorized, "incorrect user id or password")
	}

	user.Username = req.NewUsername
	if err := db.Instance().Model(user).Where("id = ?", req.UserID).Update("username", user.Username).Error; err != nil {
		log.Printf("failed to rename user %s: %v\n", req.UserID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.UserModel{
		UserID:   user.ID,
		Username: user.Username,
	}, nil
}

func (s *userService) ChangeRole(ctx context.Context, req *models.UserChangeRoleRequest) (*models.UserModel, app.Error) {
	if !api.IsAdmin(ctx) {
		return nil, app.ErrPermissionDenied
	}

	user := new(entities.User)
	if err := db.Instance().First(user, "id = ?", req.UserID).Error; err != nil {
		log.Printf("failed to get user %s: %v\n", req.UserID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, fmt.Sprintf("user %s does not exist", req.UserID))
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	user.Role = req.NewRole
	if err := db.Instance().Model(user).Where("id = ?", req.UserID).Update("role", user.Role).Error; err != nil {
		log.Printf("failed to change user %s role: %v\n", req.UserID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.UserModel{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

func (s *userService) ResetPassword(ctx context.Context, req *models.UserResetPasswordRequest) (*models.UserModel, app.Error) {
	user := new(entities.User)
	if err := db.Instance().First(user, "id = ?", req.UserID).Error; err != nil {
		log.Printf("failed to get user %s: %v\n", req.UserID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, fmt.Sprintf("user %s does not exist", req.UserID))
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if !api.IsAdmin(ctx) {
		if req.Password == "" {
			log.Println("origin password is required")
			return nil, app.NewError(http.StatusBadRequest, "origin password is required")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			return nil, app.NewError(http.StatusUnauthorized, "incorrect user id or password")
		}
	}

	newPasswordHashBytes, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, app.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to generate password hash: %v", err))
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		user.Password = string(newPasswordHashBytes)
		if err := tx.Model(user).Where("id = ?", req.UserID).Update("password", user.Password).Error; err != nil {
			log.Printf("failed to reset user %s password: %v\n", req.UserID, err)
			return err
		}

		if err := tx.Delete(&entities.UserToken{}, "user_id = ? AND token_type = ?", user.ID, app.TokenTypeRefreshToken).Error; err != nil {
			log.Printf("failed to delete user %s refresh token: %v\n", user.ID, err)
			return err
		}

		log.Printf("user %s password reset successfully", user.ID)
		return nil
	}); err != nil {
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.UserModel{
		UserID:   user.ID,
		Username: user.Username,
	}, nil
}

func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*models.UserModel, app.Error) {
	result := &models.UserModel{
		RefreshToken: refreshToken,
	}
	mc, err := app.ValidateToken(refreshToken)
	if err != nil {
		return result, app.NewError(http.StatusUnauthorized, fmt.Sprintf("invalid refresh token %s: %v", refreshToken, err))
	}

	userToken := &entities.UserToken{}
	if err := db.Instance().First(userToken, "token = ? AND token_type = ?", refreshToken, app.TokenTypeRefreshToken).Error; err != nil {
		log.Printf("failed to find refresh access token %s: %v\n", refreshToken, err)
		return result, app.NewError(http.StatusUnauthorized, fmt.Sprintf("invalid refresh token %s", refreshToken))
	}

	result.UserID = mc.UserID
	result.Username = mc.Username
	result.Email = mc.Email

	if mc.TokenType != app.TokenTypeRefreshToken {
		return result, app.NewError(http.StatusUnauthorized, fmt.Sprintf("not a refresh token: %s", refreshToken))
	}

	accessToken, _, err := app.GenerateToken(app.TokenClaims{
		UserID:    mc.UserID,
		Username:  mc.Username,
		Email:     mc.Email,
		UserRole:  mc.UserRole,
		TokenType: app.TokenTypeAccessToken,
	})
	if err != nil {
		log.Printf("failed to generate access token for user %s: %v\n", mc.UserID, err)
		return result, app.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to generate access token for user %s", mc.UserID))
	}

	result.AccessToken = accessToken
	return result, nil
}

func (s *userService) Delete(ctx context.Context, req *models.DeleteUserRequest) app.Error {
	if err := s.validateDeleteUser(req); err != nil {
		return app.NewError(http.StatusBadRequest, err.Error())
	}

	if api.UserID(ctx) != req.UserID && req.Password == "" {
		return app.NewError(http.StatusBadRequest, "password is required to delete another user")
	}

	user := new(entities.User)
	if err := db.Instance().First(user, "id = ?", req.UserID).Error; err != nil {
		log.Printf("failed to get user %s: %v\n", req.UserID, err)
		if db.IsErrRecordNotFound(err) {
			return app.NewError(http.StatusNotFound, fmt.Sprintf("user %s does not exist", req.UserID))
		}
		return app.ErrDatabaseOperationFailed
	}

	if api.UserID(ctx) == req.UserID {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			return app.NewError(http.StatusUnauthorized, "incorrect user id or password")
		}
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&entities.UserToken{}, "user_id = ?", user.ID).Error; err != nil {
			log.Printf("failed to delete user %s tokens: %v\n", user.ID, err)
			return err
		}

		if err := tx.Delete(&entities.User{}, "id = ?", user.ID).Error; err != nil {
			log.Printf("failed to delete user %s: %v\n", user.ID, err)
			return err
		}

		return nil
	}); err != nil {
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s *userService) validateSignUpUser(req *models.UserSignUpRequest) app.Error {
	for _, c := range req.Username {
		if !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '-' {
			return app.NewError(http.StatusBadRequest, fmt.Sprintf("username %s contains invalid character: %c", req.Username, c))
		}
	}

	if req.Username[0] == '-' || req.Username[len(req.Username)-1] == '-' {
		return app.NewError(http.StatusBadRequest, fmt.Sprintf("user id cannot start or end with '-'"))
	}

	var (
		hasLowercase = false
		hasUppercase = false
		hasNumber    = false
		hasSpecial   = false
	)
	for _, c := range req.Password {
		if unicode.IsLower(c) {
			hasLowercase = true
		}
		if unicode.IsUpper(c) {
			hasUppercase = true
		}
		if unicode.IsNumber(c) {
			hasNumber = true
		}
		if unicode.IsPunct(c) || unicode.IsSymbol(c) {
			hasSpecial = true
		}
	}
	if !hasLowercase || !hasUppercase || !hasNumber || !hasSpecial {
		return app.NewError(http.StatusBadRequest, "password must contain at least one lowercase letter, one uppercase letter, one number, and one special character")
	}

	if len(req.Phone) > 0 && !phoneRgx.MatchString(req.Phone) {
		return s.invalidPhoneNumber(req.Phone)
	}

	return nil
}

func (s *userService) validateDeleteUser(user *models.DeleteUserRequest) error {
	if user.UserID == "" {
		return fmt.Errorf("user id is empty")
	}

	if user.Password == "" {
		return fmt.Errorf("password is empty")
	}

	return nil
}

func (s *userService) invalidPhoneNumber(phone string) app.Error {
	return app.NewError(http.StatusBadRequest, fmt.Sprintf("invalid phone number: %s", phone))
}
