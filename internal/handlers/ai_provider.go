package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListCurrentUserAIProviders(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	providers, err := services.ListUserAIProviders(claims.UserID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, providers)
}

func CreateCurrentUserAIProvider(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req models.CreateAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	provider, err := services.CreateUserAIProvider(claims.UserID, &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidBuilderRegistryAlias) || errors.Is(err, services.ErrInvalidAIProviderBaseURL) {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, provider)
}

func UpdateCurrentUserAIProvider(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req models.CreateAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	provider, err := services.UpdateUserAIProvider(claims.UserID, c.Param("providerID"), &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidBuilderRegistryAlias) || errors.Is(err, services.ErrInvalidAIProviderBaseURL) {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, provider)
}

func DeleteCurrentUserAIProvider(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	if err := services.DeleteUserAIProvider(claims.UserID, c.Param("providerID")); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func ListProjectAIProviders(c *gin.Context) {
	projectID := c.Param("projectID")

	providers, err := services.ListProjectAIProviders(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, providers)
}

func CreateProjectAIProvider(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.CreateAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	provider, err := services.CreateProjectAIProvider(projectID, &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidBuilderRegistryAlias) || errors.Is(err, services.ErrInvalidAIProviderBaseURL) {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, provider)
}

func UpdateProjectAIProvider(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.CreateAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	provider, err := services.UpdateProjectAIProvider(projectID, c.Param("providerID"), &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidBuilderRegistryAlias) || errors.Is(err, services.ErrInvalidAIProviderBaseURL) {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, provider)
}

func DeleteProjectAIProvider(c *gin.Context) {
	projectID := c.Param("projectID")

	if err := services.DeleteProjectAIProvider(projectID, c.Param("providerID")); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}
