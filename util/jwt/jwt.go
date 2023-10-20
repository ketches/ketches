/*
Copyright 2023 The Ketches Authors.

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
package jwt

import (
	"flag"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"os"
	"time"
)

const (
	jwtIssuer          = "Ketches"
	normalTokenExpire  = time.Hour * 24
	accessTokenExpire  = time.Minute * 2
	refreshTokenExpire = time.Hour * 24 * 7
)

var jwtSecret string

func loadJwtSecret() string {
	if jwtSecret != "" {
		return jwtSecret
	}

	if val := os.Getenv("KETCHES_JWT_SECRET"); val != "" {
		jwtSecret = val
	} else {
		flag.StringVar(&jwtSecret, "jwt-secret", jwtSecret, "JWT secret")
		flag.Parse()
	}

	if jwtSecret == "" {
		log.Fatalln(`jwt secret is required. 
You can set it by environment variable "KETCHES_JWT_SECRET" or command line argument "-jwt-secret"`)
	}
	return jwtSecret
}

type MyClaims struct {
	jwt.RegisteredClaims
	AccountID  string     `json:"account_id"`
	SignInRole SignInRole `json:"sign_in_role"`
	Email      string     `json:"email"`
	TokenType  TokenType  `json:"token_type"`
}

type SignInRole string

const (
	SignInRoleAdmin SignInRole = "admin"
	SignInRoleUser  SignInRole = "user"
)

type TokenType string

const (
	TokenTypeEmpty        TokenType = ""
	TokenTypeAccessToken  TokenType = "access_token"
	TokenTypeRefreshToken TokenType = "refresh_token"
)

func GenerateToken(mc MyClaims) (string, error) {
	var expiresAt time.Time
	var now = time.Now()

	switch mc.TokenType {
	case TokenTypeAccessToken:
		expiresAt = now.Add(accessTokenExpire)
	case TokenTypeRefreshToken:
		expiresAt = now.Add(refreshTokenExpire)
	default:
		expiresAt = now.Add(normalTokenExpire)
	}

	claims := MyClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		AccountID: mc.AccountID,
		Email:     mc.Email,
		TokenType: mc.TokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(loadJwtSecret()))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateToken(tokenString string) (*MyClaims, error) {
	var ret *MyClaims
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(loadJwtSecret()), nil
	})
	if token != nil {
		ret, _ = token.Claims.(*MyClaims)
	}

	return ret, err
}
