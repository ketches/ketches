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

import "net/http"

type Error interface {
	Code() int
	Message() string
}

type appError struct {
	code    int
	message string
}

func NewError(code int, message string) Error {
	return &appError{
		code:    code,
		message: message,
	}
}

var (
	ErrNotAuthorized = &appError{
		code:    http.StatusUnauthorized,
		message: "Not authorized",
	}

	ErrUserPasswordMustReset = &appError{
		code:    http.StatusUnauthorized,
		message: "User password must be reset",
	}

	ErrPermissionDenied = &appError{
		code:    http.StatusForbidden,
		message: "Permission denied",
	}

	ErrDatabaseOperationFailed = &appError{
		code:    http.StatusInternalServerError,
		message: "Database operation failed",
	}

	ErrClusterOperationFailed = &appError{
		code:    http.StatusInternalServerError,
		message: "Cluster operation failed",
	}
)

func (e *appError) Code() int {
	return e.code
}

func (e *appError) Message() string {
	return e.message
}
