package repository

import (
	"errors"
)

var (
	ErrToSql  = errors.New("Failed to ToSql")
	ErrToScan = errors.New("Failed to Scan row")
)
