package db

import (
	"fmt"
	"log"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"golang.org/x/crypto/bcrypt"
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
		&entities.AppGateway{},
		&entities.AppVolume{},
		&entities.AppConfigFile{},
		&entities.AppProbe{},
		&entities.AppSchedulingRule{},
	); err != nil {
		log.Fatalf("failed to migrate database, %v", err)
	}

	checkOrInitAdminUser(db)
}

func checkOrInitAdminUser(db *gorm.DB) {
	var count int64
	if err := db.Model(&entities.User{}).Where("role = ?", app.UserRoleAdmin).Count(&count).Error; err != nil {
		log.Fatalf("failed to count admin users: %v", err)
	}

	if count == 0 {
		adminUserID := uuid.New()
		passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("failed to generate password hash: %v", err)
		}
		if err := db.Create(&entities.User{
			UUIDBase: entities.UUIDBase{
				ID: adminUserID,
			},
			Username:          "admin",
			Fullname:          "Ketches Admin",
			Email:             fmt.Sprintf("admin.%s@ketches.cn", adminUserID[:6]),
			Password:          string(passwordHashBytes),
			Role:              app.UserRoleAdmin,
			MustResetPassword: true,
		}).Error; err != nil {
			log.Fatalf("failed to create initial admin user: %v", err)
		}
	}
}
