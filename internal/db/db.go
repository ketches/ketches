package db

import (
	"fmt"
	"log"

	"github.com/ketches/ketches/internal/app"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	var err error
	var dialector gorm.Dialector

	switch app.Config.DBDriver {
	case "postgres":
		dialector = postgres.Open(app.Config.DBSource)
	case "mysql":
		dialector = mysql.Open(app.Config.DBSource)
	case "sqlite":
		dialector = sqlite.Open(app.Config.DBSource)
	default:
		return fmt.Errorf("unsupported database driver: %s", app.Config.DBDriver)
	}

	DB, err = gorm.Open(dialector)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if app.Config.DBAutoMigrate {
		if err := Migrate(); err != nil {
			return fmt.Errorf("failed to migrate database: %w", err)
		}
	}

	log.Printf("successfully connected to %s database", app.Config.DBDriver)
	return nil
}
