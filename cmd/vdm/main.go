package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/subagents"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// DiscordSession interface for testability
type DiscordSession interface {
	AddHandler(handler interface{}) func()
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (ccmd *discordgo.ApplicationCommand, err error)
	ApplicationCommandBulkOverwrite(appID, guildID string, commands []*discordgo.ApplicationCommand, options ...discordgo.RequestOption) (ccmds []*discordgo.ApplicationCommand, err error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	Open() error
	Close() error
	GetState() *discordgo.State
	ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) (st []*discordgo.Message, err error)
}

type discordSessionWrapper struct {
	*discordgo.Session
}

func (w *discordSessionWrapper) GetState() *discordgo.State {
	return w.Session.State
}

type Feedback struct {
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	Rating    int    `json:"rating"`
	Comments  string `json:"comments"`
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
}

func saveFeedback(f Feedback) error {
	filename := fmt.Sprintf("feedback/feedback_%d.json", time.Now().UnixNano())
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// Config holds the application configuration
type Config struct {
	Token                  string `env:"DISCORD_TOKEN"`
	RemoveCommands         bool   `env:"DISCORD_REMOVE_COMMANDS" envDefault:"false"`
	GeminiKey              string `env:"GEMINI_API_KEY"`
	GroqKey                string `env:"GROQ_API_KEY"`
	DeepSeekKey            string `env:"DEEPSEEK_API_KEY"`
	GLMKey                 string `env:"GLM_API_KEY"`
	AnthropicKey           string `env:"ANTHROPIC_API_KEY"`
	OpenAIKey              string `env:"OPENAI_API_KEY"`
	OpenRouterKey          string `env:"OPENROUTER_API_KEY"`
	OllamaURL              string `env:"OLLAMA_URL"`
	LLMProvider            string `env:"LLM_PROVIDER" envDefault:"groq"`
	LLMModel               string `env:"LLM_MODEL" envDefault:"qwen/qwen3-32b"`
	DryRun                 bool   `env:"DRY_RUN" envDefault:"false"`
	Feedback               bool   `env:"FEEDBACK" envDefault:"false"`
	ResearcherPromptFile   string `env:"RESEARCHER_PROMPT_FILE" envDefault:"config/prompt_researcher.txt"`
	VDPromptFile           string `env:"VD_PROMPT_FILE" envDefault:"config/prompt_vd.txt"`
	OrchestratorPromptFile string `env:"ORCHESTRATOR_PROMPT_FILE" envDefault:"config/prompt_orchestrator.txt"`
	HistoryFile            string `env:"VDM_HISTORY_FILE" envDefault:"vdm_history.json"`
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
	_ = godotenv.Overload()

	// Handle graceful shutdown signals for the entire stack
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, useDiscord, verbose, err := parseConfig(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "failed to parse config: %v\n", err)
		os.Exit(1)
	}
	cli.PrintDraculaBanner()

	if verbose {
		fmt.Printf("--- VERBOSE STARTUP ---\n")
		fmt.Printf("DISCORD_TOKEN: %s\n", cfg.Token)
		fmt.Printf("GEMINI_API_KEY: %s\n", cfg.GeminiKey)
		fmt.Printf("GROQ_API_KEY: %s\n", cfg.GroqKey)
		fmt.Printf("DEEPSEEK_API_KEY: %s\n", cfg.DeepSeekKey)
		fmt.Printf("GLM_API_KEY: %s\n", cfg.GLMKey)
		fmt.Printf("ANTHROPIC_API_KEY: %s\n", cfg.AnthropicKey)
		fmt.Printf("OPENAI_API_KEY: %s\n", cfg.OpenAIKey)
		fmt.Printf("OPENROUTER_API_KEY: %s\n", cfg.OpenRouterKey)
		fmt.Printf("LLM_PROVIDER: %s\n", cfg.LLMProvider)
		fmt.Printf("LLM_MODEL: %s\n", cfg.LLMModel)
		fmt.Printf("------------------------\n")
	}

	p, err := initProvider(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize provider", "error", err)
		os.Exit(1)
	}

	deps := cli.DefaultDeps()
	agents, err := initSubagents(cfg, p, deps)
	if err != nil {
		slog.Error("failed to initialize subagents", "error", err)
		os.Exit(1)
	}

	wd, _ := os.Getwd()
	python := subagents.FindPythonPath(wd)
	script := filepath.Join(wd, "py", "restricted_python.py")

	orchestratorPromptBytes, err2 := os.ReadFile(cfg.OrchestratorPromptFile)
	if err2 != nil {
		slog.Error("failed to read orchestrator prompt", "error", err2)
		os.Exit(1)
	}
	orchestratorPrompt := string(orchestratorPromptBytes)

	historyBytes := 0
	if fi, statErr := os.Stat(cfg.HistoryFile); statErr == nil {
		historyBytes = int(fi.Size())
	}
	cli.PrintStartupConfig(cfg.LLMProvider, cfg.LLMModel, historyBytes)

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
		s.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentMessageContent
		runDiscord(ctx, cfg, &discordSessionWrapper{s}, p, agents, deps, orchestratorPrompt, python, script)
	} else {
		if cfg.DryRun {
			slog.Info("DRY RUN MODE ENABLED. Prompts will be echoed back.")
		}
		runCLI(ctx, os.Stdin, os.Stdout, cfg, p, agents, deps, orchestratorPrompt, python, script)
	}
}

