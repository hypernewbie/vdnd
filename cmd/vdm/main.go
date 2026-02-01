package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// DiscordSession interface for testability
type DiscordSession interface {
	AddHandler(handler interface{}) func()
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (ccmd *discordgo.ApplicationCommand, err error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	Open() error
	Close() error
	GetState() *discordgo.State
}

type discordSessionWrapper struct {
	*discordgo.Session
}

func (w *discordSessionWrapper) GetState() *discordgo.State {
	return w.Session.State
}

// Config holds the application configuration
type Config struct {
	Token          string `env:"DISCORD_TOKEN"`
	RemoveCommands bool   `env:"DISCORD_REMOVE_COMMANDS" envDefault:"true"`
	GeminiKey      string `env:"GEMINI_API_KEY"`
	GroqKey        string `env:"GROQ_API_KEY"`
	LLMProvider    string `env:"LLM_PROVIDER" envDefault:"groq"`
	LLMModel       string `env:"LLM_MODEL" envDefault:"qwen/qwen3-32b"`
}

// loadConfig reads configuration from environment variables
func loadConfig() (*Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	// Open log file
	logFile, err := os.OpenFile("vdm.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Initialize structured logger
	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	// Load .env file if it exists
	_ = godotenv.Load()

	cfg, useDiscord, verbose, promptMode, err := parseConfig(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "failed to parse config: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("--- VERBOSE STARTUP ---\n")
		fmt.Printf("DISCORD_TOKEN: %s\n", cfg.Token)
		fmt.Printf("GEMINI_API_KEY: %s\n", cfg.GeminiKey)
		fmt.Printf("GROQ_API_KEY: %s\n", cfg.GroqKey)
		fmt.Printf("LLM_PROVIDER: %s\n", cfg.LLMProvider)
		fmt.Printf("LLM_MODEL: %s\n", cfg.LLMModel)
		fmt.Printf("------------------------\n")
	}

	if useDiscord {
		if cfg.Token == "" {
			slog.Error("DISCORD_TOKEN is required for discord mode")
			os.Exit(1)
		}
		s, err := discordgo.New("Bot " + cfg.Token)
		if err != nil {
			slog.Error("invalid bot parameters", "error", err)
			os.Exit(1)
		}
		runDiscord(context.Background(), cfg, &discordSessionWrapper{s})
	} else {
		p, err := initProvider(context.Background(), cfg)
		if err != nil {
			slog.Error("failed to initialize provider", "error", err)
			os.Exit(1)
		}

		runCLI(context.Background(), os.Stdin, os.Stdout, cfg, p, cli.DefaultDeps(), promptMode)
	}
}

func initProvider(ctx context.Context, cfg *Config) (llm.Provider, error) {
	var p llm.Provider
	var err error

	switch cfg.LLMProvider {
	case "gemini":
		if cfg.GeminiKey != "" {
			p, err = llm.NewGeminiProvider(ctx, cfg.GeminiKey, cfg.LLMModel)
		}
	case "ollama":
		p, err = llm.NewOllamaProvider(cfg.LLMModel)
	case "groq":
		if cfg.GroqKey != "" {
			p, err = llm.NewGroqProvider(cfg.GroqKey, cfg.LLMModel)
		}
	}
	return p, err
}

func parseConfig(args []string) (cfg *Config, useDiscord bool, verbose bool, promptMode bool, err error) {
	fs := flag.NewFlagSet("vdm", flag.ContinueOnError)
	useDiscordPtr := fs.Bool("discord", false, "Run in Discord bot mode")
	verbosePtr := fs.Bool("verbose", false, "Print configuration and secrets on startup")
	providerFlag := fs.String("provider", "", "LLM provider (gemini, ollama, groq)")
	modelFlag := fs.String("model", "", "LLM model name")
	promptModeFlag := fs.Bool("prompt-mode", false, "Force schema-constrained prompting (JSON)")

	if err := fs.Parse(args); err != nil {
		return nil, false, false, false, err
	}

	cfg, err = loadConfig()
	if err != nil {
		return nil, false, false, false, err
	}

	// Flag overrides environment
	if *providerFlag != "" {
		cfg.LLMProvider = *providerFlag
	}
	if *modelFlag != "" {
		cfg.LLMModel = *modelFlag
	}

	return cfg, *useDiscordPtr, *verbosePtr, *promptModeFlag, nil
}

func runCLI(ctx context.Context, in io.Reader, out io.Writer, cfg *Config, p llm.Provider, deps cli.Deps, forcePromptMode bool) {
	slog.Info("Starting CLI mode...")

	var orch *llm.Orchestrator

	if p != nil {
		orch = llm.NewOrchestrator(ctx, p, deps)
		if forcePromptMode {
			orch.EnablePromptMode(true)
		}
		defer orch.Close()
		fmt.Fprintf(out, "LLM mode enabled (Generative DM using %s/%s). Type 'exit' to quit.\n", cfg.LLMProvider, cfg.LLMModel)
	} else {
		fmt.Fprintln(out, "Standard CLI mode enabled. Type 'exit' to quit.")
	}

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, "> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		cmd, arg, _ := strings.Cut(line, " ")
		if orch != nil {
			slog.Info("USER", "content", line)
			resp, err := orch.ProcessInput(ctx, line)
			if err != nil {
				slog.Error("orchestrator error", "error", err)
				fmt.Fprintf(out, "Error: %v\n", err)
				continue
			}
			fmt.Fprintf(out, "DM: %s\n", resp)
			continue
		}

		switch cmd {
		case "echo":
			if arg == "" {
				fmt.Fprintln(out, "Usage: echo <content>")
				continue
			}
			slog.Info("received echo command", "content", arg)
			fmt.Fprintf(out, "Echo: %s\n", arg)
		case "exit", "quit":
			slog.Info("CLI mode shutting down")
			return
		default:
			fmt.Fprintf(out, "Unknown command: %s\n", cmd)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("error reading input", "error", err)
	}
}

func runDiscord(ctx context.Context, cfg *Config, s DiscordSession) {
	// Define commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "echo",
			Description: "Echoes back your text",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "content",
					Description: "Content to echo",
					Required:    true,
				},
			},
		},
	}

	// Register handlers
	s.AddHandler(handleInteraction(s))
	s.AddHandler(handleReady)

	// Open session
	err := s.Open()
	if err != nil {
		slog.Error("cannot open the session", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	slog.Info("session opened")

	// Register commands
	slog.Info("registering commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.GetState().User.ID, "", v)
		if err != nil {
			slog.Error("cannot create command", "name", v.Name, "error", err)
			continue // Or exit/panic depending on strictness
		}
		registeredCommands[i] = cmd
		slog.Info("registered command", "name", v.Name)
	}

	// Wait here until context is canceled or term signal is received.
	slog.Info("bot is now running. Press CTRL-C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	select {
	case <-stop:
		slog.Info("interrupt signal received")
	case <-ctx.Done():
		slog.Info("context canceled")
	}

	// Cleanup
	if cfg.RemoveCommands {
		slog.Info("removing commands...")
		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.GetState().User.ID, "", v.ID)
			if err != nil {
				slog.Error("cannot delete command", "name", v.Name, "error", err)
			}
		}
	}

	slog.Info("gracefully shutting down")
}

func handleInteraction(s DiscordSession) func(sess *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(sess *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.ApplicationCommandData().Name != "echo" {
			return
		}
		// Enforce Guild-only interactions
		if i.Member == nil {
			return
		}

		options := i.ApplicationCommandData().Options
		content := options[0].StringValue()

		slog.Info("received echo command",
			"guild_id", i.GuildID,
			"user_id", i.Member.User.ID,
			"content", content,
		)

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Echo: %s", content),
			},
		})
		if err != nil {
			slog.Error("failed to respond to interaction", "error", err)
		}
	}
}

func handleReady(sess *discordgo.Session, r *discordgo.Ready) {
	slog.Info("logged in",
		"username", sess.State.User.Username,
		"discriminator", sess.State.User.Discriminator,
	)
}