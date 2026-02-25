package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/state"

	"github.com/bwmarrin/discordgo"
)

type mockDiscordSession struct {
	handlers        []interface{}
	commandsCreated []*discordgo.ApplicationCommand
	commandsDeleted []string
	openError       error
}

func (m *mockDiscordSession) AddHandler(handler interface{}) func() {
	m.handlers = append(m.handlers, handler)
	return func() {}
}

func (m *mockDiscordSession) ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	m.commandsCreated = append(m.commandsCreated, cmd)
	cmdCopy := *cmd
	cmdCopy.ID = "mock-id-" + cmd.Name
	return &cmdCopy, nil
}

func (m *mockDiscordSession) ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
	m.commandsDeleted = append(m.commandsDeleted, cmdID)
	return nil
}

func (m *mockDiscordSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	return nil
}

func (m *mockDiscordSession) Open() error {
	return m.openError
}

func (m *mockDiscordSession) Close() error { return nil }

func (m *mockDiscordSession) GetState() *discordgo.State {
	return &discordgo.State{
		Ready: discordgo.Ready{
			User: &discordgo.User{ID: "bot-id"},
		},
	}
}

func (m *mockDiscordSession) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error) {
	return nil, nil
}

func mockDeps() cli.Deps {
	return cli.Deps{
		Roller: &cli.CryptoRoller{},
		Store: &state.MemoryStore{State: &state.GameState{
			SceneName:     "Test Scene",
			Positions:     make(map[string]*state.Zone),
			Entities:      make(map[string]*state.EntityState),
			ReactionsUsed: make(map[string]bool),
		}},
		Clock:  &cli.RealClock{},
		Stderr: io.Discard,
	}
}

type mockProvider struct {
	supportsToolCalling bool
	generateResponse    string
	generateError       error
	toolCallResponse    llmtypes.GenerationResponse
}

func (m *mockProvider) Name() string              { return "mock" }
func (m *mockProvider) ModelName() string         { return "mock-model" }
func (m *mockProvider) SupportsToolCalling() bool { return m.supportsToolCalling }
func (m *mockProvider) Close() error              { return nil }
func (m *mockProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	return m.generateResponse, m.generateError
}
func (m *mockProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	return m.toolCallResponse, m.generateError
}
func (m *mockProvider) GenerateStream(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool, callback func(chunk string) error) (llmtypes.GenerationResponse, error) {
	resp, err := m.GenerateWithTools(ctx, messages, tools)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}
	if resp.Content != "" && callback != nil {
		if err := callback(resp.Content); err != nil {
			return llmtypes.GenerationResponse{}, err
		}
	}
	return resp, nil
}

type mockRLM struct {
	response string
	thinking string
	err      error
}

func (m *mockRLM) Complete(ctx context.Context, query string, contextData string, history []llmtypes.Message) (string, string, error) {
	return m.response, m.thinking, m.err
}

func TestRunCLI_EchoLoop(t *testing.T) {
	input := "echo hello world\nexit\n"
	in := strings.NewReader(input)
	out := new(bytes.Buffer)
	cfg := &Config{LLMProvider: "none"}

	runCLI(context.Background(), in, out, cfg, nil, nil, mockDeps())

	outputStr := out.String()
	if !strings.Contains(outputStr, "Standard CLI mode enabled") {
		t.Errorf("Expected startup message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "Echo: hello world") {
		t.Errorf("Expected echo response, got: %s", outputStr)
	}
}

func TestRunCLI_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Unknown command",
			input:    "magic-spell\nexit\n",
			expected: []string{"Unknown command: magic-spell"},
		},
		{
			name:     "Empty input",
			input:    "\n\nexit\n",
			expected: []string{"> > > "},
		},
		{
			name:     "Echo usage",
			input:    "echo\nexit\n",
			expected: []string{"Usage: echo <content>"},
		},
		{
			name:     "Multiple words echo",
			input:    "echo hello world again\nexit\n",
			expected: []string{"Echo: hello world again"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := new(bytes.Buffer)
			cfg := &Config{LLMProvider: "none"}

			runCLI(context.Background(), in, out, cfg, nil, nil, mockDeps())

			output := out.String()
			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("Expected output to contain '%s', got: %s", exp, output)
				}
			}
		})
	}
}

