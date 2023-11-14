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

	"slices"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/pkg/global"
	"github.com/ketches/ketches/util/conv"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

var (
	vmGKR  = kubevirtcorev1.SchemeGroupVersion.WithResource("virtualmachines")
	vmiGKR = kubevirtcorev1.SchemeGroupVersion.WithResource("virtualmachineinstances")
)

type VirtualMachineService interface {
	Create(ctx context.Context, req *model.CreateVirtualMachineRequest) (*model.CreateVirtualMachineResponse, error)
}

type virtualMachineService struct {
	Service
}

func NewVirtualMachineService() VirtualMachineService {
	return &virtualMachineService{
		Service: LoadService(),
	}
}

var _ VirtualMachineService = (*virtualMachineService)(nil)

func (s *virtualMachineService) Create(ctx context.Context, req *model.CreateVirtualMachineRequest) (*model.CreateVirtualMachineResponse, error) {
	space, err := s.InClusterStore().SpaceLister().Get(req.SpaceUri)
	if err != nil {
		return nil, err
	}

	if len(req.Ports) == 0 || !slices.ContainsFunc(req.Ports, func(port model.Port) bool {
		if port.Number == 22 {
			return true
		}
		return false
	}) {
		req.Ports = append(req.Ports, model.Port{Number: 22})
	}

	workerCluster, ok := s.InClusterStore().Clusterset().Cluster(space.Spec.Cluster)
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", space.Spec.Cluster)
	}

	got, err := workerCluster.DynamicClient().Resource(vmGKR).Get(ctx, req.Name, metav1.GetOptions{})
	if got != nil {
		return nil, fmt.Errorf("virtual machine %s already exists", req.Name)
	}

	vm := &kubevirtcorev1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: kubevirtcorev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: space.Name,
			Labels: map[string]string{
				global.OwnedResourceLabelKey: global.LabelTrueValue,
				corev1alpha1.SpaceLabelKey:   space.Name,
				"ketches.io/virtualmachine":  req.Name,
			},
		},
		Spec: kubevirtcorev1.VirtualMachineSpec{
			Running: conv.Ptr(true),
			Template: &kubevirtcorev1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						global.OwnedResourceLabelKey: global.LabelTrueValue,
						"ketches.io/virtualmachine":  req.Name,
					},
				},
				Spec: kubevirtcorev1.VirtualMachineInstanceSpec{
					Domain: kubevirtcorev1.DomainSpec{
						Devices: kubevirtcorev1.Devices{
							Disks: []kubevirtcorev1.Disk{
								{
									Name: "containerdisk",
									DiskDevice: kubevirtcorev1.DiskDevice{
										Disk: &kubevirtcorev1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{Name: "cloudinitdisk",
									DiskDevice: kubevirtcorev1.DiskDevice{
										Disk: &kubevirtcorev1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
							},
							Interfaces: []kubevirtcorev1.Interface{
								{
									Name: "default",
									InterfaceBindingMethod: kubevirtcorev1.InterfaceBindingMethod{
										Masquerade: &kubevirtcorev1.InterfaceMasquerade{},
									},
								},
							},
						},
						Resources: kubevirtcorev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"memory": resource.MustParse("64M"),
							},
						},
					},
					Networks: []kubevirtcorev1.Network{
						{
							Name: "default",
							NetworkSource: kubevirtcorev1.NetworkSource{
								Pod: &kubevirtcorev1.PodNetwork{},
							},
						},
					},
					Volumes: []kubevirtcorev1.Volume{
						{
							Name: "containerdisk",
							VolumeSource: kubevirtcorev1.VolumeSource{
								ContainerDisk: &kubevirtcorev1.ContainerDiskSource{
									Image: req.ContainerDiskImage,
								},
							},
						}, {
							Name: "cloudinitdisk",
							VolumeSource: kubevirtcorev1.VolumeSource{
								CloudInitNoCloud: &kubevirtcorev1.CloudInitNoCloudSource{
									UserData: "Hello Kubevirt!",
								},
							},
						},
					},
				},
			},
		},
	}

	err = workerCluster.KubeRuntimeClient().Create(ctx, vm)
	if err != nil {
		return nil, err
	}
	// TODO: construct services or gateways
	for _, port := range req.Ports {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", req.Name, port.Number),
				Namespace: space.Name,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:       fmt.Sprintf("%d", port.Number),
						Protocol:   corev1.ProtocolTCP,
						Port:       port.Number,
						TargetPort: intstr.FromInt32(port.Number),
					},
				},
			},
		}
		err := workerCluster.KubeRuntimeClient().Create(ctx, svc)
		if err != nil {
			return nil, err
		} else {
			// TODO: construct gateway

		}
	}
	return &model.CreateVirtualMachineResponse{
		Name:   vm.Name,
		Status: string(vm.Status.PrintableStatus),
	}, nil
}
