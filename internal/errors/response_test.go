package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name           string
		err            *AppError
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "unauthorized error",
			err:            ErrInvalidCredentials(),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   CodeInvalidCredentials,
		},
		{
			name:           "not found error",
			err:            ErrResourceNotFound("user", "123"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   CodeResourceNotFound,
		},
		{
			name:           "bad request error",
			err:            ErrInvalidField("email", "invalid format"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   CodeInvalidField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			RespondWithError(c, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if resp.Error.Code != tt.expectedCode {
				t.Errorf("code = %s, want %s", resp.Error.Code, tt.expectedCode)
			}
		})
	}
}

func TestRespondWithError_WithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := ErrResourceNotFound("gateway", "gw-123")
	RespondWithError(c, err)

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Details["resource_type"] != "gateway" {
		t.Errorf("details[resource_type] = %v, want gateway", resp.Error.Details["resource_type"])
	}
	if resp.Error.Details["identifier"] != "gw-123" {
		t.Errorf("details[identifier] = %v, want gw-123", resp.Error.Details["identifier"])
	}
}

func TestAbortWithError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	AbortWithError(c, ErrForbidden())

	if !c.IsAborted() {
		t.Error("context should be aborted")
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != CodeForbidden {
		t.Errorf("code = %s, want %s", resp.Error.Code, CodeForbidden)
	}
}

func TestHandleError_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleError(c, ErrInvalidCredentials())

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != CodeInvalidCredentials {
		t.Errorf("code = %s, want %s", resp.Error.Code, CodeInvalidCredentials)
	}
}

func TestHandleError_RegularError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleError(c, errors.New("some internal error"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != CodeInternalError {
		t.Errorf("code = %s, want %s", resp.Error.Code, CodeInternalError)
	}

	// Should not expose internal error details
	if resp.Error.Message == "some internal error" {
		t.Error("should not expose internal error message")
	}
}

func TestErrorResponseJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := ErrRateLimitExceeded(30)
	RespondWithError(c, err)

	// Verify JSON structure
	expected := `{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"rate limit exceeded","details":{"retry_after_seconds":30}}}`

	var gotJSON, expectedJSON map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &gotJSON); err != nil {
		t.Fatalf("failed to unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("failed to unmarshal expected: %v", err)
	}

	gotError := gotJSON["error"].(map[string]any)
	expectedError := expectedJSON["error"].(map[string]any)

	if gotError["code"] != expectedError["code"] {
		t.Errorf("code mismatch: got %v, want %v", gotError["code"], expectedError["code"])
	}
	if gotError["message"] != expectedError["message"] {
		t.Errorf("message mismatch: got %v, want %v", gotError["message"], expectedError["message"])
	}
}

func TestErrorResponse_OmitsEmptyDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := ErrForbidden() // No details
	RespondWithError(c, err)

	body := w.Body.String()
	if contains(body, "details") {
		t.Errorf("response should omit empty details, got: %s", body)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
