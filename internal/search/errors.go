package search

import (
	"errors"
	"fmt"
)

var (
	ErrNoResults = errors.New("no results found")
)

type SearchExecutionError struct {
	Query string
	Err   error
}

func (e *SearchExecutionError) Error() string {
	return fmt.Sprintf("search failed for %q: %v", e.Query, e.Err)
}

func (e *SearchExecutionError) Unwrap() error { return e.Err }
