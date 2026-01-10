package wireguard

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewInterfaceManager(t *testing.T) {
	logger := zap.NewNop()
	manager := NewInterfaceManager("wg0", logger)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
	if manager.name != "wg0" {
		t.Errorf("Expected name 'wg0', got %q", manager.name)
	}
	if manager.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestNewInterfaceManager_NilLogger(t *testing.T) {
	manager := NewInterfaceManager("wg0", nil)

	if manager == nil {
		t.Fatal("Expected non-nil manager even with nil logger")
	}
	if manager.name != "wg0" {
		t.Errorf("Expected name 'wg0', got %q", manager.name)
	}
}

func TestInterfaceManager_GetName(t *testing.T) {
	logger := zap.NewNop()
	manager := NewInterfaceManager("wg-test", logger)

	if manager.GetName() != "wg-test" {
		t.Errorf("Expected GetName() to return 'wg-test', got %q", manager.GetName())
	}
}

func TestInterfaceManager_GetConfig_Empty(t *testing.T) {
	logger := zap.NewNop()
	manager := NewInterfaceManager("wg0", logger)

	config := manager.GetConfig()

	// Config should be zero-valued initially
	if config.PrivateKey != "" {
		t.Errorf("Expected empty PrivateKey, got %q", config.PrivateKey)
	}
	if config.ListenPort != 0 {
		t.Errorf("Expected ListenPort 0, got %d", config.ListenPort)
	}
	if config.Address != "" {
		t.Errorf("Expected empty Address, got %q", config.Address)
	}
}

func TestInterfaceConfig_Struct(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "test-private-key-base64",
		ListenPort: 51820,
		Address:    "10.0.0.1/24",
	}

	if config.PrivateKey != "test-private-key-base64" {
		t.Errorf("Expected PrivateKey 'test-private-key-base64', got %q", config.PrivateKey)
	}
	if config.ListenPort != 51820 {
		t.Errorf("Expected ListenPort 51820, got %d", config.ListenPort)
	}
	if config.Address != "10.0.0.1/24" {
		t.Errorf("Expected Address '10.0.0.1/24', got %q", config.Address)
	}
}

func TestInterfaceStats_Struct(t *testing.T) {
	stats := InterfaceStats{
		PublicKey:  "test-public-key",
		ListenPort: 51820,
		Peers: []InterfacePeerStats{
			{
				PublicKey: "peer-public-key",
				Endpoint:  "192.168.1.100:51820",
			},
		},
	}

	if stats.PublicKey != "test-public-key" {
		t.Errorf("Expected PublicKey 'test-public-key', got %q", stats.PublicKey)
	}
	if stats.ListenPort != 51820 {
		t.Errorf("Expected ListenPort 51820, got %d", stats.ListenPort)
	}
	if len(stats.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(stats.Peers))
	}
	if stats.Peers[0].PublicKey != "peer-public-key" {
		t.Errorf("Expected peer PublicKey 'peer-public-key', got %q", stats.Peers[0].PublicKey)
	}
}

func TestInterfacePeerStats_Struct(t *testing.T) {
	stats := InterfacePeerStats{
		PublicKey:           "peer-public-key",
		Endpoint:            "192.168.1.100:51820",
		AllowedIPs:          []string{"10.0.0.2/32", "192.168.1.0/24"},
		LatestHandshake:     1704067200,
		TransferRx:          1024000,
		TransferTx:          2048000,
		PersistentKeepalive: 25,
	}

	if stats.PublicKey != "peer-public-key" {
		t.Errorf("Expected PublicKey 'peer-public-key', got %q", stats.PublicKey)
	}
	if stats.Endpoint != "192.168.1.100:51820" {
		t.Errorf("Expected Endpoint '192.168.1.100:51820', got %q", stats.Endpoint)
	}
	if len(stats.AllowedIPs) != 2 {
		t.Errorf("Expected 2 AllowedIPs, got %d", len(stats.AllowedIPs))
	}
	if stats.TransferRx != 1024000 {
		t.Errorf("Expected TransferRx 1024000, got %d", stats.TransferRx)
	}
	if stats.TransferTx != 2048000 {
		t.Errorf("Expected TransferTx 2048000, got %d", stats.TransferTx)
	}
	if stats.PersistentKeepalive != 25 {
		t.Errorf("Expected PersistentKeepalive 25, got %d", stats.PersistentKeepalive)
	}
}

