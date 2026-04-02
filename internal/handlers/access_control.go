package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/services"
)

var errForbidden = errors.New("forbidden")

func requireAdminClaims(c *gin.Context) *app.Claims {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		c.Abort()
		return nil
	}

	if claims.Role != app.UserRoleAdmin {
		api.Error(c, http.StatusForbidden, errForbidden)
		c.Abort()
		return nil
	}

	return claims
}

func requireProjectAccess(c *gin.Context, projectID string) *app.Claims {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		c.Abort()
		return nil
	}

	if err := services.EnsureProjectAccess(claims.UserID, claims.Role, projectID); err != nil {
		writeProjectAccessError(c, err)
		return nil
	}

	return claims
}

func requireClusterProjectAccess(c *gin.Context, projectID, clusterID string) *app.Claims {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		c.Abort()
		return nil
	}

	if err := services.EnsureClusterProjectAccess(claims.UserID, claims.Role, projectID, clusterID); err != nil {
		writeProjectAccessError(c, err)
		return nil
	}

	return claims
}

func queryProjectID(c *gin.Context) string {
	return strings.TrimSpace(c.Query("project_id"))
}

func writeProjectAccessError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrProjectScopeRequired):
		api.Error(c, http.StatusBadRequest, err)
	case errors.Is(err, services.ErrProjectAccessDenied), errors.Is(err, services.ErrClusterProjectDenied):
		api.Error(c, http.StatusForbidden, errForbidden)
	default:
		api.Error(c, http.StatusInternalServerError, err)
	}
	c.Abort()
}
