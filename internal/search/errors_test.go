package search

import (
	"errors"
	"testing"
)

func TestSearchExecutionErrorWraps(t *testing.T) {
	cause := errors.New("exit status 1")
	err := &SearchExecutionError{Query: "lofi", Err: cause}
	if !errors.Is(err, cause) {
		t.Fatal("expected errors.Is to unwrap to the cause")
	}
	var se *SearchExecutionError
	if !errors.As(err, &se) {
		t.Fatal("expected errors.As to match SearchExecutionError")
	}
	if err.Error() == "" {
		t.Fatal("expected a non-empty message")
	}
}
