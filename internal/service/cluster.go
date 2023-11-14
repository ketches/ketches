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

	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/pkg/kube/workercluster"
)

type ClusterService interface {
	List(ctx context.Context) ([]*model.ClusterModel, error)
	Get(ctx context.Context, name string) (*model.ClusterModel, error)
	Add(ctx context.Context, req *model.AddClusterRequest) (*model.AddClusterResponse, error)
	Update(ctx context.Context, req *model.UpdateClusterRequest) (*model.UpdateClusterResponse, error)
	Remove(ctx context.Context, name string) error
}

type clusterService struct {
	Service
}

func NewClusterService() ClusterService {
	return &clusterService{
		Service: LoadService(),
	}
}

var _ ClusterService = (*clusterService)(nil)

func (s *clusterService) List(ctx context.Context) ([]*model.ClusterModel, error) {
	var result []*model.ClusterModel
	cs := s.InClusterStore().Clusterset().List(workercluster.ListOptions{})
	for _, c := range cs {
		result = append(result, &model.ClusterModel{
			Name:        c.Name(),
			Description: c.Description(),
			KubeConfig:  c.KubeConfig(),
			CreatedAt:   c.Resource().CreationTimestamp.Time,
		})
	}
	return result, nil
}

func (s *clusterService) Get(ctx context.Context, name string) (*model.ClusterModel, error) {
	// TODO implement me
	panic("implement me")
}

func (s *clusterService) Add(ctx context.Context, req *model.AddClusterRequest) (*model.AddClusterResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (s *clusterService) Update(ctx context.Context, req *model.UpdateClusterRequest) (*model.UpdateClusterResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (s *clusterService) Remove(ctx context.Context, name string) error {
	// TODO implement me
	panic("implement me")
}
