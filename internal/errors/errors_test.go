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
		err      *AppError
		expected string
	}{
		{
			name:     "without underlying error",
			err:      New(CodeInvalidCredentials, "invalid credentials", http.StatusUnauthorized),
			expected: "AUTH_INVALID_CREDENTIALS: invalid credentials",
		},
		{
			name:     "with underlying error",
			err:      Wrap(errors.New("connection refused"), CodeDatabaseError, "database error", http.StatusInternalServerError),
			expected: "DB_ERROR: database error: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	appErr := Wrap(underlying, CodeInternalError, "wrapped", http.StatusInternalServerError)

	if !errors.Is(appErr, underlying) {
		t.Error("errors.Is should find underlying error")
	}
}

func TestAppError_WithDetails(t *testing.T) {
	err := New(CodeInvalidField, "invalid field", http.StatusBadRequest).
		WithDetails("field", "email").
		WithDetails("reason", "invalid format")

	if err.Details["field"] != "email" {
		t.Errorf("Details[field] = %v, want email", err.Details["field"])
	}
	if err.Details["reason"] != "invalid format" {
		t.Errorf("Details[reason] = %v, want 'invalid format'", err.Details["reason"])
	}
}

func TestAppError_WithError(t *testing.T) {
	underlying := errors.New("original error")
	err := New(CodeInternalError, "internal error", http.StatusInternalServerError).
		WithError(underlying)

	if err.Err != underlying {
		t.Error("WithError should set underlying error")
	}
}

func TestErrInvalidCredentials(t *testing.T) {
	err := ErrInvalidCredentials()

	if err.Code != CodeInvalidCredentials {
		t.Errorf("Code = %s, want %s", err.Code, CodeInvalidCredentials)
	}
	if err.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusUnauthorized)
	}
}

func TestErrResourceNotFound(t *testing.T) {
	err := ErrResourceNotFound("user", "123")

	if err.Code != CodeResourceNotFound {
		t.Errorf("Code = %s, want %s", err.Code, CodeResourceNotFound)
	}
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusNotFound)
	}
	if err.Details["resource_type"] != "user" {
		t.Errorf("Details[resource_type] = %v, want user", err.Details["resource_type"])
	}
	if err.Details["identifier"] != "123" {
		t.Errorf("Details[identifier] = %v, want 123", err.Details["identifier"])
	}
}

func TestErrResourceExists(t *testing.T) {
	err := ErrResourceExists("user", "john@example.com")

	if err.Code != CodeResourceExists {
		t.Errorf("Code = %s, want %s", err.Code, CodeResourceExists)
	}
	if err.HTTPStatus != http.StatusConflict {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusConflict)
	}
}

func TestErrInvalidField(t *testing.T) {
	err := ErrInvalidField("email", "must be a valid email address")

	if err.Code != CodeInvalidField {
		t.Errorf("Code = %s, want %s", err.Code, CodeInvalidField)
	}
	if err.Details["field"] != "email" {
		t.Errorf("Details[field] = %v, want email", err.Details["field"])
	}
}

func TestErrMissingField(t *testing.T) {
	err := ErrMissingField("username")

	if err.Code != CodeMissingField {
		t.Errorf("Code = %s, want %s", err.Code, CodeMissingField)
	}
	if err.Details["field"] != "username" {
		t.Errorf("Details[field] = %v, want username", err.Details["field"])
	}
}

func TestErrInvalidJSON(t *testing.T) {
	underlying := errors.New("unexpected EOF")
	err := ErrInvalidJSON(underlying)

	if err.Code != CodeInvalidJSON {
		t.Errorf("Code = %s, want %s", err.Code, CodeInvalidJSON)
	}
	if !errors.Is(err, underlying) {
		t.Error("should wrap underlying error")
	}
}

func TestErrForbidden(t *testing.T) {
	err := ErrForbidden()

	if err.Code != CodeForbidden {
		t.Errorf("Code = %s, want %s", err.Code, CodeForbidden)
	}
	if err.HTTPStatus != http.StatusForbidden {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusForbidden)
	}
}

