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
	"github.com/ketches/ketches/api/spec"
	"log"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/ketches/ketches/api/core/v1alpha1"
	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/pkg/ketches"
	"github.com/ketches/ketches/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	k8sstrings "k8s.io/utils/strings"
	"sigs.k8s.io/yaml"
)

type ApplicationService interface {
	List(ctx context.Context, af *model.ApplicationFilter) ([]*model.ApplicationModel, error)
	Get(ctx context.Context, spaceID string, appID string) (*model.ApplicationModel, error)
	Create(ctx context.Context, req *model.CreateApplicationRequest) (*model.CreateApplicationResponse, error)
	Start(ctx context.Context, spaceID string, appID string) (*model.ApplicationModel, error)
	Stop(ctx context.Context, spaceID string, appID string) (*model.ApplicationModel, error)
	Restart(ctx context.Context, spaceID string, appID string) (*model.ApplicationModel, error)
	GetPodsAndContainers(ctx context.Context, req *model.GetPodsAndContainersRequest) (*model.GetPodsAndContainersResponse, error)
	GetContainerLogs(ctx context.Context, req *model.GetApplicationContainerLogsRequest) (*model.GetApplicationContainerLogsResponse, error)
	Delete(ctx context.Context, spaceID string, appID string) error
	Export(ctx context.Context, req *model.ExportApplicationsRequest) (*model.ExportApplicationsResponse, error)
	Import(ctx context.Context, req *model.ImportApplicationsRequest) (*model.ImportApplicationsResponse, error)
	Backup(ctx context.Context, req *model.BackupApplicationRequest) (*model.BackupApplicationResponse, error)
	ListBackups(ctx context.Context, req *model.ListApplicationBackupsRequest) ([]*model.ListApplicationBackupsResponse, error)
	CreateBackupSchedule(ctx context.Context, req *model.CreateApplicationBackupScheduleRequest) (*model.CreateApplicationBackupScheduleResponse, error)
	Restore(ctx context.Context, req *model.RestoreApplicationsRequest) (*model.RestoreApplicationsResponse, error)
	ListRestores(ctx context.Context, spaceID string, appID string) ([]*model.BackupApplicationResponse, error)
}

type applicationService struct {
	Service
}

func NewApplicationService() ApplicationService {
	return &applicationService{
		Service: LoadService(),
	}
}

var _ ApplicationService = (*applicationService)(nil)

func (s *applicationService) List(ctx context.Context, af *model.ApplicationFilter) ([]*model.ApplicationModel, error) {
	listed, err := ketches.Store().ApplicationLister().Applications(af.SpaceID).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	listed, _ = model.PagedResult(listed, af)

	var ret []*model.ApplicationModel
	for _, app := range listed {
		ret = append(ret, &model.ApplicationModel{
			Name:        app.Name,
			DisplayName: app.Spec.DisplayName,
			// Description: app.Spec.Description,
			Image:    app.Spec.Image,
			Replicas: app.Spec.Replicas,
			Status:   app.Status.Phase.String(),
		})
	}

	return ret, nil
}

func (s *applicationService) Get(ctx context.Context, spaceID, appID string) (*model.ApplicationModel, error) {
	app, err := ketches.Store().ApplicationLister().Applications(spaceID).Get(appID)
	if err != nil {
		return nil, err
	}

	return &model.ApplicationModel{
		Name:        app.Name,
		DisplayName: app.Spec.DisplayName,
		Description: app.Spec.Description,
		Image:       app.Spec.Image,
		Replicas:    app.Spec.Replicas,
		Status:      app.Status.Phase.String(),
	}, nil
}

func (s *applicationService) Create(ctx context.Context, req *model.CreateApplicationRequest) (*model.CreateApplicationResponse, error) {
	if err := s.validateCreateApplication(req); err != nil {
		return nil, err
	}

	got, err := ketches.Store().ApplicationLister().Applications(req.SpaceID).Get(req.Name)
	if err == nil {
		return nil, fmt.Errorf("application %s already exists", got.Name)
	}

	var env []corev1.EnvVar
	for _, e := range req.EnvVars {
		env = append(env, corev1.EnvVar{
			Name:  e.Key,
			Value: e.Value,
		})
	}

	var ports []*v1alpha1.Port
	for _, p := range req.Ports {
		port := &v1alpha1.Port{
			Number: p.Number,
			Target: p.Number,
		}
		for _, g := range p.Gateways {
			// TODO: port advanced options
			port.Gateways = append(port.Gateways, v1alpha1.Gateway{
				Name:     fmt.Sprintf("%s-%d-%s", strings.ToLower(g.Type), p.Number, k8sstrings.ShortenString(uuid.NewString(), 5)),
				Type:     v1alpha1.GatewayType(g.Type),
				NodePort: g.NodePort,
				Host:     g.Host,
				Path:     g.Path,
			})
		}
		ports = append(ports, port)
	}

	newApp := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.SpaceID,
			Labels: map[string]string{
				corev1alpha1.ApplicationEditionLabelKey: corev1alpha1.NewApplicationEditionLabelValue(),
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Type: v1alpha1.WorkloadType(req.Type),
			ViewSpec: spec.ViewSpec{
				DisplayName: req.DisplayName,
				Description: req.Description,
			},
			DesiredState:     v1alpha1.DesiredStateRunning,
			Image:            req.Image,
			Replicas:         req.Replicas,
			Command:          strings.Split(req.Command, " "),
			Args:             strings.Split(req.Command, " "),
			Env:              env,
			Ports:            ports,
			Resources:        corev1.ResourceRequirements{},
			Healthz:          nil,
			Autoscaler:       nil,
			Sidecars:         nil,
			MountFiles:       nil,
			MountDirectories: nil,
			Privileged:       false,
		},
	}
	created, err := s.KetchesClient().CoreV1alpha1().Applications(req.SpaceID).Create(ctx, newApp, metav1.CreateOptions{})
	if err != nil {
		log.Printf("failed to create application %s/%s: %v", req.SpaceID, req.Name, err)
		return nil, fmt.Errorf("failed to create application %s/%s", req.SpaceID, req.Name)
	}
	return &model.CreateApplicationResponse{
		ApplicationModel: model.ApplicationModel{
			Name:        req.Name,
			Type:        req.Type,
			DisplayName: req.DisplayName,
			Description: req.Description,
			Image:       req.Image,
			Replicas:    req.Replicas,
			Status:      created.Status.Phase.String(),
		},
	}, nil
}

