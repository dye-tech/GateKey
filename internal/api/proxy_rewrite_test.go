package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/config"
	"github.com/gatekey-project/gatekey/internal/db"
)

func TestReverseProxyRewrite(t *testing.T) {
	// A closed pool exercises the documented fallback to config settings without
	// requiring a database connection for this HTTP proxy regression test.
	pool, err := pgxpool.New(context.Background(), "postgres://localhost/gatekey_test")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	s := &Server{
		config:        &config.Config{},
		logger:        zap.NewNop(),
		settingsStore: db.NewSettingsStore(&db.DB{Pool: pool}),
	}

	for _, preserveHost := range []bool{false, true} {
		name := "target-host"
		if preserveHost {
			name = "original-host"
		}
		t.Run(name, func(t *testing.T) {
			requests := make(chan *http.Request, 1)
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.Clone(r.Context())
				w.WriteHeader(http.StatusNoContent)
			}))
			defer backend.Close()
			target, err := url.Parse(backend.URL + "/base")
			if err != nil {
				t.Fatal(err)
			}
			app := &db.ProxyApplication{
				StripPrefix:        true,
				PreserveHostHeader: preserveHost,
				InjectHeaders:      map[string]string{"X-App-Header": "configured"},
			}
			writer := httptest.NewRecorder()
			c, router := gin.CreateTestContext(writer)
			if err := router.SetTrustedProxies(nil); err != nil {
				t.Fatal(err)
			}
			c.Request = httptest.NewRequest(http.MethodGet, "https://vpn.example/proxy/demo/items?q=hello%20world", nil)
			c.Request.RemoteAddr = "192.0.2.10:1234"
			// A client must not be able to remove authenticated identity headers by
			// listing them in Connection, or inject an arbitrary forwarding chain.
			c.Request.Header.Set("Connection", "X-Forwarded-User, X-App-Header")
			c.Request.Header.Set("X-Forwarded-User", "attacker")
			c.Request.Header.Set("X-Forwarded-For", "198.51.100.20")
			c.Request.Header.Set("Forwarded", "for=198.51.100.20")
			proxy := s.createReverseProxy(app, target, "user-123", "user@example.com", []string{"staff", "ops"}, "demo", c)
			defer proxy.Transport.(*http.Transport).CloseIdleConnections()
			proxy.ServeHTTP(writer, c.Request)
			if writer.Code != http.StatusNoContent {
				t.Fatalf("proxy status = %d: %s", writer.Code, writer.Body.String())
			}
			r := <-requests
			if r.URL.Path != "/base/items" || r.URL.Query().Get("q") != "hello world" {
				t.Errorf("unexpected target URL: %s", r.URL)
			}
			wantHost := target.Host
			if preserveHost {
				wantHost = "vpn.example"
			}
			if r.Host != wantHost {
				t.Errorf("host = %q, want %q", r.Host, wantHost)
			}
			for header, want := range map[string]string{
				"X-Forwarded-User":   "user-123",
				"X-Forwarded-Email":  "user@example.com",
				"X-Forwarded-Groups": "staff,ops",
				"X-Forwarded-For":    "192.0.2.10",
				"X-Forwarded-Host":   "vpn.example",
				"X-App-Header":       "configured",
				"Connection":         "",
				"Forwarded":          "",
			} {
				if got := r.Header.Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}
