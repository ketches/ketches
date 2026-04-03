package app

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAccessTokenIncludesIssuerAudienceAndTokenType(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	Config.JWTSecret = "jwt-test-secret"
	Config.JWTIssuer = "ketches.test"
	Config.JWTAudience = "ketches-ui"
	Config.AccessTokenTTLMinutes = 15
	Config.RefreshTokenTTLHours = 24 * 7

	user := &entities.User{
		Base:     entities.Base{ID: "user-1"},
		Username: "alice",
		Role:     UserRoleUser,
	}

	tokenString, err := GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := parseClaimsForTest(tokenString)
	require.NoError(t, err)

	assert.Equal(t, "ketches.test", claims.Issuer)
	assert.Equal(t, jwt.ClaimStrings{"ketches-ui"}, claims.Audience)
	assert.Equal(t, "access", claims.TokenType)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, "alice", claims.Username)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateRefreshTokenIncludesIssuerAudienceAndTokenType(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	Config.JWTSecret = "jwt-test-secret"
	Config.JWTIssuer = "ketches.test"
	Config.JWTAudience = "ketches-ui"
	Config.AccessTokenTTLMinutes = 15
	Config.RefreshTokenTTLHours = 24 * 7

	user := &entities.User{
		Base: entities.Base{ID: "user-1"},
	}

	tokenString, err := GenerateRefreshToken(user)
	require.NoError(t, err)

	claims, err := parseClaimsForTest(tokenString)
	require.NoError(t, err)

	assert.Equal(t, "ketches.test", claims.Issuer)
	assert.Equal(t, jwt.ClaimStrings{"ketches-ui"}, claims.Audience)
	assert.Equal(t, "refresh", claims.TokenType)
	assert.Equal(t, "user-1", claims.Subject)
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), claims.ExpiresAt.Time, 5*time.Second)
}

func parseClaimsForTest(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(Config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}
