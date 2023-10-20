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

func ListSpaces(c *gin.Context) {
	spaceService := service.NewSpaceService()

	resp, err := spaceService.List(c)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, resp)
}

func GetSpace(c *gin.Context) {
	spaceID := c.Param("space_id")
	spaceService := service.NewSpaceService()

	resp, err := spaceService.Get(c, spaceID)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, resp)
}

func CreateSpace(c *gin.Context) {
	var req model.CreateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	spaceService := service.NewSpaceService()
	resp, err := spaceService.Create(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, resp)
}

func UpdateSpace(c *gin.Context) {
	spaceID := c.Param("space_id")
	var req model.UpdateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	req.SpaceID = spaceID

	spaceService := service.NewSpaceService()
	resp, err := spaceService.Update(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, resp)
}

func DeleteSpace(c *gin.Context) {
	spaceID := c.Param("space_id")
	spaceService := service.NewSpaceService()

	if err := spaceService.Delete(c, spaceID); err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, nil)
}

func AddMembers(c *gin.Context) {
	spaceID := c.Param("space_id")
	var req model.AddMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	req.SpaceID = spaceID

	spaceService := service.NewSpaceService()
	if err := spaceService.AddMembers(c, &req); err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, nil)
}
