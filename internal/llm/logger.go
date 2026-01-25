package llm

import (
	"fmt"
	"os"
	"time"
)

const LogFile = "logging.txt"

// LogActivity appends a formatted message to the project's logging.txt file.
func LogActivity(role, content string) {
	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s: %s\n---\n", timestamp, role, content)

	if _, err := f.WriteString(entry); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write to log file: %v\n", err)
	}
}
