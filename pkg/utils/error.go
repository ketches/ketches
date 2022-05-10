package utils

import (
	"errors"
	"fmt"
)

func ResourceNotFound(resource string) error {
	return errors.New(fmt.Sprintf("%s not found", resource))
}
