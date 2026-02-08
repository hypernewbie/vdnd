package vdhelpers

import (
	"encoding/json"
	"fmt"
	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/cmdparser"
)

type VDResult struct {
	Stdout   string `json:"stdout"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

var cliRun = cli.Run

// ExecuteGenericVD runs a raw command string via cli.Run.
func ExecuteGenericVD(cmd string, deps cli.Deps) string {
	args := cmdparser.Parse(cmd)
	if len(args) == 0 {
		return mustJSON(VDResult{Error: "empty command"})
	}
	stdout, exitCode := cliRun(args, deps)
	return mustJSON(VDResult{Stdout: stdout, ExitCode: exitCode})
}

// ExecuteStructuredVD maps a structured tool call to argv.
// This is a placeholder; actual mapping will be done in orchestrator's mapToolToArgs.
func ExecuteStructuredVD(toolName string, args map[string]any, deps cli.Deps) string {
	// Delegated to orchestrator's existing mapToolToArgs logic
	// Returns same JSON result format
	return mustJSON(VDResult{Error: "not implemented yet"})
}

// mustJSON marshals a VDResult to JSON string (panics on error, should never happen).
func mustJSON(r VDResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		panic(fmt.Sprintf("vdhelpers: failed to marshal JSON: %v", err))
	}
	return string(b)
}
