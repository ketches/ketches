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
	"unicode"

	"github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/pkg/ketches"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

type SpaceService interface {
	List(ctx context.Context) (*model.ListSpacesResponse, error)
	Get(ctx context.Context, space string) (*model.GetSpaceResponse, error)
	Create(ctx context.Context, req *model.CreateSpaceRequest) (*model.CreateSpaceResponse, error)
	Update(ctx context.Context, req *model.UpdateSpaceRequest) (*model.UpdateSpaceResponse, error)
	AddMembers(ctx context.Context, req *model.AddMembersRequest) error
	ListMembers(ctx context.Context, space string) (*model.ListSpaceMembersResponse, error)
	RemoveMembers(ctx context.Context, req *model.RemoveSpacesRequest) error
	Delete(ctx context.Context, space string) error
}

type spaceService struct {
	Service
}

func NewSpaceService() SpaceService {
	return &spaceService{
		Service: LoadService(),
	}
}

func (s *spaceService) List(ctx context.Context) (*model.ListSpacesResponse, error) {
	user, err := ketches.Store().UserLister().Get(s.AccountID(ctx))
	if err != nil {
		return nil, err
	}

	listed, err := ketches.Store().SpaceLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}

	result := &model.ListSpacesResponse{
		Spaces: make([]model.SpaceModel, len(listed)),
	}

	for _, space := range listed {
		if _, ok := space.Spec.Members[user.Name]; !ok && !s.AdminAccount(ctx) {
			continue
		}
		result.Spaces = append(result.Spaces, model.SpaceModel{
			Name:        space.Name,
			DisplayName: space.Spec.DisplayName,
		})
	}
	return result, nil
}

func (s *spaceService) Get(ctx context.Context, space string) (*model.GetSpaceResponse, error) {
	got, err := ketches.Store().SpaceLister().Get(space)
	if err != nil {
		return nil, err
	}
	result := &model.GetSpaceResponse{
		SpaceModel: model.SpaceModel{
			Name:        got.Name,
			DisplayName: got.Spec.DisplayName,
		},
	}

	result.Members = readMembers(got)
	return result, nil
}

func (s *spaceService) Create(ctx context.Context, req *model.CreateSpaceRequest) (*model.CreateSpaceResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("space name is required")
	}

	for _, r := range req.Name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			continue
		}
		return nil, fmt.Errorf("space name must be alphanumeric or '-'")
	}

	if req.Name[0] == '-' || req.Name[len(req.Name)-1] == '-' {
		return nil, fmt.Errorf("space name must not start or end with '-'")
	}

	if req.DisplayName == "" {
		req.DisplayName = req.Name
	}

	_, err := ketches.Store().SpaceLister().Get(req.Name)
	if err != nil && errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("space name %s already exists", req.Name)
	}

	space := &v1alpha1.Space{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
		Spec: v1alpha1.SpaceSpec{
			DisplayName:   req.DisplayName,
			LimitRange:    &v1alpha1.LimitRange{},
			ResourceQuota: &corev1.ResourceRequirements{},
			Members: map[string]v1alpha1.SpaceMemberRoles{
				s.AccountID(ctx): {v1alpha1.SpaceMemberRoleOwner},
			},
		},
	}
	_, err = s.KetchesClient().CoreV1alpha1().Spaces().Create(ctx, space, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	result := &model.CreateSpaceResponse{
		SpaceModel: model.SpaceModel{
			Name:        space.Name,
			DisplayName: space.Spec.DisplayName,
		},
	}
	result.Members = readMembers(space)
	return result, nil
}

func (s *spaceService) Update(ctx context.Context, req *model.UpdateSpaceRequest) (*model.UpdateSpaceResponse, error) {
	got, err := ketches.Store().SpaceLister().Get(req.SpaceID)
	if err != nil {
		return nil, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		newest, err := ketches.Store().SpaceLister().Get(req.SpaceID)
		if err != nil {
			return err
		}
		newest.Spec.DisplayName = req.DisplayName
		_, err = s.KetchesClient().CoreV1alpha1().Spaces().Update(ctx, got, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	result := &model.UpdateSpaceResponse{
		SpaceModel: model.SpaceModel{
			Name:        got.Name,
			DisplayName: req.DisplayName,
		},
	}
	result.Members = readMembers(got)
	return result, nil
}

func (s *spaceService) AddMembers(ctx context.Context, req *model.AddMembersRequest) error {
	got, err := ketches.Store().SpaceLister().Get(req.SpaceID)
	if err != nil {
		return err
	}

	members := got.Spec.Members

	if members == nil {
		members = map[string]v1alpha1.SpaceMemberRoles{}
	}

	for _, user := range req.Members {
		if _, ok := members[user.AccountID]; !ok {
			members[user.AccountID] = v1alpha1.StringSliceToSpaceMemberRoles(user.Roles)
		}
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		newest, err := ketches.Store().SpaceLister().Get(req.SpaceID)
		if err != nil {
			return err
		}
		newest.Spec.Members = members
		_, err = s.KetchesClient().CoreV1alpha1().Spaces().Update(ctx, got, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func (s *spaceService) ListMembers(ctx context.Context, space string) (*model.ListSpaceMembersResponse, error) {
	got, err := ketches.Store().SpaceLister().Get(space)
	if err != nil {
		return nil, err
	}
	result := &model.ListSpaceMembersResponse{
		Members: make([]model.SpaceMemberDetailModel, len(got.Spec.Members)),
	}
	for accountID, roles := range got.Spec.Members {
		user, err := ketches.Store().UserLister().Get(accountID)
		if err != nil {
			continue
		}
		result.Members = append(result.Members, model.SpaceMemberDetailModel{
			AccountID: accountID,
			FullName:  user.Spec.FullName,
			Email:     user.Spec.Email,
			Roles:     roles.StringSlice(),
		})
	}
	return result, nil
}

func (s *spaceService) RemoveMembers(ctx context.Context, req *model.RemoveSpacesRequest) error {
	got, err := ketches.Store().SpaceLister().Get(req.SpaceID)
	if err != nil {
		return err
	}

	members := got.Spec.Members
	for _, user := range req.Members {
		delete(members, user)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		newest, err := ketches.Store().SpaceLister().Get(req.SpaceID)
		if err != nil {
			return err
		}
		newest.Spec.Members = members
		_, err = s.KetchesClient().CoreV1alpha1().Spaces().Update(ctx, got, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func (s *spaceService) Delete(ctx context.Context, space string) error {
	got, err := ketches.Store().SpaceLister().Get(space)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if got.Status.ApplicationCount > 0 {
		return fmt.Errorf("space %s is still owning %d applications", space, got.Status.ApplicationCount)
	}
	return s.KetchesClient().CoreV1alpha1().Spaces().Delete(ctx, space, metav1.DeleteOptions{})
}

func readMembers(space *v1alpha1.Space) []model.SpaceMemberDetailModel {
	result := make([]model.SpaceMemberDetailModel, len(space.Spec.Members))
	for accountID, roles := range space.Spec.Members {
		user, err := ketches.Store().UserLister().Get(accountID)
		if err != nil {
			continue
		}
		result = append(result, model.SpaceMemberDetailModel{
			AccountID: accountID,
			FullName:  user.Spec.FullName,
			Email:     user.Spec.Email,
			Roles:     roles.StringSlice(),
		})
	}
	return result
}