func (s *applicationService) validateCreateApplication(am *model.CreateApplicationRequest) error {
	if am == nil {
		return fmt.Errorf("no application provided")
	}
	if am.Name == "" {
		return fmt.Errorf("application name is empty")
	}

	if len(am.Name) > 32 {
		return fmt.Errorf("application name cannot be longer than 32 characters")
	}

	for _, r := range am.Name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			continue
		}
		return fmt.Errorf("application name can only contain letters, digits, and '-'")
	}

	if am.Name[0] == '-' || am.Name[len(am.Name)-1] == '-' {
		return fmt.Errorf("application name cannot start or end with '-'")
	}

	return nil
}

func (s *applicationService) Start(ctx context.Context, spaceID, appID string) (*model.ApplicationModel, error) {
	var app *v1alpha1.Application
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		got, err := ketches.Store().ApplicationLister().Applications(spaceID).Get(appID)
		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("application %s/%s not found", spaceID, appID)
			}
			log.Printf("failed to get application %s/%s: %v", spaceID, appID, err)
			return fmt.Errorf("failed to get application %s/%s", spaceID, appID)
		}
		got.Spec.DesiredState = v1alpha1.DesiredStateRunning
		app, err = s.KetchesClient().CoreV1alpha1().Applications(spaceID).Update(ctx, got, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("failed to start application %s/%s: %v", spaceID, appID, err)
			return fmt.Errorf("failed to start application %s/%s", spaceID, appID)
		}
		return nil
	})

	var ret *model.ApplicationModel
	if app != nil {
		ret = &model.ApplicationModel{
			Name:        app.Name,
			DisplayName: app.Spec.DisplayName,
			Description: app.Spec.Description,
			Image:       app.Spec.Image,
			Replicas:    app.Spec.Replicas,
			Status:      app.Status.Phase.String(),
		}
	}
	return ret, err
}

func (s *applicationService) Stop(ctx context.Context, spaceID, appID string) (*model.ApplicationModel, error) {
	var newApp *v1alpha1.Application
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		app, err := ketches.Store().ApplicationLister().Applications(spaceID).Get(appID)
		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("application %s/%s not found", spaceID, appID)
			}
			log.Printf("failed to get application %s/%s: %v", spaceID, appID, err)
			return fmt.Errorf("failed to get application %s/%s", spaceID, appID)
		}
		app.Spec.DesiredState = v1alpha1.DesiredStateStopped
		newApp, err = s.KetchesClient().CoreV1alpha1().Applications(spaceID).Update(ctx, app, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("failed to stop application %s/%s: %v", spaceID, appID, err)
			return fmt.Errorf("failed to stop application %s/%s", spaceID, appID)
		}
		return nil
	})

	var ret *model.ApplicationModel
	if newApp != nil {
		ret = &model.ApplicationModel{
			Name:        newApp.Name,
			DisplayName: newApp.Spec.DisplayName,
			Description: newApp.Spec.Description,
			Image:       newApp.Spec.Image,
			Replicas:    newApp.Spec.Replicas,
			Status:      newApp.Status.Phase.String(),
		}
	}
	return ret, err
}

