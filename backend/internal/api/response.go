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

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
)

// "github.com/gin-gonic/gin"

// var (
// 	errBadRequest   = errors.New(http.StatusText(http.StatusBadRequest))
// 	errUnauthorized = errors.New(http.StatusText(http.StatusUnauthorized))
// )

// Append status code for api response.
// const (
// 	// 400XXX
// 	StatusInvalidParameter = 4004100

// 	// 401XXX
// 	StatusAccessTokenExpired  = 401200
// 	StatusInvalidAccessToken  = 401201
// 	StatusRefreshTokenExpired = 401210
// 	StatusInvalidRefreshToken = 401211
// )

// func StatusCodeText(code int) string {
// 	if code >= 100 && code <= 511 {
// 		return http.StatusText(int(code))
// 	}
// 	switch code {
// 	case StatusInvalidParameter:
// 		return "Invalid Parameter"
// 	case StatusAccessTokenExpired:
// 		return "Access-Token Expired"
// 	case StatusInvalidAccessToken:
// 		return "Invalid Access-Token"
// 	case StatusRefreshTokenExpired:
// 		return "Refresh-Token Expired"
// 	case StatusInvalidRefreshToken:
// 		return "Invalid Refresh-Token"
// 	default:
// 		return ""
// 	}
// }

// type Responser interface {
// 	Success(c *gin.Context, data interface{})
// 	Error(c *gin.Context, err error)
// 	ErrorWithStatusCode(c *gin.Context, statusCode int, err error)
// 	Abort(c *gin.Context)
// }

type Response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Data: data,
	})
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Data: data,
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, err app.Error) {
	c.AbortWithStatusJSON(err.Code(), Response{
		Error: err.Message(),
	})
}

// func Status(c *gin.Context, code int, data any, err error) {
// 	c.JSON(code, Response{
// 		Data:  data,
// 		Error: err.Error(),
// 	})
// }

// func Success(c *gin.Context, data interface{}) {
// 	c.JSON(http.StatusOK, Response{
// 		Data: data,
// 	})
// }

// func BadRequest(c *gin.Context) {
// 	c.JSON(http.StatusBadRequest, Response{
// 		Error: http.StatusText(http.StatusBadRequest),
// 	})
// }

// func BadRequestWithError(c *gin.Context, err error) {
// 	c.JSON(http.StatusBadRequest, Response{
// 		Error: err.Error(),
// 	})
// }

// func Unauthorized(c *gin.Context) {
// 	c.JSON(http.StatusUnauthorized, Response{
// 		Error: http.StatusText(http.StatusUnauthorized),
// 	})
// }

// func UnauthorizedWithError(c *gin.Context, err error) {
// 	c.JSON(http.StatusUnauthorized, Response{
// 		Error: err.Error(),
// 	})
// }

// func InternalServerError(c *gin.Context) {
// 	c.JSON(http.StatusInternalServerError, Response{
// 		Error: http.StatusText(http.StatusInternalServerError),
// 	})
// }

// func AccessTokenExpired(c *gin.Context) {
// 	c.JSON(http.StatusUnauthorized, Response{
// 		Error: "Access-Token Expired",
// 	})
// }

// func InvalidAccessToken(c *gin.Context) {
// 	c.JSON(http.StatusUnauthorized, Response{
// 		Error: "Invalid Access-Token",
// 	})
// }

// func RefreshTokenExpired(c *gin.Context) {
// 	c.JSON(http.StatusUnauthorized, Response{
// 		Error: "Refresh-Token Expired",
// 	})
// }

// func InvalidRefreshToken(c *gin.Context) {
// 	c.JSON(http.StatusUnauthorized, Response{
// 		Error: "Invalid Refresh-Token",
// 	})
// }

// func Forbidden(c *gin.Context) {
// 	c.JSON(http.StatusForbidden, Response{
// 		Error:
// 	})
// }

// func ForbiddenWithError(c *gin.Context, err error) {
// 	c.JSON(http.StatusForbidden, Response{
// 		Code:  http.StatusForbidden,
// 		Error: err.Error(),
// 	})
// }

// func Error(c *gin.Context, err error) {
// 	c.JSON(http.StatusOK, Response{
// 		Code:  http.StatusOK,
// 		Error: err.Error(),
// 	})
// }

// func Abort(c *gin.Context) {
// 	c.Abort()
// }

// func AbortWithNoContent(c *gin.Context) {
// 	c.AbortWithStatus(http.StatusNoContent)
// }

// func AbortWithBadRequest(c *gin.Context) {
// 	BadRequest(c)
// 	c.Abort()
// }

// func AbortWithUnauthorized(c *gin.Context) {
// 	Unauthorized(c)
// 	c.Abort()
// }

// func AbortWithAccessTokenExpired(c *gin.Context) {
// 	AccessTokenExpired(c)
// 	c.Abort()
// }

// func AbortWithInvalidAccessToken(c *gin.Context) {
// 	InvalidAccessToken(c)
// 	c.Abort()
// }

// func AbortWithForbidden(c *gin.Context) {
// 	Forbidden(c)
// 	c.Abort()
// }
