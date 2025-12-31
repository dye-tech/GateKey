package api

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/db"
)

// Hop-by-hop headers that should not be forwarded
var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// Headers to strip from responses (security)
var stripResponseHeaders = []string{
	"X-Powered-By",
	"Server",
}

// handleProxyRequest is the main reverse proxy handler
func (s *Server) handleProxyRequest(c *gin.Context) {
	slug := c.Param("slug")
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	s.logger.Info("Proxy request received",
		zap.String("slug", slug),
		zap.String("path", path),
		zap.String("method", c.Request.Method),
		zap.String("client_ip", c.ClientIP()),
		zap.String("user_agent", c.Request.UserAgent()))

	// 1. Authenticate user
	userID, groups, err := s.getCurrentUserInfo(c)
	if err != nil {
		// For browser requests, redirect to login
		if isHTMLRequest(c) {
			returnURL := url.QueryEscape(c.Request.URL.String())
			c.Redirect(http.StatusFound, "/login?return_to="+returnURL)
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Get user email for headers
	userEmail := s.getUserEmailFromContext(c, userID)

	// 2. Check access permission
	hasAccess, app, err := s.proxyAppStore.CanUserAccessApp(c.Request.Context(), userID, groups, slug)
	if err != nil {
		s.logger.Error("Failed to check proxy access",
			zap.String("slug", slug),
			zap.String("user", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if !hasAccess || app == nil {
		s.logger.Warn("Proxy access denied",
			zap.String("slug", slug),
			zap.String("user", userID))
		if isHTMLRequest(c) {
			c.HTML(http.StatusForbidden, "", `
				<!DOCTYPE html>
				<html>
				<head><title>Access Denied</title></head>
				<body style="font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0;">
					<div style="text-align: center;">
						<h1>Access Denied</h1>
						<p>You don't have permission to access this application.</p>
						<a href="/" style="color: #2563eb;">Return to Dashboard</a>
					</div>
				</body>
				</html>
			`)
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if !app.IsActive {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "application is disabled"})
		return
	}

	// 3. Parse target URL
	targetURL, err := url.Parse(app.InternalURL)
	if err != nil {
		s.logger.Error("Invalid internal URL",
			zap.String("slug", slug),
			zap.String("url", app.InternalURL),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target URL"})
		return
	}

	// 4. Handle WebSocket upgrade
	if isWebSocketRequest(c) {
		if app.WebsocketEnabled {
			s.handleWebSocketProxy(c, app, targetURL, path, userID, userEmail, groups)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "websocket not enabled for this application"})
		}
		return
	}

	// 5. Set proxy context cookie for this slug (used by NoRoute handler for redirects)
	c.SetCookie("gatekey_proxy_context", slug, 3600, "/", "", false, false)

	// 6. Create and execute reverse proxy
	start := time.Now()

	s.logger.Info("Proxying request",
		zap.String("slug", slug),
		zap.String("path", path),
		zap.String("target_url", app.InternalURL),
		zap.String("target_host", targetURL.Host),
		zap.String("user", userID),
		zap.Bool("strip_prefix", app.StripPrefix),
		zap.Int("timeout_seconds", app.TimeoutSeconds))

	proxy := s.createReverseProxy(app, targetURL, userID, userEmail, groups, slug, c)

	// Serve the proxy request
	proxy.ServeHTTP(c.Writer, c.Request)

	s.logger.Info("Proxy request completed",
		zap.String("slug", slug),
		zap.String("path", path),
		zap.Int("status", c.Writer.Status()),
		zap.Duration("duration", time.Since(start)))

	// 6. Log the access (async)
	responseTime := time.Since(start)
	go s.logProxyAccessAsync(c, app, path, c.Writer.Status(), responseTime, userID, userEmail)
}

// createReverseProxy creates a configured reverse proxy for the application
func (s *Server) createReverseProxy(app *db.ProxyApplication, targetURL *url.URL, userID, userEmail string, groups []string, slug string, c *gin.Context) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Set target scheme and host
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host

			// Handle path
			originalPath := req.URL.Path
			if app.StripPrefix {
				// Strip /proxy/{slug} prefix
				req.URL.Path = strings.TrimPrefix(originalPath, "/proxy/"+slug)
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
			}
			// Prepend target base path if any
			if targetURL.Path != "" && targetURL.Path != "/" {
				req.URL.Path = singleJoiningSlash(targetURL.Path, req.URL.Path)
			}

			// Handle host header
			if app.PreserveHostHeader {
				req.Host = c.Request.Host
			} else {
				req.Host = targetURL.Host
			}

			// Remove hop-by-hop headers
			for _, h := range hopByHopHeaders {
				req.Header.Del(h)
			}

			// Inject custom headers from app config
			for k, v := range app.InjectHeaders {
				req.Header.Set(k, v)
			}

			// Add forwarded headers for user identity
			req.Header.Set("X-Forwarded-User", userID)
			req.Header.Set("X-Forwarded-Email", userEmail)
			req.Header.Set("X-Forwarded-Groups", strings.Join(groups, ","))

			// Standard proxy headers
			if clientIP := c.ClientIP(); clientIP != "" {
				if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
					req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
				} else {
					req.Header.Set("X-Forwarded-For", clientIP)
				}
			}
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Forwarded-Host", c.Request.Host)
			req.Header.Set("X-Real-IP", c.ClientIP())

			// Force desktop User-Agent to avoid mobile-specific UIs that use CORS/WASM
			// which don't work well with cookie-based authentication through proxies
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

			s.logger.Debug("Proxying request",
				zap.String("slug", slug),
				zap.String("method", req.Method),
				zap.String("target", req.URL.String()))
		},
		ModifyResponse: func(resp *http.Response) error {
			proxyHost := c.Request.Host

			// Rewrite URLs in HTML responses for path-based routing
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "text/html") {
				// Get the current request path for resolving relative URLs
				currentPath := c.Request.URL.Path
				s.rewriteHTMLResponse(resp, slug, currentPath)
			}

			// Rewrite Location headers for redirects
			if location := resp.Header.Get("Location"); location != "" {
				resp.Header.Set("Location", s.rewriteLocationHeader(location, slug, targetURL, proxyHost))
			}

			// Rewrite Set-Cookie headers to scope to proxy path
			if cookies := resp.Header.Values("Set-Cookie"); len(cookies) > 0 {
				resp.Header.Del("Set-Cookie")
				for _, cookie := range cookies {
					resp.Header.Add("Set-Cookie", s.rewriteSetCookieHeader(cookie, slug))
				}
			}

			// Strip internal headers
			for _, h := range stripResponseHeaders {
				resp.Header.Del(h)
			}

			// Add security headers
			resp.Header.Set("X-Frame-Options", "SAMEORIGIN")
			resp.Header.Set("X-Content-Type-Options", "nosniff")
			resp.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			s.logger.Error("Proxy error",
				zap.String("slug", slug),
				zap.String("path", r.URL.Path),
				zap.String("target_host", targetURL.Host),
				zap.String("target_scheme", targetURL.Scheme),
				zap.String("method", r.Method),
				zap.String("error_type", fmt.Sprintf("%T", err)),
				zap.Error(err))

			// Determine error type for appropriate response
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.logger.Error("Proxy timeout", zap.String("slug", slug), zap.Bool("is_timeout", netErr.Timeout()))
				http.Error(w, "Application timeout - please try again", http.StatusGatewayTimeout)
			} else if strings.Contains(err.Error(), "connection refused") {
				http.Error(w, "Application unavailable - connection refused", http.StatusBadGateway)
			} else if strings.Contains(err.Error(), "context canceled") {
				s.logger.Warn("Proxy request canceled by client", zap.String("slug", slug))
				http.Error(w, "Request canceled", http.StatusBadGateway)
			} else {
				http.Error(w, "Application unavailable: "+err.Error(), http.StatusBadGateway)
			}
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Allow self-signed certs for internal services
			},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: time.Duration(app.TimeoutSeconds) * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return proxy
}

