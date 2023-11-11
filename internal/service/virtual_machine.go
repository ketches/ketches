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

	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/pkg/global"
	"github.com/ketches/ketches/pkg/ketches"
	"github.com/ketches/ketches/util/conv"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (c *virtualMachineService) Create(ctx context.Context, req *model.CreateVirtualMachineRequest) (*model.CreateVirtualMachineResponse, error) {
	space, err := ketches.Store().SpaceLister().Get(req.SpaceUri)
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

	cluster, ok := ketches.Store().Clusterset().Cluster(space.Spec.Cluster)
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", space.Spec.Cluster)
	}

	vm := &kubevirtcorev1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: space.Name,
		},
		Spec: kubevirtcorev1.VirtualMachineSpec{
			Running: conv.Ptr(true),
			Template: &kubevirtcorev1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						global.OwnedResourceLabelKey: global.LabelTrueValue,
						"kubevirt.io/size":           "small",
						"kubevirt.io/domain":         req.Name,
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
	err = cluster.KubevirtRuntimeClient().Get(ctx, client.ObjectKeyFromObject(vm), &kubevirtcorev1.VirtualMachine{})
	if err != nil {
		if errors.IsNotFound(err) {
			err = cluster.KubevirtRuntimeClient().Create(ctx, vm)
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
				cluster.KubeRuntimeClient().Create(ctx, svc)
			}
			return &model.CreateVirtualMachineResponse{Name: vm.Name, Status: string(vm.Status.PrintableStatus)}, nil
		}
		return nil, err
	}
	return nil, fmt.Errorf("virtual machine %s already exists", vm.Name)
}
