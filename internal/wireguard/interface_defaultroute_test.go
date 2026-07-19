package wireguard

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// TestAddPeer_SkipsDefaultRoute verifies that AddPeer never issues an
// "ip route add" for a default route (0.0.0.0/0 or ::/0). Injecting a default
// route into the main table clobbers the host's real default route and creates
// a routing loop that prevents the WireGuard handshake from completing. Normal
// subnet AllowedIPs must still get their routes; /32 and /128 host routes are
// skipped.
func TestAddPeer_SkipsDefaultRoute(t *testing.T) {
	ctx := context.Background()

	var routeAdds []string
	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name == "ip" && len(args) >= 3 && args[0] == "route" && args[1] == "add" {
				routeAdds = append(routeAdds, args[2])
			}
			return []byte{}, nil
		},
	}

	m := NewInterfaceManagerWithExecutor("wg0", zap.NewNop(), mock)

	peer := PeerConfig{
		PublicKey: "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXl0ZXN0MTI9",
		Endpoint:  "hub.example.com:51820",
		AllowedIPs: []string{
			"0.0.0.0/0",     // default route -> must be skipped
			"::/0",          // IPv6 default route -> must be skipped
			"172.30.0.2/32", // host route -> skipped (single host)
			"10.0.0.0/16",   // real subnet -> must be routed
		},
	}

	if err := m.AddPeer(ctx, peer); err != nil {
		t.Fatalf("AddPeer returned error: %v", err)
	}

	for _, r := range routeAdds {
		if r == "0.0.0.0/0" || r == "::/0" {
			t.Errorf("AddPeer added a default route %q; it must never touch the default route", r)
		}
	}

	if !contains(routeAdds, "10.0.0.0/16") {
		t.Errorf("AddPeer did not add route for real subnet 10.0.0.0/16; got routes: %v", routeAdds)
	}
	if contains(routeAdds, "172.30.0.2/32") {
		t.Errorf("AddPeer added a /32 host route %q; host routes should be skipped", "172.30.0.2/32")
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.EqualFold(h, needle) {
			return true
		}
	}
	return false
}