// rewriteHTMLResponse rewrites URLs in HTML content for path-based proxy routing
func (s *Server) rewriteHTMLResponse(resp *http.Response, slug string, currentPath string) {
	// Check if response is gzipped
	isGzip := resp.Header.Get("Content-Encoding") == "gzip"
	s.logger.Info("HTML rewrite starting",
		zap.String("slug", slug),
		zap.Bool("is_gzip", isGzip),
		zap.Int64("original_content_length", resp.ContentLength),
		zap.String("content_type", resp.Header.Get("Content-Type")),
		zap.String("transfer_encoding", resp.Header.Get("Transfer-Encoding")))

	// Read the raw body first to see what we're getting
	rawBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		s.logger.Warn("Failed to read raw response body", zap.Error(err))
		return
	}

	s.logger.Info("Raw body read",
		zap.String("slug", slug),
		zap.Int("raw_size", len(rawBody)),
		zap.Bool("is_gzip", isGzip))

	// Decompress if needed
	var body []byte
	if isGzip {
		gzReader, err := gzip.NewReader(bytes.NewReader(rawBody))
		if err != nil {
			s.logger.Warn("Failed to create gzip reader", zap.Error(err))
			// Fall back to raw body
			body = rawBody
		} else {
			body, err = io.ReadAll(gzReader)
			gzReader.Close()
			if err != nil {
				s.logger.Warn("Failed to decompress gzip response", zap.Error(err))
				body = rawBody
			}
		}
	} else {
		body = rawBody
	}

	s.logger.Info("HTML body read",
		zap.String("slug", slug),
		zap.Int("body_size", len(body)))

	// Rewrite absolute paths
	proxyPrefix := "/proxy/" + slug

	// Calculate base directory for resolving relative paths
	// currentPath is like "/proxy/nas/portal/" - we need the directory part
	baseDir := currentPath
	if !strings.HasSuffix(baseDir, "/") {
		// If path is like /proxy/nas/portal (no trailing slash), get directory
		if idx := strings.LastIndex(baseDir, "/"); idx > 0 {
			baseDir = baseDir[:idx]
		}
	} else {
		// Remove trailing slash for processing
		baseDir = strings.TrimSuffix(baseDir, "/")
	}

	s.logger.Info("HTML rewrite path info",
		zap.String("slug", slug),
		zap.String("current_path", currentPath),
		zap.String("base_dir", baseDir))

	// Rewrite src="/...", href="/...", action="/..." to include proxy prefix
	// Only rewrite absolute paths that don't already have the proxy prefix
	patterns := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		// Double-quoted attributes with absolute paths
		{regexp.MustCompile(`(src|href|action)="(/[^"]*?)"`), `$1="` + proxyPrefix + `$2"`},
		// Single-quoted attributes with absolute paths
		{regexp.MustCompile(`(src|href|action)='(/[^']*?)'`), `$1='` + proxyPrefix + `$2'`},
		// url() in inline CSS
		{regexp.MustCompile(`url\("(/[^"]*?)"\)`), `url("` + proxyPrefix + `$1")`},
		{regexp.MustCompile(`url\('(/[^']*?)'\)`), `url('` + proxyPrefix + `$1')`},
		{regexp.MustCompile(`url\((/[^)]*?)\)`), `url(` + proxyPrefix + `$1)`},
		// JavaScript import statements with absolute paths
		{regexp.MustCompile(`import\s+(\w+)\s+from\s+"(/[^"]+)"`), `import $1 from "` + proxyPrefix + `$2"`},
		{regexp.MustCompile(`import\s+(\w+)\s+from\s+'(/[^']+)'`), `import $1 from '` + proxyPrefix + `$2'`},
		// JavaScript fetch() calls with absolute paths
		{regexp.MustCompile(`fetch\("(/[^"]+)"\)`), `fetch("` + proxyPrefix + `$1")`},
		{regexp.MustCompile(`fetch\('(/[^']+)'\)`), `fetch('` + proxyPrefix + `$1')`},
	}

	content := string(body)
	originalContent := content
	rewriteCount := 0

	// First pass: rewrite absolute paths
	for _, p := range patterns {
		content = p.pattern.ReplaceAllStringFunc(content, func(match string) string {
			// Don't double-prefix if already has /proxy/
			if strings.Contains(match, "/proxy/") {
				return match
			}
			rewriteCount++
			return p.pattern.ReplaceAllString(match, p.replacement)
		})
	}

	// Second pass: rewrite relative paths that start with ../
	// These need to be resolved relative to the current page's directory
	relativePatterns := []struct {
		pattern *regexp.Regexp
		quote   string
	}{
		{regexp.MustCompile(`(src|href|action)="(\.\./[^"]*?)"`), `"`},
		{regexp.MustCompile(`(src|href|action)='(\.\./[^']*?)'`), `'`},
	}

	for _, p := range relativePatterns {
		content = p.pattern.ReplaceAllStringFunc(content, func(match string) string {
			// Extract the relative path
			submatch := p.pattern.FindStringSubmatch(match)
			if len(submatch) < 3 {
				return match
			}
			attr := submatch[1]
			relativePath := submatch[2]

			// Resolve the relative path
			resolved := resolveRelativePath(baseDir, relativePath)
			rewriteCount++

			s.logger.Debug("Resolving relative path",
				zap.String("base", baseDir),
				zap.String("relative", relativePath),
				zap.String("resolved", resolved))

			return attr + `=` + p.quote + resolved + p.quote
		})
	}

	s.logger.Info("HTML rewrite completed",
		zap.String("slug", slug),
		zap.Int("original_size", len(originalContent)),
		zap.Int("new_size", len(content)),
		zap.Int("rewrites", rewriteCount))

	// Create new body
	newBody := []byte(content)
	if isGzip {
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		gzWriter.Write(newBody)
		gzWriter.Close()
		newBody = buf.Bytes()
	} else {
		// Remove Content-Encoding if we're not re-gzipping
		resp.Header.Del("Content-Encoding")
	}

	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
}

