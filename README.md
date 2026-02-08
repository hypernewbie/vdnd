# uaa/vdnd

A Pathfinder 2nd Edition (PF2E) rules engine and CLI toolkit designed for LLM orchestration.

## Overview

This repository provides a deterministic rules engine for PF2E, allowing AI agents or users to interact with game mechanics through a structured, stateful interface.

## Quick Start

### Prerequisites

- **Go:** 1.25.6 or higher.
- **Dependencies:** All dependencies are managed via Go modules.

### Building

The project consists of two primary binaries:

#### 1. The `vd` CLI Tool
The main interface for executing rules and managing game state.
```bash
go build -o vd ./cmd/vd
```

#### 2. The `vdm` Component
The Virtual Dungeon Master / Discord bot implementation.
```bash
go build -o vdm ./cmd/vdm
```

Alternatively, you can skip the build step and run directly using `go run`:
```bash
go run ./cmd/vd [args]
go run ./cmd/vdm [args]
```

The application uses environment variables for configuration. Create your own `.env` file by copying the provided `.env.example` template and filling in your API keys:

```bash
cp .env.example .env
```

### Configuration Variables

| Variable | Description |
|----------|-------------|
| `DISCORD_TOKEN` | Your Discord bot token (required for `--discord` mode) |
| `DISCORD_CHANNEL_ID` | Optional Discord channel ID to restrict the bot to |
| `GEMINI_API_KEY` | Google Gemini API Key |
| `GROQ_API_KEY` | Groq Cloud API Key |
| `DEEPSEEK_API_KEY` | DeepSeek API Key |
| `GLM_API_KEY` | Zhipu AI GLM API Key |
| `ANTHROPIC_API_KEY` | Anthropic API Key |
| `OLLAMA_URL` | Custom Ollama server URL (defaults to `http://127.0.0.1:11434`) |
| `LLM_PROVIDER` | Default provider (`gemini`, `groq`, `ollama`, `deepseek`, `glm`, `anthropic`) |
| `LLM_MODEL` | Default model name for the chosen provider |

### Running

#### Rules Engine CLI (`vd`)
Execute PF2E rules directly from the command line:
```bash
./vd status
./vd action strike hero orc
# Or using go run:
go run ./cmd/vd status
```

#### Virtual DM CLI (`vdm`)
Run the interactive DM chat mode:
```bash
./vdm
# Or using go run:
go run ./cmd/vdm
```

#### Discord Bot
Start the application in Discord bot mode:
```bash
./vdm --discord
# Or using go run:
go run ./cmd/vdm --discord
```

### Running Tests

To verify the installation and core logic:
```bash
# Run all tests
go test ./...

# Run tests with coverage for the rules engine
go test -cover ./pkg/rules/...
```

## Repository Structure

- `pkg/rules/`: Core PF2E mechanics (pure logic).
- `cmd/vd/`: Entry point for the `vd` CLI tool.
- `cmd/vdm/`: Entry point for the `vdm` component.
- `internal/`: Shared state management, parsing, and CLI handlers.
- `rules/`: Game data and rule definitions.
