package wireguard

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockExecutor is a configurable mock for testing.
type mockExecutor struct {
	runFunc    func(ctx context.Context, name string, args ...string) error
	outputFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
	lookPath   func(name string) (string, error)
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, name, args...)
	}
	return nil
}

func (m *mockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.outputFunc != nil {
		return m.outputFunc(ctx, name, args...)
	}
	return []byte{}, nil
}

func (m *mockExecutor) LookPath(name string) (string, error) {
	if m.lookPath != nil {
		return m.lookPath(name)
	}
	return "/usr/bin/" + name, nil
}

func TestDefaultExecutor(t *testing.T) {
	executor := DefaultExecutor()
	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}
}

func TestRealExecutor_LookPath_NotFound(t *testing.T) {
	executor := DefaultExecutor()
	_, err := executor.LookPath("nonexistent-command-12345")
	if err == nil {
		t.Error("Expected error for non-existent command")
	}
}

func TestInterfaceManager_Setup_Success(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			// First call: ip link show (should fail to indicate interface doesn't exist)
			if callCount == 0 {
				callCount++
				return errors.New("interface not found")
			}
			return nil
		},
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	config := InterfaceConfig{
		PrivateKey: "dGVzdC1wcml2YXRlLWtleS1iYXNlNjQtc3RyaW5nMTIz",
		ListenPort: 51820,
		Address:    "10.0.0.1/24",
	}

	err := manager.Setup(ctx, config)
	if err != nil {
		t.Errorf("Setup failed: %v", err)
	}
}

func TestInterfaceManager_Setup_CreateInterfaceFails(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			return errors.New("interface not found")
		},
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("command failed"), errors.New("ip link add failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	config := InterfaceConfig{
		PrivateKey: "dGVzdC1wcml2YXRlLWtleS1iYXNlNjQtc3RyaW5nMTIz",
		ListenPort: 51820,
		Address:    "10.0.0.1/24",
	}

	err := manager.Setup(ctx, config)
	if err == nil {
		t.Error("Expected error when createInterface fails")
	}
}

func TestInterfaceManager_Teardown_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			return nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.Teardown(ctx)
	if err != nil {
		t.Errorf("Teardown failed: %v", err)
	}
}

func TestInterfaceManager_Teardown_NotFound(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			return errors.New("Cannot find device \"wg-test\"")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.Teardown(ctx)
	// Should not return error if device not found
	if err != nil {
		t.Errorf("Teardown should not fail if device not found: %v", err)
	}
}

func TestInterfaceManager_Teardown_OtherError(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			return errors.New("some other error")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.Teardown(ctx)
	if err == nil {
		t.Error("Expected error for non 'cannot find device' error")
	}
}

func TestInterfaceManager_GetStats_Success(t *testing.T) {
	ctx := context.Background()

	wgOutput := "(hidden)\tabcdefghijklmnopqrstuvwxyz1234567890AB=\t51820\toff\n" +
		"peerkey123456789012345678901234567890AB=\t(none)\t192.168.1.100:51820\t10.0.0.2/32\t1704067200\t1024\t2048\t25"

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte(wgOutput), nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	stats, err := manager.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.PublicKey != "abcdefghijklmnopqrstuvwxyz1234567890AB=" {
		t.Errorf("Expected public key, got %q", stats.PublicKey)
	}
	if stats.ListenPort != 51820 {
		t.Errorf("Expected ListenPort 51820, got %d", stats.ListenPort)
	}
	if len(stats.Peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(stats.Peers))
	}
}

func TestInterfaceManager_GetStats_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, errors.New("wg command failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	_, err := manager.GetStats(ctx)
	if err == nil {
		t.Error("Expected error when wg show fails")
	}
}

func TestInterfaceManager_IsInterfaceUp_True(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			return nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	if !manager.IsInterfaceUp(ctx) {
		t.Error("Expected IsInterfaceUp to return true")
	}
}

func TestInterfaceManager_IsInterfaceUp_False(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			return errors.New("interface not up")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	if manager.IsInterfaceUp(ctx) {
		t.Error("Expected IsInterfaceUp to return false")
	}
}

func TestInterfaceManager_AddPeer_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	peer := PeerConfig{
		PublicKey:           "testpeerkey123456789012345678901234=",
		AllowedIPs:          []string{"10.0.0.2/32"},
		PersistentKeepalive: 25,
	}

	err := manager.AddPeer(ctx, peer)
	if err != nil {
		t.Errorf("AddPeer failed: %v", err)
	}
}

