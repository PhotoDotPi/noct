package dep

import "errors"

type MissingDependencyError struct {
	Name string
	Err  error
}

func (e *MissingDependencyError) Error() string {
	return e.Name + " is required but was not found: " + e.Err.Error()
}

func (e *MissingDependencyError) Unwrap() error { return e.Err }

func IsMissing(err error) bool {
	var depErr *MissingDependencyError
	return errors.As(err, &depErr)
}
