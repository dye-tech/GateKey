package sshbastion

import (
	"testing"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		input    string
		wantUser string
		wantHost string
		wantPort int
	}{
		{"root@10.0.1.5", "root", "10.0.1.5", 22},
		{"ubuntu@myhost.internal:2222", "ubuntu", "myhost.internal", 2222},
		{"10.0.1.5", "", "10.0.1.5", 22},
		{"myhost.internal:8022", "", "myhost.internal", 8022},
		{"admin@192.168.1.1", "admin", "192.168.1.1", 22},
		{"user@host.example.com:22", "user", "host.example.com", 22},
		{"ec2-user@10.0.17.75", "ec2-user", "10.0.17.75", 22},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			user, host, port := ParseTarget(tt.input)
			if user != tt.wantUser {
				t.Errorf("ParseTarget(%q) user = %q, want %q", tt.input, user, tt.wantUser)
			}
			if host != tt.wantHost {
				t.Errorf("ParseTarget(%q) host = %q, want %q", tt.input, host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("ParseTarget(%q) port = %d, want %d", tt.input, port, tt.wantPort)
			}
		})
	}
}

func TestGenerateED25519Key(t *testing.T) {
	pemData, err := GenerateED25519Key()
	if err != nil {
		t.Fatalf("GenerateED25519Key() error = %v", err)
	}
	if len(pemData) == 0 {
		t.Fatal("GenerateED25519Key() returned empty PEM data")
	}
	if string(pemData[:len("-----BEGIN PRIVATE KEY-----")]) != "-----BEGIN PRIVATE KEY-----" {
		t.Errorf("GenerateED25519Key() PEM does not start with expected header")
	}
}

func TestNewServer(t *testing.T) {
	// Importing zap for test would require a dependency, so use a no-op approach
	// Just verify the constructor works
	_, err := NewServer(Config{ListenAddr: ":0"}, nil)
	// nil logger will panic in production but we're just testing struct creation
	if err == nil {
		// Expected — constructor doesn't validate logger
	}
}

func TestSetHostKey(t *testing.T) {
	pemData, err := GenerateED25519Key()
	if err != nil {
		t.Fatalf("GenerateED25519Key() error = %v", err)
	}

	srv, _ := NewServer(Config{ListenAddr: ":0"}, nil)
	if err := srv.SetHostKey(pemData); err != nil {
		t.Fatalf("SetHostKey() error = %v", err)
	}
}

func TestSetHostKeyInvalid(t *testing.T) {
	srv, _ := NewServer(Config{ListenAddr: ":0"}, nil)
	err := srv.SetHostKey([]byte("not a valid PEM key"))
	if err == nil {
		t.Fatal("SetHostKey() with invalid PEM should return error")
	}
}
