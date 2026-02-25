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

// mustJSON marshals a VDResult to JSON string.
func mustJSON(r VDResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		fallback, _ := json.Marshal(VDResult{
			ExitCode: 1,
			Error:    fmt.Sprintf("vdhelpers: failed to marshal JSON: %v", err),
		})
		return string(fallback)
	}
	return string(b)
}
