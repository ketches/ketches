package db

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/ketches/ketches/internal/app"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func newGormConfig() *gorm.Config {
	return &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	}
}

func InitDB() error {
	var err error
	var dialector gorm.Dialector

	switch app.Config.DBDriver {
	case "postgres", "postgresql":
		dialector = postgres.Open(app.Config.DBSource)
	case "mysql":
		dialector = mysql.Open(app.Config.DBSource)
	default:
		return app.NewErrorf("unsupported database driver: %s", app.Config.DBDriver)
	}

	DB, err = gorm.Open(dialector, newGormConfig())
	if err != nil {
		return app.WrapError("failed to connect to database", err)
	}

	if err := migrateConfiguredDatabase(); err != nil {
		return app.WrapError("failed to migrate database", err)
	}

	if err := configureConnectionPool(DB); err != nil {
		return app.WrapError("configure database connection pool", err)
	}

	slog.Info("database connected", "driver", app.Config.DBDriver)
	return nil
}

func migrateConfiguredDatabase() error {
	if !app.Config.DBAutoMigrate {
		return nil
	}
	return Migrate()
}

func configureConnectionPool(gormDB *gorm.DB) error {
	if gormDB == nil {
		return nil
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	applyPoolConfig(sqlDB)
	return nil
}

func applyPoolConfig(sqlDB *sql.DB) {
	if sqlDB == nil {
		return
	}

	sqlDB.SetMaxIdleConns(app.Config.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(app.Config.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(app.Config.DBConnMaxLifetimeMinutes) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(app.Config.DBConnMaxIdleTimeMinutes) * time.Minute)
}
