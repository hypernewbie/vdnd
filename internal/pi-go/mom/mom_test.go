package mom

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- Sandbox Tests ---

func TestParseSandboxArg(t *testing.T) {
	tests := []struct {
		input     string
		wantType  SandboxType
		wantError bool
	}{
		{"host", SandboxHost, false},
		{"docker:my-container", SandboxDocker, false},
		{"docker:", "", true},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			config, err := ParseSandboxArg(tt.input)
			if tt.wantError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config.Type != tt.wantType {
				t.Errorf("type = %q, want %q", config.Type, tt.wantType)
			}
		})
	}
}

func TestHostExecutor(t *testing.T) {
	exec := &hostExecutor{}
	result, err := exec.Exec(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "hello" {
		t.Errorf("stdout = %q, want 'hello'", result.Stdout)
	}
	if result.Code != 0 {
		t.Errorf("code = %d, want 0", result.Code)
	}
}

func TestHostExecutor_ExitCode(t *testing.T) {
	exec := &hostExecutor{}
	result, err := exec.Exec(context.Background(), "exit 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 42 {
		t.Errorf("code = %d, want 42", result.Code)
	}
}

func TestHostExecutor_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	exec := &hostExecutor{}
	_, err := exec.Exec(ctx, "sleep 10")
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}

func TestShellEscape(t *testing.T) {
	result := shellEscape("hello 'world'")
	if !strings.Contains(result, "hello") {
		t.Error("should contain original text")
	}
}

// --- Store Tests ---

