package ripgrep

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Match struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

type Result struct {
	RGInstalled bool    `json:"rg_installed"`
	Warning     string  `json:"warning,omitempty"`
	Matches     []Match `json:"matches"`
}

var execLookPath = exec.LookPath
var execCommand = exec.Command

// Search executes ripgrep with the given pattern and optional path.
// If rg is not installed, a loud warning is printed to stderr.
func Search(pattern, path string) (*Result, error) {
	// Detection
	rgPath, err := execLookPath("rg")
	installed := err == nil

	result := &Result{
		RGInstalled: installed,
		Matches:     []Match{},
	}

	if !installed {
		warning := `🔥🔥🔥 RIPGREP NOT INSTALLED! 🔥🔥🔥
For fast rule searching, install ripgrep:
    sudo apt update && sudo apt install ripgrep
Falling back to slower Python regex search.
`
		fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", warning)
		result.Warning = warning
		return result, nil
	}

	if path == "" {
		path = "rules/"
	}

	cmd := execCommand(rgPath, "-n", "-i", pattern, path)
	output, err := cmd.Output()
	if err != nil {
		// rg returns exit code 1 when no matches are found; treat as empty result
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return result, nil
		}
		return nil, err
	}

	result.Matches = parseRGOutput(string(output))
	return result, nil
}

func parseRGOutput(output string) []Match {
	var matches []Match
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		var match Match
		fmt.Sscanf(parts[1], "%d", &match.Line)
		match.File = parts[0]
		match.Text = parts[2]
		matches = append(matches, match)
	}
	return matches
}

// ToJSON returns the result as a JSON string suitable for tool responses.
func (r *Result) ToJSON() string {
	b, _ := json.Marshal(r)
	return string(b)
}