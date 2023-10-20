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

package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/http"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/internal/service"
)

func ListApplications(c *gin.Context) {
	spaceID := c.Param("space_id")

	var af model.ApplicationFilter
	if err := c.ShouldBindQuery(&af); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	af.SpaceID = spaceID

	applicationService := service.NewApplicationService()
	resp, err := applicationService.List(c, &af)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func GetApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	appID := c.Param("application_id")
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Get(c, spaceID, appID)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func CreateApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	var app model.CreateApplicationRequest
	if err := c.ShouldBindJSON(&app); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	app.SpaceID = spaceID
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Create(c, &app)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func StartApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	appID := c.Param("application_id")
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Start(c, spaceID, appID)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func StopApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	appID := c.Param("application_id")
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Stop(c, spaceID, appID)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func RestartApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	appID := c.Param("application_id")
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Restart(c, spaceID, appID)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func GetApplicationPodsAndContainers(c *gin.Context) {
	spaceID := c.Param("space_id")
	appID := c.Param("application_id")
	var req model.GetPodsAndContainersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.SpaceID = spaceID
	req.ApplicationID = appID
	applicationService := service.NewApplicationService()
	resp, err := applicationService.GetPodsAndContainers(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func GetApplicationContainerLogs(c *gin.Context) {
	spaceID := c.Param("space_id")
	appID := c.Param("application_id")
	var req model.GetApplicationContainerLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.SpaceID = spaceID
	req.ApplicationID = appID
	applicationService := service.NewApplicationService()
	resp, err := applicationService.GetContainerLogs(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func DeleteApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	appID := c.Param("application_id")
	applicationService := service.NewApplicationService()
	err := applicationService.Delete(c, spaceID, appID)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, nil)
}

func ExportApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	var req model.ExportApplicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.SpaceID = spaceID
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Export(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func ImportApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	var req model.ImportApplicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.SpaceID = spaceID
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Import(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func BackupApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	var req model.BackupApplicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.SpaceID = spaceID
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Backup(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}

func RestoreApplication(c *gin.Context) {
	spaceID := c.Param("space_id")
	var req model.RestoreApplicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.SpaceID = spaceID
	applicationService := service.NewApplicationService()
	resp, err := applicationService.Restore(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}
	http.Success(c, resp)
}
