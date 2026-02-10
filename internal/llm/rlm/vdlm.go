package rlm

import (
	"context"
	"encoding/json"
	"log/slog"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/vdengine"
	"uaa/vdnd/internal/llm/vdhelpers"
)

func NewVDLM(provider llmtypes.Provider, deps cli.Deps, promptBuilder SystemPromptBuilder) *RLM {
	engine := vdengine.New(deps)
	return NewRLMWithConfig(provider, Config{
		MaxIterations:       50,
		MaxDepth:            1,
		Tools:               engine.Tools(),
		ToolHandlers:        VDLMHandlers(engine),
		SessionFactory:      NewVDSessionFactory(deps),
		SystemPromptBuilder: promptBuilder,
	})
}

func VDLMHandlers(engine *vdengine.VDEngine) map[string]ToolHandler {
	handlers := make(map[string]ToolHandler)
	for _, tool := range engine.Tools() {
		toolName := tool.Name
		handlers[toolName] = func(ctx context.Context, call llmtypes.ToolCall, session any) (string, error) {
			stdout, exitCode, cmdArgs, err := engine.ExecuteTool(call)
			if err != nil {
				return "", err
			}

			// We don't have direct access to Orchestrator's logger here,
			// so we just log the basics.
			slog.Info("TOOL_CALL",
				"tool", call.Name,
				"arguments", call.Arguments,
				"mapped_args", cmdArgs,
				"exit_code", exitCode,
			)

			res := vdhelpers.VDResult{
				Stdout:   stdout,
				ExitCode: exitCode,
			}
			if exitCode != 0 {
				res.Error = "Command failed"
			}
			b, _ := json.Marshal(res)
			return string(b), nil
		}
	}
	return handlers
}

func NewVDSessionFactory(deps cli.Deps) SessionFactory {
	return func() (any, func(), error) {
		// VDLM doesn't need special session state per call for now,
		// as engine holds deps.
		return deps, nil, nil
	}
}
