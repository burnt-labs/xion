package indexer

import "errors"

var (
	ErrGrantNotFound     = errors.New("grant not found")
	ErrAllowanceNotFound = errors.New("allowance not found")
)
