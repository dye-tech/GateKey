# GateKey Development Guide

This document covers development practices, linting configuration, and code style guidelines for GateKey.

## Development Setup

### Prerequisites

- Go 1.23+
- Node.js 20+
- PostgreSQL 15+
- Docker (optional, for testing)
- golangci-lint (for linting)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/dye-tech/GateKey.git
cd GateKey

# Install Go dependencies
go mod download

# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install goimports
go install golang.org/x/tools/cmd/goimports@latest

# Install frontend dependencies
cd web && npm install && cd ..

# Set up the database
createdb gatekey
export DATABASE_URL="postgres://localhost/gatekey?sslmode=disable"
make migrate-up

# Run in development mode
make dev
```

## Linting

GateKey uses [golangci-lint](https://golangci-lint.run/) for Go code linting. The configuration is defined in `.golangci.yml`.

### Running the Linter

```bash
# Run all linters
make lint

# Or directly
golangci-lint run ./...

# Fix formatting issues
goimports -local github.com/gatekey-project/gatekey -w .
```

### Enabled Linters

| Linter | Purpose |
|--------|---------|
| `errcheck` | Check for unchecked errors |
| `gosimple` | Simplify code |
| `govet` | Report suspicious constructs |
| `ineffassign` | Detect ineffectual assignments |
| `staticcheck` | Static analysis checks |
| `unused` | Find unused code |
| `bodyclose` | Check HTTP response body is closed |
| `errorlint` | Check error wrapping |
| `exhaustive` | Check switch exhaustiveness |
| `gofmt` | Check formatting |
| `goimports` | Check import formatting |
| `gosec` | Security checks |
| `misspell` | Spell checking |
| `sqlclosecheck` | Check SQL rows/statements are closed |
| `stylecheck` | Style checks |

### Disabled Linters

Some linters are intentionally disabled as they are too strict for this project:

| Linter | Reason |
|--------|--------|
| `dupl` | Duplicate code detection - too strict, database queries often similar |
| `funlen` | Function length - too strict for some complex handlers |
| `goconst` | Constants - too strict for status strings |
| `gocritic` | Too opinionated |
| `gocyclo` | Cyclomatic complexity - too strict |
| `godot` | Comment periods - too pedantic |
| `godox` | TODO/FIXME comments - we use these intentionally |
| `lll` | Line length - handled by gofmt |
| `mnd` | Magic numbers - too strict for configs |
| `noctx` | Context in HTTP - sometimes impractical |
| `prealloc` | Slice preallocation - premature optimization |
| `revive` | Too opinionated with many rules |
| `unparam` | Unused params - sometimes needed for interface compliance |

### Linter Exceptions

#### Error Checking Exclusions

The following functions have their error return values intentionally ignored (via `errcheck` configuration):

```yaml
exclude-functions:
  - io.Copy
  - io.ReadAll
  - (io.Closer).Close
  - encoding/json.Marshal
  - encoding/json.Unmarshal
  - fmt.Fprintf
  - fmt.Printf
  - os.Remove
  - os.RemoveAll
  - os.MkdirAll
  - crypto/rand.Read
```

#### Gosec Exclusions

The following security warnings are excluded:

| Rule | Reason |
|------|--------|
| G101 | Hardcoded credentials - false positives for const names containing "key", "token", etc. |
| G104 | Audit errors not checked - handled by errcheck |
| G107 | HTTP request with variable URL - necessary for API calls |
| G115 | Integer overflow - context-dependent |
| G204 | Subprocess with variable - necessary for exec |
| G304 | File path provided as taint input - necessary for config files |
| G306 | Poor file permissions - we intentionally set specific permissions (e.g., 0644 for readable certs) |
| G402 | TLS InsecureSkipVerify - needed for internal services |

#### Path-Based Exclusions

Some linters are relaxed for specific paths:

| Path | Excluded Linters | Reason |
|------|------------------|--------|
| `_test.go` | errcheck, gosec | Tests don't need strict error checking |
| `cmd/` | errcheck, unused | CLI code has more flexibility |
| `internal/client/` | errcheck, unused | Client code is work-in-progress |
| `internal/firewall/` | errcheck | System calls often ignore errors intentionally |
| `internal/openvpn/` | unused | Some helper functions for future use |

### Common Lint Fixes

#### Unchecked Error Returns

When intentionally ignoring an error, use explicit assignment:

```go
// Bad - lint will complain
s.store.Delete(ctx, id)

