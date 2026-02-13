// vmom is a Discord bot that runs a coding agent in a workspace.
// It listens for @mentions and runs the pi-go/mom agent loop with
// bash/read/edit/write tools against a local workspace directory.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"uaa/vdnd/internal/pi-go/agent"
	"uaa/vdnd/internal/pi-go/ai"
	_ "uaa/vdnd/internal/pi-go/ai/anthropic"       // Register Anthropic provider
	_ "uaa/vdnd/internal/pi-go/ai/google"          // Register Google provider
	_ "uaa/vdnd/internal/pi-go/ai/openaicompat"    // Register OpenAI Completions provider
	_ "uaa/vdnd/internal/pi-go/ai/openairesponses" // Register OpenAI Responses provider
	"uaa/vdnd/internal/pi-go/mom"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds the application configuration, loaded from env vars / .env file.
type Config struct {
	Token           string `env:"DISCORD_TOKEN"`
	AnthropicKey    string `env:"ANTHROPIC_API_KEY"`
	OpenAIKey       string `env:"OPENAI_API_KEY"`
	DeepSeekKey     string `env:"DEEPSEEK_API_KEY"`
	GoogleKey       string `env:"GOOGLE_API_KEY"`
	Provider        string `env:"VMOM_PROVIDER" envDefault:"anthropic"`
	Model           string `env:"VMOM_MODEL" envDefault:"claude-sonnet-4-5"`
	BaseURL         string `env:"VMOM_BASE_URL"` // Custom API base URL (e.g. https://api.deepseek.com)
	Workspace       string `env:"VMOM_WORKSPACE" envDefault:"./workspace"`
	Sandbox         string `env:"VMOM_SANDBOX" envDefault:"host"`
	AllowedChannels string `env:"VMOM_ALLOWED_CHANNELS"` // comma-separated channel IDs, empty = all
	RemoveCommands  bool   `env:"DISCORD_REMOVE_COMMANDS" envDefault:"true"`
}

func loadConfig() (*Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// resolveAPIKey returns the API key for the configured provider.
func (c *Config) resolveAPIKey() (string, error) {
	switch c.Provider {
	case "anthropic":
		if c.AnthropicKey != "" {
			return c.AnthropicKey, nil
		}
	case "openai":
		if c.OpenAIKey != "" {
			return c.OpenAIKey, nil
		}
	case "deepseek":
		if c.DeepSeekKey != "" {
			return c.DeepSeekKey, nil
		}
	case "google":
		if c.GoogleKey != "" {
			return c.GoogleKey, nil
		}
	}
	// Fallback to env lookup
	key := ai.GetEnvApiKey(c.Provider)
	if key != "" {
		return key, nil
	}
	return "", fmt.Errorf("no API key found for provider %q", c.Provider)
}

// isChannelAllowed checks if a channel is in the allow list (empty = all allowed).
func (c *Config) isChannelAllowed(channelID string) bool {
	if c.AllowedChannels == "" {
		return true
	}
	for _, id := range strings.Split(c.AllowedChannels, ",") {
		if strings.TrimSpace(id) == channelID {
			return true
		}
	}
	return false
}

func main() {
	// Open log file
	logFile, err := os.OpenFile("vmom.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	// Load .env file if it exists
	_ = godotenv.Load()

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "DISCORD_TOKEN is required")
		os.Exit(1)
	}

	// Validate sandbox
	sandboxConfig, err := mom.ParseSandboxArg(cfg.Sandbox)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid VMOM_SANDBOX: %v\n", err)
		os.Exit(1)
	}

	if err := mom.ValidateSandbox(context.Background(), sandboxConfig); err != nil {
		fmt.Fprintf(os.Stderr, "sandbox validation failed: %v\n", err)
		os.Exit(1)
	}

	// Ensure workspace exists
	os.MkdirAll(cfg.Workspace, 0755)

	// Create the single global runner
	runner := mom.NewAgentRunner(mom.RunnerConfig{
		SandboxConfig: sandboxConfig,
		ChannelID:     "global",
		ChannelDir:    cfg.Workspace,
		WorkspacePath: cfg.Workspace,
		BaseURL:       cfg.BaseURL,
		Provider:      cfg.Provider,
		ModelID:       cfg.Model,
		GetAPIKey: func() (string, error) {
			return cfg.resolveAPIKey()
		},
		Settings: mom.NewSettingsManager(cfg.Workspace),
	})

	// Create Discord session
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		slog.Error("invalid bot parameters", "error", err)
		os.Exit(1)
	}
	s.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentMessageContent

	// Bot state
	bot := &Bot{
		cfg:    cfg,
		runner: runner,
	}

	// Register handlers
	s.AddHandler(bot.handleReady)
	s.AddHandler(bot.handleMessage)
	s.AddHandler(bot.handleInteraction(s))

	// Open session
	if err := s.Open(); err != nil {
		slog.Error("cannot open session", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	// Register slash commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "vstop",
			Description: "Abort the current vmom agent run",
		},
		{
			Name:        "vstatus",
			Description: "Show vmom agent status",
		},
	}

	registered := make([]*discordgo.ApplicationCommand, 0, len(commands))
	for _, cmd := range commands {
		c, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			slog.Error("cannot create command", "name", cmd.Name, "error", err)
			continue
		}
		registered = append(registered, c)
		slog.Info("registered command", "name", cmd.Name)
	}

	slog.Info("vmom is running", "workspace", cfg.Workspace, "provider", cfg.Provider, "model", cfg.Model)
	fmt.Printf("vmom is running (workspace: %s, provider: %s, model: %s)\n", cfg.Workspace, cfg.Provider, cfg.Model)
	fmt.Println("Press CTRL-C to exit.")

	// Wait for shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	slog.Info("shutting down...")
	fmt.Println("\nShutting down...")

	// Abort any running agent
	runner.Abort()

	// Cleanup commands
	if cfg.RemoveCommands {
		for _, cmd := range registered {
			s.ApplicationCommandDelete(s.State.User.ID, "", cmd.ID)
		}
	}
}

