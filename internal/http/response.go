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

package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
)

// var (
// 	errBadRequest   = errors.New(http.StatusText(http.StatusBadRequest))
// 	errUnauthorized = errors.New(http.StatusText(http.StatusUnauthorized))
// )

// Append status code for api response.
const (
	// 400XXX
	StatusInvalidParameter = 4004100

	// 401XXX
	StatusAccessTokenExpired  = 401200
	StatusInvalidAccessToken  = 401201
	StatusRefreshTokenExpired = 401210
	StatusInvalidRefreshToken = 401211
)

func StatusCodeText(code int) string {
	if code >= 100 && code <= 511 {
		return stdhttp.StatusText(int(code))
	}
	switch code {
	case StatusInvalidParameter:
		return "Invalid Parameter"
	case StatusAccessTokenExpired:
		return "Access-Token Expired"
	case StatusInvalidAccessToken:
		return "Invalid Access-Token"
	case StatusRefreshTokenExpired:
		return "Refresh-Token Expired"
	case StatusInvalidRefreshToken:
		return "Invalid Refresh-Token"
	default:
		return ""
	}
}

// type Responser interface {
// 	Success(c *gin.Context, data interface{})
// 	Error(c *gin.Context, err error)
// 	ErrorWithStatusCode(c *gin.Context, statusCode int, err error)
// 	Abort(c *gin.Context)
// }

type Response struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(stdhttp.StatusOK, Response{
		Code: stdhttp.StatusOK,
		Data: data,
	})
}

func BadRequest(c *gin.Context) {
	c.JSON(stdhttp.StatusBadRequest, Response{
		Code:  stdhttp.StatusBadRequest,
		Error: stdhttp.StatusText(stdhttp.StatusBadRequest),
	})
}

func BadRequestWithError(c *gin.Context, err error) {
	c.JSON(stdhttp.StatusBadRequest, Response{
		Code:  stdhttp.StatusBadRequest,
		Error: err.Error(),
	})
}

func Unauthorized(c *gin.Context) {
	c.JSON(stdhttp.StatusUnauthorized, Response{
		Code:  stdhttp.StatusUnauthorized,
		Error: stdhttp.StatusText(stdhttp.StatusUnauthorized),
	})
}

func UnauthorizedWithError(c *gin.Context, err error) {
	c.JSON(stdhttp.StatusUnauthorized, Response{
		Code:  stdhttp.StatusUnauthorized,
		Error: err.Error(),
	})
}

func InternalServerError(c *gin.Context) {
	c.JSON(stdhttp.StatusInternalServerError, Response{
		Code:  stdhttp.StatusInternalServerError,
		Error: stdhttp.StatusText(stdhttp.StatusInternalServerError),
	})
}

func AccessTokenExpired(c *gin.Context) {
	c.JSON(stdhttp.StatusUnauthorized, Response{
		Code:  StatusAccessTokenExpired,
		Error: StatusCodeText(StatusAccessTokenExpired),
	})
}

func InvalidAccessToken(c *gin.Context) {
	c.JSON(stdhttp.StatusUnauthorized, Response{
		Code:  StatusInvalidAccessToken,
		Error: StatusCodeText(StatusInvalidAccessToken),
	})
}

func RefreshTokenExpired(c *gin.Context) {
	c.JSON(stdhttp.StatusUnauthorized, Response{
		Code:  StatusRefreshTokenExpired,
		Error: StatusCodeText(StatusRefreshTokenExpired),
	})
}

func InvalidRefreshToken(c *gin.Context) {
	c.JSON(stdhttp.StatusUnauthorized, Response{
		Code:  StatusInvalidRefreshToken,
		Error: StatusCodeText(StatusInvalidRefreshToken),
	})
}

func Forbidden(c *gin.Context) {
	c.JSON(stdhttp.StatusForbidden, Response{
		Code:  stdhttp.StatusForbidden,
		Error: stdhttp.StatusText(stdhttp.StatusForbidden),
	})
}

func ForbiddenWithError(c *gin.Context, err error) {
	c.JSON(stdhttp.StatusForbidden, Response{
		Code:  stdhttp.StatusForbidden,
		Error: err.Error(),
	})
}

func Error(c *gin.Context, err error) {
	c.JSON(stdhttp.StatusOK, Response{
		Code:  stdhttp.StatusOK,
		Error: err.Error(),
	})
}

func Abort(c *gin.Context) {
	c.Abort()
}

func AbortWithNoContent(c *gin.Context) {
	c.AbortWithStatus(stdhttp.StatusNoContent)
}

func AbortWithBadRequest(c *gin.Context) {
	BadRequest(c)
	c.Abort()
}

func AbortWithUnauthorized(c *gin.Context) {
	Unauthorized(c)
	c.Abort()
}

func AbortWithAccessTokenExpired(c *gin.Context) {
	AccessTokenExpired(c)
	c.Abort()
}

func AbortWithInvalidAccessToken(c *gin.Context) {
	InvalidAccessToken(c)
	c.Abort()
}

func AbortWithForbidden(c *gin.Context) {
	Forbidden(c)
	c.Abort()
}