func TestInterfaceManager_AddPeer_WithPSK(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	peer := PeerConfig{
		PublicKey:    "testpeerkey123456789012345678901234=",
		PresharedKey: "pskkey12345678901234567890123456789=",
		AllowedIPs:   []string{"10.0.0.2/32"},
	}

	err := manager.AddPeer(ctx, peer)
	if err != nil {
		t.Errorf("AddPeer with PSK failed: %v", err)
	}
}

func TestInterfaceManager_AddPeer_WithEndpoint(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	peer := PeerConfig{
		PublicKey:  "testpeerkey123456789012345678901234=",
		AllowedIPs: []string{"10.0.0.2/32"},
		Endpoint:   "192.168.1.100:51820",
	}

	err := manager.AddPeer(ctx, peer)
	if err != nil {
		t.Errorf("AddPeer with endpoint failed: %v", err)
	}
}

func TestInterfaceManager_AddPeer_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("error"), errors.New("wg set failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	peer := PeerConfig{
		PublicKey:  "testpeerkey123456789012345678901234=",
		AllowedIPs: []string{"10.0.0.2/32"},
	}

	err := manager.AddPeer(ctx, peer)
	if err == nil {
		t.Error("Expected error when wg set fails")
	}
}

func TestInterfaceManager_RemovePeer_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.RemovePeer(ctx, "testpeerkey123456789012345678901234=")
	if err != nil {
		t.Errorf("RemovePeer failed: %v", err)
	}
}

func TestInterfaceManager_RemovePeer_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("error"), errors.New("wg set remove failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.RemovePeer(ctx, "testpeerkey123456789012345678901234=")
	if err == nil {
		t.Error("Expected error when wg set remove fails")
	}
}

func TestInterfaceManager_setPrivateKey_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	manager.config = InterfaceConfig{
		PrivateKey: "dGVzdC1wcml2YXRlLWtleS1iYXNlNjQtc3RyaW5nMTIz",
	}

	err := manager.setPrivateKey(ctx)
	if err != nil {
		t.Errorf("setPrivateKey failed: %v", err)
	}
}

func TestInterfaceManager_setPrivateKey_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("error"), errors.New("wg set private-key failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	manager.config = InterfaceConfig{
		PrivateKey: "dGVzdC1wcml2YXRlLWtleS1iYXNlNjQtc3RyaW5nMTIz",
	}

	err := manager.setPrivateKey(ctx)
	if err == nil {
		t.Error("Expected error when wg set private-key fails")
	}
}

func TestInterfaceManager_setListenPort_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	manager.config = InterfaceConfig{
		ListenPort: 51820,
	}

	err := manager.setListenPort(ctx)
	if err != nil {
		t.Errorf("setListenPort failed: %v", err)
	}
}

func TestInterfaceManager_setListenPort_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("error"), errors.New("wg set listen-port failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	manager.config = InterfaceConfig{
		ListenPort: 51820,
	}

	err := manager.setListenPort(ctx)
	if err == nil {
		t.Error("Expected error when wg set listen-port fails")
	}
}

func TestInterfaceManager_setAddress_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	manager.config = InterfaceConfig{
		Address: "10.0.0.1/24",
	}

	err := manager.setAddress(ctx)
	if err != nil {
		t.Errorf("setAddress failed: %v", err)
	}
}

func TestInterfaceManager_setAddress_FileExists(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("RTNETLINK answers: File exists"), errors.New("address exists")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	manager.config = InterfaceConfig{
		Address: "10.0.0.1/24",
	}

	err := manager.setAddress(ctx)
	if err != nil {
		t.Errorf("setAddress should not fail for 'File exists': %v", err)
	}
}

func TestInterfaceManager_setAddress_OtherError(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("some error"), errors.New("ip address add failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	manager.config = InterfaceConfig{
		Address: "10.0.0.1/24",
	}

	err := manager.setAddress(ctx)
	if err == nil {
		t.Error("Expected error for non 'File exists' error")
	}
}

func TestInterfaceManager_bringUp_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.bringUp(ctx)
	if err != nil {
		t.Errorf("bringUp failed: %v", err)
	}
}

func TestInterfaceManager_bringUp_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("error"), errors.New("ip link set up failed")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.bringUp(ctx)
	if err == nil {
		t.Error("Expected error when ip link set up fails")
	}
}

func TestInterfaceManager_createInterface_ExistsAndDeleted(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			callCount++
			// First call: ip link show - interface exists
			// Second call: ip link delete - success
			// Third call: ip link show (in loop) - success
			if callCount <= 2 {
				return nil
			}
			return errors.New("not found")
		},
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.createInterface(ctx)
	if err != nil {
		t.Errorf("createInterface failed: %v", err)
	}
}