func (s *applicationService) Restart(ctx context.Context, spaceID, appID string) (*model.ApplicationModel, error) {
	var app *v1alpha1.Application
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		got, err := ketches.Store().ApplicationLister().Applications(spaceID).Get(appID)
		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("application %s/%s not found", spaceID, appID)
			}
			log.Printf("failed to get application %s/%s: %v", spaceID, appID, err)
			return fmt.Errorf("failed to get application %s/%s", spaceID, appID)
		}
		got.Labels[corev1alpha1.ApplicationEditionLabelKey] = corev1alpha1.NewApplicationEditionLabelValue()
		app, err = s.KetchesClient().CoreV1alpha1().Applications(spaceID).Update(ctx, got, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("failed to restart application %s/%s: %v", spaceID, appID, err)
			return fmt.Errorf("failed to restart application %s/%s", spaceID, appID)
		}
		return nil
	})

	var ret *model.ApplicationModel
	if app != nil {
		ret = &model.ApplicationModel{
			Name:        app.Name,
			DisplayName: app.Spec.DisplayName,
			Description: app.Spec.Description,
			Image:       app.Spec.Image,
			Replicas:    app.Spec.Replicas,
			Status:      app.Status.Phase.String(),
		}
	}
	return ret, err
}

func (s *applicationService) GetPodsAndContainers(ctx context.Context, req *model.GetPodsAndContainersRequest) (*model.GetPodsAndContainersResponse, error) {
	selector := labels.SelectorFromSet(labels.Set{corev1alpha1.ApplicationLabelKey: req.ApplicationID})
	pods, err := kube.Store().PodLister().Pods(req.SpaceID).List(selector)
	if err != nil {
		return nil, err
	}

	var result = new(model.GetPodsAndContainersResponse)
	for _, pod := range pods {
		containers := make([]model.Container, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))
		for i, c := range pod.Spec.Containers {
			containers[i] = model.Container{
				Type: "main",
				Name: c.Name,
			}
		}

		for i, c := range pod.Spec.InitContainers {
			containers[i+len(pod.Spec.Containers)] = model.Container{
				Type: "init",
				Name: c.Name,
			}
		}

		result.Pods = append(result.Pods, model.PodContainers{
			Name:       pod.Name,
			Containers: containers,
		})
	}
	return result, err
}

func (s *applicationService) GetContainerLogs(ctx context.Context, req *model.GetApplicationContainerLogsRequest) (*model.GetApplicationContainerLogsResponse, error) {
	selector := labels.SelectorFromSet(labels.Set{corev1alpha1.ApplicationLabelKey: req.ApplicationID})
	pods, err := kube.Store().PodLister().Pods(req.SpaceID).List(selector)
	if err != nil {
		return nil, err
	}

	result := new(model.GetApplicationContainerLogsResponse)
	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			if c.Name == req.Container {
				result.Body, err = kube.GetContainerLogs(ctx, req.SpaceID, pod.Name, req.Container, kube.GetContainerOptions{
					Follow:    req.Follow,
					TailLines: &req.TailLines,
					Previous:  req.Previous,
				})
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}

func (s *applicationService) Delete(ctx context.Context, spaceID, appID string) error {
	return s.KetchesClient().CoreV1alpha1().Applications(spaceID).Delete(ctx, appID, metav1.DeleteOptions{})
}

func (s *applicationService) Export(ctx context.Context, req *model.ExportApplicationsRequest) (*model.ExportApplicationsResponse, error) {
	ls := new(v1alpha1.ApplicationList)

	for _, app := range req.Applications {
		got, err := ketches.Store().ApplicationLister().Applications(req.SpaceID).Get(app)
		if err != nil {
			return nil, err
		}
		ls.Items = append(ls.Items, *got)
	}

	body, err := yaml.Marshal(ls)
	if err != nil {
		return nil, err
	}
	return &model.ExportApplicationsResponse{
		Body: body,
	}, nil
}

func (s *applicationService) Import(ctx context.Context, req *model.ImportApplicationsRequest) (*model.ImportApplicationsResponse, error) {
	result := new(model.ImportApplicationsResponse)
	var ls v1alpha1.ApplicationList
	err := yaml.Unmarshal(req.Body, &ls)
	if err != nil {
		result.Message = "Parse imported data failed"
		return result, err
	}
	result.Message = "Import applications successfully"
	return result, nil
}

// TODO: Backup & Restore with Velero and MinIO 0
func (s *applicationService) Backup(ctx context.Context, req *model.BackupApplicationRequest) (*model.BackupApplicationResponse, error) {
	return nil, nil
}

func (s applicationService) DeleteBackup(ctx context.Context, app string) error {
	return nil
}

func (s *applicationService) ListBackups(ctx context.Context, req *model.ListApplicationBackupsRequest) ([]*model.ListApplicationBackupsResponse, error) {
	return nil, nil
}

func (s *applicationService) CreateBackupSchedule(ctx context.Context, req *model.CreateApplicationBackupScheduleRequest) (*model.CreateApplicationBackupScheduleResponse, error) {
	return nil, nil
}

func (s *applicationService) Restore(ctx context.Context, req *model.RestoreApplicationsRequest) (*model.RestoreApplicationsResponse, error) {
	return nil, nil
}

func (s *applicationService) DeleteRestoreRecord(ctx context.Context, app string) error {
	return nil
}

func (s *applicationService) ListRestores(ctx context.Context, spaceID, appID string) ([]*model.BackupApplicationResponse, error) {
	return nil, nil
}

// TODO: Backup & Restore with Velero and MinIO 1
