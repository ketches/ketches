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
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entity"
	"github.com/ketches/ketches/pkg/kube/incluster"

	"k8s.io/client-go/kubernetes"
)

// Service is an interface for all services
type Service interface {
	KubeClient() kubernetes.Interface
	InvalidParams() app.Error
	InvalidName(name string) app.Error
	CheckProjectPermissions(ctx context.Context, projectID string) (projectRole string, err app.Error)
}

var serviceInstance Service

type service struct {
	inclusterKubeClient kubernetes.Interface
}

func LoadService() Service {
	if serviceInstance == nil {
		serviceInstance = &service{
			inclusterKubeClient: incluster.Client(),
		}
	}

	return serviceInstance
}

func (s *service) KubeClient() kubernetes.Interface {
	return s.inclusterKubeClient
}

func (s *service) InvalidParams() app.Error {
	return app.NewError(http.StatusBadRequest, "invalid params")
}

func (s *service) InvalidName(name string) app.Error {
	return app.NewError(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", name))
}

func (s *service) CheckProjectPermissions(ctx context.Context, projectID string) (string, app.Error) {
	if api.IsAdmin(ctx) {
		return app.ProjectRoleOwner, nil
	}

	projectMember := &entity.ProjectMember{}
	if err := db.Instance().Model(&entity.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, api.UserID(ctx)).First(projectMember).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return "", app.ErrPermissionDenied
		}
		return "", app.ErrDatabaseOperationFailed
	}
	return projectMember.ProjectRole, nil
}