// Bot holds the bot state and handles Discord events.
type Bot struct {
	cfg    *Config
	runner *mom.AgentRunner
	botID  string
	mu     sync.Mutex
}

func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	b.botID = r.User.ID
	slog.Info("bot ready", "user", r.User.Username, "id", r.User.ID)
}

// handleMessage processes @mention messages and runs the agent.
func (b *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore own messages
	if m.Author.ID == b.botID {
		return
	}

	// Check if this channel is allowed
	if !b.cfg.isChannelAllowed(m.ChannelID) {
		return
	}

	// Only respond to @mentions of the bot
	if !isMentioned(m.Message, b.botID) {
		return
	}

	// Strip the mention from the message text
	text := stripMention(m.Content, b.botID)
	text = strings.TrimSpace(text)
	if text == "" {
		text = "Hello! What can I help you with?"
	}

	userName := getDisplayName(m.Author)
	slog.Info("received mention",
		"channel", m.ChannelID,
		"user", userName,
		"text", text,
	)

	// Check if agent is busy
	if b.runner.IsRunning() {
		s.ChannelMessageSend(m.ChannelID, "⏳ I'm currently working on something. Use `/vstop` to abort, or wait for me to finish.")
		return
	}

	// Show typing indicator
	s.ChannelTyping(m.ChannelID)

	// Subscribe to agent events to stream responses
	var responseText strings.Builder
	unsubscribe := b.runner.Subscribe(func(e agent.Event) {
		switch e.Type {
		case agent.EventMessageEnd:
			if e.Message != nil && e.Message.Assistant != nil {
				for _, block := range e.Message.Assistant.Content {
					if block.Type == ai.ContentTypeText && block.Text != "" {
						responseText.WriteString(block.Text)
					}
				}
			}
		case agent.EventToolExecutionEnd:
			// Refresh typing indicator during tool execution
			s.ChannelTyping(m.ChannelID)
		}
	})

	// Run agent in background
	go func() {
		defer unsubscribe()

		prompt := fmt.Sprintf("[%s] %s", userName, text)
		result, err := b.runner.Run(context.Background(), prompt)

		slog.Info("agent finished",
			"stopReason", result.StopReason,
			"error", result.ErrorMessage,
		)

		response := responseText.String()
		if err != nil && response == "" {
			response = fmt.Sprintf("❌ Error: %s", err.Error())
		}
		if response == "" {
			response = "✅ Done (no text output)"
		}

		// Send response, chunked for Discord's 2000-char limit
		sendChunked(s, m.ChannelID, response)
	}()
}

// handleInteraction handles slash commands (/vstop, /vstatus).
func (b *Bot) handleInteraction(sess *discordgo.Session) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		name := i.ApplicationCommandData().Name

		switch name {
		case "vstop":
			if b.runner.IsRunning() {
				b.runner.Abort()
				respond(s, i, "🛑 Agent has been aborted.")
			} else {
				respond(s, i, "Agent is idle.")
			}

		case "vstatus":
			status := "idle"
			if b.runner.IsRunning() {
				status = "running"
			}
			respond(s, i, fmt.Sprintf(
				"**vmom status**\n"+
					"**Status:** %s\n"+
					"**Provider:** %s\n"+
					"**Model:** %s\n"+
					"**Workspace:** %s\n"+
					"**Sandbox:** %s",
				status, b.cfg.Provider, b.cfg.Model, b.cfg.Workspace, b.cfg.Sandbox,
			))
		}
	}
}

// --- Helpers ---

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}

func isMentioned(m *discordgo.Message, botID string) bool {
	for _, u := range m.Mentions {
		if u.ID == botID {
			return true
		}
	}
	return false
}

func stripMention(content string, botID string) string {
	// Discord mentions look like <@123456> or <@!123456>
	content = strings.ReplaceAll(content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")
	return strings.TrimSpace(content)
}

func getDisplayName(u *discordgo.User) string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}

// sendChunked sends a message split into chunks respecting Discord's 2000-char limit.
func sendChunked(s *discordgo.Session, channelID string, text string) {
	const maxLen = 1900 // Leave some margin

	for len(text) > 0 {
		chunk := text
		if len(chunk) > maxLen {
			// Try to break at a newline
			idx := strings.LastIndex(chunk[:maxLen], "\n")
			if idx < maxLen/2 {
				idx = maxLen // No good newline break, just cut
			}
			chunk = text[:idx]
			text = text[idx:]
		} else {
			text = ""
		}

		_, err := s.ChannelMessageSend(channelID, chunk)
		if err != nil {
			slog.Error("failed to send message", "error", err)
			return
		}

		// Small delay between chunks to avoid rate limits
		if text != "" {
			time.Sleep(500 * time.Millisecond)
		}
	}
}
