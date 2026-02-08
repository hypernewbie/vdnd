package rlm

import (
	"fmt"
	"os"
)

// NewDMSystemPromptBuilder creates a SystemPromptBuilder that captures the DM prompt from a file.
func NewDMSystemPromptBuilder(path string) (SystemPromptBuilder, error) {
	if path == "" {
		path = "vdm_prompt.txt"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	template := string(data)
	return func(contextSize int, depth int) string {
		return fmt.Sprintf(template, depth)
	}, nil
}
