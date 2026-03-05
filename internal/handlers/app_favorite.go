package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/services"
)

// ListFavoriteApps returns all apps favorited by the current user in the given env.
func ListFavoriteApps(c *gin.Context) {
	envID := c.Param("envID")
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, nil)
		return
	}
	favorites, err := services.ListFavoriteApps(c.Request.Context(),claims.UserID, envID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, favorites)
}

// GetAppFavoriteStatus returns whether the current user has favorited the app.
func GetAppFavoriteStatus(c *gin.Context) {
	appID := c.Param("appID")
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, nil)
		return
	}
	// Resolve envID from app record
	var app entities.App
	if err := db.DB.Select("env_id").First(&app, "id = ?", appID).Error; err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	isFav := services.IsFavoriteApp(claims.UserID, appID, app.EnvID)
	api.Success(c, gin.H{"is_favorite": isFav})
}

// AddFavoriteApp adds an app to the current user's favorites.
func AddFavoriteApp(c *gin.Context) {
	appID := c.Param("appID")
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, nil)
		return
	}
	// Resolve envID from app record
	var app entities.App
	if err := db.DB.Select("env_id").First(&app, "id = ?", appID).Error; err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	fav, err := services.AddFavoriteApp(claims.UserID, appID, app.EnvID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, fav)
}

// RemoveFavoriteApp removes an app from the current user's favorites.
func RemoveFavoriteApp(c *gin.Context) {
	appID := c.Param("appID")
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, nil)
		return
	}
	// Resolve envID from app record
	var app entities.App
	if err := db.DB.Select("env_id").First(&app, "id = ?", appID).Error; err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	if err := services.RemoveFavoriteApp(claims.UserID, appID, app.EnvID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
