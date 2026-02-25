package llm

import (
	"context"

	"uaa/vdnd/internal/llm/llmtypes"
)

// Subagent is a specialized worker tool callable by the orchestrator.
type Subagent interface {
	Name() string
	Description() string
	Run(ctx context.Context, query string, history []llmtypes.Message) (string, error)
	ToolDefinition() llmtypes.Tool
}