// TestRunCLI_LLMMode tests the CLI with LLM providers
func TestRunCLI_LLMMode(t *testing.T) {
	t.Run("BasicInteraction", func(t *testing.T) {
		input := "Hello DM\nexit\n"
		in := strings.NewReader(input)
		out := new(bytes.Buffer)
		cfg := &Config{
			LLMProvider: "mock",
			LLMModel:    "test-model",
			HistoryFile: t.TempDir() + "/history.json",
		}

		mockProv := &mockProvider{
			supportsToolCalling: true,
			toolCallResponse: llmtypes.GenerationResponse{
				Content:      "Welcome adventurer!",
				FinishReason: "stop",
			},
		}

		runCLI(context.Background(), in, out, cfg, mockProv, nil, mockDeps())

		output := out.String()
		if !strings.Contains(output, "LLM mode enabled") {
			t.Errorf("Expected LLM mode enabled message")
		}
		if !strings.Contains(output, "DM: Welcome adventurer!") {
			t.Errorf("Expected DM response, got: %s", output)
		}
	})

	t.Run("ProviderError", func(t *testing.T) {
		// Test behavior when provider fails
		input := "Hello DM\nexit\n"
		in := strings.NewReader(input)
		out := new(bytes.Buffer)
		cfg := &Config{
			LLMProvider: "mock",
			HistoryFile: t.TempDir() + "/history.json",
		}
		mockProv := &mockProvider{generateError: fmt.Errorf("api failure")}

		runCLI(context.Background(), in, out, cfg, mockProv, nil, mockDeps())

		output := out.String()
		if !strings.Contains(output, "Error: api failure") {
			t.Errorf("Expected error message, got: %s", output)
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		os.Unsetenv("DISCORD_TOKEN")
		os.Unsetenv("DISCORD_REMOVE_COMMANDS")
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GROQ_API_KEY")
		os.Unsetenv("LLM_PROVIDER")
		os.Unsetenv("LLM_MODEL")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if cfg.LLMProvider != "groq" {
			t.Errorf("Expected LLMProvider default 'groq', got '%s'", cfg.LLMProvider)
		}
	})

	t.Run("EnvVarOverrides", func(t *testing.T) {
		t.Setenv("DISCORD_TOKEN", "test-token")
		t.Setenv("LLM_PROVIDER", "gemini")
		t.Setenv("DISCORD_REMOVE_COMMANDS", "false")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if cfg.Token != "test-token" {
			t.Errorf("Expected Token 'test-token', got '%s'", cfg.Token)
		}
		if cfg.RemoveCommands != false {
			t.Errorf("Expected RemoveCommands false, got %v", cfg.RemoveCommands)
		}
	})
}

func TestParseConfig(t *testing.T) {
	t.Run("FlagOverrides", func(t *testing.T) {
		t.Setenv("LLM_PROVIDER", "groq")
		args := []string{"-provider", "gemini", "-discord", "-verbose"}
		cfg, useDiscord, verbose, err := parseConfig(args)
		if err != nil {
			t.Fatalf("parseConfig failed: %v", err)
		}

		if cfg.LLMProvider != "gemini" {
			t.Errorf("Expected LLMProvider 'gemini', got '%s'", cfg.LLMProvider)
		}
		if !useDiscord || !verbose {
			t.Errorf("Expected flags true, got useDiscord=%v, verbose=%v", useDiscord, verbose)
		}
	})

	t.Run("InvalidFlag", func(t *testing.T) {
		args := []string{"-invalid"}
		_, _, _, err := parseConfig(args)
		if err == nil {
			t.Errorf("Expected error for invalid flag")
		}
	})
}

func TestRunDiscord(t *testing.T) {
	cfg := &Config{
		Token:          "test-token",
		RemoveCommands: true,
	}
	m := &mockDiscordSession{}

	ctx, cancel := context.WithCancel(context.Background())
	// Stop after a very short time
	go func() {
		cancel()
	}()

	runDiscord(ctx, cfg, m, nil, nil, mockDeps())

	if len(m.commandsCreated) == 0 {
		t.Errorf("Expected commands to be created")
	}
	if len(m.commandsDeleted) == 0 {
		t.Errorf("Expected commands to be deleted on cleanup")
	}
}

func TestInitProvider(t *testing.T) {

	t.Run("Ollama", func(t *testing.T) {

		cfg := &Config{LLMProvider: "ollama", LLMModel: "llama3"}

		p, err := initProvider(context.Background(), cfg)

		if err != nil {

			t.Fatalf("initProvider failed: %v", err)

		}

		if p == nil || p.Name() != "ollama" {

			t.Errorf("Expected ollama provider")

		}

	})

	t.Run("GeminiMissingKey", func(t *testing.T) {

		cfg := &Config{LLMProvider: "gemini"}

		p, err := initProvider(context.Background(), cfg)

		if err != nil {

			t.Fatalf("initProvider failed: %v", err)

		}

		if p != nil {

			t.Errorf("Expected nil provider for missing key")

		}

	})

	t.Run("GeminiWithKey", func(t *testing.T) {

		cfg := &Config{LLMProvider: "gemini", GeminiKey: "test-key"}

		p, err := initProvider(context.Background(), cfg)

		if err != nil {

			t.Fatalf("initProvider failed: %v", err)

		}

		if p == nil || p.Name() != "gemini" {

			t.Errorf("Expected gemini provider")

		}

	})

	t.Run("GroqWithKey", func(t *testing.T) {

		cfg := &Config{LLMProvider: "groq", GroqKey: "test-key"}

		p, err := initProvider(context.Background(), cfg)

		if err != nil {

			t.Fatalf("initProvider failed: %v", err)

		}

		if p == nil || p.Name() != "groq" {

			t.Errorf("Expected groq provider")

		}

	})

	t.Run("GroqMissingKey", func(t *testing.T) {

		cfg := &Config{LLMProvider: "groq"}

		p, err := initProvider(context.Background(), cfg)

		if err != nil {

			t.Fatalf("initProvider failed: %v", err)

		}

		if p != nil {

			t.Errorf("Expected nil provider for missing key")

		}

	})

}

func TestDiscordHandlers(t *testing.T) {

	t.Run("HandleReady", func(t *testing.T) {

		sess := &discordgo.Session{

			State: &discordgo.State{

				Ready: discordgo.Ready{

					User: &discordgo.User{

						Username: "test-bot",

						Discriminator: "1234",
					},
				},
			},
		}

		handleReady(sess, &discordgo.Ready{})

	})

	t.Run("HandleInteraction_Echo", func(t *testing.T) {

		m := &mockDiscordSession{}

		handler := handleInteraction(m, nil, false, NewMessageCache(10))

		i := &discordgo.InteractionCreate{

			Interaction: &discordgo.Interaction{

				Type: discordgo.InteractionApplicationCommand,

				Data: discordgo.ApplicationCommandInteractionData{

					Name: "echo",

					Options: []*discordgo.ApplicationCommandInteractionDataOption{

						{

							Type: discordgo.ApplicationCommandOptionString,

							Value: "hello world",
						},
					},
				},

				Member: &discordgo.Member{

					User: &discordgo.User{ID: "user-id"},
				},

				GuildID: "guild-id",
			},
		}

		handler(nil, i)

	})

	t.Run("HandleInteraction_vcomment", func(t *testing.T) {
		m := &mockDiscordSession{}
		cache := NewMessageCache(10)
		handler := handleInteraction(m, nil, false, cache)

		i := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "vcomment",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{
							Type:  discordgo.ApplicationCommandOptionString,
							Value: "Unit test comment",
						},
					},
				},
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "user-id", Username: "tester"},
				},
				GuildID:   "guild-id",
				ChannelID: "channel-id",
				ID:        "interaction-id",
			},
		}

		handler(nil, i)

		// Check if message was added to cache
		msgs := cache.Get("channel-id")
		if len(msgs) != 1 {
			t.Errorf("Expected 1 message in cache, got %d", len(msgs))
		} else if msgs[0].Content != "Unit test comment" {
			t.Errorf("Expected message content 'Unit test comment', got '%s'", msgs[0].Content)
		}
	})

	t.Run("HandleMessageCreate_NoAutoCache", func(t *testing.T) {
		cache := NewMessageCache(10)
		handler := handleMessageCreate(cache)

		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content:   "Automatic message",
				ChannelID: "channel-id",
				Author:    &discordgo.User{ID: "user-id", Bot: false},
			},
		}

		handler(nil, m)

		// Check if message was NOT added to cache
		msgs := cache.Get("channel-id")
		if len(msgs) != 0 {
			t.Errorf("Expected 0 messages in cache, got %d", len(msgs))
		}
	})

}
