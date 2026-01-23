package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
)

// Config holds the application configuration
type Config struct {
	Token          string `env:"DISCORD_TOKEN"`
	RemoveCommands bool   `env:"DISCORD_REMOVE_COMMANDS" envDefault:"true"`
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
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Parse flags
	useDiscord := flag.Bool("discord", false, "Run in Discord bot mode")
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if *useDiscord {
		if cfg.Token == "" {
			logger.Error("DISCORD_TOKEN is required for discord mode")
			os.Exit(1)
		}
		runDiscord(logger, cfg)
	} else {
		runCLI(logger)
	}
}

func runCLI(logger *slog.Logger) {
	logger.Info("Starting CLI mode...")
	fmt.Println("CLI mode enabled. Type 'exit' to quit.")

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
		switch cmd {
		case "echo":
			if arg == "" {
				fmt.Println("Usage: echo <content>")
				continue
			}
			logger.Info("received echo command", "content", arg)
			fmt.Printf("Echo: %s\n", arg)
		case "exit", "quit":
			logger.Info("CLI mode shutting down")
			return
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("error reading input", "error", err)
	}
}

func runDiscord(logger *slog.Logger, cfg *Config) {
	// Initialize Discord session
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		logger.Error("invalid bot parameters", "error", err)
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

			logger.Info("received echo command",
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
				logger.Error("failed to respond to interaction", "error", err)
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
		logger.Info("logged in",
			"username", s.State.User.Username,
			"discriminator", s.State.User.Discriminator,
		)
	})

	// Open session
	err = s.Open()
	if err != nil {
		logger.Error("cannot open the session", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	logger.Info("session opened")

	// Register commands
	logger.Info("registering commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			logger.Error("cannot create command", "name", v.Name, "error", err)
			continue // Or exit/panic depending on strictness
		}
		registeredCommands[i] = cmd
		logger.Info("registered command", "name", v.Name)
	}

	// Wait here until CTRL-C or other term signal is received.
	logger.Info("bot is now running. Press CTRL-C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	// Cleanup
	if cfg.RemoveCommands {
		logger.Info("removing commands...")
		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
			if err != nil {
				logger.Error("cannot delete command", "name", v.Name, "error", err)
			}
		}
	}

	logger.Info("gracefully shutting down")
}
