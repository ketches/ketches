package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

const refreshTokenCookiePath = "/api/v1/users/refresh-token"

func GetSignUpConfig(c *gin.Context) {
	enabled, err := services.GetPublicSignUpEnabled()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	verificationRequired, err := services.GetSignUpEmailVerificationRequired()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.PublicSignUpSettingsResponse{
		Enabled:                   enabled,
		EmailVerificationRequired: verificationRequired,
	})
}

func RequestSignUpVerificationCode(c *gin.Context) {
	var req models.SignUpVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	resp, err := services.RequestSignUpVerificationCode(req.Email)
	if err != nil {
		status := http.StatusBadRequest
		if err == services.ErrPublicSignUpDisabled {
			status = http.StatusForbidden
		}
		api.Error(c, status, err)
		return
	}

	api.Success(c, resp)
}

func RefreshToken(c *gin.Context) {
	refreshTokenCookie, err := c.Cookie(app.RefreshTokenCookieName)
	if err != nil || strings.TrimSpace(refreshTokenCookie) == "" {
		api.Error(c, http.StatusUnauthorized, services.ErrInvalidRefreshToken)
		return
	}

	refreshToken, err := url.QueryUnescape(refreshTokenCookie)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, services.ErrInvalidRefreshToken)
		return
	}

	claims, err := app.ParseToken(refreshToken, app.TokenTypeRefresh)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, services.ErrInvalidRefreshToken)
		return
	}

	user, err := services.GetUser(claims.UserID)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, services.ErrInvalidRefreshToken)
		return
	}
	if err := services.VerifyRefreshToken(user, refreshToken); err != nil {
		api.Error(c, http.StatusUnauthorized, err)
		return
	}

	if err := issueUserSession(c, user, user.MustChangePassword); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.SignInResponse{
		User:               toUserResponse(user),
		MustChangePassword: user.MustChangePassword,
	})
}

func SignOut(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	if err := services.ClearRefreshToken(claims.UserID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	clearSessionCookies(c)
	api.NoContent(c)
}

func UpdateUserLock(c *gin.Context) {
	userID := c.Param("userID")

	var req models.UpdateUserLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.SetUserLockState(userID, req.Locked, req.Reason)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toUserResponse(user))
}

func issueUserSession(c *gin.Context, user *entities.User, mustChangePassword bool) error {
	accessToken, err := app.GenerateAccessToken(user)
	if err != nil {
		return err
	}

	refreshToken, err := app.GenerateRefreshToken(user)
	if err != nil {
		return err
	}

	if err := services.StoreRefreshToken(user.ID, refreshToken); err != nil {
		return err
	}

	csrfToken, err := services.GenerateCSRFToken()
	if err != nil {
		return err
	}

	setCookie(c, &http.Cookie{
		Name:     app.AccessTokenCookieName,
		Value:    url.QueryEscape(accessToken),
		Path:     "/",
		MaxAge:   app.Config.AccessTokenTTLMinutes * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(c),
	})
	setCookie(c, &http.Cookie{
		Name:     app.RefreshTokenCookieName,
		Value:    url.QueryEscape(refreshToken),
		Path:     refreshTokenCookiePath,
		MaxAge:   app.Config.RefreshTokenTTLHours * int(time.Hour/time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(c),
	})
	setCookie(c, &http.Cookie{
		Name:     app.CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		MaxAge:   app.Config.RefreshTokenTTLHours * int(time.Hour/time.Second),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(c),
	})
	return nil
}

func clearSessionCookies(c *gin.Context) {
	clearCookie(c, app.AccessTokenCookieName, "/")
	clearCookie(c, app.RefreshTokenCookieName, refreshTokenCookiePath)
	setCookie(c, &http.Cookie{
		Name:     app.CSRFCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(c),
	})
}

func setCookie(c *gin.Context, cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	http.SetCookie(c.Writer, cookie)
}

func clearCookie(c *gin.Context, name, path string) {
	setCookie(c, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(c),
	})
}

func requestUsesHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func toUserResponse(user *entities.User) models.UserResponse {
	return models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Fullname:  user.Fullname,
		Bio:       user.Bio,
		Role:      user.Role,
		IsLocked:  user.IsLocked,
		LockedAt:  user.LockedAt,
		CreatedAt: user.CreatedAt,
	}
}
