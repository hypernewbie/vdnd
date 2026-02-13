package mom

import (
	"context"
	"fmt"
	"strings"
	"time"

	"uaa/vdnd/internal/pi-go/agent"
	"uaa/vdnd/internal/pi-go/codingagent"
)

// CreateMomTools creates the tool set for a mom agent runner.
// Tools execute commands via the given Executor (host or Docker).
func CreateMomTools(executor Executor, cwd string) []agent.Tool {
	ops := executorFileOps(executor, cwd)
	bashOps := executorBashOps(executor, cwd)

	return []agent.Tool{
		codingagent.CreateReadTool(cwd, ops),
		codingagent.CreateBashTool(cwd, bashOps),
		codingagent.CreateEditTool(cwd, ops),
		codingagent.CreateWriteTool(cwd, ops),
	}
}

// executorFileOps wraps an Executor as codingagent.FileOps.
// Read/write operations go through the executor's command execution for remote support.
func executorFileOps(executor Executor, _ string) *codingagent.FileOps {
	ops := codingagent.DefaultFileOps()

	// Override with executor-based operations for Docker support
	if _, ok := executor.(*dockerExecutor); ok {
		ops.ReadFile = func(path string) ([]byte, error) {
			result, err := executor.Exec(context.Background(), fmt.Sprintf("cat %s", shellEscape(path)))
			if err != nil {
				return nil, err
			}
			if result.Code != 0 {
				return nil, fmt.Errorf("cat failed: %s", strings.TrimSpace(result.Stderr))
			}
			return []byte(result.Stdout), nil
		}
		ops.WriteFile = func(path string, content []byte) error {
			// Use base64 to safely transfer content
			result, err := executor.Exec(context.Background(),
				fmt.Sprintf("echo %s | base64 -d > %s", shellEscape(string(content)), shellEscape(path)))
			if err != nil {
				return err
			}
			if result.Code != 0 {
				return fmt.Errorf("write failed: %s", strings.TrimSpace(result.Stderr))
			}
			return nil
		}
		ops.Access = func(path string) error {
			result, err := executor.Exec(context.Background(), fmt.Sprintf("test -e %s", shellEscape(path)))
			if err != nil {
				return err
			}
			if result.Code != 0 {
				return fmt.Errorf("file not found: %s", path)
			}
			return nil
		}
	}

	return &ops
}

// executorBashOps wraps an Executor as codingagent.BashOps.
func executorBashOps(executor Executor, _ string) *codingagent.BashOps {
	return &codingagent.BashOps{
		Exec: func(ctx context.Context, command string, cwd string) (string, int, error) {
			// Wrap command with cd for the executor
			fullCmd := fmt.Sprintf("cd %s && %s", shellEscape(cwd), command)
			result, err := executor.Exec(ctx, fullCmd)
			if err != nil {
				return "", -1, err
			}
			output := result.Stdout
			if result.Stderr != "" {
				if output != "" {
					output += "\n"
				}
				output += result.Stderr
			}
			return output, result.Code, nil
		},
	}
}

// --- System Prompt ---

// ChannelInfo describes a Slack channel for the system prompt.
type ChannelInfo struct {
	ID   string
	Name string
}

// UserInfo describes a Slack user for the system prompt.
type UserInfo struct {
	ID          string
	Name        string
	DisplayName string
}

// BuildMomSystemPrompt builds the system prompt for a mom agent.
func BuildMomSystemPrompt(workspacePath string, channelID string, memory string, channels []ChannelInfo, users []UserInfo) string {
	var b strings.Builder

	b.WriteString("You are mom, an AI assistant that lives in a Slack workspace. ")
	b.WriteString("You help users by answering questions, writing code, running commands, and managing files.\n\n")

	// Tools
	b.WriteString("You have access to tools: read, bash, edit, write.\n")
	b.WriteString("Use bash for command execution, read to examine files, edit for precise changes, write for new files.\n\n")

	// Workspace info
	b.WriteString(fmt.Sprintf("Workspace path: %s\n", workspacePath))
	b.WriteString(fmt.Sprintf("Current channel: %s\n\n", channelID))

	// Channels
	if len(channels) > 0 {
		b.WriteString("Slack channels:\n")
		for _, ch := range channels {
			b.WriteString(fmt.Sprintf("- #%s (%s)\n", ch.Name, ch.ID))
		}
		b.WriteString("\n")
	}

	// Users
	if len(users) > 0 {
		b.WriteString("Users:\n")
		for _, u := range users {
			name := u.DisplayName
			if name == "" {
				name = u.Name
			}
			b.WriteString(fmt.Sprintf("- @%s (%s)\n", name, u.ID))
		}
		b.WriteString("\n")
	}

	// Memory
	if memory != "" {
		b.WriteString("Your memory (persistent notes):\n")
		b.WriteString(memory)
		b.WriteString("\n\n")
	}

	// Guidelines
	b.WriteString("Guidelines:\n")
	b.WriteString("- Be concise in Slack messages\n")
	b.WriteString("- Use code blocks for code\n")
	b.WriteString("- Break long responses into multiple messages\n")
	b.WriteString("- Read files before editing\n")

	// Add timestamp
	b.WriteString(fmt.Sprintf("\nCurrent date and time: %s\n",
		time.Now().Format(time.RFC1123)))

	return b.String()
}
