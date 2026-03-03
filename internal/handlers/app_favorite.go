package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/services"
)

// ListFavoriteApps returns all apps favorited by the current user.
func ListFavoriteApps(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, nil)
		return
	}
	favorites, err := services.ListFavoriteApps(claims.UserID)
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
	isFav := services.IsFavoriteApp(claims.UserID, appID)
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
	fav, err := services.AddFavoriteApp(claims.UserID, appID)
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
	if err := services.RemoveFavoriteApp(claims.UserID, appID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
