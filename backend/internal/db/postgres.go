package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newPostgres creates a new PostgreSQL database connection.
func newPostgres(dns string) (*gorm.DB, error) {
	return newDB(postgres.New(postgres.Config{
		DSN: dns,
	}))
}