// rewriteLocationHeader rewrites redirect Location headers
func (s *Server) rewriteLocationHeader(location, slug string, targetURL *url.URL, proxyHost string) string {
	locURL, err := url.Parse(location)
	if err != nil {
		return location
	}

	// Only rewrite if it's pointing to the target host
	if locURL.Host == targetURL.Host || locURL.Host == "" {
		proxyPrefix := "/proxy/" + slug

		// Absolute path or relative path on same host
		if locURL.IsAbs() {
			// Full URL - rewrite to our proxy
			newPath := proxyPrefix + locURL.Path
			// Preserve query string and rewrite any redirect URLs within it
			if locURL.RawQuery != "" {
				newPath += "?" + s.rewriteQueryParams(locURL.RawQuery, slug, targetURL, proxyHost)
			}
			return newPath
		} else if strings.HasPrefix(location, "/") {
			// Absolute path
			return proxyPrefix + location
		}
	}

	return location
}

// rewriteQueryParams rewrites URLs embedded in query parameters
func (s *Server) rewriteQueryParams(query, slug string, targetURL *url.URL, proxyHost string) string {
	params, err := url.ParseQuery(query)
	if err != nil {
		return query
	}

	// Common OAuth/redirect parameter names
	redirectParams := []string{"return_url", "redirect_uri", "redirect_url", "callback", "return", "next", "continue", "destination"}

	proxyBase := "https://" + proxyHost + "/proxy/" + slug
	targetBase := targetURL.Scheme + "://" + targetURL.Host

	for _, param := range redirectParams {
		if val := params.Get(param); val != "" {
			// Decode the value if it's URL encoded
			decoded, err := url.QueryUnescape(val)
			if err != nil {
				decoded = val
			}

			// If the redirect points back to our proxy, that's fine
			if strings.HasPrefix(decoded, proxyBase) {
				continue
			}

			// If it points to the target, rewrite to our proxy
			if strings.HasPrefix(decoded, targetBase) || strings.HasPrefix(decoded, "/") {
				var newVal string
				if strings.HasPrefix(decoded, "/") {
					newVal = proxyBase + decoded
				} else {
					// Replace target base with proxy base
					newVal = strings.Replace(decoded, targetBase, proxyBase, 1)
				}
				params.Set(param, newVal)
			}
		}
	}

	return params.Encode()
}

