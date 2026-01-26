package llm

import (
	"testing"
)

func TestExtractThinking(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "with thought tags",
			content:  "<thought>thinking here</thought>actual content",
			expected: "thinking here",
		},
		{
			name:     "no thought tags",
			content:  "just content",
			expected: "",
		},
		{
			name:     "empty tags",
			content:  "<thought></thought>content",
			expected: "",
		},
		{
			name:     "multiple tags (first one extracted)",
			content:  "<thought>first</thought><thought>second</thought>",
			expected: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractThinking(tt.content)
			if got != tt.expected {
				t.Errorf("ExtractThinking() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStripThinking(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "with thought tags",
			content:  "<thought>thinking here</thought>actual content",
			expected: "actual content",
		},
		{
			name:     "no thought tags",
			content:  "just content",
			expected: "just content",
		},
		{
			name:     "multiple tags",
			content:  "<thought>first</thought>middle<thought>second</thought>last",
			expected: "middlelast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripThinking(tt.content)
			if got != tt.expected {
				t.Errorf("StripThinking() = %v, want %v", got, tt.expected)
			}
		})
	}
}
