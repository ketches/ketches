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
	"fmt"
	"log/slog"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/pkg/kube/incluster"
	"github.com/ketches/ketches/pkg/kube/workercluster"
)

func getSpace(space string) (*corev1alpha1.Space, error) {
	result, err := incluster.Store().SpaceLister().Get(space)
	if err != nil {
		slog.Error("failed to get space", "space", space, "error", err)
		return nil, fmt.Errorf("failed to get space %s", space)
	}

	return result, nil
}

func getWorkerCluster(cluster string) (workercluster.Cluster, error) {
	result, ok := incluster.Store().Clusterset().Cluster(cluster)
	if !ok {
		slog.Error("failed to get cluster", "cluster", cluster)
		return nil, fmt.Errorf("failed to get cluster %s", cluster)
	}

	return result, nil
}

func getWorkerClusterBySpace(space string) (workercluster.Cluster, error) {
	s, err := getSpace(space)
	if err != nil {
		return nil, err
	}

	return getWorkerCluster(s.Spec.Cluster)
}
