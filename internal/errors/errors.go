// Package errors provides structured error types for the GateKey application.
// It defines error codes that clients can use to handle errors programmatically,
// and provides a consistent error response format across all API endpoints.
package errors

import (
	"fmt"
	"net/http"
)

// Error codes for client consumption.
// These codes are stable and can be relied upon for programmatic error handling.
const (
	// Authentication errors (AUTH_*)
	CodeInvalidCredentials  = "AUTH_INVALID_CREDENTIALS"
	CodeSessionExpired      = "AUTH_SESSION_EXPIRED"
	CodeSessionNotFound     = "AUTH_SESSION_NOT_FOUND"
	CodeTokenInvalid        = "AUTH_TOKEN_INVALID"
	CodeTokenExpired        = "AUTH_TOKEN_EXPIRED"
	CodeUserNotFound        = "AUTH_USER_NOT_FOUND"
	CodeUserDisabled        = "AUTH_USER_DISABLED"
	CodeProviderUnavailable = "AUTH_PROVIDER_UNAVAILABLE"
	CodeMFARequired         = "AUTH_MFA_REQUIRED"
	CodeMFAInvalid          = "AUTH_MFA_INVALID"

	// Authorization errors (AUTHZ_*)
	CodeForbidden         = "AUTHZ_FORBIDDEN"
	CodeInsufficientScope = "AUTHZ_INSUFFICIENT_SCOPE"
	CodeAdminRequired     = "AUTHZ_ADMIN_REQUIRED"

	// Validation errors (VALIDATION_*)
	CodeInvalidRequest = "VALIDATION_INVALID_REQUEST"
	CodeInvalidField   = "VALIDATION_INVALID_FIELD"
	CodeMissingField   = "VALIDATION_MISSING_FIELD"
	CodeInvalidFormat  = "VALIDATION_INVALID_FORMAT"
	CodeOutOfRange     = "VALIDATION_OUT_OF_RANGE"
	CodeInvalidJSON    = "VALIDATION_INVALID_JSON"

	// Resource errors (RESOURCE_*)
	CodeResourceNotFound = "RESOURCE_NOT_FOUND"
	CodeResourceExists   = "RESOURCE_EXISTS"
	CodeResourceConflict = "RESOURCE_CONFLICT"
	CodeResourceLocked   = "RESOURCE_LOCKED"

	// PKI errors (PKI_*)
	CodeCertificateInvalid  = "PKI_CERTIFICATE_INVALID"
	CodeCertificateExpired  = "PKI_CERTIFICATE_EXPIRED"
	CodeCertificateRevoked  = "PKI_CERTIFICATE_REVOKED"
	CodeCANotInitialized    = "PKI_CA_NOT_INITIALIZED"
	CodeKeyGenerationFailed = "PKI_KEY_GENERATION_FAILED"
	CodeSigningFailed       = "PKI_SIGNING_FAILED"
	CodeCRLGenerationFailed = "PKI_CRL_GENERATION_FAILED"

	// Database errors (DB_*)
	CodeDatabaseError       = "DB_ERROR"
	CodeConnectionFailed    = "DB_CONNECTION_FAILED"
	CodeQueryFailed         = "DB_QUERY_FAILED"
	CodeTransactionFailed   = "DB_TRANSACTION_FAILED"
	CodeConstraintViolation = "DB_CONSTRAINT_VIOLATION"
	CodeRecordNotFound      = "DB_RECORD_NOT_FOUND"

	// Gateway errors (GATEWAY_*)
	CodeGatewayNotFound       = "GATEWAY_NOT_FOUND"
	CodeGatewayUnreachable    = "GATEWAY_UNREACHABLE"
	CodeGatewayTokenInvalid   = "GATEWAY_TOKEN_INVALID"
	CodeGatewayNotProvisioned = "GATEWAY_NOT_PROVISIONED"

	// Rate limiting errors (RATE_*)
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	CodeTooManyRequests   = "RATE_TOO_MANY_REQUESTS"

	// Internal errors (INTERNAL_*)
	CodeInternalError      = "INTERNAL_ERROR"
	CodeServiceUnavailable = "INTERNAL_SERVICE_UNAVAILABLE"
	CodeTimeout            = "INTERNAL_TIMEOUT"
)

