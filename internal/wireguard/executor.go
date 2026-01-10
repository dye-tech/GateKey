package wireguard

import (
	"context"
	"os/exec"
)

// CommandExecutor abstracts command execution for testing.
type CommandExecutor interface {
	// Run executes a command and returns an error if it fails.
	Run(ctx context.Context, name string, args ...string) error
	// Output executes a command and returns its combined output.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	// LookPath searches for an executable.
	LookPath(name string) (string, error)
}

// realExecutor is the default executor that uses os/exec.
type realExecutor struct{}

// DefaultExecutor returns the real command executor.
func DefaultExecutor() CommandExecutor {
	return &realExecutor{}
}

func (e *realExecutor) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

func (e *realExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func (e *realExecutor) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
