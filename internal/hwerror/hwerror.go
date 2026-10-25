package hwerror

import "errors"

const (
	CodeItemUnavailable = "ITEM_UNAVAILABLE"
	CodeRequestConflict = "REQUEST_CONFLICT"
)

type Error struct {
	Code    string
	Message string
}

func New(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func (e *Error) Error() string {
	return e.Message + " [" + e.Code + "]"
}

func Code(err error) string {
	var hwErr *Error
	if errors.As(err, &hwErr) {
		return hwErr.Code
	}

	return ""
}
