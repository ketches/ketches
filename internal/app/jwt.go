package app

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ketches/ketches/internal/db/entities"
)

type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(user *entities.User) (string, error) {
	return generateToken(user, TokenTypeAccess, time.Duration(Config.AccessTokenTTLMinutes)*time.Minute)
}

func GenerateRefreshToken(user *entities.User) (string, error) {
	return generateToken(user, TokenTypeRefresh, time.Duration(Config.RefreshTokenTTLHours)*time.Hour)
}

func ParseToken(tokenString, expectedTokenType string) (*Claims, error) {
	claims := &Claims{}
	parseOptions := make([]jwt.ParserOption, 0, 2)
	if Config.JWTIssuer != "" {
		parseOptions = append(parseOptions, jwt.WithIssuer(Config.JWTIssuer))
	}
	if Config.JWTAudience != "" {
		parseOptions = append(parseOptions, jwt.WithAudience(Config.JWTAudience))
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(Config.JWTSecret), nil
	}, parseOptions...)
	if err != nil || !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	if expectedTokenType != "" && claims.TokenType != expectedTokenType {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func generateToken(user *entities.User, tokenType string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		switch tokenType {
		case TokenTypeRefresh:
			ttl = 7 * 24 * time.Hour
		default:
			ttl = 15 * time.Minute
		}
	}

	now := time.Now()
	registeredClaims := jwt.RegisteredClaims{
		Subject:   user.ID,
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	if Config.JWTIssuer != "" {
		registeredClaims.Issuer = Config.JWTIssuer
	}
	if Config.JWTAudience != "" {
		registeredClaims.Audience = jwt.ClaimStrings{Config.JWTAudience}
	}

	claims := Claims{
		UserID:           user.ID,
		Username:         user.Username,
		Role:             user.Role,
		TokenType:        tokenType,
		RegisteredClaims: registeredClaims,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(Config.JWTSecret))
}
