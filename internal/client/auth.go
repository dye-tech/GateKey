package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// AuthManager handles authentication for the client.
type AuthManager struct {
	config *Config
}

// TokenData holds the authentication token and metadata.
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserEmail    string    `json:"user_email,omitempty"`
	UserName     string    `json:"user_name,omitempty"`
}

// NewAuthManager creates a new authentication manager.
func NewAuthManager(config *Config) *AuthManager {
	return &AuthManager{config: config}
}

// Login performs browser-based authentication.
func (a *AuthManager) Login(ctx context.Context, noBrowser bool) error {
	if a.config.ServerURL == "" {
		return fmt.Errorf("server URL not configured")
	}

	// Find an available port for the callback server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Channel to receive the token
	tokenChan := make(chan *TokenData, 1)
	errChan := make(chan error, 1)

	// Start callback server
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.handleCallback(w, r, tokenChan, errChan)
		}),
	}

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Build login URL
	loginURL, err := url.Parse(a.config.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	loginURL.Path = "/api/v1/auth/cli/login"
	q := loginURL.Query()
	q.Set("callback", callbackURL)
	loginURL.RawQuery = q.Encode()

	fmt.Println("Authenticating with GateKey server...")

	if noBrowser {
		fmt.Println("\nOpen this URL in your browser to log in:")
		fmt.Printf("\n  %s\n\n", loginURL.String())
	} else {
		fmt.Println("Opening browser for authentication...")
		if err := openBrowser(loginURL.String()); err != nil {
			fmt.Printf("Could not open browser automatically.\n")
			fmt.Println("Please open this URL manually:")
			fmt.Printf("\n  %s\n\n", loginURL.String())
		}
	}

	fmt.Println("Waiting for authentication...")

	// Wait for callback or timeout
	select {
	case token := <-tokenChan:
		if err := a.saveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}
		fmt.Println("\nAuthentication successful!")
		if token.UserEmail != "" {
			fmt.Printf("Logged in as: %s\n", token.UserEmail)
		}
		return nil

	case err := <-errChan:
		return err

	case <-ctx.Done():
		return ctx.Err()

	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authentication timed out")
	}
}

// handleCallback processes the OAuth callback.
func (a *AuthManager) handleCallback(w http.ResponseWriter, r *http.Request, tokenChan chan<- *TokenData, errChan chan<- error) {
	if r.URL.Path != "/callback" {
		http.NotFound(w, r)
		return
	}

	// Check for error
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		if errDesc == "" {
			errDesc = errMsg
		}
		a.writeCallbackPage(w, false, errDesc)
		errChan <- fmt.Errorf("authentication failed: %s", errDesc)
		return
	}

	// Get token from query params or POST body
	var token TokenData

	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&token); err != nil {
			a.writeCallbackPage(w, false, "Invalid response from server")
			errChan <- fmt.Errorf("failed to decode token: %w", err)
			return
		}
	} else {
		token.AccessToken = r.URL.Query().Get("token")
		token.RefreshToken = r.URL.Query().Get("refresh_token")
		token.UserEmail = r.URL.Query().Get("email")
		token.UserName = r.URL.Query().Get("name")

		if expiresIn := r.URL.Query().Get("expires_in"); expiresIn != "" {
			var seconds int
			fmt.Sscanf(expiresIn, "%d", &seconds)
			token.ExpiresAt = time.Now().Add(time.Duration(seconds) * time.Second)
		}
	}

	if token.AccessToken == "" {
		a.writeCallbackPage(w, false, "No token received")
		errChan <- fmt.Errorf("no token received in callback")
		return
	}

	a.writeCallbackPage(w, true, "")
	tokenChan <- &token
}

// writeCallbackPage writes an HTML response for the callback.
func (a *AuthManager) writeCallbackPage(w http.ResponseWriter, success bool, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var html string
	if success {
		html = `<!DOCTYPE html>
<html>
<head>
    <title>GateKey - Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
               display: flex; justify-content: center; align-items: center;
               height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; padding: 40px; background: white;
                     border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .success { color: #10b981; font-size: 48px; margin-bottom: 20px; }
        h1 { color: #1f2937; margin: 0 0 10px 0; }
        p { color: #6b7280; }
    </style>
</head>
<body>
    <div class="container">
        <div class="success">✓</div>
        <h1>Authentication Successful</h1>
        <p>You can close this window and return to the terminal.</p>
    </div>
</body>
</html>`
	} else {
		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>GateKey - Authentication Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
               display: flex; justify-content: center; align-items: center;
               height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; padding: 40px; background: white;
                     border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .error { color: #ef4444; font-size: 48px; margin-bottom: 20px; }
        h1 { color: #1f2937; margin: 0 0 10px 0; }
        p { color: #6b7280; }
        .msg { color: #ef4444; font-family: monospace; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="error">✗</div>
        <h1>Authentication Failed</h1>
        <p>An error occurred during authentication.</p>
        <p class="msg">%s</p>
    </div>
</body>
</html>`, errMsg)
	}

	w.Write([]byte(html))
}

// Logout clears saved credentials.
func (a *AuthManager) Logout() error {
	tokenPath := a.config.TokenPath()

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token: %w", err)
	}

	fmt.Println("Logged out successfully.")
	return nil
}

// GetToken returns the saved token if valid.
func (a *AuthManager) GetToken() (*TokenData, error) {
	tokenPath := a.config.TokenPath()

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in. Run 'gatekey login' first")
		}
		return nil, fmt.Errorf("failed to read token: %w", err)
	}

	var token TokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check expiration
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("session expired. Run 'gatekey login' to re-authenticate")
	}

	return &token, nil
}

// saveToken writes the token to disk securely.
func (a *AuthManager) saveToken(token *TokenData) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	tokenPath := a.config.TokenPath()
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}

	return nil
}

// IsLoggedIn checks if there's a valid token.
func (a *AuthManager) IsLoggedIn() bool {
	token, err := a.GetToken()
	return err == nil && token != nil
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		// Try common browsers
		browsers := []string{"xdg-open", "sensible-browser", "x-www-browser", "gnome-open"}
		for _, browser := range browsers {
			if path, err := exec.LookPath(browser); err == nil {
				cmd = exec.Command(path, url)
				break
			}
		}
		if cmd == nil {
			return fmt.Errorf("no browser found")
		}
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// RefreshToken attempts to refresh the access token.
func (a *AuthManager) RefreshToken(ctx context.Context) error {
	token, err := a.GetToken()
	if err != nil {
		return err
	}

	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Build refresh URL
	refreshURL, err := url.Parse(a.config.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	refreshURL.Path = "/api/v1/auth/refresh"

	// Make refresh request
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL.String(),
		strings.NewReader(fmt.Sprintf(`{"refresh_token":"%s"}`, token.RefreshToken)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var newToken TokenData
	if err := json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Preserve user info if not in response
	if newToken.UserEmail == "" {
		newToken.UserEmail = token.UserEmail
	}
	if newToken.UserName == "" {
		newToken.UserName = token.UserName
	}

	return a.saveToken(&newToken)
}
