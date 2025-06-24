package db

import (
	"log"

	"github.com/ketches/ketches/internal/db/entity"
	"gorm.io/gorm"
)

// Migrate creates or updates the database tables.
// Append new database entities here, database migrations
// will be handled on the application startup.
func Migrate(db *gorm.DB) {
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.UserToken{},
		&entity.Cluster{},
		&entity.Cert{},
		&entity.Project{},
		&entity.ProjectMember{},
		&entity.Env{},
		&entity.App{},
		&entity.AppEnvVar{},
		&entity.AppPort{},
		&entity.AppGateway{},
	); err != nil {
		log.Fatalf("failed to migrate database, %v", err)
	}
}
