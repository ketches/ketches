package app

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrClusterNotFound = errors.New("cluster not found")
	ErrBadRequest      = errors.New("bad request")
)

func NewErrorf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	return errors.New(message)
}

func WrapError(message string, err error) error {
	if err == nil {
		return errors.New(message)
	}
	return errors.Join(errors.New(message), err)
}

func WrapErrorf(err error, format string, args ...any) error {
	return WrapError(fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...), err)
}
