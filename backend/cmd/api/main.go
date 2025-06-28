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

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/ketches/ketches/internal/routes"
	_ "github.com/ketches/ketches/openapi"
)

// @title Ketches API Server
// @description Ketches api server
// @version v1
func main() {
	server := http.Server{
		Addr:    fmt.Sprintf("%s:%s", app.GetEnv("APP_HOST", "0.0.0.0"), app.GetEnv("APP_PORT", "8080")), // default to 8080 if not set
		Handler: newHttpHandler(),
	}

	go func() {
		log.Printf("Ketches api server is listening on http://%s\n", server.Addr)
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
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middlewares.Cors())

	routes.NewRoute(r).Register()

	return r
}
