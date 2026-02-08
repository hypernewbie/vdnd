package rlm

import (
	"context"
	"log/slog"
	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/vdengine"
)

func NewVDLM(provider llmtypes.Provider, deps cli.Deps) *RLM {
	engine := vdengine.New(deps)
	return NewRLMWithConfig(provider, Config{
		MaxIterations:  50,
		MaxDepth:       1,
		Tools:          engine.Tools(),
		ToolHandlers:   VDLMHandlers(engine),
		SessionFactory: NewVDSessionFactory(deps),
		SystemPromptBuilder: BuildVDLMSystemPrompt,
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

			// VDEngine.ExecuteTool for ripgrep returns the JSON result directly
			// For others it returns stdout. We should probably wrap in VDResult 
			// if it's not already, but vdengine.ExecuteTool is generic.
			// Actually VDLM expectation is the observation string.
			return stdout, nil
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
