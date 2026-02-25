# Virtual Dungeon Master (vdm)

`vdm` is the AI-powered Dungeon Master for the Pathfinder 2nd Edition rules engine. It orchestrates the interaction between human players (via CLI or Discord), a Large Language Model (LLM), and the deterministic rules logic in `vd`.

## Architecture

The `vdm` component is built on a modular architecture:

### 1. Orchestrator (`internal/llm/orchestrator.go`)
The central coordinator. It maintains conversation history and manages the main generation loop.
- **Tool Calling**: When a provider supports native tool calls (e.g., Gemini, Groq), the orchestrator delegates tasks to specialized subagents.
- **Marker Handling**: For models that don't support tools, it parses text markers like `>VD_SUGGEST_CMD` from the LLM's response and executes them.

### 2. Subagents (`internal/llm/subagents/`)
Specialized synchronous workers managed by the orchestrator.
- **Research Subagent**: Uses the Python sandbox (`py/restricted_python.py`) to search rules and summarize findings.
- **Execution Subagent**: Uses `vdengine` tools to mutate game state and report concrete outcomes.

## Operation Modes

### CLI Mode
The default mode. Provides an interactive `>` prompt. It uses `internal/cli` to simulate running the `vd` binary internally, avoiding process overhead.

### Discord mode (`--discord`)
Uses `discordgo` to register slash commands (like `/echo`) and listen for interactions. It leverages the same Orchestrator logic to provide automated DM responses in a guild channel.

## Future Development

When extending `vdm`, consider:
- **Adding Tools**: New deterministic actions should be added to `internal/llm/vdengine/engine.go` so the execution subagent can use them.
- **System Prompts**: Prompt logic is split between `orchestrator.go` and subagent prompt configuration in `cmd/vdm/main.go`.
- **Python Environment**: Security is critical. Any new Python helpers must be whitelisted in `py/restricted_python.py`.