// rewriteSetCookieHeader rewrites Set-Cookie headers for path-based proxy
func (s *Server) rewriteSetCookieHeader(cookie, slug string) string {
	proxyPath := "/proxy/" + slug

	// If cookie has Path=/, change it to our proxy path
	// This ensures cookies set by the app are scoped to the proxy path
	if strings.Contains(cookie, "Path=/;") || strings.HasSuffix(cookie, "Path=/") {
		cookie = strings.Replace(cookie, "Path=/;", "Path="+proxyPath+";", 1)
		cookie = strings.Replace(cookie, "Path=/", "Path="+proxyPath, 1)
	} else if !strings.Contains(strings.ToLower(cookie), "path=") {
		// Add path if not present
		cookie = cookie + "; Path=" + proxyPath
	}

	return cookie
}

// handleWebSocketProxy handles WebSocket connections
func (s *Server) handleWebSocketProxy(c *gin.Context, app *db.ProxyApplication, targetURL *url.URL, path, userID, userEmail string, groups []string) {
	// Build WebSocket target URL
	wsScheme := "ws"
	if targetURL.Scheme == "https" {
		wsScheme = "wss"
	}

	targetPath := path
	if app.StripPrefix {
		targetPath = strings.TrimPrefix(path, "/proxy/"+app.Slug)
		if targetPath == "" {
			targetPath = "/"
		}
	}

	wsURL := fmt.Sprintf("%s://%s%s", wsScheme, targetURL.Host, singleJoiningSlash(targetURL.Path, targetPath))
	if c.Request.URL.RawQuery != "" {
		wsURL += "?" + c.Request.URL.RawQuery
	}

	s.logger.Debug("Proxying WebSocket",
		zap.String("slug", app.Slug),
		zap.String("target", wsURL))

	// Dial the target WebSocket
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	targetConn, err := dialer.Dial("tcp", targetURL.Host)
	if err != nil {
		s.logger.Error("WebSocket dial failed", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to connect to application"})
		return
	}
	defer targetConn.Close()

	// Hijack the client connection
	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		s.logger.Error("Hijacking not supported")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "websocket not supported"})
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		s.logger.Error("Failed to hijack connection", zap.Error(err))
		return
	}
	defer clientConn.Close()

	// Forward the WebSocket handshake
	// Write the original request to target
	req := c.Request
	req.URL.Path = targetPath
	req.Host = targetURL.Host
	req.Header.Set("X-Forwarded-User", userID)
	req.Header.Set("X-Forwarded-Email", userEmail)
	req.Header.Set("X-Forwarded-Groups", strings.Join(groups, ","))

	if err := req.Write(targetConn); err != nil {
		s.logger.Error("Failed to forward WebSocket handshake", zap.Error(err))
		return
	}

	// Bidirectional copy
	errChan := make(chan error, 2)
	go func() {
		_, err := io.Copy(targetConn, clientConn)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, targetConn)
		errChan <- err
	}()

	// Wait for either direction to finish
	<-errChan
}