// AppError is the base application error type.
// It implements the error interface and provides structured error information.
type AppError struct {
	// Code is a stable error code for programmatic handling
	Code string `json:"code"`

	// Message is a human-readable error description
	Message string `json:"message"`

	// Details contains additional context about the error
	Details map[string]any `json:"details,omitempty"`

	// HTTPStatus is the HTTP status code to return (not serialized to JSON)
	HTTPStatus int `json:"-"`

	// Err is the underlying error (not serialized to JSON)
	Err error `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error and returns it for chaining.
func (e *AppError) WithDetails(key string, value any) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithError wraps an underlying error and returns it for chaining.
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// New creates a new AppError with the given code, message, and HTTP status.
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap wraps an existing error with an AppError.
func Wrap(err error, code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// --- Authentication Errors ---

// ErrInvalidCredentials returns an error for invalid username or password.
func ErrInvalidCredentials() *AppError {
	return New(CodeInvalidCredentials, "invalid username or password", http.StatusUnauthorized)
}

// ErrSessionExpired returns an error for expired sessions.
func ErrSessionExpired() *AppError {
	return New(CodeSessionExpired, "session has expired", http.StatusUnauthorized)
}

// ErrSessionNotFound returns an error when session is not found.
func ErrSessionNotFound() *AppError {
	return New(CodeSessionNotFound, "session not found", http.StatusUnauthorized)
}

// ErrTokenInvalid returns an error for invalid tokens.
func ErrTokenInvalid() *AppError {
	return New(CodeTokenInvalid, "invalid or malformed token", http.StatusUnauthorized)
}

// ErrTokenExpired returns an error for expired tokens.
func ErrTokenExpired() *AppError {
	return New(CodeTokenExpired, "token has expired", http.StatusUnauthorized)
}

// ErrUserNotFound returns an error when user is not found.
func ErrUserNotFound() *AppError {
	return New(CodeUserNotFound, "user not found", http.StatusNotFound)
}

// ErrUserDisabled returns an error when user account is disabled.
func ErrUserDisabled() *AppError {
	return New(CodeUserDisabled, "user account is disabled", http.StatusForbidden)
}

// ErrProviderUnavailable returns an error when auth provider is unavailable.
func ErrProviderUnavailable(provider string) *AppError {
	return New(CodeProviderUnavailable, "authentication provider unavailable", http.StatusServiceUnavailable).
		WithDetails("provider", provider)
}

// --- Authorization Errors ---

// ErrForbidden returns an error for forbidden access.
func ErrForbidden() *AppError {
	return New(CodeForbidden, "access denied", http.StatusForbidden)
}

// ErrInsufficientScope returns an error for insufficient permissions.
func ErrInsufficientScope(required string) *AppError {
	return New(CodeInsufficientScope, "insufficient permissions", http.StatusForbidden).
		WithDetails("required_scope", required)
}

// ErrAdminRequired returns an error when admin access is required.
func ErrAdminRequired() *AppError {
	return New(CodeAdminRequired, "administrator access required", http.StatusForbidden)
}

// --- Validation Errors ---

// ErrInvalidRequest returns an error for invalid requests.
func ErrInvalidRequest(reason string) *AppError {
	return New(CodeInvalidRequest, reason, http.StatusBadRequest)
}

// ErrInvalidField returns an error for invalid field values.
func ErrInvalidField(field, reason string) *AppError {
	return New(CodeInvalidField, fmt.Sprintf("invalid value for field '%s': %s", field, reason), http.StatusBadRequest).
		WithDetails("field", field).
		WithDetails("reason", reason)
}

// ErrMissingField returns an error for missing required fields.
func ErrMissingField(field string) *AppError {
	return New(CodeMissingField, fmt.Sprintf("missing required field '%s'", field), http.StatusBadRequest).
		WithDetails("field", field)
}

// ErrInvalidJSON returns an error for invalid JSON.
func ErrInvalidJSON(err error) *AppError {
	return Wrap(err, CodeInvalidJSON, "invalid JSON in request body", http.StatusBadRequest)
}

// --- Resource Errors ---

// ErrResourceNotFound returns an error when a resource is not found.
func ErrResourceNotFound(resourceType, identifier string) *AppError {
	return New(CodeResourceNotFound, fmt.Sprintf("%s not found", resourceType), http.StatusNotFound).
		WithDetails("resource_type", resourceType).
		WithDetails("identifier", identifier)
}

// ErrResourceExists returns an error when a resource already exists.
func ErrResourceExists(resourceType, identifier string) *AppError {
	return New(CodeResourceExists, fmt.Sprintf("%s already exists", resourceType), http.StatusConflict).
		WithDetails("resource_type", resourceType).
		WithDetails("identifier", identifier)
}

// ErrResourceConflict returns an error for resource conflicts.
func ErrResourceConflict(reason string) *AppError {
	return New(CodeResourceConflict, reason, http.StatusConflict)
}

// --- PKI Errors ---

// ErrCertificateInvalid returns an error for invalid certificates.
func ErrCertificateInvalid(reason string) *AppError {
	return New(CodeCertificateInvalid, reason, http.StatusBadRequest).
		WithDetails("reason", reason)
}

// ErrCertificateExpired returns an error for expired certificates.
func ErrCertificateExpired() *AppError {
	return New(CodeCertificateExpired, "certificate has expired", http.StatusUnauthorized)
}

// ErrCertificateRevoked returns an error for revoked certificates.
func ErrCertificateRevoked() *AppError {
	return New(CodeCertificateRevoked, "certificate has been revoked", http.StatusUnauthorized)
}

// ErrCANotInitialized returns an error when CA is not initialized.
func ErrCANotInitialized() *AppError {
	return New(CodeCANotInitialized, "certificate authority not initialized", http.StatusInternalServerError)
}

// --- Gateway Errors ---

// ErrGatewayNotFound returns an error when gateway is not found.
func ErrGatewayNotFound(identifier string) *AppError {
	return New(CodeGatewayNotFound, "gateway not found", http.StatusNotFound).
		WithDetails("identifier", identifier)
}

// ErrGatewayUnreachable returns an error when gateway is unreachable.
func ErrGatewayUnreachable(identifier string) *AppError {
	return New(CodeGatewayUnreachable, "gateway is unreachable", http.StatusBadGateway).
		WithDetails("identifier", identifier)
}

// ErrGatewayTokenInvalid returns an error for invalid gateway tokens.
func ErrGatewayTokenInvalid() *AppError {
	return New(CodeGatewayTokenInvalid, "invalid gateway token", http.StatusUnauthorized)
}

// --- Rate Limiting Errors ---

// ErrRateLimitExceeded returns an error when rate limit is exceeded.
func ErrRateLimitExceeded(retryAfter int) *AppError {
	return New(CodeRateLimitExceeded, "rate limit exceeded", http.StatusTooManyRequests).
		WithDetails("retry_after_seconds", retryAfter)
}

// --- Internal Errors ---

// ErrInternal returns an internal server error.
// The underlying error is logged but not exposed to clients.
func ErrInternal(err error) *AppError {
	return Wrap(err, CodeInternalError, "an internal error occurred", http.StatusInternalServerError)
}

// ErrServiceUnavailable returns a service unavailable error.
func ErrServiceUnavailable(service string) *AppError {
	return New(CodeServiceUnavailable, "service temporarily unavailable", http.StatusServiceUnavailable).
		WithDetails("service", service)
}

// ErrTimeout returns a timeout error.
func ErrTimeout(operation string) *AppError {
	return New(CodeTimeout, "operation timed out", http.StatusGatewayTimeout).
		WithDetails("operation", operation)
}

// --- Database Errors ---

// ErrDatabase returns a database error.
func ErrDatabase(err error) *AppError {
	return Wrap(err, CodeDatabaseError, "database error", http.StatusInternalServerError)
}

// ErrRecordNotFound returns an error when a database record is not found.
func ErrRecordNotFound(table string) *AppError {
	return New(CodeRecordNotFound, "record not found", http.StatusNotFound).
		WithDetails("table", table)
}
