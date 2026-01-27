package auth

import (
	"testing"
	"time"
)

func TestNewLocalAuthProvider(t *testing.T) {
	provider := NewLocalAuthProvider()

	if provider == nil {
		t.Fatal("NewLocalAuthProvider returned nil")
	}

	if provider.users == nil {
		t.Error("users map should be initialized")
	}

	// Check Argon2 parameters are set
	if provider.time == 0 {
		t.Error("time parameter should be set")
	}
	if provider.memory == 0 {
		t.Error("memory parameter should be set")
	}
	if provider.threads == 0 {
		t.Error("threads parameter should be set")
	}
	if provider.keyLen == 0 {
		t.Error("keyLen parameter should be set")
	}
}

func TestCreateUser(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("testuser", "password123", "test@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Verify user was created
	user, err := provider.GetUser("testuser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %q", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %q", user.Email)
	}
	if user.IsAdmin {
		t.Error("User should not be admin")
	}
	if user.PasswordHash == "" {
		t.Error("Password hash should be set")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestCreateUser_AdminUser(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("admin", "adminpass", "admin@example.com", true)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := provider.GetUser("admin")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if !user.IsAdmin {
		t.Error("User should be admin")
	}
}

func TestCreateUser_Duplicate(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("testuser", "password123", "test@example.com", false)
	if err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	// Try to create same user again
	err = provider.CreateUser("testuser", "different", "other@example.com", true)
	if err != ErrUserExists {
		t.Errorf("Expected ErrUserExists, got %v", err)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("testuser", "password123", "test@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := provider.Authenticate("testuser", "password123")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}

	if user == nil {
		t.Fatal("Authenticate returned nil user")
	}
	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %q", user.Username)
	}
	if user.LastLogin.IsZero() {
		t.Error("LastLogin should be set after authentication")
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("testuser", "password123", "test@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	_, err = provider.Authenticate("testuser", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticate_UserNotFound(t *testing.T) {
	provider := NewLocalAuthProvider()

	_, err := provider.Authenticate("nonexistent", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestGetUser_Success(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("testuser", "password123", "test@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := provider.GetUser("testuser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %q", user.Username)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	provider := NewLocalAuthProvider()

	_, err := provider.GetUser("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdatePassword_Success(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("testuser", "oldpassword", "test@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Update password
	err = provider.UpdatePassword("testuser", "newpassword")
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	// Verify old password no longer works
	_, err = provider.Authenticate("testuser", "oldpassword")
	if err != ErrInvalidCredentials {
		t.Error("Old password should not work")
	}

	// Verify new password works
	user, err := provider.Authenticate("testuser", "newpassword")
	if err != nil {
		t.Fatalf("Authenticate with new password failed: %v", err)
	}
	if user == nil {
		t.Error("User should not be nil")
	}
}

func TestUpdatePassword_UserNotFound(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.UpdatePassword("nonexistent", "newpassword")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.CreateUser("testuser", "password123", "test@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err = provider.DeleteUser("testuser")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify user is gone
	_, err = provider.GetUser("testuser")
	if err != ErrUserNotFound {
		t.Error("User should be deleted")
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	provider := NewLocalAuthProvider()

	err := provider.DeleteUser("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	provider := NewLocalAuthProvider()

	// Empty list
	users := provider.ListUsers()
	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}

	// Add users
	provider.CreateUser("user1", "pass1", "user1@example.com", false)
	provider.CreateUser("user2", "pass2", "user2@example.com", true)
	provider.CreateUser("user3", "pass3", "user3@example.com", false)

	users = provider.ListUsers()
	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}

	// Verify users are in the list
	usernames := make(map[string]bool)
	for _, u := range users {
		usernames[u.Username] = true
	}

	for _, expected := range []string{"user1", "user2", "user3"} {
		if !usernames[expected] {
			t.Errorf("User %q not found in list", expected)
		}
	}
}

func TestHasUsers(t *testing.T) {
	provider := NewLocalAuthProvider()

	if provider.HasUsers() {
		t.Error("HasUsers should return false when no users exist")
	}

	provider.CreateUser("testuser", "password123", "test@example.com", false)

	if !provider.HasUsers() {
		t.Error("HasUsers should return true when users exist")
	}
}

func TestInitDefaultAdmin_NoUsers(t *testing.T) {
	provider := NewLocalAuthProvider()

	password, created := provider.InitDefaultAdmin()

	if !created {
		t.Error("InitDefaultAdmin should create user when no users exist")
	}
	if password == "" {
		t.Error("InitDefaultAdmin should return generated password")
	}

	// Verify admin user was created
	user, err := provider.GetUser("admin")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if !user.IsAdmin {
		t.Error("admin user should be an admin")
	}

	// Verify password works
	authUser, err := provider.Authenticate("admin", password)
	if err != nil {
		t.Fatalf("Authenticate with generated password failed: %v", err)
	}
	if authUser == nil {
		t.Error("authUser should not be nil")
	}
}

func TestInitDefaultAdmin_UsersExist(t *testing.T) {
	provider := NewLocalAuthProvider()

	// Create a user first
	provider.CreateUser("existinguser", "password123", "existing@example.com", false)

	password, created := provider.InitDefaultAdmin()

	if created {
		t.Error("InitDefaultAdmin should not create user when users exist")
	}
	if password != "" {
		t.Error("InitDefaultAdmin should return empty password when users exist")
	}
}

func TestHashPassword_Determinism(t *testing.T) {
	provider := NewLocalAuthProvider()

	// Hash same password twice - should produce different hashes due to random salt
	hash1, err := provider.hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}

	hash2, err := provider.hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Same password should produce different hashes due to random salt")
	}
}

func TestVerifyPassword_ValidFormats(t *testing.T) {
	provider := NewLocalAuthProvider()

	// Create user and get the hash
	err := provider.CreateUser("testuser", "testpass", "test@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, _ := provider.GetUser("testuser")
	hash := user.PasswordHash

	// Verify correct password
	if !provider.verifyPassword("testpass", hash) {
		t.Error("verifyPassword should return true for correct password")
	}

	// Verify incorrect password
	if provider.verifyPassword("wrongpass", hash) {
		t.Error("verifyPassword should return false for incorrect password")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	provider := NewLocalAuthProvider()

	testCases := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"not enough parts", "$argon2id$v=19"},
		{"wrong format", "not-a-valid-hash"},
		{"invalid version", "$argon2id$v=invalid$m=65536,t=3,p=4$salt$hash"},
		{"invalid params", "$argon2id$v=19$invalid$salt$hash"},
		{"invalid salt encoding", "$argon2id$v=19$m=65536,t=3,p=4$!!!invalid!!!$hash"},
		{"invalid hash encoding", "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$!!!invalid!!!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if provider.verifyPassword("password", tc.hash) {
				t.Error("verifyPassword should return false for invalid hash format")
			}
		})
	}
}

func TestLocalUser_Fields(t *testing.T) {
	now := time.Now()
	user := &LocalUser{
		Username:     "testuser",
		PasswordHash: "somehash",
		Email:        "test@example.com",
		IsAdmin:      true,
		CreatedAt:    now,
		LastLogin:    now,
	}

	if user.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got %q", user.Username)
	}
	if user.PasswordHash != "somehash" {
		t.Errorf("Expected PasswordHash 'somehash', got %q", user.PasswordHash)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got %q", user.Email)
	}
	if !user.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestConcurrentAccess(t *testing.T) {
	provider := NewLocalAuthProvider()

	// Create initial user
	provider.CreateUser("testuser", "password123", "test@example.com", false)

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			provider.GetUser("testuser")
			provider.ListUsers()
			provider.HasUsers()
			done <- true
		}()
	}

	// Concurrent authentication
	for i := 0; i < 10; i++ {
		go func() {
			provider.Authenticate("testuser", "password123")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}
