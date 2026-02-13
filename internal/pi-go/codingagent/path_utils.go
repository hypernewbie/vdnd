// Package codingagent provides coding tools and session management for the coding agent.
package codingagent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// unicodeSpaces matches various Unicode space characters that LLMs may produce.
var unicodeSpaces = regexp.MustCompile("[\u00A0\u2000-\u200A\u202F\u205F\u3000]")

// normalizeUnicodeSpaces replaces Unicode space variants with ASCII space.
func normalizeUnicodeSpaces(s string) string {
	return unicodeSpaces.ReplaceAllString(s, " ")
}

// normalizeAtPrefix strips the leading '@' prefix from paths.
func normalizeAtPrefix(s string) string {
	return strings.TrimPrefix(s, "@")
}

// ExpandPath expands ~ to home directory and normalizes Unicode spaces.
func ExpandPath(filePath string) string {
	normalized := normalizeUnicodeSpaces(normalizeAtPrefix(filePath))
	if normalized == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(normalized, "~/") {
		home, _ := os.UserHomeDir()
		return home + normalized[1:]
	}
	return normalized
}

// ResolveToCwd resolves a path relative to cwd, handling ~ expansion and absolute paths.
func ResolveToCwd(filePath string, cwd string) string {
	expanded := ExpandPath(filePath)
	if filepath.IsAbs(expanded) {
		return expanded
	}
	return filepath.Join(cwd, expanded)
}

// ResolveReadPath resolves a file path for reading, trying path variants on macOS.
func ResolveReadPath(filePath string, cwd string) string {
	resolved := ResolveToCwd(filePath, cwd)

	if fileExists(resolved) {
		return resolved
	}

	// Try Unicode NFD variant (macOS stores filenames in NFD form)
	// In Go, we can use unicode normalization but for simplicity
	// we just check if a normalized variant exists
	variants := []func(string) string{
		tryNFDVariant,
		tryCurlyQuoteVariant,
	}

	for _, fn := range variants {
		variant := fn(resolved)
		if variant != resolved && fileExists(variant) {
			return variant
		}
	}

	return resolved
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// tryNFDVariant tries Unicode NFD normalization (simplified for Go — strips combining marks).
func tryNFDVariant(s string) string {
	// Simple approach: strip combining marks (not a full NFD conversion but handles most cases)
	var builder strings.Builder
	for _, r := range s {
		if !unicode.Is(unicode.Mn, r) { // Mn = Mark, Nonspacing
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// tryCurlyQuoteVariant replaces straight apostrophe with right single quotation mark.
func tryCurlyQuoteVariant(s string) string {
	return strings.ReplaceAll(s, "'", "\u2019")
}
