package cli

import (
	"io"
	"strings"
	"testing"
)

func TestRun_Basic(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		contains string
	}{
		{
			name:     "no args",
			args:     []string{},
			wantCode: 0,
			contains: "Usage:",
		},
		{
			name:     "help",
			args:     []string{"help"},
			wantCode: 0,
			contains: "Usage:",
		},
		{
			name:     "unknown command",
			args:     []string{"foo"},
			wantCode: 1,
			contains: "unknown command",
		},
	}

	deps := DefaultDeps()
	deps.Stderr = io.Discard

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, code := Run(tt.args, deps)
			if code != tt.wantCode {
				t.Errorf("Run() code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(out, tt.contains) {
				t.Errorf("Run() output = %q, want to contain %q", out, tt.contains)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	args := []string{"pos1", "--flag1", "val1", "pos2", "--boolFlag"}
	pos, flags := ParseFlags(args)

	if len(pos) != 2 || pos[0] != "pos1" || pos[1] != "pos2" {
		t.Errorf("ParseFlags positional = %v, want [pos1 pos2]", pos)
	}
	if flags["flag1"] != "val1" {
		t.Errorf("ParseFlags flag1 = %q, want val1", flags["flag1"])
	}
	if flags["boolFlag"] != "true" {
		t.Errorf("ParseFlags boolFlag = %q, want true", flags["boolFlag"])
	}
}
