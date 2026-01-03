package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Common error codes
const (
	CodeNotFound          = "NOT_FOUND"
	CodeBadRequest        = "BAD_REQUEST"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeForbidden         = "FORBIDDEN"
	CodeConflict          = "CONFLICT"
	CodeInternalError     = "INTERNAL_ERROR"
	CodeValidationError   = "VALIDATION_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// Common application errors
var (
	ErrNotFound           = &AppError{Code: CodeNotFound, Message: "resource not found", Status: http.StatusNotFound}
	ErrBadRequest         = &AppError{Code: CodeBadRequest, Message: "bad request", Status: http.StatusBadRequest}
	ErrUnauthorized       = &AppError{Code: CodeUnauthorized, Message: "unauthorized", Status: http.StatusUnauthorized}
	ErrForbidden          = &AppError{Code: CodeForbidden, Message: "forbidden", Status: http.StatusForbidden}
	ErrConflict           = &AppError{Code: CodeConflict, Message: "resource conflict", Status: http.StatusConflict}
	ErrInternalError      = &AppError{Code: CodeInternalError, Message: "internal server error", Status: http.StatusInternalServerError}
	ErrServiceUnavailable = &AppError{Code: CodeServiceUnavailable, Message: "service unavailable", Status: http.StatusServiceUnavailable}
)

// New creates a new AppError
func New(code string, message string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// Wrap wraps an error with an AppError
func Wrap(err error, appErr *AppError) *AppError {
	return &AppError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Status:  appErr.Status,
		Err:     err,
	}
}

// WithMessage returns a new AppError with a custom message
func (e *AppError) WithMessage(message string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: message,
		Status:  e.Status,
		Err:     e.Err,
	}
}

// WithError returns a new AppError with a wrapped error
func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Status:  e.Status,
		Err:     err,
	}
}

// Is checks if the error is a specific AppError
func Is(err error, target *AppError) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == target.Code
	}
	return false
}

// GetStatus returns the HTTP status from an error
func GetStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return http.StatusInternalServerError
}
