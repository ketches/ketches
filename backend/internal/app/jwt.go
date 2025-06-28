/*
Copyright 2025 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package app

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	NormalTokenTTL  = time.Hour * 24     // 24 hours
	AccessTokenTTL  = time.Hour * 2      // 2 hours
	RefreshTokenTTL = time.Hour * 24 * 7 // 7 days
)

var jwtSecret string

func loadJwtSecret() string {
	return GetEnv("APP_JWT_SECRET", "ketches")
}

type TokenClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"userID"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	UserRole  string `json:"userRole"`
	TokenType string `json:"tokenType"`
}

const (
	TokenTypeAccessToken  = "access_token"
	TokenTypeRefreshToken = "refresh_token"
)

func GenerateToken(mc TokenClaims) (string, time.Time, error) {
	var expiresAt time.Time
	var now = time.Now()

	switch mc.TokenType {
	case TokenTypeAccessToken:
		expiresAt = now.Add(AccessTokenTTL)
	case TokenTypeRefreshToken:
		expiresAt = now.Add(RefreshTokenTTL)
	default:
		expiresAt = now.Add(NormalTokenTTL)
	}

	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			// Issuer:    jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    mc.UserID,
		Username:  mc.Username,
		Email:     mc.Email,
		UserRole:  mc.UserRole,
		TokenType: mc.TokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(loadJwtSecret()))
	if err != nil {
		return "", expiresAt, err
	}
	return tokenString, expiresAt, nil
}

func ValidateToken(tokenString string) (*TokenClaims, error) {
	var result *TokenClaims
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(loadJwtSecret()), nil
	})
	if err != nil {
		return nil, err
	}

	if token != nil {
		claims, ok := token.Claims.(*TokenClaims)
		if !ok {
			return nil, err
		}
		result = claims
	}

	return result, err
}
