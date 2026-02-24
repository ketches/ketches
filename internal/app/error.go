package app

import "errors"

var (
	ErrClusterNotFound = errors.New("cluster not found")
	ErrBadRequest      = errors.New("bad request")
)
