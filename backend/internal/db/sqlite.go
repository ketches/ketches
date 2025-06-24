package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewSQLite creates a new SQLite database connection.
func NewSQLite(dns string) (*gorm.DB, error) {
	return newDB(sqlite.Open(dns))
}
