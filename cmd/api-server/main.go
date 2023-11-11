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

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/initializer"
	"github.com/ketches/ketches/internal/middleware"
	"github.com/ketches/ketches/internal/route"
	_ "github.com/ketches/ketches/openapi"
	"github.com/ketches/ketches/pkg/global"
	"github.com/spf13/cast"
)

// @title Ketches Http Server
// @description Ketches Http server.
// @version v1
// @contact.name Ketches Support Team
// @contact.email ketches@ketchees.cn
// @contact.url https://support.ketches.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0
func main() {
	server := http.Server{
		Addr:    ":" + cast.ToString(global.ApiServerPort()),
		Handler: newHttpHandler(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("listen and serve err: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server ...")

	shutdownTimeoutErr := fmt.Errorf("shutting down server timeout")
	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, shutdownTimeoutErr)
	defer func() {
		cancel()
		if errors.Is(context.Cause(ctx), shutdownTimeoutErr) {
			log.Fatal(shutdownTimeoutErr)
		}
	}()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func newHttpHandler() http.Handler {
	color.Yellow("➜ Served at http://localhost:%d\n", global.ApiServerPort())
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.Cors())

	route.NewRoute(r).Register()

	return r
}

func init() {
	// init builtin resources
	err := initializer.InitializePlatform()
	if err != nil {
		log.Fatalf("initialize platform err: %v", err)
	}
}
