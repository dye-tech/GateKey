package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
)

// LocalUser represents a local admin user.
type LocalUser struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Email        string    `json:"email"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

// LocalAuthProvider handles local username/password authentication.
type LocalAuthProvider struct {
	mu    sync.RWMutex
	users map[string]*LocalUser

	// Argon2 parameters
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

// NewLocalAuthProvider creates a new local auth provider.
func NewLocalAuthProvider() *LocalAuthProvider {
	return &LocalAuthProvider{
		users:   make(map[string]*LocalUser),
		time:    1,
		memory:  64 * 1024, // 64MB
		threads: 4,
		keyLen:  32,
	}
}

// CreateUser creates a new local user.
func (p *LocalAuthProvider) CreateUser(username, password, email string, isAdmin bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.users[username]; exists {
		return ErrUserExists
	}

	hash, err := p.hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	p.users[username] = &LocalUser{
		Username:     username,
		PasswordHash: hash,
		Email:        email,
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now(),
	}

	return nil
}

// Authenticate validates username and password.
func (p *LocalAuthProvider) Authenticate(username, password string) (*LocalUser, error) {
	p.mu.RLock()
	user, exists := p.users[username]
	p.mu.RUnlock()

	if !exists {
		// Still do password comparison to prevent timing attacks
		p.hashPassword(password)
		return nil, ErrInvalidCredentials
	}

	if !p.verifyPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	p.mu.Lock()
	user.LastLogin = time.Now()
	p.mu.Unlock()

	return user, nil
}

// GetUser returns a user by username.
func (p *LocalAuthProvider) GetUser(username string) (*LocalUser, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	user, exists := p.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// UpdatePassword updates a user's password.
func (p *LocalAuthProvider) UpdatePassword(username, newPassword string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	user, exists := p.users[username]
	if !exists {
		return ErrUserNotFound
	}

	hash, err := p.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = hash
	return nil
}

// DeleteUser removes a user.
func (p *LocalAuthProvider) DeleteUser(username string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.users[username]; !exists {
		return ErrUserNotFound
	}

	delete(p.users, username)
	return nil
}

// ListUsers returns all users (without password hashes).
func (p *LocalAuthProvider) ListUsers() []*LocalUser {
	p.mu.RLock()
	defer p.mu.RUnlock()

	users := make([]*LocalUser, 0, len(p.users))
	for _, u := range p.users {
		users = append(users, u)
	}
	return users
}

// HasUsers returns true if any users exist.
func (p *LocalAuthProvider) HasUsers() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.users) > 0
}

// hashPassword generates an Argon2id hash of the password.
func (p *LocalAuthProvider) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, p.keyLen)

	// Encode as: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.time, p.threads, b64Salt, b64Hash), nil
}

// verifyPassword checks if a password matches the hash.
func (p *LocalAuthProvider) verifyPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	var version int
	var memory, time uint32
	var threads uint8

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}

// InitDefaultAdmin creates a default admin user if no users exist.
// Returns the generated password if a new user was created.
func (p *LocalAuthProvider) InitDefaultAdmin() (string, bool) {
	if p.HasUsers() {
		return "", false
	}

	// Generate a random password
	passwordBytes := make([]byte, 16)
	rand.Read(passwordBytes)
	password := base64.RawURLEncoding.EncodeToString(passwordBytes)

	err := p.CreateUser("admin", password, "admin@localhost", true)
	if err != nil {
		return "", false
	}

	return password, true
}