func initProvider(ctx context.Context, cfg *Config) (llmtypes.Provider, error) {
	if cfg.DryRun {
		return llm.NewDummyProvider(cfg.LLMModel), nil
	}

	var p llmtypes.Provider
	var err error

	switch cfg.LLMProvider {
	case "gemini":
		if cfg.GeminiKey != "" {
			p, err = llm.NewGeminiProvider(ctx, cfg.GeminiKey, cfg.LLMModel)
		}
	case "ollama":
		p, err = llm.NewOllamaProvider(cfg.LLMModel, cfg.OllamaURL)
	case "groq":
		if cfg.GroqKey != "" {
			p, err = llm.NewGroqProvider(cfg.GroqKey, cfg.LLMModel)
		}
	case "deepseek":
		if cfg.DeepSeekKey != "" {
			p, err = llm.NewDeepSeekProvider(cfg.DeepSeekKey, cfg.LLMModel)
		}
	case "glm":
		if cfg.GLMKey != "" {
			p, err = llm.NewGLMProvider(cfg.GLMKey, cfg.LLMModel)
		}
	case "anthropic":
		if cfg.AnthropicKey != "" {
			p, err = llm.NewAnthropicProvider(cfg.AnthropicKey, cfg.LLMModel)
		}
	case "chatgpt":
		if cfg.OpenAIKey != "" {
			p, err = llm.NewChatGPTProvider(cfg.OpenAIKey, cfg.LLMModel)
		}
	case "openrouter":
		if cfg.OpenRouterKey != "" {
			p, err = llm.NewOpenRouterProvider(cfg.OpenRouterKey, cfg.LLMModel)
		}
	}
	return p, err
}

func initSubagents(cfg *Config, p llmtypes.Provider, deps cli.Deps) ([]llmtypes.Subagent, error) {
	if p == nil {
		return nil, nil
	}

	wd, _ := os.Getwd()
	python := subagents.FindPythonPath(wd)
	script := filepath.Join(wd, "py", "restricted_python.py")

	researchAgent := subagents.NewResearchSubagent(p, python, script)
	researcherPrompt, err := os.ReadFile(cfg.ResearcherPromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read researcher prompt: %w", err)
	}
	researchAgent.SetPrompt(string(researcherPrompt))

	pythonReadOnlyAgent, err := subagents.NewPythonReadOnlySubagent(python, script)
	if err != nil {
		return nil, fmt.Errorf("failed to start read-only python subagent: %w", err)
	}

	execAgent := subagents.NewExecutionSubagentWithPython(p, deps, pythonReadOnlyAgent)
	vdPrompt, err := os.ReadFile(cfg.VDPromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read execution prompt: %w", err)
	}
	execAgent.SetPrompt(string(vdPrompt))

	return []llmtypes.Subagent{researchAgent, execAgent}, nil
}

