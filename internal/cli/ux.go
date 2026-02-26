package cli

import (
	"fmt"
	"strings"
)

const (
	ansiReset     = "\033[0m"
	draculaFG     = "\033[38;2;248;248;242m"
	draculaMute   = "\033[38;2;98;114;164m"
	draculaCyan   = "\033[38;2;139;233;253m"
	draculaGreen  = "\033[38;2;80;250;123m"
	draculaOrange = "\033[38;2;255;184;108m"
	draculaPink   = "\033[38;2;255;121;198m"
	draculaPurple = "\033[38;2;189;147;249m"
	draculaRed    = "\033[38;2;255;85;85m"
	draculaYellow = "\033[38;2;241;250;140m"
)

func uxPrintf(color string, format string, args ...any) {
	fmt.Printf("%s%s%s", color, fmt.Sprintf(format, args...), ansiReset)
}

func truncateUX(text string, max int) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= max {
		return trimmed
	}
	if max <= 3 {
		return trimmed[:max]
	}
	return trimmed[:max-3] + "..."
}

func PrintDraculaBanner() {
	banner := []struct {
		color string
		text  string
	}{
		{draculaPurple, "      _,."},
		{draculaPink, "    ,` -.)"},
		{draculaCyan, "   ( _/-\\-._"},
		{draculaPurple, "  /,|`--._,-^|            V I R T U A L"},
		{draculaPink, "  \\_| |`-._/||            D U N G E O N"},
		{draculaCyan, "    |  `-, / |            M A S T E R"},
		{draculaPurple, "    |     || |"},
		{draculaPink, "     `r-._||/   ^"},
		{draculaCyan, " __,-<_     )`-/  ` ."},
		{draculaPurple, "'  \\   `---'   \\     '"},
		{draculaPink, "    |           |.--'"},
		{draculaCyan, "    /           /"},
	}
	for _, line := range banner {
		uxPrintf(line.color, "%s\n", line.text)
	}
	uxPrintf(draculaFG, "\n")
}

func PrintStartupConfig(provider, model string, contextBytes int) {
	uxPrintf(draculaCyan, "[Startup] Provider=%s Model=%s ContextBytes=%d\n", provider, model, contextBytes)
}

func PrintPlayerInput(input string) {
	uxPrintf(draculaFG, "[Player] %s\n", truncateUX(input, 180))
}

func PrintSubagentInvocation(name, payload string) {
	summary := truncateUX(payload, 140)
	switch name {
	case "call_research_assistant":
		uxPrintf(draculaPurple, "[Orchestrator -> ResearchAssistant] %s\n", summary)
	case "call_vdm_execution":
		uxPrintf(draculaOrange, "[Orchestrator -> ExecutionAgent] %s\n", summary)
	case "execute_python":
		uxPrintf(draculaGreen, "[Orchestrator -> Sandbox] %s\n", summary)
	default:
		uxPrintf(draculaCyan, "[Orchestrator -> %s] %s\n", name, summary)
	}
}

func PrintSandboxExecution() {
	uxPrintf(draculaMute, "[Sandbox] Executing Python...\n")
}

func PrintVDEngineExecution(toolName, args string) {
	uxPrintf(draculaGreen, "  [VD Engine] %s %s\n", toolName, truncateUX(args, 120))
}

func PrintWarning(msg string) {
	uxPrintf(draculaYellow, "[Warning] %s\n", truncateUX(msg, 220))
}

func PrintError(msg string) {
	uxPrintf(draculaRed, "[Error] %s\n", truncateUX(msg, 220))
}

func PrintInfo(msg string) {
	uxPrintf(draculaCyan, "[Info] %s\n", truncateUX(msg, 220))
}
