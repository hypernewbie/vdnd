package output

import "strings"

// Output provides a fluent builder API for consistent markdown formatting.
type Output struct {
	buf strings.Builder
}

// New creates a new Output builder.
func New() *Output {
	return &Output{}
}

// Header adds a markdown header with the given level and text.
func (o *Output) Header(level int, text string) *Output {
	o.buf.WriteString(strings.Repeat("#", level) + " " + text + "\n")
	return o
}

// Field adds a bold field label with a value.
func (o *Output) Field(label, value string) *Output {
	o.buf.WriteString("**" + label + ":** " + value + "\n")
	return o
}

// Section adds a level 2 section header.
func (o *Output) Section(title string) *Output {
	o.buf.WriteString("\n## " + title + "\n")
	return o
}

// ListItem adds a bulleted list item.
func (o *Output) ListItem(text string) *Output {
	o.buf.WriteString("- " + text + "\n")
	return o
}

// Table adds a markdown table with headers and rows.
func (o *Output) Table(headers []string, rows [][]string) *Output {
	o.buf.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	o.buf.WriteString("|" + strings.Repeat("---|", len(headers)) + "\n")
	for _, row := range rows {
		o.buf.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	return o
}

// Result adds a result entry with bold label (alias for ListItem with bold label).
func (o *Output) Result(label, value string) *Output {
	o.buf.WriteString("- **" + label + ":** " + value + "\n")
	return o
}

// String returns the built markdown string.
func (o *Output) String() string {
	return o.buf.String()
}
