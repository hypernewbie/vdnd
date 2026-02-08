package rlm

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
	Query       string `json:"query,omitempty"`
	Context     string `json:"context,omitempty"`
	CodeSnippet string `json:"code_snippet,omitempty"`
}

// RecursiveHandler is a callback function for handling recursive LLM calls.
type RecursiveHandler func(query, context string) (string, error)

// REPLExecutor manages a persistent Python REPL process.
type REPLExecutor struct {
	cmd              *exec.Cmd
	stdin            io.WriteCloser
	stdout           io.ReadCloser
	reader           *bufio.Reader
	RecursiveHandler RecursiveHandler
}

// NewREPLExecutor creates and starts a new Python REPL executor.
func NewREPLExecutor(pythonPath, scriptPath string) (*REPLExecutor, error) {
	cmd := exec.Command(pythonPath, scriptPath)
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
		reader: bufio.NewReader(stdout),
	}, nil
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
			if r.RecursiveHandler == nil {
				r.sendRecursiveResponse("Error: Recursive calls not supported in this environment")
				continue
			}
			res, err := r.RecursiveHandler(msg.Query, msg.Context)
			if err != nil {
				r.sendRecursiveResponse(fmt.Sprintf("Error: %v", err))
			} else {
				r.sendRecursiveResponse(res)
			}
		case "result":
			return &REPLResult{
				Stdout: msg.Stdout,
				Error:  msg.Error,
			}, nil
		default:
			// Unexpected message type, ignore or handle as error?
			// For now, continue reading if it's not a result.
		}
	}
}

func (r *REPLExecutor) sendRecursiveResponse(result string) {
	response := map[string]string{
		"type":   "recursive_response",
		"result": result,
	}
	data, _ := json.Marshal(response)
	fmt.Fprintln(r.stdin, string(data))
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
	// Priority 1: .venv/bin/python
	venvPath := filepath.Join(rootDir, ".venv", "bin", "python3")
	if _, err := os.Stat(venvPath); err == nil {
		return venvPath
	}
	// Priority 2: System python3
	return "python3"
}
