package service

import "fmt"

func wrapError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}

// serviceResult keeps failed operations from returning partial values and retains the error cause.
func serviceResult[T any](value T, err error, action string) (T, error) {
	if err != nil {
		var zero T
		return zero, wrapError(action, err)
	}
	return value, nil
}
