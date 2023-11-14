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

package model

type VirtualMachine struct {
}

type CreateVirtualMachineRequest struct {
	SpaceUri           string `json:",inline"`
	Name               string `json:"name" binding:"required"`
	ContainerDiskImage string `json:"container_disk_image" binding:"required"`
	Ports              []Port `json:"ports"`
}

type CreateVirtualMachineResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
