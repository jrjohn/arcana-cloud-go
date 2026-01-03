package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name: "error without wrapped error",
			appErr: &AppError{
				Code:    CodeNotFound,
				Message: "resource not found",
			},
			expected: "resource not found",
		},
		{
			name: "error with wrapped error",
			appErr: &AppError{
				Code:    CodeInternalError,
				Message: "internal error",
				Err:     errors.New("database connection failed"),
			},
			expected: "internal error: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appErr.Error(); got != tt.expected {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := &AppError{
		Code:    CodeInternalError,
		Message: "wrapped error",
		Err:     originalErr,
	}

	if unwrapped := appErr.Unwrap(); unwrapped != originalErr {
		t.Errorf("AppError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// Test without wrapped error
	appErrNoWrap := &AppError{
		Code:    CodeBadRequest,
		Message: "no wrap",
	}
	if unwrapped := appErrNoWrap.Unwrap(); unwrapped != nil {
		t.Errorf("AppError.Unwrap() = %v, want nil", unwrapped)
	}
}

func TestNew(t *testing.T) {
	appErr := New(CodeBadRequest, "bad request test", http.StatusBadRequest)

	if appErr.Code != CodeBadRequest {
		t.Errorf("New() Code = %v, want %v", appErr.Code, CodeBadRequest)
	}
	if appErr.Message != "bad request test" {
		t.Errorf("New() Message = %v, want %v", appErr.Message, "bad request test")
	}
	if appErr.Status != http.StatusBadRequest {
		t.Errorf("New() Status = %v, want %v", appErr.Status, http.StatusBadRequest)
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("connection timeout")
	wrapped := Wrap(originalErr, ErrInternalError)

	if wrapped.Code != ErrInternalError.Code {
		t.Errorf("Wrap() Code = %v, want %v", wrapped.Code, ErrInternalError.Code)
	}
	if wrapped.Err != originalErr {
		t.Errorf("Wrap() Err = %v, want %v", wrapped.Err, originalErr)
	}
	if wrapped.Status != ErrInternalError.Status {
		t.Errorf("Wrap() Status = %v, want %v", wrapped.Status, ErrInternalError.Status)
	}
}

func TestAppError_WithMessage(t *testing.T) {
	original := ErrNotFound
	customMessage := "user with ID 123 not found"

	withMsg := original.WithMessage(customMessage)

	if withMsg.Message != customMessage {
		t.Errorf("WithMessage() Message = %v, want %v", withMsg.Message, customMessage)
	}
	if withMsg.Code != original.Code {
		t.Errorf("WithMessage() Code = %v, want %v", withMsg.Code, original.Code)
	}
	if withMsg.Status != original.Status {
		t.Errorf("WithMessage() Status = %v, want %v", withMsg.Status, original.Status)
	}
	// Original should be unchanged
	if original.Message == customMessage {
		t.Error("Original error was modified")
	}
}

func TestAppError_WithError(t *testing.T) {
	original := ErrInternalError
	wrappedErr := errors.New("database error")

	withErr := original.WithError(wrappedErr)

	if withErr.Err != wrappedErr {
		t.Errorf("WithError() Err = %v, want %v", withErr.Err, wrappedErr)
	}
	if withErr.Code != original.Code {
		t.Errorf("WithError() Code = %v, want %v", withErr.Code, original.Code)
	}
	// Original should be unchanged
	if original.Err != nil {
		t.Error("Original error was modified")
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   *AppError
		expected bool
	}{
		{
			name:     "same error",
			err:      ErrNotFound,
			target:   ErrNotFound,
			expected: true,
		},
		{
			name:     "wrapped error with same code",
			err:      Wrap(errors.New("original"), ErrNotFound),
			target:   ErrNotFound,
			expected: true,
		},
		{
			name:     "different error codes",
			err:      ErrBadRequest,
			target:   ErrNotFound,
			expected: false,
		},
		{
			name:     "non-AppError",
			err:      errors.New("plain error"),
			target:   ErrNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			target:   ErrNotFound,
			expected: false,
		},
		{
			name:     "wrapped in fmt.Errorf",
			err:      fmt.Errorf("wrapped: %w", ErrUnauthorized),
			target:   ErrUnauthorized,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.target); got != tt.expected {
				t.Errorf("Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "NotFound error",
			err:      ErrNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "BadRequest error",
			err:      ErrBadRequest,
			expected: http.StatusBadRequest,
		},
		{
			name:     "Unauthorized error",
			err:      ErrUnauthorized,
			expected: http.StatusUnauthorized,
		},
		{
			name:     "Forbidden error",
			err:      ErrForbidden,
			expected: http.StatusForbidden,
		},
		{
			name:     "Conflict error",
			err:      ErrConflict,
			expected: http.StatusConflict,
		},
		{
			name:     "InternalError",
			err:      ErrInternalError,
			expected: http.StatusInternalServerError,
		},
		{
			name:     "ServiceUnavailable error",
			err:      ErrServiceUnavailable,
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "wrapped AppError",
			err:      fmt.Errorf("wrapped: %w", ErrNotFound),
			expected: http.StatusNotFound,
		},
		{
			name:     "plain error",
			err:      errors.New("plain error"),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStatus(tt.err); got != tt.expected {
				t.Errorf("GetStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// Verify all common errors have expected values
	tests := []struct {
		name   string
		err    *AppError
		code   string
		status int
	}{
		{"ErrNotFound", ErrNotFound, CodeNotFound, http.StatusNotFound},
		{"ErrBadRequest", ErrBadRequest, CodeBadRequest, http.StatusBadRequest},
		{"ErrUnauthorized", ErrUnauthorized, CodeUnauthorized, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, CodeForbidden, http.StatusForbidden},
		{"ErrConflict", ErrConflict, CodeConflict, http.StatusConflict},
		{"ErrInternalError", ErrInternalError, CodeInternalError, http.StatusInternalServerError},
		{"ErrServiceUnavailable", ErrServiceUnavailable, CodeServiceUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("%s Code = %v, want %v", tt.name, tt.err.Code, tt.code)
			}
			if tt.err.Status != tt.status {
				t.Errorf("%s Status = %v, want %v", tt.name, tt.err.Status, tt.status)
			}
		})
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify error codes are as expected
	codes := map[string]string{
		"CodeNotFound":           CodeNotFound,
		"CodeBadRequest":         CodeBadRequest,
		"CodeUnauthorized":       CodeUnauthorized,
		"CodeForbidden":          CodeForbidden,
		"CodeConflict":           CodeConflict,
		"CodeInternalError":      CodeInternalError,
		"CodeValidationError":    CodeValidationError,
		"CodeServiceUnavailable": CodeServiceUnavailable,
	}

	expected := map[string]string{
		"CodeNotFound":           "NOT_FOUND",
		"CodeBadRequest":         "BAD_REQUEST",
		"CodeUnauthorized":       "UNAUTHORIZED",
		"CodeForbidden":          "FORBIDDEN",
		"CodeConflict":           "CONFLICT",
		"CodeInternalError":      "INTERNAL_ERROR",
		"CodeValidationError":    "VALIDATION_ERROR",
		"CodeServiceUnavailable": "SERVICE_UNAVAILABLE",
	}

	for name, code := range codes {
		if code != expected[name] {
			t.Errorf("%s = %v, want %v", name, code, expected[name])
		}
	}
}

func TestAppError_ErrorsAs(t *testing.T) {
	// Test that errors.As works correctly with AppError
	appErr := &AppError{
		Code:    CodeNotFound,
		Message: "test error",
		Status:  http.StatusNotFound,
	}

	var target *AppError
	if !errors.As(appErr, &target) {
		t.Error("errors.As should return true for *AppError")
	}

	if target.Code != appErr.Code {
		t.Errorf("errors.As target Code = %v, want %v", target.Code, appErr.Code)
	}
}

// Benchmarks
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(CodeBadRequest, "benchmark error", http.StatusBadRequest)
	}
}

func BenchmarkWrap(b *testing.B) {
	err := errors.New("original error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Wrap(err, ErrInternalError)
	}
}

func BenchmarkIs(b *testing.B) {
	err := Wrap(errors.New("test"), ErrNotFound)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Is(err, ErrNotFound)
	}
}

func BenchmarkGetStatus(b *testing.B) {
	err := Wrap(errors.New("test"), ErrNotFound)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetStatus(err)
	}
}