// logProxyAccessAsync logs proxy access asynchronously
func (s *Server) logProxyAccessAsync(c *gin.Context, app *db.ProxyApplication, path string, status int, duration time.Duration, userID, userEmail string) {
	log := &db.ProxyAccessLog{
		ProxyAppID:     app.ID,
		UserID:         userID,
		UserEmail:      userEmail,
		RequestMethod:  c.Request.Method,
		RequestPath:    path,
		ResponseStatus: status,
		ResponseTimeMs: int(duration.Milliseconds()),
		ClientIP:       c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	}

	if err := s.proxyAppStore.LogProxyAccess(c.Request.Context(), log); err != nil {
		s.logger.Warn("Failed to log proxy access", zap.Error(err))
	}
}

// Helper functions

func isHTMLRequest(c *gin.Context) bool {
	accept := c.Request.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

func isWebSocketRequest(c *gin.Context) bool {
	return strings.ToLower(c.Request.Header.Get("Upgrade")) == "websocket"
}

func (s *Server) getUserEmailFromContext(c *gin.Context, userID string) string {
	user, err := s.getAuthenticatedUser(c)
	if err != nil {
		return userID
	}
	if user.Email != "" {
		return user.Email
	}
	return userID
}

// handleProxyContextRedirect handles requests that don't match any route
// It redirects them to the appropriate proxy based on Referer header or cookie
func (s *Server) handleProxyContextRedirect(c *gin.Context) {
	requestPath := c.Request.URL.Path

	// Skip known non-proxy paths (these should go to frontend or return 404)
	if strings.HasPrefix(requestPath, "/api/") ||
		strings.HasPrefix(requestPath, "/proxy/") ||
		strings.HasPrefix(requestPath, "/health") ||
		strings.HasPrefix(requestPath, "/ready") ||
		strings.HasPrefix(requestPath, "/metrics") ||
		strings.HasPrefix(requestPath, "/login") ||
		strings.HasPrefix(requestPath, "/scripts/") ||
		strings.HasPrefix(requestPath, "/downloads/") ||
		strings.HasPrefix(requestPath, "/bin/") ||
		// Frontend routes - these should NOT be redirected to proxy
		requestPath == "/" ||
		strings.HasPrefix(requestPath, "/admin") ||
		strings.HasPrefix(requestPath, "/web-access") ||
		strings.HasPrefix(requestPath, "/dashboard") ||
		strings.HasPrefix(requestPath, "/settings") ||
		strings.HasPrefix(requestPath, "/gateways") ||
		strings.HasPrefix(requestPath, "/assets/") ||
		strings.HasPrefix(requestPath, "/static/") ||
		strings.HasPrefix(requestPath, "/favicon") ||
		strings.HasSuffix(requestPath, ".js") && !strings.Contains(requestPath, "/") ||
		strings.HasSuffix(requestPath, ".css") && !strings.Contains(requestPath, "/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var slug string

	// First, try to extract proxy context from Referer header
	referer := c.Request.Header.Get("Referer")
	if referer != "" {
		refURL, err := url.Parse(referer)
		if err == nil && strings.HasPrefix(refURL.Path, "/proxy/") {
			parts := strings.SplitN(strings.TrimPrefix(refURL.Path, "/proxy/"), "/", 2)
			if len(parts) > 0 && parts[0] != "" {
				slug = parts[0]
			}
		}
	}

	// Fall back to cookie if no Referer
	if slug == "" {
		if cookie, err := c.Cookie("gatekey_proxy_context"); err == nil && cookie != "" {
			slug = cookie
		}
	}

	// If we have a proxy context, redirect to the proxy path
	if slug != "" {
		newPath := "/proxy/" + slug + requestPath
		if c.Request.URL.RawQuery != "" {
			newPath += "?" + c.Request.URL.RawQuery
		}

		s.logger.Info("Redirecting to proxy context",
			zap.String("slug", slug),
			zap.String("original", requestPath),
			zap.String("new_path", newPath),
			zap.String("referer", referer))

		// Use 307 to preserve the request method (important for POST/PUT)
		c.Redirect(http.StatusTemporaryRedirect, newPath)
		return
	}

	// No proxy context found, return 404
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// resolveRelativePath resolves a relative path against a base directory
// For example: base="/proxy/nas/portal", relative="../libs/ext.js" -> "/proxy/nas/libs/ext.js"
func resolveRelativePath(base, relative string) string {
	// Split base into path components
	parts := strings.Split(strings.Trim(base, "/"), "/")

	// Process the relative path
	relParts := strings.Split(relative, "/")
	for _, part := range relParts {
		if part == ".." {
			// Go up one directory
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
		} else if part != "." && part != "" {
			// Add the path component
			parts = append(parts, part)
		}
	}

	return "/" + strings.Join(parts, "/")
}
