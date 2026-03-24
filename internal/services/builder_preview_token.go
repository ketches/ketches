package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ketches/ketches/internal/app"
)

const BuilderPreviewSessionCookieName = "X-Ketches-Builder-Preview"

type BuilderPreviewClaims struct {
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
	jwt.RegisteredClaims
}

func MintBuilderPreviewSessionToken(userID, projectID, sessionID, runID string) (string, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(projectID) == "" || strings.TrimSpace(sessionID) == "" || strings.TrimSpace(runID) == "" {
		return "", errors.New("builder preview session token fields are required")
	}

	claims := BuilderPreviewClaims{
		UserID:    userID,
		ProjectID: projectID,
		SessionID: sessionID,
		RunID:     runID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("builder-preview:%s:%s:%s:%s", userID, projectID, sessionID, runID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(app.Config.JWTSecret))
}

func ParseBuilderPreviewSessionToken(tokenString string) (*BuilderPreviewClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, jwt.ErrTokenSignatureInvalid
	}

	claims := &BuilderPreviewClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(app.Config.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	return claims, nil
}
