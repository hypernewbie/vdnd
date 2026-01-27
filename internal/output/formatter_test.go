package output

import (
	"strings"
	"testing"
)

func TestOutputBuilder(t *testing.T) {
	o := New().
		Header(1, "Test Title").
		Field("Key", "Value").
		Section("Details").
		ListItem("Item 1").
		ListItem("Item 2").
		Table(
			[]string{"Col1", "Col2"},
			[][]string{
				{"A", "B"},
				{"C", "D"},
			},
		).
		Result("Outcome", "Success")

	result := o.String()

	// Verify key elements
	if !strings.Contains(result, "# Test Title") {
		t.Error("Expected header not found")
	}
	if !strings.Contains(result, "**Key:** Value") {
		t.Error("Expected field not found")
	}
	if !strings.Contains(result, "## Details") {
		t.Error("Expected section not found")
	}
	if !strings.Contains(result, "- Item 1") {
		t.Error("Expected list item not found")
	}
	if !strings.Contains(result, "| Col1 | Col2 |") {
		t.Error("Expected table header not found")
	}
	if !strings.Contains(result, "- **Outcome:** Success") {
		t.Error("Expected result not found")
	}
}
