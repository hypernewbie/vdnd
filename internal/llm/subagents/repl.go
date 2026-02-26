package subagents

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

// REPLResult represents the output from the Python sandbox.
type REPLResult struct {
	Stdout string `json:"stdout"`
	Error  string `json:"error"`
}

// REPLMessage represents a message from the Python sandbox.
type REPLMessage struct {
	Type        string `json:"type"`
	Stdout      string `json:"stdout,omitempty"`
	Error       string `json:"error,omitempty"`
	CodeSnippet string `json:"code_snippet,omitempty"`
}

// REPLExecutor manages a persistent Python REPL process.
type REPLExecutor struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
}

// NewREPLExecutorWithEnv creates and starts a new Python REPL executor with custom environment variables.
func NewREPLExecutorWithEnv(pythonPath, scriptPath string, env []string) (*REPLExecutor, error) {
	cmd := exec.Command(pythonPath, scriptPath)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start REPL process: %w", err)
	}

	return &REPLExecutor{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		reader: bufio.NewReaderSize(stdout, 1024*1024), // 1MB buffer
	}, nil
}

// NewREPLExecutor creates and starts a new Python REPL executor.
func NewREPLExecutor(pythonPath, scriptPath string) (*REPLExecutor, error) {
	return NewREPLExecutorWithEnv(pythonPath, scriptPath, nil)
}

// Execute sends code to the REPL and returns the result.
func (r *REPLExecutor) Execute(code string) (*REPLResult, error) {
	input := map[string]string{"code": code}
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	if _, err := fmt.Fprintln(r.stdin, string(data)); err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %w", err)
	}

	for {
		line, err := r.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdout: %w", err)
		}

		var msg REPLMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w (line: %q)", err, line)
		}

		switch msg.Type {
		case "python_state_mutation_attempt":
			slog.Warn("PYTHON_STATE_ATTEMPT", "snippet", msg.CodeSnippet)
		case "recursive_call":
			return &REPLResult{Error: "recursive_call is not supported in subagent mode"}, nil
		case "result":
			return &REPLResult{
				Stdout: msg.Stdout,
				Error:  msg.Error,
			}, nil
		}
	}
}

// Close terminates the REPL process.
func (r *REPLExecutor) Close() error {
	if r.stdin != nil {
		r.stdin.Close()
	}
	if r.cmd != nil {
		return r.cmd.Wait()
	}
	return nil
}

// FindPythonPath attempts to find a usable python executable.
func FindPythonPath(rootDir string) string {
	venvPath := filepath.Join(rootDir, ".venv", "bin", "python3")
	if _, err := os.Stat(venvPath); err == nil {
		return venvPath
	}
	return "python3"
}