func TestErrInsufficientScope(t *testing.T) {
	err := ErrInsufficientScope("admin:write")

	if err.Code != CodeInsufficientScope {
		t.Errorf("Code = %s, want %s", err.Code, CodeInsufficientScope)
	}
	if err.Details["required_scope"] != "admin:write" {
		t.Errorf("Details[required_scope] = %v, want admin:write", err.Details["required_scope"])
	}
}

func TestErrRateLimitExceeded(t *testing.T) {
	err := ErrRateLimitExceeded(60)

	if err.Code != CodeRateLimitExceeded {
		t.Errorf("Code = %s, want %s", err.Code, CodeRateLimitExceeded)
	}
	if err.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusTooManyRequests)
	}
	if err.Details["retry_after_seconds"] != 60 {
		t.Errorf("Details[retry_after_seconds] = %v, want 60", err.Details["retry_after_seconds"])
	}
}

func TestErrInternal(t *testing.T) {
	underlying := errors.New("nil pointer dereference")
	err := ErrInternal(underlying)

	if err.Code != CodeInternalError {
		t.Errorf("Code = %s, want %s", err.Code, CodeInternalError)
	}
	if err.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusInternalServerError)
	}
	// Should not expose underlying error in message
	if err.Message == underlying.Error() {
		t.Error("should not expose underlying error in message")
	}
	// But should be unwrappable
	if !errors.Is(err, underlying) {
		t.Error("should wrap underlying error")
	}
}

func TestErrGatewayNotFound(t *testing.T) {
	err := ErrGatewayNotFound("gw-123")

	if err.Code != CodeGatewayNotFound {
		t.Errorf("Code = %s, want %s", err.Code, CodeGatewayNotFound)
	}
	if err.Details["identifier"] != "gw-123" {
		t.Errorf("Details[identifier] = %v, want gw-123", err.Details["identifier"])
	}
}

func TestErrCertificateInvalid(t *testing.T) {
	err := ErrCertificateInvalid("certificate is not a CA")

	if err.Code != CodeCertificateInvalid {
		t.Errorf("Code = %s, want %s", err.Code, CodeCertificateInvalid)
	}
	if err.Details["reason"] != "certificate is not a CA" {
		t.Errorf("Details[reason] = %v, want 'certificate is not a CA'", err.Details["reason"])
	}
}

func TestErrProviderUnavailable(t *testing.T) {
	err := ErrProviderUnavailable("oidc")

	if err.Code != CodeProviderUnavailable {
		t.Errorf("Code = %s, want %s", err.Code, CodeProviderUnavailable)
	}
	if err.Details["provider"] != "oidc" {
		t.Errorf("Details[provider] = %v, want oidc", err.Details["provider"])
	}
}

func TestIsAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{
			name:     "matching AppError",
			err:      ErrInvalidCredentials(),
			code:     CodeInvalidCredentials,
			expected: true,
		},
		{
			name:     "non-matching AppError",
			err:      ErrInvalidCredentials(),
			code:     CodeForbidden,
			expected: false,
		},
		{
			name:     "wrapped AppError",
			err:      fmt.Errorf("wrapped: %w", ErrInvalidCredentials()),
			code:     CodeInvalidCredentials,
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			code:     CodeInternalError,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			code:     CodeInternalError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAppError(tt.err, tt.code); got != tt.expected {
				t.Errorf("IsAppError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetAppError(t *testing.T) {
	t.Run("returns AppError", func(t *testing.T) {
		err := ErrInvalidCredentials()
		if got := GetAppError(err); got != err {
			t.Errorf("GetAppError() = %v, want %v", got, err)
		}
	})

	t.Run("returns wrapped AppError", func(t *testing.T) {
		appErr := ErrInvalidCredentials()
		wrapped := fmt.Errorf("wrapped: %w", appErr)
		if got := GetAppError(wrapped); got != appErr {
			t.Errorf("GetAppError() = %v, want %v", got, appErr)
		}
	})

	t.Run("returns nil for regular error", func(t *testing.T) {
		err := errors.New("regular error")
		if got := GetAppError(err); got != nil {
			t.Errorf("GetAppError() = %v, want nil", got)
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		if got := GetAppError(nil); got != nil {
			t.Errorf("GetAppError() = %v, want nil", got)
		}
	})
}
