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

package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/handlers"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
)

type RootRoute struct {
	*gin.Engine
}

func NewRoute(e *gin.Engine) *RootRoute {
	return &RootRoute{e}
}

func (r *RootRoute) Register() {
	r.GET("/openapi/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	r.GET("/version", handlers.Version)
	r.GET("/healthz", handlers.Healthz)

	NewAPIV1Route(r.Engine).Register()
}
