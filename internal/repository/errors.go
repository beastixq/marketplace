package repository

import (
	"errors"
)

var (
	ErrToSql             = errors.New("Failed to ToSql")
	ErrToScan            = errors.New("Failed to Scan row")
	ErrExec              = errors.New("Failed to Exec")
	ErrQuery             = errors.New("Failed to Query")
	ErrRowsIteration     = errors.New("Error occured at the end of rows iteration")
	ErrNoChangesInUpdate = errors.New("All update fields are nil")
)
