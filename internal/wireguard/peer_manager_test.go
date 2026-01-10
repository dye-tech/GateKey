package wireguard

import (
	"testing"
	"time"
)

func TestNewPeerManager(t *testing.T) {
	pm := NewPeerManager("wg0")

	if pm == nil {
		t.Fatal("Expected non-nil PeerManager")
	}
	if pm.interfaceName != "wg0" {
		t.Errorf("Expected interface name 'wg0', got '%s'", pm.interfaceName)
	}
	if pm.peers == nil {
		t.Error("Expected peers map to be initialized")
	}
}

func TestPeerManager_GetPeers_Empty(t *testing.T) {
	pm := NewPeerManager("wg0")

	peers := pm.GetPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(peers))
	}
}

func TestPeerManager_GetPeer_NotFound(t *testing.T) {
	pm := NewPeerManager("wg0")

	peer, found := pm.GetPeer("nonexistent-key")
	if found {
		t.Error("Expected peer not to be found")
	}
	if peer != nil {
		t.Error("Expected nil peer")
	}
}

func TestPeerManager_PeerCount_Empty(t *testing.T) {
	pm := NewPeerManager("wg0")

	if pm.PeerCount() != 0 {
		t.Errorf("Expected peer count 0, got %d", pm.PeerCount())
	}
}

func TestPeerManager_InternalPeersMap(t *testing.T) {
	pm := NewPeerManager("wg0")

	// Add a peer directly to the internal map (simulating what AddPeer does)
	pm.peers["test-public-key"] = &Peer{
		PublicKey:  "test-public-key",
		AllowedIPs: []string{"10.0.0.2/32"},
		UserID:     "user-1",
		UserEmail:  "test@example.com",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	// Test GetPeer
	peer, found := pm.GetPeer("test-public-key")
	if !found {
		t.Fatal("Expected peer to be found")
	}
	if peer.UserEmail != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", peer.UserEmail)
	}

	// Test GetPeers
	peers := pm.GetPeers()
	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	// Test PeerCount
	if pm.PeerCount() != 1 {
		t.Errorf("Expected peer count 1, got %d", pm.PeerCount())
	}
}

func TestPeerManager_GetPeer_ReturnsCopy(t *testing.T) {
	pm := NewPeerManager("wg0")

	pm.peers["test-key"] = &Peer{
		PublicKey: "test-key",
		UserEmail: "original@example.com",
	}

	// Get peer and modify it
	peer, _ := pm.GetPeer("test-key")
	peer.UserEmail = "modified@example.com"

	// Original should be unchanged
	original, _ := pm.GetPeer("test-key")
	if original.UserEmail != "original@example.com" {
		t.Error("GetPeer should return a copy, not the original")
	}
}

func TestPeerManager_GetPeers_ReturnsCopies(t *testing.T) {
	pm := NewPeerManager("wg0")

	pm.peers["test-key"] = &Peer{
		PublicKey: "test-key",
		UserEmail: "original@example.com",
	}

	// Get peers and modify
	peers := pm.GetPeers()
	peers[0].UserEmail = "modified@example.com"

	// Original should be unchanged
	originalPeers := pm.GetPeers()
	if originalPeers[0].UserEmail != "original@example.com" {
		t.Error("GetPeers should return copies, not originals")
	}
}

func TestPeer_Fields(t *testing.T) {
	expires := time.Now().Add(24 * time.Hour)
	peer := Peer{
		PublicKey:    "public-key-123",
		PresharedKey: "psk-456",
		AllowedIPs:   []string{"10.0.0.5/32", "fd00::5/128"},
		UserID:       "user-uuid",
		UserEmail:    "user@example.com",
		ConfigID:     "config-uuid",
		ExpiresAt:    expires,
	}

	if peer.PublicKey != "public-key-123" {
		t.Errorf("Expected public key 'public-key-123', got '%s'", peer.PublicKey)
	}
	if len(peer.AllowedIPs) != 2 {
		t.Errorf("Expected 2 allowed IPs, got %d", len(peer.AllowedIPs))
	}
	if peer.ExpiresAt != expires {
		t.Errorf("Expected expires at %v, got %v", expires, peer.ExpiresAt)
	}
}