func parseConfig(args []string) (cfg *Config, useDiscord bool, verbose bool, err error) {
	fs := flag.NewFlagSet("vdm", flag.ContinueOnError)
	useDiscordPtr := fs.Bool("discord", false, "Run in Discord bot mode")
	providerFlag := fs.String("provider", "", "LLM provider (gemini, ollama, groq, anthropic)")
	modelFlag := fs.String("model", "", "LLM model name")
	promptModeFlag := fs.Bool("prompt-mode", false, "Force schema-constrained prompting (JSON)")
	dryRunFlag := fs.Bool("dry-run", false, "Enable dry run mode (echo prompts)")
	feedbackFlag := fs.Bool("feedback", false, "Collect feedback after CLI session")
	verbosePtr := fs.Bool("verbose", false, "Print configuration and secrets on startup")

	if err := fs.Parse(args); err != nil {
		return nil, false, false, err
	}

	cfg, err = loadConfig()
	if err != nil {
		return nil, false, false, err
	}

	// Flag overrides environment
	if *providerFlag != "" {
		cfg.LLMProvider = *providerFlag
	}
	if *modelFlag != "" {
		cfg.LLMModel = *modelFlag
	}

	if *promptModeFlag {
		cfg.LLMProvider = "prompt" // internal marker for prompt mode
	}

	if *dryRunFlag {
		cfg.DryRun = true
	}

	if *feedbackFlag {
		cfg.Feedback = true
	}

	return cfg, *useDiscordPtr, *verbosePtr, nil
}

func runCLI(ctx context.Context, in io.Reader, out io.Writer, cfg *Config, p llmtypes.Provider, agents []llmtypes.Subagent, deps cli.Deps, orchestratorPrompt string, pythonPath, scriptPath string) {
	slog.Info("Starting CLI mode...")

	var orch *llm.Orchestrator

	if p != nil {
		var err error
		orch, err = llm.NewOrchestrator(ctx, p, deps, pythonPath, scriptPath)
		if err != nil {
			slog.Error("failed to initialize orchestrator", "error", err)
			os.Exit(1)
		}
		orch.SetPrompt(orchestratorPrompt)
		orch.RegisterSubagents(agents...)
		// Load history
		if err := orch.LoadHistory(cfg.HistoryFile); err != nil {
			slog.Warn("failed to load history", "error", err)
		}
		defer func() {
			cli.PrintInfo("\n[VDM] Shutting down...")
			if err := orch.SaveHistory(cfg.HistoryFile); err != nil {
				slog.Error("failed to save history", "error", err)
			}
			orch.Close()
			cli.PrintInfo("[VDM] Cleaning up sandbox...")
			cli.PrintInfo("[VDM] Goodbye.")
		}()
		fmt.Fprintf(out, "LLM mode enabled (Generative DM using %s/%s with subagents). Type 'exit' to quit.\n", cfg.LLMProvider, cfg.LLMModel)
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
			cli.PrintPlayerInput(line)
			resp, err := orch.ProcessInput(ctx, line, nil)
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

	if cfg.Feedback {
		collectCLIFeedback(in, out, cfg, p)
	}
}

func collectCLIFeedback(in io.Reader, out io.Writer, cfg *Config, p llmtypes.Provider) {
	fmt.Fprintln(out, "\n--- FEEDBACK ---")
	fmt.Fprint(out, "How would you rate the DM's performance? (1-5 stars): ")

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return
	}

	rating, _ := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if rating < 1 || rating > 5 {
		fmt.Fprintln(out, "Invalid rating. Skipping feedback.")
		return
	}

	fmt.Fprint(out, "Any additional comments? (optional): ")
	if !scanner.Scan() {
		return
	}
	comments := strings.TrimSpace(scanner.Text())

	provider, model := "none", "none"
	if p != nil {
		provider = p.Name()
		model = p.ModelName()
	}

	fb := Feedback{
		Timestamp: time.Now().Format(time.RFC3339),
		User:      "cli",
		Rating:    rating,
		Comments:  comments,
		Provider:  provider,
		Model:     model,
	}

	if err := saveFeedback(fb); err != nil {
		fmt.Fprintf(out, "Error saving feedback: %v\n", err)
	} else {
		fmt.Fprintln(out, "Thank you for your feedback!")
	}
}

