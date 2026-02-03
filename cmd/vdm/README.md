# Virtual Dungeon Master (vdm)

`vdm` is the AI-powered Dungeon Master for the Pathfinder 2nd Edition rules engine. It orchestrates the interaction between human players (via CLI or Discord), a Large Language Model (LLM), and the deterministic rules logic in `vd`.

## Architecture

The `vdm` component is built on a modular architecture:

### 1. Orchestrator (`internal/llm/orchestrator.go`)
The central coordinator. It maintains conversation history, selects the system prompt based on the provider's capabilities, and manages the main generation loop. 
- **Tool Calling**: When a provider supports native tool calls (e.g., Gemini, Groq), the orchestrator maps these to CLI commands (see `vd_status`, `vd_scene_new`, etc.).
- **Marker Handling**: For models that don't support tools, it parses text markers like `>VD_SUGGEST_CMD` from the LLM's response and executes them.

### 2. Recursive Learning Model (RLM) (`internal/llm/rlm/`)
An advanced reasoning layer that allows the DM to "research" rules before responding.
- **Python Sandbox**: Executes code in a restricted Python environment (`py/restricted_python.py`) to search rule files and parse data.
- **Native Tooling**: Uses the `execute_python` tool to bridge the LLM with the Python sandbox.
- **Recursion**: Can trigger nested LLM calls to solve sub-problems during its research phase.

## Operation Modes

### CLI Mode
The default mode. Provides an interactive `>` prompt. It uses `internal/cli` to simulate running the `vd` binary internally, avoiding process overhead.

### Discord mode (`--discord`)
Uses `discordgo` to register slash commands (like `/echo`) and listen for interactions. It leverages the same Orchestrator logic to provide automated DM responses in a guild channel.

## Future Development

When extending `vdm`, consider:
- **Adding Tools**: New deterministic actions should be added to `defineTools()` in `orchestrator.go` and mapped in `mapToolToArgs`.
- **System Prompts**: Prompt logic is split between `orchestrator.go` (standard DM) and `rlm/prompts.go` (researcher).
- **Python Environment**: Security is critical. Any new Python helpers must be whitelisted in `py/restricted_python.py`.
