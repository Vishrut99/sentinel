package domain

import "fmt"

// AppError represents a standardized API error payload.
type AppError struct {
	Code    string `json:"error"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is allows errors.Is comparisons by AppError code.
func (e *AppError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithErr returns a copy of the AppError with wrapped inner error.
func (e *AppError) WithErr(err error) *AppError {
	if e == nil {
		return nil
	}
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

var (
	ErrBadRequest = &AppError{
		Code:    "BAD_REQUEST",
		Message: "invalid request",
	}
	ErrUnauthorized = &AppError{
		Code:    "UNAUTHORIZED",
		Message: "unauthorized",
	}
	ErrForbidden = &AppError{
		Code:    "FORBIDDEN",
		Message: "forbidden",
	}
	ErrNotFound = &AppError{
		Code:    "NOT_FOUND",
		Message: "resource not found",
	}
	ErrConflict = &AppError{
		Code:    "CONFLICT",
		Message: "resource conflict",
	}
)
