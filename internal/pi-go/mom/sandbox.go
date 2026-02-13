// Package mom provides multi-agent orchestration.
package mom

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// SandboxType identifies the sandbox execution environment.
type SandboxType string

const (
	SandboxHost   SandboxType = "host"
	SandboxDocker SandboxType = "docker"
)

// SandboxConfig configures the sandbox execution environment.
type SandboxConfig struct {
	Type      SandboxType
	Container string // Docker container name (only for SandboxDocker)
}

// ParseSandboxArg parses a sandbox argument string (e.g., "host" or "docker:<container>").
func ParseSandboxArg(value string) (SandboxConfig, error) {
	if value == "host" {
		return SandboxConfig{Type: SandboxHost}, nil
	}
	if strings.HasPrefix(value, "docker:") {
		container := strings.TrimPrefix(value, "docker:")
		if container == "" {
			return SandboxConfig{}, fmt.Errorf("docker sandbox requires container name (e.g., docker:mom-sandbox)")
		}
		return SandboxConfig{Type: SandboxDocker, Container: container}, nil
	}
	return SandboxConfig{}, fmt.Errorf("invalid sandbox type %q: use 'host' or 'docker:<container-name>'", value)
}

// ExecResult holds the result of a command execution.
type ExecResult struct {
	Stdout string
	Stderr string
	Code   int
}

// Executor runs commands in a sandbox environment.
type Executor interface {
	// Exec executes a bash command and returns the result.
	Exec(ctx context.Context, command string) (ExecResult, error)
	// WorkspacePath returns the workspace path as seen by the executor.
	// Host returns the actual path; Docker returns /workspace.
	WorkspacePath(hostPath string) string
}

// CreateExecutor creates an Executor for the given sandbox configuration.
func CreateExecutor(config SandboxConfig) Executor {
	if config.Type == SandboxDocker {
		return &dockerExecutor{container: config.Container}
	}
	return &hostExecutor{}
}

// --- Host executor ---

type hostExecutor struct{}

func (e *hostExecutor) Exec(ctx context.Context, command string) (ExecResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return ExecResult{}, fmt.Errorf("command aborted: %w", ctx.Err())
		} else {
			return ExecResult{}, fmt.Errorf("command failed: %w", err)
		}
	}

	// Truncate large outputs (10MB max)
	const maxOutput = 10 * 1024 * 1024
	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	if len(stdoutStr) > maxOutput {
		stdoutStr = stdoutStr[:maxOutput]
	}
	if len(stderrStr) > maxOutput {
		stderrStr = stderrStr[:maxOutput]
	}

	return ExecResult{Stdout: stdoutStr, Stderr: stderrStr, Code: code}, nil
}

func (e *hostExecutor) WorkspacePath(hostPath string) string {
	return hostPath
}

// --- Docker executor ---

type dockerExecutor struct {
	container string
}

func (e *dockerExecutor) Exec(ctx context.Context, command string) (ExecResult, error) {
	dockerCmd := fmt.Sprintf("docker exec %s sh -c %s", e.container, shellEscape(command))
	host := &hostExecutor{}
	return host.Exec(ctx, dockerCmd)
}

func (e *dockerExecutor) WorkspacePath(_ string) string {
	return "/workspace"
}

// ValidateSandbox checks that the sandbox environment is available.
func ValidateSandbox(ctx context.Context, config SandboxConfig) error {
	if config.Type == SandboxHost {
		return nil
	}

	executor := &hostExecutor{}

	// Check Docker is available
	result, err := executor.Exec(ctx, "docker --version")
	if err != nil {
		return fmt.Errorf("docker is not installed or not in PATH")
	}
	_ = result

	// Check container is running
	result, err = executor.Exec(ctx, fmt.Sprintf("docker inspect -f '{{.State.Running}}' %s", config.Container))
	if err != nil {
		return fmt.Errorf("container %q does not exist", config.Container)
	}
	if strings.TrimSpace(result.Stdout) != "true" {
		return fmt.Errorf("container %q is not running", config.Container)
	}

	return nil
}

// shellEscape escapes a string for passing to sh -c.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// KillProcessTree kills a process and all its children.
func KillProcessTree(pid int) {
	// Try to kill the process group first
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	// Fallback to killing just the process
	p, err := os.FindProcess(pid)
	if err == nil {
		_ = p.Kill()
	}
}
