package codingagent

import (
	"fmt"
	"strings"
	"time"
)

// ToolDescription maps tool names to short descriptions for the system prompt.
var ToolDescriptions = map[string]string{
	"read":  "Read file contents",
	"bash":  "Execute bash commands (ls, grep, find, etc.)",
	"edit":  "Make surgical edits to files (find exact text and replace)",
	"write": "Create or overwrite files",
	"grep":  "Search file contents for patterns (respects .gitignore)",
	"find":  "Find files by glob pattern (respects .gitignore)",
	"ls":    "List directory contents",
}

// ContextFile is a pre-loaded context file for the system prompt.
type ContextFile struct {
	Path    string
	Content string
}

// BuildSystemPromptOptions configures system prompt construction.
type BuildSystemPromptOptions struct {
	CustomPrompt       string
	SelectedTools      []string
	AppendSystemPrompt string
	Cwd                string
	ContextFiles       []ContextFile
}

// BuildSystemPrompt constructs the full system prompt with tool descriptions and guidelines.
func BuildSystemPrompt(opts BuildSystemPromptOptions) string {
	cwd := opts.Cwd
	if cwd == "" {
		cwd = "."
	}

	now := time.Now()
	dateTime := now.Format("Monday, January 2, 2006 3:04:05 PM MST")

	appendSection := ""
	if opts.AppendSystemPrompt != "" {
		appendSection = "\n\n" + opts.AppendSystemPrompt
	}

	// Custom prompt path
	if opts.CustomPrompt != "" {
		prompt := opts.CustomPrompt
		if appendSection != "" {
			prompt += appendSection
		}

		if len(opts.ContextFiles) > 0 {
			prompt += "\n\n# Project Context\n\nProject-specific instructions and guidelines:\n\n"
			for _, cf := range opts.ContextFiles {
				prompt += fmt.Sprintf("## %s\n\n%s\n\n", cf.Path, cf.Content)
			}
		}

		prompt += fmt.Sprintf("\nCurrent date and time: %s", dateTime)
		prompt += fmt.Sprintf("\nCurrent working directory: %s", cwd)
		return prompt
	}

	// Default system prompt
	tools := opts.SelectedTools
	if len(tools) == 0 {
		tools = []string{"read", "bash", "edit", "write"}
	}

	// Filter to known tools only
	var knownTools []string
	for _, t := range tools {
		if _, ok := ToolDescriptions[t]; ok {
			knownTools = append(knownTools, t)
		}
	}

	toolsList := "(none)"
	if len(knownTools) > 0 {
		var lines []string
		for _, t := range knownTools {
			lines = append(lines, fmt.Sprintf("- %s: %s", t, ToolDescriptions[t]))
		}
		toolsList = strings.Join(lines, "\n")
	}

	// Build guidelines
	toolSet := map[string]bool{}
	for _, t := range knownTools {
		toolSet[t] = true
	}

	var guidelines []string

	hasBash := toolSet["bash"]
	hasEdit := toolSet["edit"]
	hasWrite := toolSet["write"]
	hasGrep := toolSet["grep"]
	hasFind := toolSet["find"]
	hasLs := toolSet["ls"]
	hasRead := toolSet["read"]

	if hasBash && !hasGrep && !hasFind && !hasLs {
		guidelines = append(guidelines, "Use bash for file operations like ls, rg, find")
	} else if hasBash && (hasGrep || hasFind || hasLs) {
		guidelines = append(guidelines, "Prefer grep/find/ls tools over bash for file exploration (faster, respects .gitignore)")
	}

	if hasRead && hasEdit {
		guidelines = append(guidelines, "Use read to examine files before editing. You must use this tool instead of cat or sed.")
	}
	if hasEdit {
		guidelines = append(guidelines, "Use edit for precise changes (old text must match exactly)")
	}
	if hasWrite {
		guidelines = append(guidelines, "Use write only for new files or complete rewrites")
	}
	if hasEdit || hasWrite {
		guidelines = append(guidelines, "When summarizing your actions, output plain text directly - do NOT use cat or bash to display what you did")
	}

	guidelines = append(guidelines, "Be concise in your responses")
	guidelines = append(guidelines, "Show file paths clearly when working with files")

	guidelinesText := ""
	for _, g := range guidelines {
		guidelinesText += "- " + g + "\n"
	}

	prompt := fmt.Sprintf(`You are an expert coding assistant operating inside pi, a coding agent harness. You help users by reading files, executing commands, editing code, and writing new files.

Available tools:
%s

In addition to the tools above, you may have access to other custom tools depending on the project.

Guidelines:
%s`, toolsList, guidelinesText)

	if appendSection != "" {
		prompt += appendSection
	}

	if len(opts.ContextFiles) > 0 {
		prompt += "\n\n# Project Context\n\nProject-specific instructions and guidelines:\n\n"
		for _, cf := range opts.ContextFiles {
			prompt += fmt.Sprintf("## %s\n\n%s\n\n", cf.Path, cf.Content)
		}
	}

	prompt += fmt.Sprintf("\nCurrent date and time: %s", dateTime)
	prompt += fmt.Sprintf("\nCurrent working directory: %s", cwd)

	return prompt
}
