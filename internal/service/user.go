/*
Copyright 2023 The Ketches Authors.

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

package service

import (
	"context"
	"fmt"
	"github.com/ketches/ketches/pkg/ketches"
	"log"
	"log/slog"
	"regexp"
	"unicode"

	"github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/util/jwt"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

var (
	emailRgx = regexp.MustCompile(`^[a-zA-Z0-9._-]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	phoneRgx = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type UserService interface {
	List(ctx context.Context, filter *model.UserFilter) ([]*model.UserModel, error)
	Get(ctx context.Context, accountID string) (*model.UserModel, error)
	SignUp(ctx context.Context, user *model.UserSignUpModel) (*model.UserModel, error)
	SignIn(ctx context.Context, user *model.UserSignInRequest) (*model.UserModel, error)
	RefreshToken(ctx context.Context, refreshToken string) (*model.UserModel, error)
	Update(ctx context.Context, user *model.UserUpdateRequest) (*model.UserUpdateResponse, error)
	Rename(ctx context.Context, user *model.UserRenameRequest) (*model.UserRenameResponse, error)
	ResetPassword(ctx context.Context, in *model.UserResetPasswordRequest) (*model.UserModel, error)
	Delete(ctx context.Context, user *model.DeleteUserRequest) error
}

type userService struct {
	Service
}

func NewUserService() UserService {
	return &userService{
		Service: LoadService(),
	}
}

func (s *userService) List(ctx context.Context, filter *model.UserFilter) ([]*model.UserModel, error) {
	selector := labels.Everything()
	if filter.SpaceID != "" {
		selector = labels.SelectorFromSet(labels.Set{
			"space.core.ketches.io/" + filter.SpaceID: "true",
		})
	}
	userList, err := ketches.Store().UserLister().List(selector)
	if err != nil {
		return nil, err
	}

	userList, _ = model.PagedResult(userList, filter)

	var users []*model.UserModel
	for _, user := range userList {
		users = append(users, &model.UserModel{
			AccountID:         user.Name,
			FullName:          user.Spec.FullName,
			Email:             user.Spec.Email,
			Phone:             user.Spec.Phone,
			MustResetPassword: user.Spec.MustResetPassword,
		})
	}

	return users, nil
}

func (s *userService) Get(ctx context.Context, accountID string) (*model.UserModel, error) {
	user, err := ketches.Store().UserLister().Get(accountID)
	if err != nil {
		return nil, err
	}

	return &model.UserModel{
		AccountID:         user.Name,
		FullName:          user.Spec.FullName,
		Email:             user.Spec.Email,
		Phone:             user.Spec.Phone,
		MustResetPassword: user.Spec.MustResetPassword,
	}, nil
}

func (s *userService) SignUp(ctx context.Context, user *model.UserSignUpModel) (*model.UserModel, error) {
	if err := s.validateSignUpUser(user); err != nil {
		return nil, err
	}

	_, err := ketches.Store().UserLister().Get(user.AccountID)
	if err == nil {
		return nil, fmt.Errorf("user %s already exists", user.AccountID)
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password hash: %v", err)
	}

	_, err = s.KetchesClient().CoreV1alpha1().Users().Create(ctx, &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: user.AccountID,
		},
		Spec: v1alpha1.UserSpec{
			FullName:          user.FullName,
			Email:             user.Email,
			Phone:             user.Phone,
			PasswordHash:      string(passwordHashBytes),
			MustResetPassword: user.MustResetPassword,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		log.Printf("failed to create user %s: %v\n", user.AccountID, err)
		return nil, fmt.Errorf("failed to create user %s", user.AccountID)
	}

	return &model.UserModel{
		AccountID:         user.AccountID,
		FullName:          user.FullName,
		Email:             user.Email,
		Phone:             user.Phone,
		MustResetPassword: user.MustResetPassword,
	}, nil
}

func (s *userService) SignIn(ctx context.Context, user *model.UserSignInRequest) (*model.UserModel, error) {
	if err := s.validateSignInUser(user); err != nil {
		return nil, err
	}

	got, err := ketches.Store().UserLister().Get(user.AccountID)
	if err != nil {
		return nil, fmt.Errorf("user %s does not exist", user.AccountID)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(got.Spec.PasswordHash), []byte(user.Password)); err != nil {
		return nil, fmt.Errorf("incorrect account id or password")
	}
	return &model.UserModel{
		AccountID:         got.Name,
		FullName:          got.Spec.FullName,
		Email:             got.Spec.Email,
		Phone:             got.Spec.Phone,
		MustResetPassword: got.Spec.MustResetPassword,
	}, nil
}

func (s *userService) Update(ctx context.Context, user *model.UserUpdateRequest) (*model.UserUpdateResponse, error) {
	if user == nil {
		return nil, fmt.Errorf("no user provided")
	}

	if user.AccountID == "" {
		return nil, fmt.Errorf("account id is empty")
	}

	if user.Email == "" {
		return nil, fmt.Errorf("email is empty")
	}

	got, err := ketches.Store().UserLister().Get(user.AccountID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("user %s does not exist", user.AccountID)
		}
		slog.ErrorContext(ctx, "failed to get user "+user.AccountID, "error", err)
		return nil, fmt.Errorf("failed to get user")
	}

	got.Spec.FullName = user.FullName
	got.Spec.Email = user.Email
	got.Spec.Phone = user.Phone

	_, err = s.KetchesClient().CoreV1alpha1().Users().Update(ctx, got, metav1.UpdateOptions{})
	if err != nil {
		slog.Error("failed to update user "+user.AccountID, "error", err)
		return nil, fmt.Errorf("failed to update user %s", user.AccountID)
	}

	return &model.UserUpdateResponse{
		UserModel: model.UserModel{
			AccountID:         got.Name,
			FullName:          got.Spec.FullName,
			Email:             got.Spec.Email,
			Phone:             got.Spec.Phone,
			MustResetPassword: got.Spec.MustResetPassword,
		},
	}, nil
}

func (s *userService) Rename(ctx context.Context, req *model.UserRenameRequest) (*model.UserRenameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("no request provided")
	}

	if req.AccountID == "" {
		return nil, fmt.Errorf("account id is empty")
	}

	if req.NewName == "" {
		return nil, fmt.Errorf("new name is empty")
	}

	if req.Password == "" {
		return nil, fmt.Errorf("password is empty")
	}

	got, err := ketches.Store().UserLister().Get(req.AccountID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("user %s does not exist", req.AccountID)
		}
		slog.ErrorContext(ctx, "failed to get user "+req.AccountID, "error", err)
		return nil, fmt.Errorf("failed to get user")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(got.Spec.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("incorrect account id or password")
	}

	// Create a new user with the new name
	created, err := s.KetchesClient().CoreV1alpha1().Users().Create(ctx, &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.NewName,
			Labels:      got.Labels,
			Annotations: got.Annotations,
		},
		Spec: got.Spec,
	}, metav1.CreateOptions{})
	if err != nil {
		slog.Error("failed to create user "+req.NewName, "error", err)
		return nil, fmt.Errorf("failed to rename user %s", req.AccountID)
	}

	// Update the new user status
	created.Status = got.Status
	_, err = s.KetchesClient().CoreV1alpha1().Users().UpdateStatus(ctx, created, metav1.UpdateOptions{})
	if err != nil {
		slog.Error("failed to update user "+req.NewName+" status", "error", err)
	}

	// Delete the old user
	err = s.KetchesClient().CoreV1alpha1().Users().Delete(ctx, req.AccountID, metav1.DeleteOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete user "+req.AccountID, "error", err)
		return nil, fmt.Errorf("failed to rename user %s", req.AccountID)
	}

	return &model.UserRenameResponse{
		UserModel: model.UserModel{
			AccountID:         req.NewName,
			FullName:          got.Spec.FullName,
			Email:             got.Spec.Email,
			Phone:             got.Spec.Phone,
			MustResetPassword: got.Spec.MustResetPassword,
		},
	}, nil
}

func (s *userService) ResetPassword(ctx context.Context, in *model.UserResetPasswordRequest) (*model.UserModel, error) {
	got, err := ketches.Store().UserLister().Get(in.AccountID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("user %s does not exist", in.AccountID)
		}
		slog.ErrorContext(ctx, "failed to get user "+in.AccountID, "error", err)
		return nil, fmt.Errorf("failed to get user")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(got.Spec.PasswordHash), []byte(in.Password)); err != nil {
		return nil, fmt.Errorf("incorrect account id or password")
	}

	newPasswordHashBytes, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password hash: %v", err)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		newest, err := ketches.Store().UserLister().Get(in.AccountID)
		if err != nil {
			return err
		}

		newest.Spec.PasswordHash = string(newPasswordHashBytes)
		newest.Spec.MustResetPassword = false
		_, err = s.KetchesClient().CoreV1alpha1().Users().Update(ctx, newest, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "failed to reset user "+in.AccountID+" password", "error", err)
		return nil, fmt.Errorf("failed to reset user " + in.AccountID + " password")
	}

	return &model.UserModel{
		AccountID:         got.Name,
		FullName:          got.Spec.FullName,
		Email:             got.Spec.Email,
		Phone:             got.Spec.Phone,
		MustResetPassword: got.Spec.MustResetPassword,
	}, nil
}

func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*model.UserModel, error) {
	ret := &model.UserModel{
		RefreshToken: refreshToken,
	}
	mc, err := jwt.ValidateToken(refreshToken)
	if err != nil {
		return ret, err
	}

	ret.AccountID = mc.AccountID
	ret.Email = mc.Email

	if mc.TokenType != jwt.TokenTypeRefreshToken {
		return ret, fmt.Errorf("not a refresh token: %s", refreshToken)
	}

	accessToken, err := jwt.GenerateToken(jwt.MyClaims{
		AccountID: mc.AccountID,
		Email:     mc.Email,
		TokenType: jwt.TokenTypeAccessToken,
	})

	ret.AccessToken = accessToken
	return ret, err
}

func (s *userService) Delete(ctx context.Context, user *model.DeleteUserRequest) error {
	if err := s.validateDeleteUser(user); err != nil {
		return err
	}

	got, err := ketches.Store().UserLister().Get(user.AccountID)
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("user %s does not exist", user.AccountID)
	}

	// TODO:
	// 1. check if user is platform admin, admin can delete any user
	// 2. check if user is the space owner, owner can delete any user in the space
	// 3. check if user is the user itself, user can delete itself
	if user.AccountID != s.AccountID(ctx) {
		return fmt.Errorf("you can only delete your own account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(got.Spec.PasswordHash), []byte(user.Password)); err != nil {
		return fmt.Errorf("incorrect account id or password")
	}

	err = s.KetchesClient().CoreV1alpha1().Users().Delete(ctx, user.AccountID, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("failed to delete user %s: %v\n", user.AccountID, err)
		return fmt.Errorf("failed to delete user %s", user.AccountID)
	}
	return nil
}

func (s *userService) validateSignUpUser(user *model.UserSignUpModel) error {
	if user == nil {
		return fmt.Errorf("no user provided")
	}

	// Username validation
	if user.AccountID == "" {
		return fmt.Errorf("account id is empty")
	}

	if len(user.AccountID) < 4 || len(user.AccountID) > 32 {
		return fmt.Errorf("account id must be between 4 and 32 characters")
	}

	for _, c := range user.AccountID {
		if !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '-' {
			return s.InvalidName(user.AccountID)
		}
	}

	if user.AccountID[0] == '-' || user.AccountID[len(user.AccountID)-1] == '-' {
		return fmt.Errorf("account id cannot start or end with '-'")
	}

	// Password validation
	if user.Password == "" {
		return fmt.Errorf("password is empty")
	}

	if len(user.Password) < 8 || len(user.Password) > 32 {
		return fmt.Errorf("password must be between 8 and 32 characters")
	}

	var (
		hasLowercase = false
		hasUppercase = false
		hasNumber    = false
		hasSpecial   = false
	)
	for _, c := range user.Password {
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
		return fmt.Errorf("password must contain at least one lowercase letter, one uppercase letter, one number, and one special character")
	}

	// Email and phone validation
	if !emailRgx.MatchString(user.Email) {
		return s.invalidEmail(user.Email)
	}
	if len(user.Phone) > 0 && !phoneRgx.MatchString(user.Phone) {
		return s.invalidPhoneNumber(user.Phone)
	}

	return nil
}

func (s *userService) validateSignInUser(user *model.UserSignInRequest) error {
	if user.AccountID == "" {
		return fmt.Errorf("account id is empty")
	}

	if user.Password == "" {
		return fmt.Errorf("password is empty")
	}

	return nil
}

func (s *userService) validateDeleteUser(user *model.DeleteUserRequest) error {
	if user.AccountID == "" {
		return fmt.Errorf("account id is empty")
	}

	if user.Password == "" {
		return fmt.Errorf("password is empty")
	}

	return nil
}

func (s *userService) invalidEmail(email string) error {
	return fmt.Errorf("invalid email: %s", email)
}

func (s *userService) invalidPhoneNumber(phone string) error {
	return fmt.Errorf("invalid phone number: %s", phone)
}