func TestEqualStringSlices(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "equal single element",
			a:        []string{"10.0.0.1/32"},
			b:        []string{"10.0.0.1/32"},
			expected: true,
		},
		{
			name:     "equal multiple elements",
			a:        []string{"10.0.0.1/32", "10.0.0.2/32"},
			b:        []string{"10.0.0.1/32", "10.0.0.2/32"},
			expected: true,
		},
		{
			name:     "different length",
			a:        []string{"10.0.0.1/32"},
			b:        []string{"10.0.0.1/32", "10.0.0.2/32"},
			expected: false,
		},
		{
			name:     "different content",
			a:        []string{"10.0.0.1/32"},
			b:        []string{"10.0.0.2/32"},
			expected: false,
		},
		{
			name:     "same elements different order",
			a:        []string{"10.0.0.1/32", "10.0.0.2/32"},
			b:        []string{"10.0.0.2/32", "10.0.0.1/32"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalStringSlices(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalStringSlices(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestNewIPAllocator(t *testing.T) {
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}
	if allocator == nil {
		t.Fatal("Expected non-nil allocator")
	}
}

func TestNewIPAllocator_InvalidCIDR(t *testing.T) {
	_, err := NewIPAllocator("not-a-cidr")
	if err == nil {
		t.Error("Expected error for invalid CIDR")
	}
}

func TestIPAllocator_Allocate(t *testing.T) {
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	// First allocation should be .2 (skips .0 network and .1 server)
	ip1, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip1 != "10.0.0.2/32" {
		t.Errorf("Expected first IP '10.0.0.2/32', got '%s'", ip1)
	}

	// Second allocation should be .3
	ip2, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip2 != "10.0.0.3/32" {
		t.Errorf("Expected second IP '10.0.0.3/32', got '%s'", ip2)
	}
}

func TestIPAllocator_AllocateIPv6(t *testing.T) {
	allocator, err := NewIPAllocator("fd00::/120")
	if err != nil {
		t.Fatalf("Failed to create IPv6 allocator: %v", err)
	}

	ip1, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip1 != "fd00::2/32" {
		t.Errorf("Expected first IPv6 'fd00::2/32', got '%s'", ip1)
	}
}

func TestIPAllocator_Release(t *testing.T) {
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	ip1, _ := allocator.Allocate()
	ip2, _ := allocator.Allocate()

	// Release first IP
	allocator.Release(ip1)

	// Next allocation should reuse the released IP
	ip3, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed after release: %v", err)
	}
	if ip3 != ip1 {
		t.Errorf("Expected released IP to be reused, got '%s' instead of '%s'", ip3, ip1)
	}

	// Ensure ip2 is still allocated (just checking it's different)
	_ = ip2
}

func TestIPAllocator_Release_WithCIDR(t *testing.T) {
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	ip1, _ := allocator.Allocate() // "10.0.0.2/32"

	// Release with CIDR suffix
	allocator.Release("10.0.0.2/32")

	// Should be able to allocate the same IP again
	ip2, _ := allocator.Allocate()
	if ip2 != ip1 {
		t.Errorf("Expected released IP to be reused")
	}
}

func TestIPAllocator_MarkAllocated(t *testing.T) {
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	// Mark .2 and .3 as already allocated
	allocator.MarkAllocated("10.0.0.2")
	allocator.MarkAllocated("10.0.0.3/32") // with CIDR

	// First allocation should skip to .4
	ip, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip != "10.0.0.4/32" {
		t.Errorf("Expected '10.0.0.4/32' (skipping marked IPs), got '%s'", ip)
	}
}

func TestIPAllocator_ExhaustedSubnet(t *testing.T) {
	// Use a /30 subnet which has 4 IPs total
	allocator, err := NewIPAllocator("10.0.0.0/30")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	// /30 has 4 IPs: .0 (network), .1 (server), .2 (client), .3 (client/broadcast)
	// The allocator will allow .2 and .3 (it doesn't specifically check for broadcast)
	ip1, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("First allocate failed: %v", err)
	}
	if ip1 != "10.0.0.2/32" {
		t.Errorf("Expected '10.0.0.2/32', got '%s'", ip1)
	}

	ip2, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Second allocate failed: %v", err)
	}
	if ip2 != "10.0.0.3/32" {
		t.Errorf("Expected '10.0.0.3/32', got '%s'", ip2)
	}

	// Third allocation should fail (no IPs left in /30)
	_, err = allocator.Allocate()
	if err == nil {
		t.Error("Expected error when subnet is exhausted")
	}
}

