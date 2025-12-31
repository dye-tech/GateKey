# Contributing to GateKey

Thank you for your interest in contributing to GateKey! This document provides guidelines and information about contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## How to Contribute

### Reporting Bugs

Before creating a bug report, please check existing issues to avoid duplicates.

When creating a bug report, include:

- **Clear title** describing the issue
- **Steps to reproduce** the behavior
- **Expected behavior** vs actual behavior
- **Environment details** (OS, Go version, browser, etc.)
- **Logs or error messages** (sanitized of sensitive data)
- **Screenshots** if applicable

### Suggesting Features

Feature suggestions are welcome! Please include:

- **Use case**: Why is this feature needed?
- **Proposed solution**: How should it work?
- **Alternatives considered**: Other approaches you've thought of
- **Additional context**: Mockups, examples, etc.

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow the coding style** (see below)
3. **Write tests** for new functionality
4. **Update documentation** as needed
5. **Ensure CI passes** before requesting review
6. **Keep PRs focused** - one feature/fix per PR

## Development Setup

### Prerequisites

- Go 1.23+
- Node.js 20+
- PostgreSQL 15+
- Docker (optional, for testing)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/dye-tech/GateKey.git
cd GateKey

# Install Go dependencies
go mod download

# Install frontend dependencies
cd web && npm install && cd ..

# Set up the database
createdb gatekey
export DATABASE_URL="postgres://localhost/gatekey?sslmode=disable"
make migrate-up

# Run in development mode
make dev
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run linter
make lint

# Run frontend tests
cd web && npm test
```

### Building

```bash
# Build all binaries
make build

# Build for all platforms
make build-all

# Build frontend
make frontend-build
```

## Coding Style

### Go

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Write meaningful comments for exported functions
- Keep functions focused and small
- Handle errors explicitly

```go
// Good
func (s *Server) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := s.store.GetUser(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user %s: %w", id, err)
    }
    return user, nil
}
```

### TypeScript/React

- Use TypeScript for all new code
- Follow React best practices (hooks, functional components)
- Use Tailwind CSS for styling
- Keep components small and focused

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```
feat(api): add endpoint for listing user sessions
fix(gateway): handle connection timeout gracefully
docs: update API documentation for v2 endpoints
```

## Project Structure

```
GateKey/
├── cmd/                    # Application entry points
│   ├── gatekey/           # VPN client
│   ├── gatekey-server/    # Control plane server
│   ├── gatekey-gateway/   # Gateway agent
│   └── gatekey-admin/     # Admin CLI
├── internal/              # Private application code
│   ├── api/              # REST API handlers
│   ├── auth/             # Authentication (OIDC, SAML)
│   ├── db/               # Database operations
│   ├── firewall/         # nftables management
│   ├── pki/              # Certificate authority
│   └── ...
├── pkg/                   # Public libraries
├── web/                   # React frontend
├── migrations/            # Database migrations
├── docs/                  # Documentation
└── deploy/               # Deployment configs
```

## Testing Guidelines

- Write unit tests for business logic
- Write integration tests for API endpoints
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for meaningful coverage, not 100%

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

## Documentation

- Update README.md for user-facing changes
- Update docs/ for detailed documentation
- Include godoc comments for exported types/functions
- Add examples where helpful

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create a git tag: `git tag v1.2.3`
4. Push the tag: `git push origin v1.2.3`
5. GitHub Actions will build and publish the release

## Getting Help

- **GitHub Issues**: For bugs and feature requests
- **Discussions**: For questions and community chat
- **Documentation**: Check docs/ directory

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