func TestParseWgShowDump_Basic(t *testing.T) {
	// Simulated output from "wg show wg0 dump"
	// Format: private-key\tpublic-key\tlisten-port\tfwmark
	// Then peer lines: public-key\tpreshared-key\tendpoint\tallowed-ips\tlatest-handshake\ttransfer-rx\ttransfer-tx\tpersistent-keepalive
	output := `(hidden)	abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGH=	51820	off
peerkey123456789012345678901234567890AB=	(none)	192.168.1.100:51820	10.0.0.2/32	1704067200	1024000	2048000	25`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	if stats.PublicKey != "abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGH=" {
		t.Errorf("Expected PublicKey, got %q", stats.PublicKey)
	}
	if stats.ListenPort != 51820 {
		t.Errorf("Expected ListenPort 51820, got %d", stats.ListenPort)
	}
	if len(stats.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(stats.Peers))
	}

	peer := stats.Peers[0]
	if peer.PublicKey != "peerkey123456789012345678901234567890AB=" {
		t.Errorf("Expected peer PublicKey, got %q", peer.PublicKey)
	}
	if peer.Endpoint != "192.168.1.100:51820" {
		t.Errorf("Expected Endpoint '192.168.1.100:51820', got %q", peer.Endpoint)
	}
	if peer.TransferRx != 1024000 {
		t.Errorf("Expected TransferRx 1024000, got %d", peer.TransferRx)
	}
	if peer.TransferTx != 2048000 {
		t.Errorf("Expected TransferTx 2048000, got %d", peer.TransferTx)
	}
	if peer.PersistentKeepalive != 25 {
		t.Errorf("Expected PersistentKeepalive 25, got %d", peer.PersistentKeepalive)
	}
}

func TestParseWgShowDump_MultiplePeers(t *testing.T) {
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
peer1key1234567890123456789012345678901234=	(none)	192.168.1.100:51820	10.0.0.2/32	1704067200	1024000	2048000	25
peer2key1234567890123456789012345678901234=	(none)	192.168.1.101:51820	10.0.0.3/32	1704067100	512000	1024000	25
peer3key1234567890123456789012345678901234=	(none)	(none)	10.0.0.4/32	0	0	0	0`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	if len(stats.Peers) != 3 {
		t.Fatalf("Expected 3 peers, got %d", len(stats.Peers))
	}

	// Check third peer has empty endpoint
	if stats.Peers[2].Endpoint != "(none)" {
		t.Errorf("Expected empty endpoint for third peer, got %q", stats.Peers[2].Endpoint)
	}
}

func TestParseWgShowDump_NoPeers(t *testing.T) {
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	if len(stats.Peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(stats.Peers))
	}
	if stats.ListenPort != 51820 {
		t.Errorf("Expected ListenPort 51820, got %d", stats.ListenPort)
	}
}

func TestParseWgShowDump_MultipleAllowedIPs(t *testing.T) {
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
peerkey123456789012345678901234567890AB=	(none)	192.168.1.100:51820	10.0.0.2/32,192.168.1.0/24,172.16.0.0/16	1704067200	1024000	2048000	25`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	if len(stats.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(stats.Peers))
	}

	peer := stats.Peers[0]
	if len(peer.AllowedIPs) != 3 {
		t.Errorf("Expected 3 AllowedIPs, got %d: %v", len(peer.AllowedIPs), peer.AllowedIPs)
	}
}