// Good - explicit ignore
_ = s.store.Delete(ctx, id)

// Better - with comment explaining why
_ = s.store.Delete(ctx, id) // Best effort cleanup
```

#### Import Formatting

Use goimports with local prefix:

```bash
goimports -local github.com/gatekey-project/gatekey -w .
```

Import order should be:
1. Standard library
2. External packages
3. Local packages (with blank line before)

```go
import (
    "context"
    "fmt"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/gatekey-project/gatekey/internal/db"
)
```

#### Exhaustive Switch Statements

When using switch on enum types, include all cases:

```go
switch rule.Protocol {
case ProtocolTCP:
    proto = 6
case ProtocolUDP:
    proto = 17
case ProtocolICMP:
    proto = 1
case ProtocolAny:
    // No-op, included for exhaustiveness
}
```

## Code Style

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Keep functions focused and small
- Handle errors explicitly
- Use meaningful variable names

### Error Handling Patterns

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to get user %s: %w", id, err)
}

// Best-effort operations (explicitly ignore)
_ = s.store.Cleanup(ctx) // Best effort, don't fail main operation

// Timing attack prevention (ignore result)
_, _ = s.hashPassword(password) // Ignore result, just for timing
```

### Comments

- Document exported functions and types
- Don't over-comment obvious code
- Use TODO/FIXME for tracked issues

```go
// GetUser retrieves a user by ID from the database.
// Returns ErrNotFound if the user doesn't exist.
func (s *Store) GetUser(ctx context.Context, id string) (*User, error) {
    // ...
}
```

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/api/...

# Run with verbose output
go test -v ./...
```

### Test Patterns

Use table-driven tests:

```go
func TestGetUser(t *testing.T) {
    tests := []struct {
        name    string
        userID  string
        want    *User
        wantErr bool
    }{
        {
            name:   "valid user",
            userID: "user-123",
            want:   &User{ID: "user-123", Name: "Test"},
        },
        {
            name:    "not found",
            userID:  "unknown",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Building

```bash
# Build all binaries
make build

# Build for all platforms
make build-all

# Build specific binary
go build -o bin/gatekey ./cmd/gatekey

# Build frontend
make frontend-build
```

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bin/gatekey-linux-amd64 ./cmd/gatekey

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o bin/gatekey-linux-arm64 ./cmd/gatekey

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o bin/gatekey-darwin-amd64 ./cmd/gatekey

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bin/gatekey-darwin-arm64 ./cmd/gatekey

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/gatekey-windows-amd64.exe ./cmd/gatekey
```

## CI/CD

The project uses GitHub Actions for CI/CD:

- **Lint**: Runs golangci-lint on every push/PR
- **Test**: Runs all tests with coverage
- **Build**: Builds binaries for all platforms
- **Security**: Runs CodeQL and gosec security scans
- **Release**: Publishes binaries to GitHub Releases on tag

### Pre-commit Checklist

Before pushing:

1. Run `make lint` - ensure no lint errors
2. Run `make test` - ensure tests pass
3. Run `go build ./...` - ensure code compiles
4. Run `goimports -w .` - ensure imports are formatted

## Troubleshooting

### "File is not properly formatted (goimports)"

Run goimports with the local prefix:

```bash
goimports -local github.com/gatekey-project/gatekey -w .
```

### "Error return value is not checked (errcheck)"

Either:
1. Handle the error properly
2. Explicitly ignore with `_ = someFunc()`
3. Add the function to errcheck exclusions in `.golangci.yml`

### "missing cases in switch (exhaustive)"

Add all enum cases to the switch statement, even if they're no-ops:

```go
case SomeEnumValue:
    // No-op, included for exhaustiveness
```

### "directive is unused for linter (nolintlint)"

The `//nolint:` comment is no longer needed because the rule is now excluded globally. Remove the comment.
