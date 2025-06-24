package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newMySQL creates a new MySQL database connection.
func newMySQL(dns string) (*gorm.DB, error) {
	return newDB(mysql.New(mysql.Config{
		DSN:               dns,
		DefaultStringSize: 256,
		// Disable datetime precision, which not supported before MySQL 5.6
		// If disabled, datetime format will be: `0000-00-00 00:00:00`,
		// otherwise will be `0000-00-00 00:00:00.000`
		// DisableDatetimePrecision: true,
	}))
}
