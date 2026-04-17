package entities

import (
	"database/sql/driver"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type JSONBlob []byte

func (j JSONBlob) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONBlob) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
		return nil
	case string:
		*j = append((*j)[:0], []byte(v)...)
		return nil
	default:
		return fmt.Errorf("unsupported JSONBlob source type %T", value)
	}
}

func (JSONBlob) GormDataType() string {
	return "json"
}

func (JSONBlob) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	switch db.Dialector.Name() {
	case "postgres":
		return "JSONB"
	case "mysql", "sqlite":
		return "JSON"
	default:
		return "JSON"
	}
}