func runDiscord(ctx context.Context, cfg *Config, s DiscordSession, p llmtypes.Provider, agents []llmtypes.Subagent, deps cli.Deps, orchestratorPrompt string, pythonPath, scriptPath string) {
	cache := NewMessageCache(100)
	// Define commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "vdm",
			Description: "Talk to the Virtual Dungeon Master",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "input",
					Description: "Your message to the DM",
					Required:    true,
				},
			},
		},
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
		{
			Name:        "vstop",
			Description: "Interrupt the DM's current thinking process",
		},
		{
			Name:        "vstatus",
			Description: "Report current DM provider and model status",
		},
		{
			Name:        "vfeedback",
			Description: "Provide feedback on the DM's performance",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "rating",
					Description: "Rate the DM from 1 to 5 stars",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "1 Star", Value: 1},
						{Name: "2 Stars", Value: 2},
						{Name: "3 Stars", Value: 3},
						{Name: "4 Stars", Value: 4},
						{Name: "5 Stars", Value: 5},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "comments",
					Description: "Additional comments or suggestions",
					Required:    false,
				},
			},
		},
		{
			Name:        "vcomment",
			Description: "Add a comment to the DM's context",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "comment",
					Description: "The comment to add",
					Required:    true,
				},
			},
		},
	}

	var orch *llm.Orchestrator
	if p != nil {
		var err error
		orch, err = llm.NewOrchestrator(ctx, p, deps, pythonPath, scriptPath)
		if err != nil {
			slog.Error("failed to initialize orchestrator", "error", err)
			os.Exit(1)
		}
		orch.SetPrompt(orchestratorPrompt)
		orch.RegisterSubagents(agents...)
		// Load history
		if err := orch.LoadHistory(cfg.HistoryFile); err != nil {
			slog.Warn("failed to load history", "error", err)
		}
		defer func() {
			cli.PrintInfo("\n[VDM] Shutting down...")
			if err := orch.SaveHistory(cfg.HistoryFile); err != nil {
				slog.Error("failed to save history", "error", err)
			}
			orch.Close()
			cli.PrintInfo("[VDM] Cleaning up sandbox...")
			cli.PrintInfo("[VDM] Goodbye.")
		}()
	}

	// Register handlers
	s.AddHandler(handleMessageCreate(cache))
	s.AddHandler(handleInteraction(s, orch, cfg.DryRun, cache))
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
	cli.PrintInfo("[Discord] Registering slash commands (Bulk Sync)...")
	registeredCommands, err := s.ApplicationCommandBulkOverwrite(s.GetState().User.ID, "", commands)
	if err != nil {
		slog.Error("cannot bulk overwrite commands", "error", err)
	} else {
		cli.PrintInfo(fmt.Sprintf("[Discord] %d commands synced successfully.", len(registeredCommands)))
	}

	// Wait here until context is canceled or term signal is received.
	slog.Info("bot is now running. Press CTRL-C to exit.")

	select {
	case <-ctx.Done():
		slog.Info("context canceled or interrupt received")
	}

	// Cleanup
	if cfg.RemoveCommands {
		cli.PrintInfo("[Discord] Removing commands (Cleanup)...")
		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.GetState().User.ID, "", v.ID)
			if err != nil {
				slog.Error("cannot delete command", "name", v.Name, "error", err)
			}
		}
		cli.PrintInfo("[Discord] Commands removed.")
	}

	slog.Info("gracefully shutting down")
}

