package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListNotifications(c *gin.Context) {
	claims := api.GetClaims(c)

	req := models.DefaultPaginationRequest()
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, rows, err := services.ListNotifications(claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	items := make([]models.NotificationResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, models.NotificationResponse(r))
	}

	api.Success(c, models.ListNotificationResponse{
		Items:      items,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func GetUnreadCount(c *gin.Context) {
	claims := api.GetClaims(c)
	count, err := services.GetUnreadCount(claims.UserID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.UnreadCountResponse{Count: count})
}

func HandleNotificationAction(c *gin.Context) {
	claims := api.GetClaims(c)
	notifID := c.Param("notificationID")

	var req models.NotificationActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.HandleNotificationAction(notifID, claims.UserID, req.Action); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.NoContent(c)
}

func MarkAllNotificationsRead(c *gin.Context) {
	claims := api.GetClaims(c)
	if err := services.MarkAllRead(claims.UserID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
