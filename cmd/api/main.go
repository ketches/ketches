package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/routes"
	"github.com/ketches/ketches/internal/services"
)

func init() {
	app.PrintVersionBanner()
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("failed to load .env file: %v", err)
	}

	app.InitConfig()

	if err := db.InitDB(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := services.EnsureBootstrapAdmin(); err != nil {
		log.Fatalf("failed to ensure bootstrap admin: %v", err)
	}

	if err := services.EnsureBuiltinExtensions(); err != nil {
		log.Fatalf("failed to ensure builtin extensions: %v", err)
	}

	if err := services.InitClusters(); err != nil {
		log.Fatalf("failed to initialize clusters: %v", err)
	}

	// Recover active build watchers
	core.GlobalBuildWatcher.RecoverActiveBuilds()

	r := gin.Default()
	routes.SetupRoutes(r)

	log.Printf("server starting on :%s", app.Config.Port)
	if err := r.Run(":" + app.Config.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
