package main

import "testing"

func TestFindShardSplit(t *testing.T) {
	t.Run("short content", func(t *testing.T) {
		got := findShardSplit("hello", 10)
		if got != 5 {
			t.Fatalf("expected 5, got %d", got)
		}
	})

	t.Run("prefers nearby newline", func(t *testing.T) {
		content := "aaaaaaaaaa\nrest"
		got := findShardSplit(content, 11)
		if got != 11 {
			t.Fatalf("expected split at newline boundary 11, got %d", got)
		}
	})

	t.Run("falls back to hard limit", func(t *testing.T) {
		content := "abcdefghijklmnopqrstuvwxyz"
		got := findShardSplit(content, 10)
		if got != 10 {
			t.Fatalf("expected hard limit 10, got %d", got)
		}
	})
}
