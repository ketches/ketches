package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
)

const contextKeyUserID = "user_id"
const contextKeyUserRole = "user_role"
const contextKeyProjectRole = "project_role"
const contextKeyRequestID = "request_id"

func SetUserID(ctx *gin.Context, userID string) {
	ctx.Set(contextKeyUserID, userID)
}

func SetUserRole(ctx *gin.Context, userRole string) {
	ctx.Set(contextKeyUserRole, userRole)
}

func SetProjectRole(ctx *gin.Context, projectRole string) {
	ctx.Set(contextKeyProjectRole, projectRole)
}

func UserID(ctx context.Context) string {
	userID := ctx.Value(contextKeyUserID)
	if userID == nil {
		return ""
	}
	return userID.(string)
}

func UserRole(ctx context.Context) string {
	userRole := ctx.Value(contextKeyUserRole)
	if userRole == nil {
		return ""
	}
	return userRole.(string)
}

func ProjectRole(ctx context.Context) string {
	projectRole := ctx.Value(contextKeyProjectRole)
	if projectRole == nil {
		return ""
	}
	return projectRole.(string)
}

func IsAdmin(ctx context.Context) bool {
	return UserRole(ctx) == app.UserRoleAdmin
}

func IsProjectOwner(ctx context.Context) bool {
	return ProjectRole(ctx) == app.ProjectRoleOwner
}

func IsProjectDeveloper(ctx context.Context) bool {
	return ProjectRole(ctx) == app.ProjectRoleDeveloper
}

func IsProjectViewer(ctx context.Context) bool {
	return ProjectRole(ctx) == app.ProjectRoleViewer
}

func SetRequestID(ctx *gin.Context, requestID string) {
	ctx.Set(contextKeyRequestID, requestID)
}

func RequestID(ctx context.Context) string {
	requestID := ctx.Value(contextKeyRequestID)
	if requestID == nil {
		return ""
	}
	return requestID.(string)
}
