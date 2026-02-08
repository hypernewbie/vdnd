package llm

import (
	"strings"
)

func ExtractThinking(content string) string {
	startTag := "<thought>"
	endTag := "</thought>"

	start := strings.Index(content, startTag)
	end := strings.Index(content, endTag)

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(content[start+len(startTag) : end])
	}

	return ""
}

func StripThinking(content string) string {
	startTag := "<thought>"
	endTag := "</thought>"

	for {
		start := strings.Index(content, startTag)
		end := strings.Index(content, endTag)

		if start != -1 && end != -1 && end > start {
			content = content[:start] + content[end+len(endTag):]
			continue
		}
		break
	}
	return strings.TrimSpace(content)
}