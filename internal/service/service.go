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

	"github.com/ketches/ketches/pkg/generated/clientset/versioned"
	"github.com/ketches/ketches/pkg/kube/incluster"

	"github.com/ketches/ketches/pkg/global"
	"k8s.io/client-go/kubernetes"
)

// Service is an interface for all services
type Service interface {
	InClusterStore() incluster.StoreInterface
	KubeClient() kubernetes.Interface
	KetchesClient() versioned.Interface
	AccountID(ctx context.Context) string
	AdminAccount(ctx context.Context) bool
	InvalidParams() error
	InvalidName(name string) error
}

var serviceInstance Service

type service struct {
	inclusterKubeClient    kubernetes.Interface
	inclusterKetchesClient versioned.Interface
}

func LoadService() Service {
	if serviceInstance == nil {
		serviceInstance = &service{
			inclusterKubeClient:    incluster.Client(),
			inclusterKetchesClient: incluster.KetchesClient(),
		}
	}

	return serviceInstance
}

func (s *service) InClusterStore() incluster.StoreInterface {
	return incluster.Store()
}

func (s *service) KubeClient() kubernetes.Interface {
	return s.inclusterKubeClient
}

func (s *service) KetchesClient() versioned.Interface {
	return s.inclusterKetchesClient
}

func (s *service) AccountID(ctx context.Context) string {
	return ctx.Value(global.ContextKeyAccountID).(string)
}

func (s *service) AdminAccount(ctx context.Context) bool {
	return s.AccountID(ctx) == "admin" && ctx.Value(global.ContextKeySignInRole).(string) == "admin"
}

func (s *service) InvalidParams() error {
	return fmt.Errorf("invalid params")
}

func (s *service) InvalidName(name string) error {
	return fmt.Errorf("invalid name: %s", name)
}
