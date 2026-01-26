package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
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

	// Parse flags
	useDiscord := flag.Bool("discord", false, "Run in Discord bot mode")
	verbose := flag.Bool("verbose", false, "Print configuration and secrets on startup")
	providerFlag := flag.String("provider", "", "LLM provider (gemini, ollama, groq)")
	modelFlag := flag.String("model", "", "LLM model name")
	promptModeFlag := flag.Bool("prompt-mode", false, "Force schema-constrained prompting (JSON)")
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Flag overrides environment
	if *providerFlag != "" {
		cfg.LLMProvider = *providerFlag
	}
	if *modelFlag != "" {
		cfg.LLMModel = *modelFlag
	}

	if *verbose {
		fmt.Printf("--- VERBOSE STARTUP ---\n")
		fmt.Printf("DISCORD_TOKEN: %s\n", cfg.Token)
		fmt.Printf("GEMINI_API_KEY: %s\n", cfg.GeminiKey)
		fmt.Printf("GROQ_API_KEY: %s\n", cfg.GroqKey)
		fmt.Printf("LLM_PROVIDER: %s\n", cfg.LLMProvider)
		fmt.Printf("LLM_MODEL: %s\n", cfg.LLMModel)
		fmt.Printf("------------------------\n")
	}

	if *useDiscord {
		if cfg.Token == "" {
			slog.Error("DISCORD_TOKEN is required for discord mode")
			os.Exit(1)
		}
		runDiscord(cfg)
	} else {
		runCLI(cfg, *promptModeFlag)
	}
}

func runCLI(cfg *Config, forcePromptMode bool) {
	slog.Info("Starting CLI mode...")

	var orch *llm.Orchestrator

	// Initialize Provider
	var p llm.Provider
	var err error

	switch cfg.LLMProvider {
	case "gemini":
		if cfg.GeminiKey != "" {
			p, err = llm.NewGeminiProvider(context.Background(), cfg.GeminiKey, cfg.LLMModel)
		}
	case "ollama":
		p, err = llm.NewOllamaProvider(cfg.LLMModel)
	case "groq":
		if cfg.GroqKey != "" {
			p, err = llm.NewGroqProvider(cfg.GroqKey, cfg.LLMModel)
		}
	}

	if err != nil {
		slog.Error("failed to initialize provider", "error", err)
		os.Exit(1)
	}

	if p != nil {
		orch = llm.NewOrchestrator(context.Background(), p, cli.DefaultDeps())
		if forcePromptMode {
			orch.EnablePromptMode(true)
		}
		defer orch.Close()
		fmt.Printf("LLM mode enabled (Generative DM using %s/%s). Type 'exit' to quit.\n", cfg.LLMProvider, cfg.LLMModel)
	} else {
		fmt.Println("Standard CLI mode enabled. Type 'exit' to quit.")
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
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
			resp, err := orch.ProcessInput(context.Background(), line)
			if err != nil {
				slog.Error("orchestrator error", "error", err)
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Printf("DM: %s\n", resp)
			continue
		}

		switch cmd {
		case "echo":
			if arg == "" {
				fmt.Println("Usage: echo <content>")
				continue
			}
			slog.Info("received echo command", "content", arg)
			fmt.Printf("Echo: %s\n", arg)
		case "exit", "quit":
			slog.Info("CLI mode shutting down")
			return
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("error reading input", "error", err)
	}
}

func runDiscord(cfg *Config) {
	// Initialize Discord session
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		slog.Error("invalid bot parameters", "error", err)
		os.Exit(1)
	}

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

	// Define command handlers
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"echo": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		},
	}

	// Register handlers
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("logged in",
			"username", s.State.User.Username,
			"discriminator", s.State.User.Discriminator,
		)
	})

	// Open session
	err = s.Open()
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
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			slog.Error("cannot create command", "name", v.Name, "error", err)
			continue // Or exit/panic depending on strictness
		}
		registeredCommands[i] = cmd
		slog.Info("registered command", "name", v.Name)
	}

	// Wait here until CTRL-C or other term signal is received.
	slog.Info("bot is now running. Press CTRL-C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	// Cleanup
	if cfg.RemoveCommands {
		slog.Info("removing commands...")
		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
			if err != nil {
				slog.Error("cannot delete command", "name", v.Name, "error", err)
			}
		}
	}

	slog.Info("gracefully shutting down")
}
