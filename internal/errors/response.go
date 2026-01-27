package errors

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the structured JSON error response format.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains the error details.
type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// RespondWithError writes an AppError as a JSON response.
// It sets the appropriate HTTP status code and returns a structured error response.
//
// Usage:
//
//	if err != nil {
//	    apperrors.RespondWithError(c, apperrors.ErrInvalidCredentials())
//	    return
//	}
func RespondWithError(c *gin.Context, err *AppError) {
	c.JSON(err.HTTPStatus, ErrorResponse{
		Error: ErrorBody{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
	})
}

// AbortWithError aborts the request with an AppError.
// This is useful in middleware where you want to stop further processing.
//
// Usage:
//
//	if !authorized {
//	    apperrors.AbortWithError(c, apperrors.ErrForbidden())
//	    return
//	}
func AbortWithError(c *gin.Context, err *AppError) {
	c.AbortWithStatusJSON(err.HTTPStatus, ErrorResponse{
		Error: ErrorBody{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
	})
}

// HandleError inspects an error and responds appropriately.
// If the error is an *AppError, it uses its code and status.
// Otherwise, it wraps it as an internal error.
//
// Usage:
//
//	if err := service.DoSomething(); err != nil {
//	    apperrors.HandleError(c, err)
//	    return
//	}
func HandleError(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		RespondWithError(c, appErr)
		return
	}
	// Wrap unknown errors as internal errors
	RespondWithError(c, ErrInternal(err))
}

// IsAppError checks if an error is an *AppError with a specific code.
//
// Usage:
//
//	if apperrors.IsAppError(err, apperrors.CodeUserNotFound) {
//	    // Handle user not found case
//	}
func IsAppError(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// GetAppError extracts an *AppError from an error chain.
// Returns nil if no AppError is found.
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}
