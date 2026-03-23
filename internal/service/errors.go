package service

import "errors"

// Sentinel errors for HTTP mapping.
var (
	ErrNotFound = errors.New("link not found")
	ErrConflict = errors.New("short code already exists")
)

// InputError is a client-side validation problem.
type InputError struct {
	msg string
}

func (e *InputError) Error() string {
	return e.msg
}

// BadInput wraps a validation message as an InputError.
func BadInput(msg string) error {
	return &InputError{msg: msg}
}