func TestIncrementIP(t *testing.T) {
	// Test increment via the allocator behavior
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	// Allocate several IPs and verify the sequence
	expectedIPs := []string{
		"10.0.0.2/32",
		"10.0.0.3/32",
		"10.0.0.4/32",
		"10.0.0.5/32",
	}

	for i, expected := range expectedIPs {
		ip, err := allocator.Allocate()
		if err != nil {
			t.Fatalf("Allocate %d failed: %v", i, err)
		}
		if ip != expected {
			t.Errorf("Allocation %d: expected '%s', got '%s'", i, expected, ip)
		}
	}
}

func TestIncrementIP_LargeJump(t *testing.T) {
	// Test with a large number of allocations
	allocator, err := NewIPAllocator("10.0.0.0/16")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	// Allocate 256 IPs (should cross .255 boundary)
	var lastIP string
	for i := 0; i < 256; i++ {
		ip, err := allocator.Allocate()
		if err != nil {
			t.Fatalf("Allocate %d failed: %v", i, err)
		}
		lastIP = ip
	}

	// 256th allocation from .2 should be 10.0.1.1/32
	// (starting at .2, so 256 allocations = .2 + 255 = .257 which is 10.0.1.1)
	if lastIP != "10.0.1.1/32" {
		t.Errorf("Expected '10.0.1.1/32' after 256 allocations, got '%s'", lastIP)
	}
}

func TestIncrementIP_IPv6(t *testing.T) {
	// Test that IPv6 incrementing works through the allocator
	allocator, err := NewIPAllocator("fd00::/120")
	if err != nil {
		t.Fatalf("Failed to create IPv6 allocator: %v", err)
	}

	// First allocation should be fd00::2
	ip1, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("First allocate failed: %v", err)
	}
	if ip1 != "fd00::2/32" {
		t.Errorf("Expected 'fd00::2/32', got '%s'", ip1)
	}

	// Allocate a few more
	for i := 3; i <= 5; i++ {
		_, err := allocator.Allocate()
		if err != nil {
			t.Fatalf("Allocate %d failed: %v", i, err)
		}
	}

	// Fifth allocation should be fd00::6
	ip6, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Sixth allocate failed: %v", err)
	}
	if ip6 != "fd00::6/32" {
		t.Errorf("Expected 'fd00::6/32', got '%s'", ip6)
	}
}

func TestPeerManager_MultiplePeers(t *testing.T) {
	pm := NewPeerManager("wg0")

	// Add multiple peers directly
	for i := 0; i < 5; i++ {
		pm.peers["peer-key-"+string(rune('A'+i))] = &Peer{
			PublicKey:  "peer-key-" + string(rune('A'+i)),
			UserEmail:  "user" + string(rune('0'+i)) + "@example.com",
			AllowedIPs: []string{"10.0.0." + string(rune('2'+i)) + "/32"},
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
	}

	if pm.PeerCount() != 5 {
		t.Errorf("Expected 5 peers, got %d", pm.PeerCount())
	}

	peers := pm.GetPeers()
	if len(peers) != 5 {
		t.Errorf("Expected 5 peers from GetPeers, got %d", len(peers))
	}
}

func TestPeerManager_GetPeer_AllFields(t *testing.T) {
	pm := NewPeerManager("wg0")

	expiresAt := time.Now().Add(48 * time.Hour)
	pm.peers["full-peer"] = &Peer{
		PublicKey:    "full-peer",
		PresharedKey: "psk-value",
		AllowedIPs:   []string{"10.0.0.5/32", "192.168.1.0/24"},
		UserID:       "user-uuid-123",
		UserEmail:    "fulltest@example.com",
		ConfigID:     "config-uuid-456",
		ExpiresAt:    expiresAt,
	}

	peer, found := pm.GetPeer("full-peer")
	if !found {
		t.Fatal("Expected peer to be found")
	}

	if peer.PublicKey != "full-peer" {
		t.Errorf("Expected PublicKey 'full-peer', got '%s'", peer.PublicKey)
	}
	if peer.PresharedKey != "psk-value" {
		t.Errorf("Expected PresharedKey 'psk-value', got '%s'", peer.PresharedKey)
	}
	if len(peer.AllowedIPs) != 2 {
		t.Errorf("Expected 2 AllowedIPs, got %d", len(peer.AllowedIPs))
	}
	if peer.UserID != "user-uuid-123" {
		t.Errorf("Expected UserID 'user-uuid-123', got '%s'", peer.UserID)
	}
	if peer.UserEmail != "fulltest@example.com" {
		t.Errorf("Expected UserEmail 'fulltest@example.com', got '%s'", peer.UserEmail)
	}
	if peer.ConfigID != "config-uuid-456" {
		t.Errorf("Expected ConfigID 'config-uuid-456', got '%s'", peer.ConfigID)
	}
	if !peer.ExpiresAt.Equal(expiresAt) {
		t.Errorf("Expected ExpiresAt %v, got %v", expiresAt, peer.ExpiresAt)
	}
}

func TestIPAllocator_ReleaseNonExistent(t *testing.T) {
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	// Releasing a non-allocated IP should not panic
	allocator.Release("10.0.0.50/32")

	// Should still be able to allocate normally
	ip, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed after release: %v", err)
	}
	if ip != "10.0.0.2/32" {
		t.Errorf("Expected '10.0.0.2/32', got '%s'", ip)
	}
}