func TestParseWgShowDump_NoAllowedIPs(t *testing.T) {
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
peerkey123456789012345678901234567890AB=	(none)	192.168.1.100:51820	(none)	1704067200	1024000	2048000	25`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	if len(stats.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(stats.Peers))
	}

	peer := stats.Peers[0]
	if len(peer.AllowedIPs) != 0 {
		t.Errorf("Expected 0 AllowedIPs for (none), got %d", len(peer.AllowedIPs))
	}
}

func TestParseWgShowDump_Empty(t *testing.T) {
	// Empty string results in []string{""} after TrimSpace+Split, not an empty slice
	// So the function returns an empty stats struct rather than an error
	stats, err := parseWgShowDump("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Stats should be essentially empty
	if stats.PublicKey != "" {
		t.Errorf("Expected empty PublicKey, got %q", stats.PublicKey)
	}
	if stats.ListenPort != 0 {
		t.Errorf("Expected ListenPort 0, got %d", stats.ListenPort)
	}
	if len(stats.Peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(stats.Peers))
	}
}

func TestParseWgShowDump_WithIPv6(t *testing.T) {
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
peerkey123456789012345678901234567890AB=	(none)	[2001:db8::1]:51820	10.0.0.2/32,fd00::2/128	1704067200	1024000	2048000	25`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	if len(stats.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(stats.Peers))
	}

	peer := stats.Peers[0]
	if peer.Endpoint != "[2001:db8::1]:51820" {
		t.Errorf("Expected IPv6 endpoint, got %q", peer.Endpoint)
	}
	if len(peer.AllowedIPs) != 2 {
		t.Errorf("Expected 2 AllowedIPs, got %d", len(peer.AllowedIPs))
	}
}

func TestInterfaceManager_DifferentNames(t *testing.T) {
	names := []string{"wg0", "wg1", "vpn0", "wireguard", "my-wg-interface"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			manager := NewInterfaceManager(name, nil)
			if manager.GetName() != name {
				t.Errorf("Expected name %q, got %q", name, manager.GetName())
			}
		})
	}
}

func TestInterfaceConfig_IPv6Address(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "test-private-key-base64",
		ListenPort: 51820,
		Address:    "fd00::1/64",
	}

	if config.Address != "fd00::1/64" {
		t.Errorf("Expected Address 'fd00::1/64', got %q", config.Address)
	}
}

func TestInterfaceConfig_DualStackAddress(t *testing.T) {
	// Note: WireGuard typically uses separate Address lines for dual-stack,
	// but this tests the struct can hold either format
	config := InterfaceConfig{
		PrivateKey: "test-private-key-base64",
		ListenPort: 51820,
		Address:    "10.0.0.1/24", // Primary address
	}

	if config.Address != "10.0.0.1/24" {
		t.Errorf("Expected Address '10.0.0.1/24', got %q", config.Address)
	}
}

func TestInterfacePeerStats_ZeroValues(t *testing.T) {
	// Test a peer that just connected (no handshake yet, no transfer)
	stats := InterfacePeerStats{
		PublicKey:           "new-peer-key",
		Endpoint:            "",
		AllowedIPs:          []string{"10.0.0.100/32"},
		LatestHandshake:     0,
		TransferRx:          0,
		TransferTx:          0,
		PersistentKeepalive: 0,
	}

	if stats.LatestHandshake != 0 {
		t.Errorf("Expected LatestHandshake 0, got %d", stats.LatestHandshake)
	}
	if stats.TransferRx != 0 {
		t.Errorf("Expected TransferRx 0, got %d", stats.TransferRx)
	}
}

func TestParseWgShowDump_ShortInterfaceLine(t *testing.T) {
	// Test with a short interface line (less than 3 parts)
	// The parser requires at least 3 parts to extract the public key
	output := `(hidden)	pubkey`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	// With only 2 parts, it won't parse the public key (requires >= 3 parts)
	if stats.PublicKey != "" {
		t.Errorf("Expected empty PublicKey with short line, got %q", stats.PublicKey)
	}
	if stats.ListenPort != 0 {
		t.Errorf("Expected ListenPort 0, got %d", stats.ListenPort)
	}
}

func TestParseWgShowDump_ShortPeerLine(t *testing.T) {
	// Test with short peer lines (less than 8 parts) - should be skipped
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
shortpeer	(none)	192.168.1.100:51820	10.0.0.2/32`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	// Short peer line should be skipped
	if len(stats.Peers) != 0 {
		t.Errorf("Expected 0 peers (short line skipped), got %d", len(stats.Peers))
	}
}

func TestParseWgShowDump_MixedPeerLines(t *testing.T) {
	// Test with mix of valid and invalid peer lines
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
shortpeer	(none)	192.168.1.100:51820
validpeer123456789012345678901234567890AB=	(none)	192.168.1.101:51820	10.0.0.3/32	1704067200	512000	1024000	25
anothershort	missing	fields`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	// Only one valid peer should be parsed
	if len(stats.Peers) != 1 {
		t.Fatalf("Expected 1 peer (others skipped), got %d", len(stats.Peers))
	}

	if stats.Peers[0].PublicKey != "validpeer123456789012345678901234567890AB=" {
		t.Errorf("Expected valid peer key, got %q", stats.Peers[0].PublicKey)
	}
}

