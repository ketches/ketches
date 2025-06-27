package db

import (
	"fmt"
	"log"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

// Migrate creates or updates the database tables.
// Append new database entities here, database migrations
// will be handled on the application startup.
func Migrate(db *gorm.DB) {
	if err := db.AutoMigrate(
		&entities.User{},
		&entities.UserToken{},
		&entities.Cluster{},
		&entities.Cert{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Env{},
		&entities.App{},
		&entities.AppEnvVar{},
		&entities.AppPort{},
		&entities.AppGateway{},
		&entities.AppVolume{},
	); err != nil {
		log.Fatalf("failed to migrate database, %v", err)
	}

	if err := checkOrInitAdminUser(db); err != nil {
		log.Fatalf("failed to check or initialize admin user, %v", err)
	}
}

func checkOrInitAdminUser(db *gorm.DB) app.Error {
	var count int64
	if err := db.Model(&entities.User{}).Where("role = ?", app.UserRoleAdmin).Count(&count).Error; err != nil {
		return app.ErrDatabaseOperationFailed
	}

	if count == 0 {
		adminUserID := uuid.New()
		if err := db.Create(&entities.User{
			UUIDBase: entities.UUIDBase{
				ID: adminUserID,
			},
			Username:          "admin",
			Fullname:          "Ketches Admin",
			Email:             fmt.Sprintf("%s.admin@ketches.cn", adminUserID[:6]),
			Password:          "admin",
			Role:              app.UserRoleAdmin,
			MustResetPassword: true,
		}); err != nil {
			return app.ErrDatabaseOperationFailed
		}
	}
	return nil
}
