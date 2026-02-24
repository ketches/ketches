package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func SignUp(c *gin.Context) {
	var req models.SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.SignUp(&req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Fullname: user.Fullname,
		Role:     user.Role,
	})
}

func SignIn(c *gin.Context) {
	var req models.SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.SignIn(&req)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, err)
		return
	}

	accessToken, err := app.GenerateAccessToken(user)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	refreshToken, err := app.GenerateRefreshToken(user)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.SignInResponse{
		User: models.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Fullname: user.Fullname,
			Role:     user.Role,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func ListUsers(c *gin.Context) {
	users, err := services.ListUsers()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	var res []models.UserResponse
	for _, u := range users {
		res = append(res, models.UserResponse{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			Fullname: u.Fullname,
			Role:     u.Role,
		})
	}
	api.Success(c, res)
}

func UpdateUser(c *gin.Context) {
	userID := c.Param("userID")
	var req struct {
		Fullname string `json:"fullname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.UpdateUser(userID, req.Fullname, req.Email, req.Phone)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Fullname: user.Fullname,
		Role:     user.Role,
	})
}

func DeleteUser(c *gin.Context) {
	userID := c.Param("userID")
	if err := services.DeleteUser(userID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ChangeUserRole(c *gin.Context) {
	userID := c.Param("userID")
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.ChangeUserRole(userID, req.Role); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.CreateUser(&req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Fullname: user.Fullname,
		Role:     user.Role,
	})
}