func TestParseWgShowDump_OnlyWhitespace(t *testing.T) {
	// Test with only whitespace
	stats, err := parseWgShowDump("   \n\t\n   ")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stats.PublicKey != "" {
		t.Errorf("Expected empty PublicKey, got %q", stats.PublicKey)
	}
}

func TestParseWgShowDump_VeryLongAllowedIPs(t *testing.T) {
	// Test with many allowed IPs
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
peerkey123456789012345678901234567890AB=	(none)	192.168.1.100:51820	10.0.0.2/32,10.0.1.0/24,10.0.2.0/24,10.0.3.0/24,192.168.0.0/16,172.16.0.0/12	1704067200	1024000	2048000	25`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	if len(stats.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(stats.Peers))
	}

	peer := stats.Peers[0]
	if len(peer.AllowedIPs) != 6 {
		t.Errorf("Expected 6 AllowedIPs, got %d: %v", len(peer.AllowedIPs), peer.AllowedIPs)
	}
}

func TestParseWgShowDump_LargeNumbers(t *testing.T) {
	// Test with large transfer numbers
	output := `(hidden)	serverpubkey1234567890123456789012345678=	51820	off
peerkey123456789012345678901234567890AB=	(none)	192.168.1.100:51820	10.0.0.2/32	1704067200	999999999999	888888888888	25`

	stats, err := parseWgShowDump(output)
	if err != nil {
		t.Fatalf("parseWgShowDump failed: %v", err)
	}

	peer := stats.Peers[0]
	if peer.TransferRx != 999999999999 {
		t.Errorf("Expected TransferRx 999999999999, got %d", peer.TransferRx)
	}
	if peer.TransferTx != 888888888888 {
		t.Errorf("Expected TransferTx 888888888888, got %d", peer.TransferTx)
	}
}

func TestInterfaceStats_MultiplePeersWithDifferentStates(t *testing.T) {
	stats := InterfaceStats{
		PublicKey:  "server-key",
		ListenPort: 51820,
		Peers: []InterfacePeerStats{
			{
				PublicKey:           "active-peer",
				Endpoint:            "192.168.1.100:51820",
				AllowedIPs:          []string{"10.0.0.2/32"},
				LatestHandshake:     1704067200,
				TransferRx:          1024000,
				TransferTx:          2048000,
				PersistentKeepalive: 25,
			},
			{
				PublicKey:           "idle-peer",
				Endpoint:            "",
				AllowedIPs:          []string{"10.0.0.3/32"},
				LatestHandshake:     0,
				TransferRx:          0,
				TransferTx:          0,
				PersistentKeepalive: 0,
			},
			{
				PublicKey:           "roaming-peer",
				Endpoint:            "10.0.0.50:12345",
				AllowedIPs:          []string{"10.0.0.4/32", "192.168.100.0/24"},
				LatestHandshake:     1704060000,
				TransferRx:          512,
				TransferTx:          1024,
				PersistentKeepalive: 30,
			},
		},
	}

	if len(stats.Peers) != 3 {
		t.Fatalf("Expected 3 peers, got %d", len(stats.Peers))
	}

	// Verify each peer's state
	if stats.Peers[0].Endpoint == "" {
		t.Error("Active peer should have endpoint")
	}
	if stats.Peers[1].LatestHandshake != 0 {
		t.Error("Idle peer should have zero handshake")
	}
	if len(stats.Peers[2].AllowedIPs) != 2 {
		t.Errorf("Roaming peer should have 2 AllowedIPs, got %d", len(stats.Peers[2].AllowedIPs))
	}
}

func TestInterfaceConfig_HighPort(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "test-key",
		ListenPort: 65535,
		Address:    "10.0.0.1/24",
	}

	if config.ListenPort != 65535 {
		t.Errorf("Expected ListenPort 65535, got %d", config.ListenPort)
	}
}

func TestInterfaceConfig_LowPort(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "test-key",
		ListenPort: 1,
		Address:    "10.0.0.1/24",
	}

	if config.ListenPort != 1 {
		t.Errorf("Expected ListenPort 1, got %d", config.ListenPort)
	}
}