func TestChannelStore_GetChannelDir(t *testing.T) {
	dir := t.TempDir()
	store := NewChannelStore(ChannelStoreConfig{WorkingDir: dir, BotToken: "test"})

	channelDir := store.GetChannelDir("C123")
	if !filepath.IsAbs(channelDir) {
		t.Error("expected absolute path")
	}

	info, err := os.Stat(channelDir)
	if err != nil {
		t.Fatalf("channel dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestChannelStore_LogMessage(t *testing.T) {
	dir := t.TempDir()
	store := NewChannelStore(ChannelStoreConfig{WorkingDir: dir, BotToken: "test"})

	msg := LoggedMessage{
		Ts:    "1234567890.123456",
		User:  "U123",
		Text:  "hello world",
		IsBot: false,
	}

	ok, err := store.LogMessage("C123", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected message to be logged (first time)")
	}

	// Second call should be deduplicated
	ok2, err := store.LogMessage("C123", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok2 {
		t.Error("expected duplicate to be rejected")
	}

	// Verify JSONL file
	logPath := filepath.Join(dir, "C123", "log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(data), "hello world") {
		t.Error("expected message text in log")
	}
}

func TestChannelStore_GetLastTimestamp(t *testing.T) {
	dir := t.TempDir()
	store := NewChannelStore(ChannelStoreConfig{WorkingDir: dir, BotToken: "test"})

	// No log yet
	ts := store.GetLastTimestamp("C123")
	if ts != "" {
		t.Errorf("expected empty timestamp, got %q", ts)
	}

	// Log a message
	store.LogMessage("C123", LoggedMessage{
		Ts:   "1234567890.000001",
		User: "U1",
		Text: "first",
	})

	time.Sleep(10 * time.Millisecond) // Let file write complete

	ts = store.GetLastTimestamp("C123")
	if ts != "1234567890.000001" {
		t.Errorf("expected '1234567890.000001', got %q", ts)
	}
}

func TestChannelStore_GenerateLocalFilename(t *testing.T) {
	store := &ChannelStore{}
	filename := store.GenerateLocalFilename("my file (1).png", "1732531234.567")
	if !strings.Contains(filename, "my_file__1_.png") {
		t.Errorf("unexpected filename: %q", filename)
	}
}

// --- Settings Tests ---

func TestSettingsManager(t *testing.T) {
	dir := t.TempDir()
	sm := NewSettingsManager(dir)

	// Defaults
	comp := sm.GetCompaction()
	if !comp.Enabled {
		t.Error("compaction should be enabled by default")
	}
	if comp.ReserveTokens != 16384 {
		t.Errorf("expected 16384 reserve tokens, got %d", comp.ReserveTokens)
	}

	retry := sm.GetRetry()
	if !retry.Enabled {
		t.Error("retry should be enabled by default")
	}

	// Set and read back
	sm.SetCompactionEnabled(false)
	if sm.GetCompaction().Enabled {
		t.Error("compaction should be disabled")
	}

	sm.SetDefaultModelAndProvider("openai", "gpt-4o")
	if sm.GetDefaultProvider() != "openai" {
		t.Errorf("provider = %q, want 'openai'", sm.GetDefaultProvider())
	}
	if sm.GetDefaultModel() != "gpt-4o" {
		t.Errorf("model = %q, want 'gpt-4o'", sm.GetDefaultModel())
	}

	// Persist and reload
	sm2 := NewSettingsManager(dir)
	if sm2.GetDefaultProvider() != "openai" {
		t.Error("settings not persisted")
	}
	if sm2.GetCompaction().Enabled {
		t.Error("compaction setting not persisted")
	}
}

func TestSettingsManager_ThinkingLevel(t *testing.T) {
	dir := t.TempDir()
	sm := NewSettingsManager(dir)

	if sm.GetDefaultThinkingLevel() != "off" {
		t.Errorf("expected 'off' default, got %q", sm.GetDefaultThinkingLevel())
	}

	sm.SetDefaultThinkingLevel("high")
	if sm.GetDefaultThinkingLevel() != "high" {
		t.Errorf("expected 'high', got %q", sm.GetDefaultThinkingLevel())
	}
}

// --- Events Tests ---

func TestEventsWatcher_ImmediateEvent(t *testing.T) {
	dir := t.TempDir()
	received := make(chan MomEvent, 1)

	watcher := NewEventsWatcher(dir, func(event MomEvent) {
		received <- event
	})

	// Write an immediate event file
	event := MomEvent{
		Type:      EventImmediate,
		ChannelID: "C123",
		Text:      "test immediate",
	}
	data, _ := json.Marshal(event)
	os.WriteFile(filepath.Join(dir, "test.json"), data, 0644)

	watcher.Start()
	defer watcher.Stop()

	select {
	case e := <-received:
		if e.Text != "test immediate" {
			t.Errorf("text = %q, want 'test immediate'", e.Text)
		}
		if e.ChannelID != "C123" {
			t.Errorf("channelID = %q, want 'C123'", e.ChannelID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	// File should be deleted after processing
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(dir, "test.json")); err == nil {
		t.Error("event file should be deleted after processing")
	}
}

func TestEventsWatcher_OneShotPast(t *testing.T) {
	dir := t.TempDir()
	received := make(chan MomEvent, 1)

	watcher := NewEventsWatcher(dir, func(event MomEvent) {
		received <- event
	})

	// One-shot in the past → should fire immediately
	event := MomEvent{
		Type:      EventOneShot,
		ChannelID: "C123",
		Text:      "past event",
		At:        time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}
	data, _ := json.Marshal(event)
	os.WriteFile(filepath.Join(dir, "past.json"), data, 0644)

	watcher.Start()
	defer watcher.Stop()

	select {
	case e := <-received:
		if e.Text != "past event" {
			t.Errorf("text = %q", e.Text)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for past event")
	}
}

// --- Tools Tests ---

func TestCreateMomTools(t *testing.T) {
	exec := &hostExecutor{}
	tools := CreateMomTools(exec, "/tmp")
	if len(tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Tool.Name] = true
	}
	for _, expected := range []string{"read", "bash", "edit", "write"} {
		if !names[expected] {
			t.Errorf("missing tool %q", expected)
		}
	}
}

func TestBuildMomSystemPrompt(t *testing.T) {
	prompt := BuildMomSystemPrompt("/workspace", "C123", "remember to be nice",
		[]ChannelInfo{{ID: "C123", Name: "general"}},
		[]UserInfo{{ID: "U1", Name: "mario", DisplayName: "Mario"}},
	)

	if !strings.Contains(prompt, "mom") {
		t.Error("should mention mom")
	}
	if !strings.Contains(prompt, "/workspace") {
		t.Error("should contain workspace path")
	}
	if !strings.Contains(prompt, "remember to be nice") {
		t.Error("should contain memory")
	}
	if !strings.Contains(prompt, "#general") {
		t.Error("should contain channel name")
	}
	if !strings.Contains(prompt, "@Mario") {
		t.Error("should contain user display name")
	}
}

// --- Runner Cache Tests ---

func TestGetOrCreateRunner(t *testing.T) {
	// Reset global state
	channelRunnersMu.Lock()
	channelRunners = map[string]*AgentRunner{}
	channelRunnersMu.Unlock()

	config := RunnerConfig{
		SandboxConfig: SandboxConfig{Type: SandboxHost},
		ChannelID:     "C_TEST",
		WorkspacePath: t.TempDir(),
	}

	r1 := GetOrCreateRunner(config)
	r2 := GetOrCreateRunner(config)

	if r1 != r2 {
		t.Error("expected same runner instance for same channel")
	}

	RemoveRunner("C_TEST")

	r3 := GetOrCreateRunner(config)
	if r1 == r3 {
		t.Error("expected new runner after removal")
	}

	// Cleanup
	channelRunnersMu.Lock()
	channelRunners = map[string]*AgentRunner{}
	channelRunnersMu.Unlock()
}
