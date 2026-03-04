package generators

import (
	"errors"
)

var (
	ErrHashPassword = errors.New("Failed to hash password")
	ErrToSql        = errors.New("Failed to ToSql (squirrel)")
	ErrQuery        = errors.New("Failed to Query")
	ErrScan         = errors.New("Failed to Scan")
	ErrReadRows     = errors.New("Failed to read rows")
	ErrCloseRows    = errors.New("Failed to close rows")
)
