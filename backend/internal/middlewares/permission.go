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

package middlewares

import (
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/orm"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !api.IsAdmin(c) {
			api.Error(c, app.ErrPermissionDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

// ProjectPermission is a middleware that checks if the user has permission to access the project, environment, or application
// based on their role in the project. It extracts project ID from the URL parameter and checks the user's role.
func ProjectPermission(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip admin users as they have full access
		if api.IsAdmin(c) {
			c.Next()
			return
		}

		// If no specific roles are required, allow access
		if len(requiredRoles) == 0 {
			c.Next()
			return
		}

		userProjectRole, err := getUserProjectRole(c)
		if err != nil {
			api.Error(c, err)
			c.Abort()
			return
		}

		hasRequiredRole := slices.Contains(requiredRoles, userProjectRole)
		if !hasRequiredRole {
			api.Error(c, app.ErrPermissionDenied)
			c.Abort()
			return
		}

		c.Next()
	}
}

// ProjectOwnerOnly is a middleware that checks if the user is the owner of the project
func ProjectOwnerOnly() gin.HandlerFunc {
	return ProjectPermission(app.ProjectRoleOwner)
}

// ProjectDeveloperOrAbove is a middleware that checks if the user is a developer or owner of the project
func ProjectDeveloperOrAbove() gin.HandlerFunc {
	return ProjectPermission(app.ProjectRoleOwner, app.ProjectRoleDeveloper)
}

// ProjectMember is a middleware that checks if the user is a member of the project (any role)
func ProjectMember() gin.HandlerFunc {
	return ProjectPermission()
}

func getUserProjectRole(c *gin.Context) (string, app.Error) {
	projectID, ok := c.Params.Get("projectID")
	if ok && projectID != "" {
		return orm.GetProjectRoleByProjectID(c, projectID)
	}
	envID, ok := c.Params.Get("envID")
	if ok && envID != "" {
		return getProjectRoleFromEnvIDRoute(c, envID)
	}
	appID, ok := c.Params.Get("appID")
	if ok && appID != "" {
		return getProjectRoleFromAppIDRoute(c, appID)
	}
	return "", nil
}

func getProjectRoleFromEnvIDRoute(c *gin.Context, envID string) (string, app.Error) {
	if envID == "" {
		return "", nil
	}

	projectRole, err := orm.GetProjectRoleByEnvID(c, envID)
	if err != nil {
		return "", err
	}

	return projectRole, nil
}

func getProjectRoleFromAppIDRoute(c *gin.Context, appID string) (string, app.Error) {
	projectRole, err := orm.GetProjectRoleByAppID(c, appID)
	if err != nil {
		return "", err
	}

	return projectRole, nil
}
