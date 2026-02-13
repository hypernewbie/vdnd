package mom

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CompactionSettings configures auto-compaction behavior.
type CompactionSettings struct {
	Enabled          bool `json:"enabled"`
	ReserveTokens    int  `json:"reserveTokens"`
	KeepRecentTokens int  `json:"keepRecentTokens"`
}

// RetrySettings configures auto-retry behavior.
type RetrySettings struct {
	Enabled     bool `json:"enabled"`
	MaxRetries  int  `json:"maxRetries"`
	BaseDelayMs int  `json:"baseDelayMs"`
}

// MomSettings holds all mom settings.
type MomSettings struct {
	DefaultProvider      string              `json:"defaultProvider,omitempty"`
	DefaultModel         string              `json:"defaultModel,omitempty"`
	DefaultThinkingLevel string              `json:"defaultThinkingLevel,omitempty"`
	Compaction           *CompactionSettings `json:"compaction,omitempty"`
	Retry                *RetrySettings      `json:"retry,omitempty"`
}

// Default settings
var (
	DefaultCompaction = CompactionSettings{
		Enabled:          true,
		ReserveTokens:    16384,
		KeepRecentTokens: 20000,
	}
	DefaultRetry = RetrySettings{
		Enabled:     true,
		MaxRetries:  3,
		BaseDelayMs: 2000,
	}
)

// SettingsManager manages mom settings stored in a workspace directory.
type SettingsManager struct {
	settingsPath string
	settings     MomSettings
}

// NewSettingsManager creates a settings manager for the given workspace directory.
func NewSettingsManager(workspaceDir string) *SettingsManager {
	sm := &SettingsManager{
		settingsPath: filepath.Join(workspaceDir, "mom-settings.json"),
	}
	sm.Load()
	return sm
}

// Load reads settings from disk. Returns defaults if file doesn't exist.
func (sm *SettingsManager) Load() {
	data, err := os.ReadFile(sm.settingsPath)
	if err != nil {
		sm.settings = MomSettings{}
		return
	}
	if err := json.Unmarshal(data, &sm.settings); err != nil {
		sm.settings = MomSettings{}
	}
}

// Save writes settings to disk.
func (sm *SettingsManager) Save() error {
	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(sm.settingsPath)
	os.MkdirAll(dir, 0755)
	return os.WriteFile(sm.settingsPath, data, 0644)
}

// GetCompaction returns compaction settings with defaults.
func (sm *SettingsManager) GetCompaction() CompactionSettings {
	if sm.settings.Compaction != nil {
		return *sm.settings.Compaction
	}
	return DefaultCompaction
}

// SetCompactionEnabled enables/disables compaction.
func (sm *SettingsManager) SetCompactionEnabled(enabled bool) {
	if sm.settings.Compaction == nil {
		c := DefaultCompaction
		sm.settings.Compaction = &c
	}
	sm.settings.Compaction.Enabled = enabled
	sm.Save()
}

// GetRetry returns retry settings with defaults.
func (sm *SettingsManager) GetRetry() RetrySettings {
	if sm.settings.Retry != nil {
		return *sm.settings.Retry
	}
	return DefaultRetry
}

// SetRetryEnabled enables/disables retry.
func (sm *SettingsManager) SetRetryEnabled(enabled bool) {
	if sm.settings.Retry == nil {
		r := DefaultRetry
		sm.settings.Retry = &r
	}
	sm.settings.Retry.Enabled = enabled
	sm.Save()
}

// GetDefaultModel returns the default model ID.
func (sm *SettingsManager) GetDefaultModel() string {
	return sm.settings.DefaultModel
}

// GetDefaultProvider returns the default provider.
func (sm *SettingsManager) GetDefaultProvider() string {
	return sm.settings.DefaultProvider
}

// SetDefaultModelAndProvider sets the default model and provider.
func (sm *SettingsManager) SetDefaultModelAndProvider(provider, modelID string) {
	sm.settings.DefaultProvider = provider
	sm.settings.DefaultModel = modelID
	sm.Save()
}

// GetDefaultThinkingLevel returns the default thinking level.
func (sm *SettingsManager) GetDefaultThinkingLevel() string {
	if sm.settings.DefaultThinkingLevel == "" {
		return "off"
	}
	return sm.settings.DefaultThinkingLevel
}

// SetDefaultThinkingLevel sets the default thinking level.
func (sm *SettingsManager) SetDefaultThinkingLevel(level string) {
	sm.settings.DefaultThinkingLevel = level
	sm.Save()
}
