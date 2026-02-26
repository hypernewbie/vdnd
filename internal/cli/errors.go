package cli

import (
	"fmt"
	"strings"
)

// ErrorCategory represents the type of error to help the LLM understand context.
type ErrorCategory string

const (
	CatUsage      ErrorCategory = "Usage"      // Incorrect command syntax or missing arguments
	CatNotFound   ErrorCategory = "NotFound"   // Entity, item, or zone doesn't exist
	CatRule       ErrorCategory = "Rule"       // Violation of Pathfinder 2E rules
	CatState      ErrorCategory = "State"      // Issues with the game state (e.g., no scene loaded)
	CatSystem     ErrorCategory = "System"     // I/O or internal software errors
)

// VDError provides structured error information tailored for an LLM.
type VDError struct {
	Category ErrorCategory
	Message  string
	Hint     string
}

func (e *VDError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s Error] %s", e.Category, e.Message))
	if e.Hint != "" {
		sb.WriteString("
Hint: ")
		sb.WriteString(e.Hint)
	}
	return sb.String()
}

// NewUsageError creates an error for incorrect command usage.
func NewUsageError(msg string, usage string) error {
	return &VDError{
		Category: CatUsage,
		Message:  msg,
		Hint:     fmt.Sprintf("Correct usage: %s", usage),
	}
}

// NewNotFoundError creates an error when a resource is missing.
func NewNotFoundError(resourceType string, id string, suggestions string) error {
	msg := fmt.Sprintf("%s '%s' not found.", resourceType, id)
	hint := fmt.Sprintf("Use 'vd %s list' to see available %ss.", strings.ToLower(resourceType), strings.ToLower(resourceType))
	if suggestions != "" {
		hint = suggestions
	}
	return &VDError{
		Category: CatNotFound,
		Message:  msg,
		Hint:     hint,
	}
}

// NewRuleError creates an error for rule violations.
func NewRuleError(msg string, hint string) error {
	return &VDError{
		Category: CatRule,
		Message:  msg,
		Hint:     hint,
	}
}

// NewStateError creates an error for state-related issues.
func NewStateError(msg string, hint string) error {
	return &VDError{
		Category: CatState,
		Message:  msg,
		Hint:     hint,
	}
}

// WrapSystemError humanizes low-level errors.
func WrapSystemError(err error, context string) error {
	msg := fmt.Sprintf("%s: %v", context, err)
	hint := "Ensure you have the necessary permissions and that the 'state.json' file is not corrupted."
	
	if strings.Contains(err.Error(), "no such file or directory") {
		msg = "No active scene or state file found."
		hint = "Start a new scene with 'vd scene new <name>' or load one with 'vd scene load <path>'."
	}

	return &VDError{
		Category: CatSystem,
		Message:  msg,
		Hint:     hint,
	}
}