func handleInteraction(s DiscordSession, orch *llm.Orchestrator, dryRun bool, cache *MessageCache) func(sess *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(sess *discordgo.Session, i *discordgo.InteractionCreate) {
		name := i.ApplicationCommandData().Name
		if name != "echo" && name != "vdm" && name != "vstop" && name != "vstatus" && name != "vfeedback" && name != "vcomment" {
			return
		}
		// Enforce Guild-only interactions
		if i.Member == nil {
			return
		}

		var content string
		options := i.ApplicationCommandData().Options
		if len(options) > 0 {
			content = options[0].StringValue()
		}

		if name == "vstop" {
			slog.Info("vstop command")
			if orch == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Error: Orchestrator not initialized.",
					},
				})
				return
			}
			if orch.Interrupt() {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "DM has been interrupted. Ready for new commands.",
					},
				})
			} else {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "The DM is already idle.",
					},
				})
			}
			return
		}

		if name == "vstatus" {
			providerName := "none"
			modelName := "none"
			if orch != nil {
				providerName, modelName = orch.ProviderInfo()
			}
			status := fmt.Sprintf("**VDM Status**\n**Provider:** %s\n**Model:** %s", providerName, modelName)
			if dryRun {
				status += "\n**Dry Run:** Active"
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: status,
				},
			})
			return
		}

		if name == "vfeedback" {
			rating := int(options[0].IntValue())
			comments := ""
			if len(options) > 1 {
				comments = options[1].StringValue()
			}

			provider, model := "unknown", "unknown"
			if orch != nil {
				provider, model = orch.ProviderInfo()
			}

			fb := Feedback{
				Timestamp: time.Now().Format(time.RFC3339),
				User:      fmt.Sprintf("discord:%s", i.Member.User.ID),
				Rating:    rating,
				Comments:  comments,
				Provider:  provider,
				Model:     model,
			}

			if err := saveFeedback(fb); err != nil {
				slog.Error("failed to save feedback", "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Error: Failed to save feedback.",
					},
				})
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Thank you for your feedback! It has been saved for review.",
				},
			})
			return
		}
		if name == "vcomment" {
			comment := options[0].StringValue()
			slog.Info("vcomment command",
				"guild_id", i.GuildID,
				"channel_id", i.ChannelID,
				"user_id", i.Member.User.ID,
				"comment", comment,
			)

			// Add to cache
			msg := &discordgo.Message{
				ID:        i.ID, // Use interaction ID as message ID
				ChannelID: i.ChannelID,
				Content:   comment,
				Author:    i.Member.User,
				Timestamp: time.Now(),
				Type:      discordgo.MessageTypeDefault,
			}
			cache.Add(msg)

			userDisplayName := getDisplayName(i.Member.User)
			echoContent := fmt.Sprintf("> %s: %s", userDisplayName, comment)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: echoContent,
				},
			})
			return
		}

		if name == "vdm" {
			if orch == nil && !dryRun {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Error: Orchestrator not initialized. Check your LLM configuration.",
					},
				})
				return
			}

			// Acknowledge the interaction immediately to avoid timeout
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			// Fetch recent messages for context ("Party Talk") from our live cache
			var partyTalk strings.Builder
			botID := s.GetState().User.ID
			recentMsgs := cache.Get(i.ChannelID)

			var collected []string
			// Process messages newest first, but the cache is oldest first
			for j := len(recentMsgs) - 1; j >= 0; j-- {
				m := recentMsgs[j]
				// Stop if we hit the previous bot response or a very old message
				if m.Author.ID == botID {
					break
				}
				// Don't include system messages
				if m.Type != discordgo.MessageTypeDefault && m.Type != discordgo.MessageTypeReply {
					continue
				}
				displayName := getDisplayName(m.Author)
				collected = append(collected, fmt.Sprintf("%s: %s", displayName, m.Content))
			}

			if len(collected) > 0 {
				partyTalk.WriteString("CHANNEL MESSAGES (Party Talk):\n")
				// collected is now newest first, we want oldest first for the prompt
				for j := len(collected) - 1; j >= 0; j-- {
					partyTalk.WriteString(collected[j] + "\n")
				}
				partyTalk.WriteString("\n")
			}
			// Clear cache for this channel after consumption to avoid "double-hearing" same messages
			cache.Clear(i.ChannelID)

			userDisplayName := getDisplayName(i.Member.User)
			content = fmt.Sprintf("%s: %s", userDisplayName, content)
			cli.PrintPlayerInput(content)

			fullInput := partyTalk.String() + "DM_COMMAND:\n" + content
			slog.Info("vdm command",
				"guild_id", i.GuildID,
				"channel_id", i.ChannelID,
				"user_id", i.Member.User.ID,
				"full_input", fullInput,
			)

			streamer := NewDiscordStreamer(sess, i.Interaction, i.ChannelID)
			defer streamer.Close()

			streamer.Chunk(fmt.Sprintf("> %s\n\n", content))

			if dryRun {
				streamer.Chunk("=== DRY RUN (Full Input) ===\n" + fullInput)
			} else {
				reporter := &discordReporter{streamer: streamer}
				resp, err := orch.ProcessInput(context.Background(), fullInput, reporter)
				if err != nil {
					slog.Error("orchestrator error", "error", err)
					streamer.Chunk(fmt.Sprintf("Error: %v", err))
					return
				}
				if strings.TrimSpace(resp) == "" {
					streamer.Chunk("*(The DM is silent)*")
				}
			}
			return
		}

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

func getDisplayName(u *discordgo.User) string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}

func handleMessageCreate(cache *MessageCache) func(sess *discordgo.Session, m *discordgo.MessageCreate) {
	return func(sess *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore bot messages
		if m.Author.Bot {
			return
		}
		slog.Info("channel message",
			"channel_id", m.ChannelID,
			"author", getDisplayName(m.Author),
			"content", m.Content,
		)
		// No longer adding to cache automatically
		// cache.Add(m.Message)
	}
}

func handleReady(sess *discordgo.Session, r *discordgo.Ready) {
	slog.Info("logged in",
		"username", sess.State.User.Username,
		"discriminator", sess.State.User.Discriminator,
	)
}
