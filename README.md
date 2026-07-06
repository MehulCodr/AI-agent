# AI-agent

`AI-agent` is a Go CLI foundation for an AI coding agent.

## Current Status

Day 1 CLI and LLM foundations are in place. Day 2 adds the safe tool system foundation with a registry plus basic echo, current directory, and list files tools. LLM tool calling, file editing, shell execution, and agent reasoning are intentionally left for later branches. Day 3 adds project context scanning.

## Available Commands

```sh
agent version
agent init
agent context
agent chat
agent run "task"
```

## How To Run Locally

From the repository root:

```sh
go run ./cmd version
go run ./cmd init
go run ./cmd context
go run ./cmd run "summarize this repo"
go run ./cmd chat
```

## LLM Configuration

Create a local `.env` file to use the real Gemini provider:

```sh
AI_AGENT_PROVIDER=gemini
AI_AGENT_API_KEY=your-gemini-api-key
AI_AGENT_MODEL=gemini-3.5-flash
# Optional:
# AI_AGENT_BASE_URL=https://generativelanguage.googleapis.com/v1beta
```

If no API key is configured, the CLI uses the local mock provider. Existing `GEMINI_API_KEY`, `GEMINI_MODEL`, and `GEMINI_BASE_URL` names are also supported as fallbacks.
