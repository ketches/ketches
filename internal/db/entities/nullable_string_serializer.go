package entities

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"gorm.io/gorm/schema"
)

const nullableStringSerializerName = "nullable_string"

// nullableStringSerializer maps SQL NULL to the zero value of a Go string and
// writes an empty Go string back as SQL NULL. This lets legacy rows use NULL
// without forcing every caller to handle a pointer or sql.NullString.
type nullableStringSerializer struct{}

func (nullableStringSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue any) error {
	value := ""
	switch typed := dbValue.(type) {
	case nil:
		// Keep the Go zero value for SQL NULL.
	case string:
		value = typed
	case []byte:
		value = string(typed)
	default:
		return errors.New(fmt.Sprintf("cannot scan %T into nullable string field %s", dbValue, field.Name))
	}

	field.ReflectValueOf(ctx, dst).SetString(value)
	return nil
}

func (nullableStringSerializer) Value(_ context.Context, _ *schema.Field, _ reflect.Value, fieldValue any) (any, error) {
	switch value := fieldValue.(type) {
	case nil:
		return nil, nil
	case string:
		if value == "" {
			return nil, nil
		}
		return value, nil
	default:
		return nil, errors.New(fmt.Sprintf("cannot serialize %T as nullable string", fieldValue))
	}
}

func init() {
	schema.RegisterSerializer(nullableStringSerializerName, nullableStringSerializer{})
}