func TestInterfaceManager_createInterface_WireguardGoFallback(t *testing.T) {
	ctx := context.Background()
	ipLinkCallCount := 0
	wgGoCallCount := 0

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			// ip link show calls
			if name == "ip" && len(args) > 0 && args[0] == "link" && len(args) > 1 && args[1] == "show" {
				ipLinkCallCount++
				// First call: interface doesn't exist
				// Later calls: check for interface creation
				if ipLinkCallCount == 1 {
					return errors.New("interface not found")
				}
				// After wireguard-go runs, interface exists
				return nil
			}
			return nil
		},
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// ip link add returns "Operation not supported"
			if name == "ip" && len(args) > 0 && args[0] == "link" && len(args) > 1 && args[1] == "add" {
				return []byte("Operation not supported"), errors.New("not supported")
			}
			// wireguard-go succeeds
			if name == "/usr/bin/wireguard-go" {
				wgGoCallCount++
				return []byte{}, nil
			}
			return []byte{}, nil
		},
		lookPath: func(name string) (string, error) {
			if name == "wireguard-go" {
				return "/usr/bin/wireguard-go", nil
			}
			return "", errors.New("not found")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.createInterface(ctx)
	if err != nil {
		t.Errorf("createInterface with wireguard-go fallback failed: %v", err)
	}
}

func TestInterfaceManager_createInterface_WireguardGoNotFound(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		runFunc: func(ctx context.Context, name string, args ...string) error {
			return errors.New("interface not found")
		},
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("Operation not supported"), errors.New("not supported")
		},
		lookPath: func(name string) (string, error) {
			return "", errors.New("wireguard-go not found")
		},
	}

	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)
	err := manager.createInterface(ctx)
	if err == nil {
		t.Error("Expected error when wireguard-go is not found")
	}
}

// PeerManager tests with mock executor

func TestPeerManager_AddPeer_WithMock_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)
	peer := &Peer{
		PublicKey:  "testpeerkey123456789012345678901234=",
		AllowedIPs: []string{"10.0.0.2/32"},
		UserEmail:  "test@example.com",
	}

	err := pm.AddPeer(ctx, peer)
	if err != nil {
		t.Errorf("AddPeer failed: %v", err)
	}

	// Verify peer was added to internal map
	if pm.PeerCount() != 1 {
		t.Errorf("Expected 1 peer, got %d", pm.PeerCount())
	}
}

func TestPeerManager_AddPeer_WithMock_WithPSK(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)
	peer := &Peer{
		PublicKey:    "testpeerkey123456789012345678901234=",
		PresharedKey: "pskkey12345678901234567890123456789=",
		AllowedIPs:   []string{"10.0.0.2/32"},
		UserEmail:    "test@example.com",
	}

	err := pm.AddPeer(ctx, peer)
	if err != nil {
		t.Errorf("AddPeer with PSK failed: %v", err)
	}
}

func TestPeerManager_AddPeer_WithMock_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("error"), errors.New("wg set failed")
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)
	peer := &Peer{
		PublicKey:  "testpeerkey123456789012345678901234=",
		AllowedIPs: []string{"10.0.0.2/32"},
	}

	err := pm.AddPeer(ctx, peer)
	if err == nil {
		t.Error("Expected error when wg set fails")
	}
}

func TestPeerManager_RemovePeer_WithMock_Success(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)
	// Add a peer first (directly to map since AddPeer also uses executor)
	pm.peers["testpeerkey123456789012345678901234="] = &Peer{
		PublicKey: "testpeerkey123456789012345678901234=",
	}

	err := pm.RemovePeer(ctx, "testpeerkey123456789012345678901234=")
	if err != nil {
		t.Errorf("RemovePeer failed: %v", err)
	}

	// Verify peer was removed
	if pm.PeerCount() != 0 {
		t.Errorf("Expected 0 peers after removal, got %d", pm.PeerCount())
	}
}

func TestPeerManager_RemovePeer_WithMock_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("error"), errors.New("wg set remove failed")
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)
	err := pm.RemovePeer(ctx, "testpeerkey123456789012345678901234=")
	if err == nil {
		t.Error("Expected error when wg set remove fails")
	}
}