func TestIPAllocator_MarkAllocated_Multiple(t *testing.T) {
	allocator, err := NewIPAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create allocator: %v", err)
	}

	// Mark several IPs as allocated
	allocator.MarkAllocated("10.0.0.2")
	allocator.MarkAllocated("10.0.0.3")
	allocator.MarkAllocated("10.0.0.5")
	allocator.MarkAllocated("10.0.0.6")

	// First available should be .4
	ip1, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip1 != "10.0.0.4/32" {
		t.Errorf("Expected '10.0.0.4/32', got '%s'", ip1)
	}

	// Next available should be .7 (skipping .5 and .6)
	ip2, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip2 != "10.0.0.7/32" {
		t.Errorf("Expected '10.0.0.7/32', got '%s'", ip2)
	}
}

func TestEqualStringSlices_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "nil vs empty",
			a:        nil,
			b:        []string{},
			expected: true, // Both have length 0
		},
		{
			name:     "empty vs nil",
			a:        []string{},
			b:        nil,
			expected: true,
		},
		{
			name:     "single empty string vs empty slice",
			a:        []string{""},
			b:        []string{},
			expected: false,
		},
		{
			name:     "identical empty strings",
			a:        []string{""},
			b:        []string{""},
			expected: true,
		},
		{
			name:     "many identical elements",
			a:        []string{"a", "b", "c", "d", "e"},
			b:        []string{"a", "b", "c", "d", "e"},
			expected: true,
		},
		{
			name:     "many elements, one different",
			a:        []string{"a", "b", "c", "d", "e"},
			b:        []string{"a", "b", "x", "d", "e"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalStringSlices(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalStringSlices(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestPeerManager_DifferentInterfaceNames(t *testing.T) {
	names := []string{"wg0", "wg1", "wg-vpn", "wireguard0", "vpn"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			pm := NewPeerManager(name)
			if pm.interfaceName != name {
				t.Errorf("Expected interface name '%s', got '%s'", name, pm.interfaceName)
			}
		})
	}
}

func TestPeer_EmptyAllowedIPs(t *testing.T) {
	peer := Peer{
		PublicKey:  "empty-allowed-ips",
		AllowedIPs: nil,
		UserEmail:  "test@example.com",
	}

	if peer.AllowedIPs != nil {
		t.Error("Expected nil AllowedIPs")
	}
}

func TestPeer_ZeroExpiresAt(t *testing.T) {
	peer := Peer{
		PublicKey: "zero-expires",
		UserEmail: "test@example.com",
	}

	if !peer.ExpiresAt.IsZero() {
		t.Errorf("Expected zero ExpiresAt, got %v", peer.ExpiresAt)
	}
}
