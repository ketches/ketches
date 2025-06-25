package db

import (
	// "go-apiserver-template/internal/config"
	// "go-apiserver-template/internal/global"
	// "go-apiserver-template/pkg/log"

	"errors"
	"log"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	DBTypeMySQL    = "mysql"
	DBTypePostgres = "postgres"
	DBTypeSQLite   = "sqlite"
)

var (
	instance *gorm.DB
)

// newDB is a helper function to create a new *gorm.DB instance with
// default configurations
func newDB(dialector gorm.Dialector) (*gorm.DB, error) {
	loglevel := logger.Error
	loglevel = logger.Info

	return gorm.Open(dialector, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(loglevel),
	})
}

// Instance returns a singleton instance of the database
func Instance() *gorm.DB {
	if instance != nil {
		return instance
	}

	var (
		err      error
		dbConfig = app.Config().DB
	)
	switch dbConfig.Type {
	case DBTypeMySQL:
		instance, err = newMySQL(dbConfig.DNS)
	case DBTypePostgres:
		instance, err = newPostgres(dbConfig.DNS)
	case DBTypeSQLite:
		instance, err = NewSQLite(dbConfig.DNS)
	default:
		log.Fatalf("unsupported database type, %v", dbConfig.Type)
	}

	if err != nil {
		log.Fatalf("error connecting to database, %v", err)
	}

	Migrate(instance)

	return instance
}

func Transaction(fn func(tx *gorm.DB) error) error {
	return Instance().Transaction(fn)
}

func IsErrRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func IsErrDuplicatedKey(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		// For MySQL
		strings.Contains(err.Error(), "Duplicate entry") ||
		// For PostgreSQL
		strings.Contains(err.Error(), "duplicate key value violates unique constraint") ||
		// For SQLite
		strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func CaseInsensitiveLike(db *gorm.DB, keyword string, field string, otherFields ...string) *gorm.DB {
	fields := []string{}
	if field != "" {
		fields = append(fields, field)
	}
	for _, f := range otherFields {
		if f != "" {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		return db
	}
	condition := ""
	args := []any{}
	for i, f := range fields {
		if i > 0 {
			condition += " OR "
		}
		condition += "LOWER(" + f + ") LIKE ?"
		args = append(args, "%"+strings.ToLower(keyword)+"%")
	}
	return db.Where(condition, args...)
}