func TestPeerManager_SyncPeers_WithMock_AddNew(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)

	// Sync with new peers
	authorizedPeers := []*Peer{
		{
			PublicKey:  "peer1key12345678901234567890123456=",
			AllowedIPs: []string{"10.0.0.2/32"},
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
		{
			PublicKey:  "peer2key12345678901234567890123456=",
			AllowedIPs: []string{"10.0.0.3/32"},
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
	}

	err := pm.SyncPeers(ctx, authorizedPeers)
	if err != nil {
		t.Errorf("SyncPeers failed: %v", err)
	}

	if pm.PeerCount() != 2 {
		t.Errorf("Expected 2 peers after sync, got %d", pm.PeerCount())
	}
}

func TestPeerManager_SyncPeers_WithMock_RemoveOld(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)
	// Add existing peers
	pm.peers["oldpeer12345678901234567890123456="] = &Peer{
		PublicKey: "oldpeer12345678901234567890123456=",
	}
	pm.peers["keeperpeer23456789012345678901234="] = &Peer{
		PublicKey: "keeperpeer23456789012345678901234=",
	}

	// Sync with only one peer (should remove oldpeer)
	authorizedPeers := []*Peer{
		{
			PublicKey:  "keeperpeer23456789012345678901234=",
			AllowedIPs: []string{"10.0.0.2/32"},
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
	}

	err := pm.SyncPeers(ctx, authorizedPeers)
	if err != nil {
		t.Errorf("SyncPeers failed: %v", err)
	}

	if pm.PeerCount() != 1 {
		t.Errorf("Expected 1 peer after sync, got %d", pm.PeerCount())
	}
}

func TestPeerManager_SyncPeers_WithMock_SkipExpired(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)

	// One valid, one expired peer
	authorizedPeers := []*Peer{
		{
			PublicKey:  "validpeer2345678901234567890123456=",
			AllowedIPs: []string{"10.0.0.2/32"},
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
		{
			PublicKey:  "expiredpeer345678901234567890123456",
			AllowedIPs: []string{"10.0.0.3/32"},
			ExpiresAt:  time.Now().Add(-24 * time.Hour), // Expired
		},
	}

	err := pm.SyncPeers(ctx, authorizedPeers)
	if err != nil {
		t.Errorf("SyncPeers failed: %v", err)
	}

	// Only valid peer should be added
	if pm.PeerCount() != 1 {
		t.Errorf("Expected 1 peer (expired skipped), got %d", pm.PeerCount())
	}
}

func TestPeerManager_SyncPeers_WithMock_UpdateAllowedIPs(t *testing.T) {
	ctx := context.Background()
	updateCalled := false

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			updateCalled = true
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)
	// Add existing peer with different AllowedIPs
	pm.peers["updatepeer2345678901234567890123456"] = &Peer{
		PublicKey:  "updatepeer2345678901234567890123456",
		AllowedIPs: []string{"10.0.0.2/32"},
	}

	// Sync with same peer but different AllowedIPs
	authorizedPeers := []*Peer{
		{
			PublicKey:  "updatepeer2345678901234567890123456",
			AllowedIPs: []string{"10.0.0.2/32", "192.168.1.0/24"}, // Additional network
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
	}

	err := pm.SyncPeers(ctx, authorizedPeers)
	if err != nil {
		t.Errorf("SyncPeers failed: %v", err)
	}

	if !updateCalled {
		t.Error("Expected wg command to be called for AllowedIPs update")
	}
}

func TestPeerManager_SyncPeers_WithMock_WithPSK(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecutor{
		outputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}

	pm := NewPeerManagerWithExecutor("wg0", mock)

	authorizedPeers := []*Peer{
		{
			PublicKey:    "pskpeer123456789012345678901234567=",
			PresharedKey: "pskkey12345678901234567890123456789=",
			AllowedIPs:   []string{"10.0.0.2/32"},
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		},
	}

	err := pm.SyncPeers(ctx, authorizedPeers)
	if err != nil {
		t.Errorf("SyncPeers with PSK failed: %v", err)
	}

	if pm.PeerCount() != 1 {
		t.Errorf("Expected 1 peer, got %d", pm.PeerCount())
	}
}

func TestNewPeerManagerWithExecutor(t *testing.T) {
	mock := &mockExecutor{}
	pm := NewPeerManagerWithExecutor("wg0", mock)

	if pm == nil {
		t.Fatal("Expected non-nil PeerManager")
	}
	if pm.interfaceName != "wg0" {
		t.Errorf("Expected interface name 'wg0', got '%s'", pm.interfaceName)
	}
	if pm.executor == nil {
		t.Error("Expected executor to be set")
	}
}

func TestNewInterfaceManagerWithExecutor(t *testing.T) {
	mock := &mockExecutor{}
	manager := NewInterfaceManagerWithExecutor("wg-test", nil, mock)

	if manager == nil {
		t.Fatal("Expected non-nil InterfaceManager")
	}
	if manager.GetName() != "wg-test" {
		t.Errorf("Expected name 'wg-test', got '%s'", manager.GetName())
	}
	if manager.executor == nil {
		t.Error("Expected executor to be set")
	}
}
